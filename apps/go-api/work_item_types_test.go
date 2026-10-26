package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/platform"
	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
	"github.com/CaliLuke/loom/clue/log"
)

type typeServiceRepository struct {
	releaseInput            store.ReleaseInput
	typeRemovalInput        store.TypeRemovalInput
	hierarchyInput          store.HierarchyInput
	formulaInput            store.FormulaAuthoringInput
	schemaInput             store.ItemSchemaInput
	actor, slug, action, id string
	project                 string
	projectInput            store.ProjectInput
	input                   store.Input
	failure                 error
}

func (r *typeServiceRepository) PreviewTypeRemoval(ctx context.Context, actor, slug string, input store.TypeRemovalInput) (store.TypeRemovalReview, error) {
	r.typeRemovalInput = input
	_, err := r.Catalog(ctx, actor, slug)
	return store.TypeRemovalReview{ID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", Items: 2, Drafts: 1, Projects: 1}, err
}
func (r *typeServiceRepository) CommitTypeRemoval(ctx context.Context, actor, slug string, input store.TypeRemovalInput) (store.Catalog, error) {
	r.typeRemovalInput = input
	return r.Catalog(ctx, actor, slug)
}

func (r *typeServiceRepository) Catalog(_ context.Context, actor, slug string) (store.Catalog, error) {
	r.actor, r.slug = actor, slug
	return store.Catalog{Enabled: true, CanManage: true, Revision: 4, Types: []store.Type{{ID: "type-id", Name: "Task", Default: true, Active: true, Mandatory: true}}}, r.failure
}
func (r *typeServiceRepository) Mutate(ctx context.Context, actor, slug, action, id string, input store.Input) (store.Catalog, error) {
	r.action, r.id, r.input = action, id, input
	return r.Catalog(ctx, actor, slug)
}

func (r *typeServiceRepository) ProjectCatalog(ctx context.Context, actor, slug, project string) (store.ProjectCatalog, error) {
	catalog, err := r.Catalog(ctx, actor, slug)
	r.project = project
	return store.ProjectCatalog{Enabled: true, CanManage: true, Revision: 7, ProjectName: project, DefaultTypeID: "preferred-type", Types: []store.ProjectType{{Type: catalog.Types[0], Imported: true, ProjectActive: true, Available: true}}}, err
}
func (r *typeServiceRepository) MutateProject(ctx context.Context, actor, slug, project string, input store.ProjectInput) (store.ProjectCatalog, error) {
	r.projectInput = input
	return r.ProjectCatalog(ctx, actor, slug, project)
}

type typeServiceSessions struct{}

func (typeServiceSessions) Validate(_ context.Context, credential string) (djangosession.Principal, error) {
	if credential != "valid" {
		return djangosession.Principal{}, errors.New("invalid session")
	}
	return djangosession.Principal{UserID: "actor"}, nil
}
func TestTypeServiceDoesNotDispatchWithoutAuthentication(t *testing.T) {
	repo := &typeServiceRepository{}
	service, err := NewWorkItemTypes(repo, typeServiceSessions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Catalog(context.Background(), &api.CatalogPayload{Slug: "workspace"})
	var failure *api.TypeError
	if !errors.As(err, &failure) || failure.Kind != "status_401" || repo.actor != "" {
		t.Fatalf("unauthenticated dispatch: %v %+v", err, repo)
	}
	_, err = service.(api.Authorizer).APIKeyAuth(context.Background(), "invalid", nil)
	if !errors.As(err, &failure) || failure.Kind != "status_401" {
		t.Fatalf("invalid session accepted: %v", err)
	}
}

func TestPropertyDiscoveryInternalFailureLogsSafeRequestContext(t *testing.T) {
	const requestID = "123e4567-e89b-42d3-a456-426614174000"
	const privateDetail = "sensitive property query detail"

	repository := &typeServiceRepository{failure: errors.New(privateDetail)}
	serviceValue, err := NewWorkItemTypes(repository, typeServiceSessions{})
	if err != nil {
		t.Fatal(err)
	}
	service := serviceValue.(*workItemTypesService)
	var logs bytes.Buffer
	principalCtx, err := service.APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	principal := principalCtx.Value(workItemTypesPrincipal{})
	handler := platform.RequestIDMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		ctx := log.Context(request.Context(),
			log.WithOutputs(log.Output{Writer: &logs, Format: log.FormatJSON}),
			log.WithFunc(func(ctx context.Context) []log.KV {
				id, ok := platform.RequestID(ctx)
				if !ok {
					return nil
				}
				return []log.KV{{K: log.RequestIDKey, V: id}}
			}),
			log.WithDisableBuffering(func(context.Context) bool { return true }),
		)
		ctx = context.WithValue(ctx, workItemTypesPrincipal{}, principal)
		_, readErr := service.DiscoveryProperties(ctx, &api.DiscoveryPropertiesPayload{Slug: "demo"})
		var failure *api.TypeError
		if !errors.As(readErr, &failure) || failure.Kind != "status_500" {
			t.Errorf("error=%T %v, want unchanged generic 500", readErr, readErr)
		}
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/demo/work-item-property-identities/", nil)
	request.Header.Set(platform.RequestIDHeader, requestID)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	var entries []map[string]any
	decoder := json.NewDecoder(&logs)
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("decode log entry: %v; logs=%s", err, logs.String())
		}
		entries = append(entries, entry)
	}
	if len(entries) != 1 {
		t.Fatalf("entries=%#v; logs=%s", entries, logs.String())
	}
	entry := entries[0]
	if entry[log.RequestIDKey] != requestID || entry["error_kind"] != "other" || entry["operation"] != "work_item_types.discovery_properties" {
		t.Fatalf("unsafe or incomplete classification: %#v", entry)
	}
	if strings.Contains(logs.String(), privateDetail) {
		t.Fatalf("log leaked raw repository error: %s", logs.String())
	}
}

func TestTypeServiceCarriesAuthenticatedScopeAndDefinition(t *testing.T) {
	repo := &typeServiceRepository{}
	service, err := NewWorkItemTypes(repo, typeServiceSessions{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	input := &api.TypeInput{Revision: 3, Name: "Bug", Description: "Defect", Icon: "bug", Color: "#123456", Active: true, Mandatory: true}
	for _, operation := range []struct {
		name, id string
		invoke   func() (*api.Catalog, error)
	}{
		{"", "", func() (*api.Catalog, error) { return service.Catalog(ctx, &api.CatalogPayload{Slug: "workspace"}) }},
		{"enable", "", func() (*api.Catalog, error) {
			return service.Enable(ctx, &api.EnablePayload{Slug: "workspace", Data: &api.Revision{Revision: 3}})
		}},
		{"create", "", func() (*api.Catalog, error) {
			return service.Create(ctx, &api.CreatePayload{Slug: "workspace", Data: input})
		}},
		{"update", "type-id", func() (*api.Catalog, error) {
			return service.Update(ctx, &api.UpdatePayload{Slug: "workspace", ID: "type-id", Data: input})
		}},
		{"delete", "type-id", func() (*api.Catalog, error) {
			return service.Delete(ctx, &api.DeletePayload{Slug: "workspace", ID: "type-id", Data: &api.Revision{Revision: 3}})
		}},
	} {
		result, callErr := operation.invoke()
		if callErr != nil {
			t.Fatal(callErr)
		}
		if repo.actor != "actor" || repo.slug != "workspace" || repo.action != operation.name || repo.id != operation.id {
			t.Fatalf("request scope lost: %+v", repo)
		}
		if result.Revision != 4 || len(result.Types) != 1 || !result.Types[0].Default || !result.CanManage {
			t.Fatalf("catalog lost protected metadata: %+v", result)
		}
		if operation.name != "" && repo.input.Revision != 3 {
			t.Fatal("mutation discarded the expected revision")
		}
		if operation.name == "create" || operation.name == "update" {
			if repo.input.Name != "Bug" || repo.input.Description != "Defect" || !repo.input.Active || !repo.input.Mandatory || repo.input.Icon != "bug" || repo.input.Color != "#123456" {
				t.Fatalf("definition lost: %+v", repo.input)
			}
		}
	}
	repo.failure = &store.Error{Status: 409, Message: "Types changed"}
	_, err = service.Catalog(ctx, &api.CatalogPayload{Slug: "workspace"})
	var failure *api.TypeError
	if !errors.As(err, &failure) || failure.Kind != "status_409" || failure.Message != "Types changed" {
		t.Fatalf("conflict lost: %v", err)
	}
	repo.failure = errors.New("private database details")
	_, err = service.Catalog(ctx, &api.CatalogPayload{Slug: "workspace"})
	if !errors.As(err, &failure) || failure.Kind != "status_500" || failure.Message == repo.failure.Error() {
		t.Fatalf("internal failure exposed: %v", err)
	}
}

func TestProjectTypeServiceScopesAvailabilityAndMutations(t *testing.T) {
	repo := &typeServiceRepository{}
	service, err := NewWorkItemTypes(repo, typeServiceSessions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ProjectCatalog(context.Background(), &api.ProjectCatalogPayload{Slug: "workspace", ProjectID: "private"})
	var failure *api.TypeError
	if !errors.As(err, &failure) || failure.Kind != "status_401" || repo.project != "" {
		t.Fatalf("anonymous project dispatch: %v", err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := service.ProjectCatalog(ctx, &api.ProjectCatalogPayload{Slug: "workspace", ProjectID: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.actor != "actor" || repo.slug != "workspace" || repo.project != "project" || catalog.ProjectName != "project" || catalog.DefaultTypeID != "preferred-type" || len(catalog.Types) != 1 || !catalog.Types[0].Available || !catalog.Types[0].Imported || !catalog.Types[0].ProjectActive {
		t.Fatalf("project scope or availability lost: %+v %+v", repo, catalog)
	}
	_, err = service.UpdateProject(ctx, &api.UpdateProjectPayload{Slug: "workspace", ProjectID: "project", Data: &api.ProjectTypeInput{Revision: 7, Action: "import", TypeIds: []string{"type-id"}}})
	if err != nil {
		t.Fatal(err)
	}
	if repo.projectInput.Revision != 7 || repo.projectInput.Action != "import" || len(repo.projectInput.TypeIDs) != 1 || repo.projectInput.TypeIDs[0] != "type-id" {
		t.Fatalf("import command changed: %+v", repo.projectInput)
	}
	repo.failure = &store.Error{Status: 403, Message: "Project administrator required"}
	_, err = service.UpdateProject(ctx, &api.UpdateProjectPayload{Slug: "workspace", ProjectID: "project", Data: &api.ProjectTypeInput{Revision: 7, Action: "remove", TypeIds: []string{"type-id"}}})
	if !errors.As(err, &failure) || failure.Kind != "status_403" {
		t.Fatalf("project denial lost: %v", err)
	}
}

func (r *typeServiceRepository) PropertyCatalog(ctx context.Context, actor, slug string) (store.PropertyCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.PropertyCatalog{Revision: 4, CanManage: true, Properties: []store.PropertyRecord{}, Attachments: []store.PropertyAttachment{}, Projects: []store.PropertyChoice{}, Members: []store.PropertyChoice{}}, err
}
func (r *typeServiceRepository) MutateProperty(ctx context.Context, actor, slug string, input store.PropertyInput) (store.PropertyCatalog, error) {
	return r.PropertyCatalog(ctx, actor, slug)
}
func (r *typeServiceRepository) PreviewPropertyRemoval(ctx context.Context, actor, slug string, input store.PropertyInput) (store.PropertyReview, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.PropertyReview{ID: "review", Items: 2, Types: 1, Projects: 1}, err
}

func (r *typeServiceRepository) ItemSchema(ctx context.Context, actor, slug string, input store.ItemSchemaInput) (store.ItemSchema, error) {
	_, err := r.Catalog(ctx, actor, slug)
	r.schemaInput = input
	return store.ItemSchema{Enabled: true, Revision: 7, Properties: []store.ResolvedProperty{}, Members: []store.ItemReferenceChoice{}, Releases: []store.ItemReferenceChoice{}}, err
}

func (r *typeServiceRepository) PreviewFormula(ctx context.Context, actor, slug string, input store.FormulaAuthoringInput) (store.FormulaAuthoringResult, error) {
	_, err := r.Catalog(ctx, actor, slug)
	r.formulaInput = input
	return store.FormulaAuthoringResult{Valid: true, Active: true, Contexts: []store.PropertyChoice{}, Result: &store.FormulaResult{Kind: "Number", Status: "value", Value: float64(0)}}, err
}

func (r *typeServiceRepository) HierarchyCatalog(ctx context.Context, actor, slug string) (store.HierarchyCatalog, error) {
	catalog, err := r.Catalog(ctx, actor, slug)
	return store.HierarchyCatalog{Catalog: catalog, HierarchyEnabled: true, DefaultLevel: 1}, err
}
func (r *typeServiceRepository) PreviewHierarchy(ctx context.Context, actor, slug string, input store.HierarchyInput) (store.HierarchyReview, error) {
	r.hierarchyInput = input
	_, err := r.Catalog(ctx, actor, slug)
	return store.HierarchyReview{ID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", Links: 1, RestrictedLinks: 1, Pairs: []store.HierarchyPair{}}, err
}
func (r *typeServiceRepository) CommitHierarchy(ctx context.Context, actor, slug string, input store.HierarchyInput) (store.HierarchyCatalog, error) {
	r.hierarchyInput = input
	return r.HierarchyCatalog(ctx, actor, slug)
}

func (r *typeServiceRepository) TypeIdentities(ctx context.Context, actor, slug, project string) (store.TypeIdentityCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	r.project = project
	return store.TypeIdentityCatalog{Enabled: true, Revision: 7, Types: []store.TypeIdentity{{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Name: "Retained", Icon: "task", Color: "#123456", Active: false}}}, err
}

func (r *typeServiceRepository) DiscoveryProperties(ctx context.Context, actor, slug, project string) (store.PropertyDiscoveryCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.PropertyDiscoveryCatalog{Enabled: true, Revision: 7, Fields: []store.DiscoveryProperty{}}, err
}
