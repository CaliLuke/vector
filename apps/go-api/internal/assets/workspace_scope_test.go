package assets

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

type workspaceBoundaryRepository struct {
	assetRepository
	workspaceID           uuid.UUID
	activeWorkspace       error
	workspaceRole         int
	asset                 FileAsset
	workspaceHookApplied  bool
	deleteWorkspaceErr    error
	updatedWorkspaceAsset bool
	deletedWorkspaceAsset bool
	calls                 []string
}

func (repository *workspaceBoundaryRepository) WorkspacePermissionAllowed(context.Context, uuid.UUID, string) (bool, error) {
	repository.calls = append(repository.calls, "WorkspacePermissionAllowed")
	return true, nil
}

func (repository *workspaceBoundaryRepository) ProjectPermissionAllowed(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	repository.calls = append(repository.calls, "ProjectPermissionAllowed")
	return true, nil
}

func (repository *workspaceBoundaryRepository) RelatedWorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	repository.calls = append(repository.calls, "RelatedWorkspaceIDBySlug")
	return repository.workspaceID, nil
}

func (repository *workspaceBoundaryRepository) WorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	repository.calls = append(repository.calls, "WorkspaceIDBySlug")
	return repository.workspaceID, repository.activeWorkspace
}

func (repository *workspaceBoundaryRepository) WorkspaceRole(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	repository.calls = append(repository.calls, "WorkspaceRole")
	return repository.workspaceRole, nil
}

func (repository *workspaceBoundaryRepository) AssetByWorkspaceAndID(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error) {
	repository.calls = append(repository.calls, "AssetByWorkspaceAndID")
	return repository.asset, nil
}

func (repository *workspaceBoundaryRepository) AssetParentVisible(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

func (repository *workspaceBoundaryRepository) UpdateWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, []byte, bool) (bool, error) {
	repository.calls = append(repository.calls, "UpdateWorkspaceAsset")
	repository.updatedWorkspaceAsset = true
	return repository.workspaceHookApplied, nil
}

func (repository *workspaceBoundaryRepository) DeleteWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	repository.calls = append(repository.calls, "DeleteWorkspaceAsset")
	repository.deletedWorkspaceAsset = true
	return repository.deleteWorkspaceErr
}

type workspaceBoundaryCache struct {
	deletes int
}

func (cache *workspaceBoundaryCache) Delete(context.Context, string) error {
	cache.deletes++
	return nil
}

func TestV2WorkspaceScopeUsesRelatedWorkspaceLookupAfterPermission(t *testing.T) {
	workspaceID := uuid.New()
	repository := &workspaceBoundaryRepository{
		workspaceID:     workspaceID,
		activeWorkspace: &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}},
	}
	service := &LifecycleService{assets: &Service{repository: repository}}

	got, err := service.v2WorkspaceID(context.Background(), uuid.New(), "tombstoned")
	if err != nil {
		t.Fatalf("related workspace lookup error = %v", err)
	}
	if got != workspaceID {
		t.Fatalf("related workspace ID = %s, want %s", got, workspaceID)
	}
	if len(repository.calls) != 2 || repository.calls[0] != "WorkspacePermissionAllowed" || repository.calls[1] != "RelatedWorkspaceIDBySlug" {
		t.Fatalf("workspace scope calls = %v, want permission then related lookup", repository.calls)
	}
}

func TestTombstonedWorkspaceLogoHooksPreserveSourceBehavior(t *testing.T) {
	t.Run("PATCH confirms FileAsset and skips missing Workspace cache hook", func(t *testing.T) {
		repository := &workspaceBoundaryRepository{
			workspaceID: uuid.New(),
			asset:       FileAsset{ID: uuid.NewString(), EntityType: "WORKSPACE_LOGO", StorageMetadata: map[string]any{"ContentLength": 68}},
		}
		cache := &workspaceBoundaryCache{}
		service := &LifecycleService{assets: &Service{repository: repository, cache: cache}}
		ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())

		err := service.patchWorkspaceAsset(ctx, "tombstoned", repository.asset.ID, io.NopCloser(strings.NewReader(`{"attributes":{"review":"updated"}}`)))
		if err != nil {
			t.Fatalf("PATCH tombstoned workspace logo error = %v", err)
		}
		if !repository.updatedWorkspaceAsset {
			t.Fatal("PATCH did not confirm the FileAsset")
		}
		if cache.deletes != 0 {
			t.Fatalf("workspace cache invalidations = %d, want none when active Workspace hook is skipped", cache.deletes)
		}
	})

	t.Run("DELETE stops on missing active Workspace before cache invalidation", func(t *testing.T) {
		workspaceErr := &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
		repository := &workspaceBoundaryRepository{
			workspaceID:        uuid.New(),
			asset:              FileAsset{ID: uuid.NewString(), EntityType: "WORKSPACE_LOGO", StorageMetadata: map[string]any{"ContentLength": 68}},
			deleteWorkspaceErr: workspaceErr,
		}
		cache := &workspaceBoundaryCache{}
		service := &LifecycleService{assets: &Service{repository: repository, cache: cache}}
		ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())

		err := service.deleteWorkspaceAsset(ctx, "tombstoned", repository.asset.ID)
		var httpErr *djangoapi.HTTPError
		if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 || httpErr.Body["error"] != "The required object does not exist." {
			t.Fatalf("DELETE tombstoned workspace logo error = %v, want source required-object 404", err)
		}
		if !repository.deletedWorkspaceAsset {
			t.Fatal("DELETE did not reach the FileAsset entity hook")
		}
		if cache.deletes != 0 {
			t.Fatalf("workspace cache invalidations = %d, want none after hook failure", cache.deletes)
		}
	})
}

func TestLegacyIssueAttachmentMissingFileChecksActiveWorkspaceFirst(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("size", "not-a-number"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	workspaceErr := &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	repository := &workspaceBoundaryRepository{workspaceID: uuid.New(), activeWorkspace: workspaceErr}
	service := &LifecycleService{assets: &Service{repository: repository}}
	userID, projectID, issueID := uuid.New(), uuid.New(), uuid.New()
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	ctx = WithRequestContentType(ctx, writer.FormDataContentType())

	_, err := service.createLegacyIssueAttachment(ctx, "tombstoned", projectID.String(), issueID.String(), io.NopCloser(bytes.NewReader(body.Bytes())))
	require.Same(t, workspaceErr, err, "active workspace lookup must precede missing-file validation")
	wantCalls := []string{"ProjectPermissionAllowed", "RelatedWorkspaceIDBySlug", "WorkspaceIDBySlug"}
	if len(repository.calls) != len(wantCalls) {
		t.Fatalf("repository calls = %v, want %v", repository.calls, wantCalls)
	}
	for index, want := range wantCalls {
		if repository.calls[index] != want {
			t.Fatalf("repository calls = %v, want %v", repository.calls, wantCalls)
		}
	}
}

func TestWorkspaceUploadAndDuplicateCheckActiveWorkspaceAfterSourceValidation(t *testing.T) {
	projectID, entityID := uuid.New(), uuid.New()
	activeError := &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}}
	tests := []struct {
		name       string
		body       string
		role       int
		call       func(*LifecycleService, context.Context, string) error
		wantStatus int
		wantCalls  []string
	}{
		{
			name: "workspace upload validates entity before active lookup",
			body: `{"name":"file.png","type":"image/png","size":68,"entity_type":"INVALID"}`,
			call: func(service *LifecycleService, ctx context.Context, _ string) error {
				_, err := service.createV2WorkspaceUpload(ctx, "tombstoned", io.NopCloser(strings.NewReader(`{"name":"file.png","type":"image/png","size":68,"entity_type":"INVALID"}`)))
				return err
			},
			wantStatus: 400,
			wantCalls:  []string{"WorkspacePermissionAllowed"},
		},
		{
			name: "nonadmin logo validation precedes active lookup",
			body: `{"name":"logo.png","type":"image/png","size":68,"entity_type":"WORKSPACE_LOGO"}`,
			role: 15,
			call: func(service *LifecycleService, ctx context.Context, _ string) error {
				_, err := service.createV2WorkspaceUpload(ctx, "tombstoned", io.NopCloser(strings.NewReader(`{"name":"logo.png","type":"image/png","size":68,"entity_type":"WORKSPACE_LOGO"}`)))
				return err
			},
			wantStatus: 403,
			wantCalls:  []string{"WorkspacePermissionAllowed", "RelatedWorkspaceIDBySlug", "WorkspaceRole"},
		},
		{
			name: "valid workspace upload reaches active lookup",
			body: `{"name":"file.png","type":"image/png","size":68,"entity_type":"ISSUE_ATTACHMENT","entity_identifier":"` + entityID.String() + `"}`,
			call: func(service *LifecycleService, ctx context.Context, body string) error {
				_, err := service.createV2WorkspaceUpload(ctx, "tombstoned", io.NopCloser(strings.NewReader(body)))
				return err
			},
			wantStatus: 404,
			wantCalls:  []string{"WorkspacePermissionAllowed", "WorkspaceIDBySlug"},
		},
		{
			name: "project upload validates entity before active lookup",
			body: `{"name":"file.png","type":"image/png","size":68,"entity_type":"INVALID"}`,
			call: func(service *LifecycleService, ctx context.Context, body string) error {
				_, err := service.createV2ProjectUpload(ctx, "tombstoned", projectID.String(), io.NopCloser(strings.NewReader(body)))
				return err
			},
			wantStatus: 400,
			wantCalls:  []string{"ProjectPermissionAllowed"},
		},
		{
			name: "valid project upload reaches active lookup",
			body: `{"name":"file.png","type":"image/png","size":68,"entity_type":"ISSUE_ATTACHMENT","entity_identifier":"` + entityID.String() + `"}`,
			call: func(service *LifecycleService, ctx context.Context, body string) error {
				_, err := service.createV2ProjectUpload(ctx, "tombstoned", projectID.String(), io.NopCloser(strings.NewReader(body)))
				return err
			},
			wantStatus: 404,
			wantCalls:  []string{"ProjectPermissionAllowed", "WorkspaceIDBySlug"},
		},
		{
			name: "duplicate validates missing entity type before active lookup",
			body: `{}`,
			call: func(service *LifecycleService, ctx context.Context, body string) error {
				_, err := service.duplicateWorkspaceAsset(ctx, "tombstoned", uuid.NewString(), io.NopCloser(strings.NewReader(body)))
				return err
			},
			wantStatus: 400,
			wantCalls:  []string{"WorkspacePermissionAllowed"},
		},
		{
			name: "valid duplicate reaches active lookup",
			body: `{"entity_type":"ISSUE_ATTACHMENT","entity_id":"` + entityID.String() + `"}`,
			call: func(service *LifecycleService, ctx context.Context, body string) error {
				_, err := service.duplicateWorkspaceAsset(ctx, "tombstoned", uuid.NewString(), io.NopCloser(strings.NewReader(body)))
				return err
			},
			wantStatus: 404,
			wantCalls:  []string{"WorkspacePermissionAllowed", "WorkspaceIDBySlug"},
		},
		{
			name: "attachment upload validates MIME before active lookup",
			body: `{"name":"file.bin","type":"application/x-invalid","size":68}`,
			call: func(service *LifecycleService, ctx context.Context, body string) error {
				_, err := service.createIssueAttachment(ctx, "tombstoned", projectID.String(), entityID.String(), io.NopCloser(strings.NewReader(body)))
				return err
			},
			wantStatus: 400,
			wantCalls:  []string{"ProjectPermissionAllowed", "RelatedWorkspaceIDBySlug"},
		},
		{
			name: "valid attachment upload reaches active lookup",
			body: `{"name":"file.png","type":"image/png","size":68}`,
			call: func(service *LifecycleService, ctx context.Context, body string) error {
				_, err := service.createIssueAttachment(ctx, "tombstoned", projectID.String(), entityID.String(), io.NopCloser(strings.NewReader(body)))
				return err
			},
			wantStatus: 404,
			wantCalls:  []string{"ProjectPermissionAllowed", "RelatedWorkspaceIDBySlug", "WorkspaceIDBySlug"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &workspaceBoundaryRepository{
				workspaceID:     uuid.New(),
				activeWorkspace: activeError,
				workspaceRole:   test.role,
			}
			storage, err := NewStorage(StorageConfig{AccessKeyID: "test-access-key", SecretAccessKey: "test-secret-key", Bucket: "assets-test", Region: "us-east-1"})
			if err != nil {
				t.Fatal(err)
			}
			service := &LifecycleService{assets: &Service{repository: repository, storage: storage, fileLimit: 5 << 20}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())

			err = test.call(service, ctx, test.body)
			var httpErr *djangoapi.HTTPError
			if err == nil || !errors.As(err, &httpErr) || httpErr.Status != test.wantStatus {
				t.Fatalf("request error = %v, want HTTP %d", err, test.wantStatus)
			}
			if len(repository.calls) != len(test.wantCalls) {
				t.Fatalf("repository calls = %v, want %v", repository.calls, test.wantCalls)
			}
			for index, want := range test.wantCalls {
				if repository.calls[index] != want {
					t.Fatalf("repository calls = %v, want %v", repository.calls, test.wantCalls)
				}
			}
		})
	}
}
