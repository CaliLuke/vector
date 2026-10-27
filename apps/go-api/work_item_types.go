package systemapi

import (
	"context"
	"errors"
	"strconv"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
	"github.com/CaliLuke/loom/security"
)

type workItemTypesRepository interface {
	DiscoveryProperties(context.Context, string, string, string) (store.PropertyDiscoveryCatalog, error)
	TypeIdentities(context.Context, string, string, string) (store.TypeIdentityCatalog, error)
	ReleaseCatalog(context.Context, string, string, string) (store.ReleaseCatalog, error)
	MutateReleaseCatalog(context.Context, string, string, string, store.ReleaseInput) (store.ReleaseCatalog, error)
	PreviewTypeRemoval(context.Context, string, string, store.TypeRemovalInput) (store.TypeRemovalReview, error)
	CommitTypeRemoval(context.Context, string, string, store.TypeRemovalInput) (store.Catalog, error)
	HierarchyCatalog(context.Context, string, string) (store.HierarchyCatalog, error)
	PreviewHierarchy(context.Context, string, string, store.HierarchyInput) (store.HierarchyReview, error)
	CommitHierarchy(context.Context, string, string, store.HierarchyInput) (store.HierarchyCatalog, error)
	PreviewFormula(context.Context, string, string, store.FormulaAuthoringInput) (store.FormulaAuthoringResult, error)
	ItemSchema(context.Context, string, string, store.ItemSchemaInput) (store.ItemSchema, error)
	PropertyCatalog(context.Context, string, string) (store.PropertyCatalog, error)
	MutateProperty(context.Context, string, string, store.PropertyInput) (store.PropertyCatalog, error)
	PreviewPropertyRemoval(context.Context, string, string, store.PropertyInput) (store.PropertyReview, error)
	Catalog(context.Context, string, string) (store.Catalog, error)
	Mutate(context.Context, string, string, string, string, store.Input) (store.Catalog, error)
	ProjectCatalog(context.Context, string, string, string) (store.ProjectCatalog, error)
	MutateProject(context.Context, string, string, string, store.ProjectInput) (store.ProjectCatalog, error)
}
type workItemTypesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type workItemTypesService struct {
	repository workItemTypesRepository
	sessions   workItemTypesSessions
	itemEdits  workItemEditRepository
}
type workItemTypesPrincipal struct{}

func NewWorkItemTypes(repository workItemTypesRepository, sessions workItemTypesSessions, itemEdits ...workItemEditRepository) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, errors.New("work item types repository and sessions are required")
	}
	if len(itemEdits) > 1 {
		return nil, errors.New("at most one work item edit repository may be configured")
	}
	var edits workItemEditRepository
	if len(itemEdits) == 1 {
		edits = itemEdits[0]
	}
	return &workItemTypesService{repository: repository, sessions: sessions, itemEdits: edits}, nil
}
func (s *workItemTypesService) APIKeyAuth(ctx context.Context, credential string, _ *security.APIKeyScheme) (context.Context, error) {
	if user, timezone, inspected := readServiceSessionPrincipal(ctx); inspected {
		if user == "" {
			return ctx, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
		}
		ctx = context.WithValue(ctx, workItemTypesTimezone{}, timezone)
		return context.WithValue(ctx, workItemTypesPrincipal{}, user), nil
	}
	if credential == "" {
		credential = sessionCredentialFromContext(ctx)
	}
	principal, err := s.sessions.Validate(ctx, credential)
	if err != nil || principal.UserID == "" {
		return ctx, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	ctx = context.WithValue(ctx, workItemTypesTimezone{}, principal.UserTimezone)
	return context.WithValue(ctx, workItemTypesPrincipal{}, principal.UserID), nil
}
func (s *workItemTypesService) Catalog(ctx context.Context, p *api.CatalogPayload) (*api.Catalog, error) {
	return s.run(ctx, p.Slug, "", "", store.Input{})
}
func (s *workItemTypesService) Enable(ctx context.Context, p *api.EnablePayload) (*api.Catalog, error) {
	return s.run(ctx, p.Slug, "enable", "", store.Input{Revision: p.Data.Revision})
}
func (s *workItemTypesService) Create(ctx context.Context, p *api.CreatePayload) (*api.Catalog, error) {
	return s.run(ctx, p.Slug, "create", "", typeInput(p.Data))
}
func (s *workItemTypesService) Update(ctx context.Context, p *api.UpdatePayload) (*api.Catalog, error) {
	return s.run(ctx, p.Slug, "update", p.ID, typeInput(p.Data))
}
func (s *workItemTypesService) Delete(ctx context.Context, p *api.DeletePayload) (*api.Catalog, error) {
	return s.run(ctx, p.Slug, "delete", p.ID, store.Input{Revision: p.Data.Revision})
}
func typeInput(p *api.TypeInput) store.Input {
	return store.Input{Revision: p.Revision, Name: p.Name, Description: p.Description, Icon: p.Icon, Color: p.Color, Active: p.Active, Mandatory: p.Mandatory}
}
func (s *workItemTypesService) run(ctx context.Context, slug, action, id string, input store.Input) (*api.Catalog, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	var result store.Catalog
	var err error
	if action == "" {
		result, err = s.repository.Catalog(ctx, actor, slug)
	} else {
		result, err = s.repository.Mutate(ctx, actor, slug, action, id, input)
	}
	if err != nil {
		return nil, typeServiceError(ctx, "work_item_types."+typeOperation(action), err)
	}
	out := &api.Catalog{Enabled: result.Enabled, CanManage: result.CanManage, Revision: result.Revision, ProjectCount: result.ProjectCount, Types: make([]*api.WorkItemType, 0, len(result.Types))}
	for _, item := range result.Types {
		out.Types = append(out.Types, &api.WorkItemType{ID: item.ID, Name: item.Name, Description: item.Description, Icon: item.Icon, Color: item.Color, Active: item.Active, Default: item.Default, Mandatory: item.Mandatory, Level: item.Level, Usage: item.Usage})
	}
	return out, nil
}

func typeServiceError(ctx context.Context, operation string, err error) error {
	var failure *store.Error
	if errors.As(err, &failure) {
		if failure.Status >= 500 {
			workItemLogError(ctx, operation, "execute", err)
		}
		out := &api.TypeError{Kind: "status_" + strconv.Itoa(failure.Status), Message: failure.Message}
		if failure.ReviewAgain {
			out.ReviewAgain = &failure.ReviewAgain
		}
		return out
	}
	workItemLogError(ctx, operation, "execute", err)
	return &api.TypeError{Kind: "status_500", Message: "Work item types could not be saved or loaded. Please try again."}
}

// Only canonical command names enter telemetry; never echo arbitrary input.
func typeOperation(action string) string {
	switch action {
	case "":
		return "catalog"
	case "create", "update", "delete", "enable", "attach", "detach", "set_default", "import", "remove":
		return action
	default:
		return "unknown"
	}
}
