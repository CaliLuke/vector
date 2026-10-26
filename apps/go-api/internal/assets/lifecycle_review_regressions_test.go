package assets

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	api "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

type reviewUserAssetRepository struct {
	assetRepository
	asset               FileAsset
	updatedAttributes   []byte
	updatedUploaded     bool
	updatedLegacyAttrs  []byte
	publicWorkspaceID   uuid.UUID
	publicProjectID     uuid.UUID
	publicWorkspaceSlug string
	publicCreateCalled  bool
	workspaceAssetRead  bool
	workspaceID         uuid.UUID
	projectExists       bool
	projectUploadCalled bool
	createdProjectID    uuid.UUID
	createdEntityID     string
	duplicateLookup     bool
}

func (repository *reviewUserAssetRepository) AssetParentVisible(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

func (repository *reviewUserAssetRepository) UserAssetByID(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error) {
	return repository.asset, nil
}

func (repository *reviewUserAssetRepository) UpdateUserAsset(_ context.Context, _, _ uuid.UUID, attributes []byte, uploaded, _ bool) error {
	repository.updatedAttributes = append([]byte(nil), attributes...)
	repository.updatedUploaded = uploaded
	return nil
}

func (repository *reviewUserAssetRepository) UpdateLegacyUserAsset(_ context.Context, _, _ uuid.UUID, attributes []byte, _ bool) error {
	repository.updatedLegacyAttrs = append([]byte(nil), attributes...)
	return nil
}

func (repository *reviewUserAssetRepository) PublicProjectBoard(context.Context, string, bool) (uuid.UUID, uuid.UUID, string, bool, error) {
	return repository.publicWorkspaceID, repository.publicProjectID, repository.publicWorkspaceSlug, true, nil
}

func (repository *reviewUserAssetRepository) AuthorizedWorkspaceID(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return repository.publicWorkspaceID, nil
}

func (repository *reviewUserAssetRepository) AssetByWorkspaceAndID(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error) {
	repository.workspaceAssetRead = true
	return repository.asset, nil
}

func (repository *reviewUserAssetRepository) WorkspacePermissionAllowed(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}

func (repository *reviewUserAssetRepository) WorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	return repository.workspaceID, nil
}

func (repository *reviewUserAssetRepository) ProjectPermissionAllowed(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return true, nil
}

func (repository *reviewUserAssetRepository) ProjectBelongsToWorkspace(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return repository.projectExists, nil
}

func (repository *reviewUserAssetRepository) CreateWorkspaceEntityUpload(_ context.Context, _, _, _, projectID uuid.UUID, _ string, _ []byte, _ int64, _ string, entityID string, _ bool) error {
	repository.projectUploadCalled = true
	repository.createdProjectID = projectID
	repository.createdEntityID = entityID
	return nil
}

func (repository *reviewUserAssetRepository) AssetForDuplicate(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, error) {
	repository.duplicateLookup = true
	return repository.asset, nil
}

func (repository *reviewUserAssetRepository) CreatePublicUpload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, []byte, int64, string, string) error {
	repository.publicCreateCalled = true
	return nil
}

func TestUserAssetConfirmationPreservesJSONArrayAndScalarAttributes(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want string
	}{
		{name: "absent preserves current object", body: `{}`, want: `{"name":"original.png"}`},
		{name: "explicit null", body: `{"attributes":null}`, want: `null`},
		{name: "empty object", body: `{"attributes":{}}`, want: `{}`},
		{name: "array", body: `{"attributes":[]}`, want: `[]`},
		{name: "scalar string", body: `{"attributes":"client-value"}`, want: `"client-value"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			userID, assetID := uuid.New(), uuid.New()
			repository := &reviewUserAssetRepository{asset: FileAsset{
				ID:              assetID.String(),
				EntityType:      "ISSUE_DESCRIPTION",
				StorageMetadata: map[string]any{"ContentLength": 4},
				Attributes:      map[string]any{"name": "original.png"},
			}}
			service := &LifecycleService{assets: &Service{repository: repository}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
			body := func() io.ReadCloser { return io.NopCloser(strings.NewReader(test.body)) }

			if err := service.confirmUserAssetUpload(ctx, assetID.String(), body()); err != nil {
				t.Fatalf("V2 user asset confirmation: %v", err)
			}
			if !repository.updatedUploaded || string(repository.updatedAttributes) != test.want {
				t.Fatalf("V2 persisted uploaded=%t attributes=%s, want true and %s", repository.updatedUploaded, repository.updatedAttributes, test.want)
			}

			if err := service.confirmLegacyUserAssetUpload(ctx, assetID.String(), body()); err != nil {
				t.Fatalf("API user asset confirmation: %v", err)
			}
			if string(repository.updatedLegacyAttrs) != test.want {
				t.Fatalf("API persisted attributes=%s, want %s", repository.updatedLegacyAttrs, test.want)
			}
		})
	}
}

func TestPublicAssetUploadMapsMalformedSizeToSource400(t *testing.T) {
	for _, size := range []string{`"not-a-number"`, `null`} {
		t.Run(size, func(t *testing.T) {
			userID := uuid.New()
			repository := &reviewUserAssetRepository{publicWorkspaceID: uuid.New(), publicProjectID: uuid.New(), publicWorkspaceSlug: "workspace"}
			service := &LifecycleService{assets: &Service{repository: repository, fileLimit: 5 << 20}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
			body := io.NopCloser(strings.NewReader(`{"name":"image.png","type":"image/png","size":` + size + `,"entity_type":"USER_AVATAR"}`))

			_, err := service.createPublicAsset(ctx, "anchor", body)
			var httpErr *djangoapi.HTTPError
			if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 400 {
				t.Fatalf("malformed public upload size error = %v, want HTTP 400", err)
			}
			if !reflect.DeepEqual(httpErr.Body, map[string]any{"error": "Invalid size.", "status": false}) {
				t.Fatalf("malformed public upload size body = %#v", httpErr.Body)
			}
			if repository.publicCreateCalled {
				t.Fatal("malformed size reached FileAsset creation")
			}
		})
	}
}

func TestGenericAssetGetRejectsLegacyLogicalDeletion(t *testing.T) {
	userID, assetID := uuid.New(), uuid.New()
	repository := &reviewUserAssetRepository{
		publicWorkspaceID: uuid.New(),
		asset: FileAsset{
			ID:         assetID.String(),
			Asset:      "workspace/deleted.png",
			IsUploaded: true,
			IsDeleted:  true,
			Attributes: map[string]any{"name": "deleted.png", "type": "image/png"},
		},
	}
	service := &LifecycleService{assets: &Service{repository: repository}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

	_, err := service.GetAPIV1WorkspacesBySlugAssetsByAssetID(ctx, &api.GetAPIV1WorkspacesBySlugAssetsByAssetIDPayload{
		Slug: "workspace", AssetID: assetID.String(),
	})
	if err == nil {
		t.Fatal("GET returned a signed URL for an is_deleted asset")
	}
	if !repository.workspaceAssetRead {
		t.Fatal("GET did not check the source asset row")
	}
}

func TestFileAssetAttributeReadPreservesLargeJSONIntegers(t *testing.T) {
	const encoded = `{"size":9007199254740993}`
	attributes, err := decodeFileAssetAttributes([]byte(encoded))
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(attributes)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != encoded {
		t.Fatalf("FileAsset attribute read/write = %s, want %s", got, encoded)
	}
}

func TestProjectCoverUploadRequiresMatchingEntityIdentifierBeforeCreation(t *testing.T) {
	projectID := uuid.New()
	for _, test := range []struct {
		name  string
		attrs string
	}{
		{name: "missing identifier with matching alias", attrs: `"entity_id":"` + projectID.String() + `"`},
		{name: "null identifier with matching alias", attrs: `"entity_identifier":null,"entity_id":"` + projectID.String() + `"`},
		{name: "empty identifier with matching alias", attrs: `"entity_identifier":"","entity_id":"` + projectID.String() + `"`},
		{name: "mismatched identifier and matching alias", attrs: `"entity_identifier":"` + uuid.NewString() + `","entity_id":"` + projectID.String() + `"`},
		{name: "uppercase identifier is not an exact path match", attrs: `"entity_identifier":"` + strings.ToUpper(projectID.String()) + `"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			userID := uuid.New()
			repository := &reviewUserAssetRepository{workspaceID: uuid.New()}
			service := &LifecycleService{assets: &Service{repository: repository, fileLimit: 5 << 20}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
			body := io.NopCloser(strings.NewReader(`{"name":"cover.png","type":"image/png","size":68,"entity_type":"PROJECT_COVER",` + test.attrs + `}`))

			_, err := service.createV2ProjectUpload(ctx, "workspace", projectID.String(), body)
			assertProjectCoverMismatch(t, err)
			if repository.projectUploadCalled {
				t.Fatal("mismatched project cover reached FileAsset creation")
			}
		})
	}
}

func TestProjectCoverUploadUsesEntityIdentifierAndKeepsRouteProject(t *testing.T) {
	userID, workspaceID, projectID := uuid.New(), uuid.New(), uuid.New()
	repository := &reviewUserAssetRepository{workspaceID: workspaceID, projectExists: true}
	storage, err := NewStorage(StorageConfig{
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Bucket:          "assets-test",
		Region:          "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	service := &LifecycleService{assets: &Service{repository: repository, storage: storage, fileLimit: 5 << 20}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	body := io.NopCloser(strings.NewReader(`{"name":"cover.png","type":"image/png","size":68,"entity_type":"PROJECT_COVER","entity_identifier":"` + projectID.String() + `","entity_id":"` + uuid.NewString() + `"}`))

	if _, err := service.createV2ProjectUpload(ctx, "workspace", projectID.String(), body); err != nil {
		t.Fatalf("matching route project cover upload: %v", err)
	}
	if !repository.projectUploadCalled || repository.createdProjectID != projectID || repository.createdEntityID != projectID.String() {
		t.Fatalf("created project=%s entity=%q called=%t, want path project %s", repository.createdProjectID, repository.createdEntityID, repository.projectUploadCalled, projectID)
	}
}

func TestProjectCoverUploadRejectsMissingRouteProjectBeforeCreation(t *testing.T) {
	userID, projectID := uuid.New(), uuid.New()
	repository := &reviewUserAssetRepository{workspaceID: uuid.New()}
	service := &LifecycleService{assets: &Service{repository: repository, fileLimit: 5 << 20}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	body := io.NopCloser(strings.NewReader(`{"name":"cover.png","type":"image/png","size":68,"entity_type":"PROJECT_COVER","entity_identifier":"` + projectID.String() + `"}`))

	_, err := service.createV2ProjectUpload(ctx, "workspace", projectID.String(), body)
	var httpErr *djangoapi.HTTPError
	if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 || httpErr.Body["error"] != "Project not found" || httpErr.Body["status"] != false {
		t.Fatalf("missing route project upload error = %v, want source 404", err)
	}
	if repository.projectUploadCalled {
		t.Fatal("missing route project reached FileAsset creation")
	}
}

func TestProjectCoverDuplicateRequiresEntityIDBeforeSourceLookup(t *testing.T) {
	projectID := uuid.New()
	for _, test := range []struct {
		name   string
		fields string
	}{
		{name: "missing entity id with identifier alias", fields: `"entity_identifier":"` + projectID.String() + `"`},
		{name: "null entity id with identifier alias", fields: `"entity_id":null,"entity_identifier":"` + projectID.String() + `"`},
		{name: "empty entity id with identifier alias", fields: `"entity_id":"","entity_identifier":"` + projectID.String() + `"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			userID := uuid.New()
			repository := &reviewUserAssetRepository{workspaceID: uuid.New(), projectExists: true}
			service := &LifecycleService{assets: &Service{repository: repository}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
			body := io.NopCloser(strings.NewReader(`{"entity_type":"PROJECT_COVER",` + test.fields + `}`))

			_, err := service.duplicateWorkspaceAsset(ctx, "workspace", uuid.NewString(), body)
			assertProjectCoverMismatch(t, err)
			if repository.duplicateLookup {
				t.Fatal("missing project-cover entity id reached source asset lookup")
			}
		})
	}
}

func TestProjectCoverDuplicateValidatesEffectiveProjectBeforeSourceLookup(t *testing.T) {
	workspaceID, projectID := uuid.New(), uuid.New()
	for _, test := range []struct {
		name       string
		fields     string
		projectOK  bool
		wantStatus int
		wantError  string
	}{
		{name: "foreign effective entity project", fields: `"entity_id":"` + projectID.String() + `"`, wantStatus: 404, wantError: "Project not found"},
		{name: "soft-deleted effective entity project", fields: `"entity_id":"` + projectID.String() + `"`, wantStatus: 404, wantError: "Project not found"},
		{name: "explicit foreign project precedes missing entity", fields: `"project_id":"` + projectID.String() + `"`, wantStatus: 404, wantError: "Project not found"},
		{name: "entity id overrides no alias", fields: `"project_id":"` + projectID.String() + `","entity_id":"` + uuid.NewString() + `","entity_identifier":"` + projectID.String() + `"`, projectOK: true, wantStatus: 400, wantError: "Entity project must match the requested project."},
		{name: "uppercase entity id is not an exact explicit project match", fields: `"project_id":"` + projectID.String() + `","entity_id":"` + strings.ToUpper(projectID.String()) + `"`, projectOK: true, wantStatus: 400, wantError: "Entity project must match the requested project."},
		{name: "uppercase explicit project is not an exact entity match", fields: `"project_id":"` + strings.ToUpper(projectID.String()) + `","entity_id":"` + projectID.String() + `"`, projectOK: true, wantStatus: 400, wantError: "Entity project must match the requested project."},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &reviewUserAssetRepository{workspaceID: workspaceID, projectExists: test.projectOK}
			service := &LifecycleService{assets: &Service{repository: repository}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())
			body := io.NopCloser(strings.NewReader(`{"entity_type":"PROJECT_COVER",` + test.fields + `}`))

			_, err := service.duplicateWorkspaceAsset(ctx, "workspace", uuid.NewString(), body)
			var httpErr *djangoapi.HTTPError
			if err == nil || !errors.As(err, &httpErr) || httpErr.Status != test.wantStatus {
				t.Fatalf("foreign project duplicate error = %v, want HTTP %d", err, test.wantStatus)
			}
			if httpErr.Body["error"] != test.wantError {
				t.Fatalf("foreign project duplicate body = %#v", httpErr.Body)
			}
			if repository.duplicateLookup {
				t.Fatal("foreign project duplicate reached source asset lookup")
			}
		})
	}
}

func TestDuplicateRejectsSoftDeletedExplicitProjectBeforeSourceLookup(t *testing.T) {
	projectID := uuid.New()
	repository := &reviewUserAssetRepository{workspaceID: uuid.New(), projectExists: false}
	service := &LifecycleService{assets: &Service{repository: repository}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())
	body := io.NopCloser(strings.NewReader(`{"entity_type":"WORKSPACE_LOGO","project_id":"` + projectID.String() + `"}`))

	_, err := service.duplicateWorkspaceAsset(ctx, "workspace", uuid.NewString(), body)
	var httpErr *djangoapi.HTTPError
	if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 || httpErr.Body["error"] != "Project not found" {
		t.Fatalf("soft-deleted explicit destination error = %v, want 404 Project not found", err)
	}
	if repository.duplicateLookup {
		t.Fatal("soft-deleted explicit destination reached source lookup or copy")
	}
}

func TestProjectCoverDuplicateAcceptsEntityOnlyProjectInWorkspace(t *testing.T) {
	workspaceID, projectID := uuid.New(), uuid.New()
	repository := &reviewUserAssetRepository{
		workspaceID:   workspaceID,
		projectExists: true,
		asset: FileAsset{
			ID:          uuid.NewString(),
			WorkspaceID: &workspaceID,
		},
	}
	service := &LifecycleService{assets: &Service{repository: repository}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())
	body := io.NopCloser(strings.NewReader(`{"entity_type":"PROJECT_COVER","entity_id":"` + projectID.String() + `"}`))

	_, err := service.duplicateWorkspaceAsset(ctx, "workspace", uuid.NewString(), body)
	var httpErr *djangoapi.HTTPError
	if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 || httpErr.Body["error"] != "Asset not found" {
		t.Fatalf("entity-only project duplicate error = %v, want source lookup after project validation", err)
	}
	if !repository.duplicateLookup {
		t.Fatal("valid same-workspace entity-only project did not reach source lookup")
	}
}

func assertProjectCoverMismatch(t *testing.T, err error) {
	t.Helper()
	var httpErr *djangoapi.HTTPError
	if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 400 {
		t.Fatalf("project cover validation error = %v, want HTTP 400", err)
	}
	want := map[string]any{"error": "Entity project must match the requested project.", "status": false}
	if !reflect.DeepEqual(httpErr.Body, want) {
		t.Fatalf("project cover validation body = %#v, want %#v", httpErr.Body, want)
	}
}
