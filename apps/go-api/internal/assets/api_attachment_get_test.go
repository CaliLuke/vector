package assets

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

type apiAttachmentGetRepository struct {
	assetRepository
	workspaceID           uuid.UUID
	asset                 FileAsset
	assetReadErr          error
	attachmentReadErr     error
	activeWorkspaceErr    error
	apiWorkspaceLookupErr error
	member                bool
	issueFound            bool
	issueMember           bool
	issueCreator          bool
	assetUpdated          bool
	assetDeleted          bool
	issueAssetUpdated     bool
	issueAssetDeleted     bool
	issueRows             []map[string]any
	calls                 []string
}

func (repository *apiAttachmentGetRepository) WorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	repository.calls = append(repository.calls, "WorkspaceIDBySlug")
	return repository.workspaceID, repository.activeWorkspaceErr
}

func (repository *apiAttachmentGetRepository) RelatedWorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	repository.calls = append(repository.calls, "RelatedWorkspaceIDBySlug")
	return repository.workspaceID, repository.apiWorkspaceLookupErr
}

func (repository *apiAttachmentGetRepository) ProjectMemberActive(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	repository.calls = append(repository.calls, "ProjectMemberActive")
	return repository.member, nil
}

func (repository *apiAttachmentGetRepository) ProjectMemberActiveByProject(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	repository.calls = append(repository.calls, "ProjectMemberActiveByProject")
	return repository.member, nil
}

func (repository *apiAttachmentGetRepository) AssetByProjectAndID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, error) {
	repository.calls = append(repository.calls, "AssetByProjectAndID")
	return repository.asset, repository.assetReadErr
}

func (repository *apiAttachmentGetRepository) IssuePermission(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (bool, bool, bool, error) {
	repository.calls = append(repository.calls, "IssuePermission")
	return repository.issueFound, repository.issueMember, repository.issueCreator, nil
}

func (repository *apiAttachmentGetRepository) IssueAttachment(_ context.Context, workspaceID, projectID, issueID, assetID uuid.UUID) (FileAsset, map[string]any, error) {
	repository.calls = append(repository.calls, "IssueAttachment")
	if repository.attachmentReadErr != nil {
		return FileAsset{}, nil, repository.attachmentReadErr
	}
	if repository.asset.WorkspaceID != nil && *repository.asset.WorkspaceID != workspaceID ||
		repository.asset.ProjectID != nil && *repository.asset.ProjectID != projectID ||
		repository.asset.IssueID != nil && *repository.asset.IssueID != issueID {
		return FileAsset{}, nil, &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	}
	row := map[string]any{"id": assetID.String(), "issue": issueID.String(), "project": projectID.String(), "entity_type": "ISSUE_ATTACHMENT", "is_uploaded": repository.asset.IsUploaded}
	return repository.asset, row, nil
}

func (repository *apiAttachmentGetRepository) ProjectIssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, map[string]any, error) {
	repository.calls = append(repository.calls, "ProjectIssueAttachment")
	return repository.asset, map[string]any{"id": repository.asset.ID, "issue": uuid.NewString()}, nil
}

func (repository *apiAttachmentGetRepository) IssueAttachmentsBySlug(context.Context, string, uuid.UUID, uuid.UUID, bool) ([]map[string]any, error) {
	repository.calls = append(repository.calls, "IssueAttachmentsBySlug")
	return repository.issueRows, nil
}

func (repository *apiAttachmentGetRepository) UpdateAPIFileAssetUploaded(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	repository.calls = append(repository.calls, "UpdateAPIFileAssetUploaded")
	repository.assetUpdated = true
	return nil
}

func (repository *apiAttachmentGetRepository) UpdateIssueAttachmentUploaded(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	repository.calls = append(repository.calls, "UpdateIssueAttachmentUploaded")
	repository.issueAssetUpdated = true
	return nil
}

func (repository *apiAttachmentGetRepository) DeleteIssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	repository.calls = append(repository.calls, "DeleteIssueAttachment")
	repository.issueAssetDeleted = true
	return nil
}

func (repository *apiAttachmentGetRepository) DeleteAPIFileAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) error {
	repository.calls = append(repository.calls, "DeleteAPIFileAsset")
	repository.assetDeleted = true
	return nil
}

type apiAttachmentTaskRecorder struct {
	tasks []map[string]any
}

func (recorder *apiAttachmentTaskRecorder) Task(_ context.Context, name string, args []any, kwargs map[string]any) error {
	recorder.tasks = append(recorder.tasks, map[string]any{"name": name, "args": args, "kwargs": kwargs})
	return nil
}

func TestAPIssueAttachmentGetRequiresAuthorizedPathIssueAndAssetScope(t *testing.T) {
	userID, workspaceID, projectID, issueID, assetID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	workspaceIDCopy, projectIDCopy, issueIDCopy := workspaceID, projectID, issueID
	repository := &apiAttachmentGetRepository{
		workspaceID: workspaceID,
		asset: FileAsset{
			ID: assetID.String(), Asset: "workspace/object.png", IsUploaded: true,
			Attributes: map[string]any{"name": "image.png"}, WorkspaceID: &workspaceIDCopy,
			ProjectID: &projectIDCopy, IssueID: &issueIDCopy,
		},
		issueFound: true, issueMember: true,
	}
	storage, err := NewStorage(StorageConfig{
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Bucket:          "assets-test",
		Region:          "us-east-1",
		SignedURLTTL:    0,
	})
	if err != nil {
		t.Fatal(err)
	}
	service := &LifecycleService{assets: &Service{repository: repository, storage: storage}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

	location, err := service.getIssueAttachmentV1(ctx, "workspace", projectID.String(), issueID.String(), assetID.String())
	if err != nil {
		t.Fatalf("get attachment for an authorized path issue: %v", err)
	}
	if location == nil {
		t.Fatal("attachment GET returned no signed location")
	}
	parsed, err := url.Parse(*location)
	if err != nil || parsed.Query().Get("X-Amz-Signature") == "" {
		t.Fatalf("attachment location is not signed: %q (%v)", *location, err)
	}
	wantCalls := []string{"RelatedWorkspaceIDBySlug", "IssuePermission", "IssueAttachment"}
	if len(repository.calls) != len(wantCalls) {
		t.Fatalf("repository calls = %v, want %v", repository.calls, wantCalls)
	}
	for index := range wantCalls {
		if repository.calls[index] != wantCalls[index] {
			t.Fatalf("repository calls = %v, want %v", repository.calls, wantCalls)
		}
	}
}

func TestAPIssueAttachmentGetRejectsNonmemberBeforeAssetLookup(t *testing.T) {
	userID, workspaceID, projectID, assetID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := &apiAttachmentGetRepository{workspaceID: workspaceID, issueFound: true, issueMember: false}
	service := &LifecycleService{assets: &Service{repository: repository}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

	_, err := service.getIssueAttachmentV1(ctx, "workspace", projectID.String(), uuid.NewString(), assetID.String())
	var httpErr *djangoapi.HTTPError
	if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 403 {
		t.Fatalf("nonmember GET error = %v, want source 403", err)
	}
	wantCalls := []string{"RelatedWorkspaceIDBySlug", "IssuePermission"}
	if len(repository.calls) != len(wantCalls) {
		t.Fatalf("repository calls = %v, want %v", repository.calls, wantCalls)
	}
	for index := range wantCalls {
		if repository.calls[index] != wantCalls[index] {
			t.Fatalf("repository calls = %v, want %v", repository.calls, wantCalls)
		}
	}
}

func TestAPIssueAttachmentGetResolvesSlugAfterProjectMembership(t *testing.T) {
	for _, test := range []struct {
		name         string
		found        bool
		member       bool
		asset        FileAsset
		assetErr     error
		workspaceErr error
		wantStatus   int
		wantBody     map[string]any
		wantSigned   bool
		wantCalls    []string
	}{
		{
			name:         "missing workspace fails before issue permission",
			workspaceErr: &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}},
			wantStatus:   404,
			wantBody:     map[string]any{"error": "The requested resource does not exist."},
			wantCalls:    []string{"RelatedWorkspaceIDBySlug"},
		},
		{
			name:       "missing issue is not found",
			found:      false,
			wantStatus: 404,
			wantBody:   map[string]any{"error": "The requested resource does not exist."},
			wantCalls:  []string{"RelatedWorkspaceIDBySlug", "IssuePermission"},
		},
		{
			name:       "member can download attachment for authorized issue",
			found:      true,
			member:     true,
			asset:      FileAsset{ID: uuid.NewString(), Asset: "workspace/object.png", IsUploaded: true, Attributes: map[string]any{"name": "image.png"}},
			wantSigned: true,
			wantCalls:  []string{"RelatedWorkspaceIDBySlug", "IssuePermission", "IssueAttachment"},
		},
		{
			name:       "issue nonmember is denied before asset lookup",
			found:      true,
			member:     false,
			wantStatus: 403,
			wantBody:   map[string]any{"error": "You are not allowed to download this attachment"},
			wantCalls:  []string{"RelatedWorkspaceIDBySlug", "IssuePermission"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			userID, projectID, assetID := uuid.New(), uuid.New(), uuid.New()
			repository := &apiAttachmentGetRepository{
				workspaceID: uuid.New(), asset: test.asset, attachmentReadErr: test.assetErr, apiWorkspaceLookupErr: test.workspaceErr, member: test.member,
				activeWorkspaceErr: &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}},
				issueFound:         test.found, issueMember: test.member,
			}
			storage, err := NewStorage(StorageConfig{AccessKeyID: "test-access-key", SecretAccessKey: "test-secret-key", Bucket: "assets-test", Region: "us-east-1"})
			if err != nil {
				t.Fatal(err)
			}
			service := &LifecycleService{assets: &Service{repository: repository, storage: storage}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

			location, err := service.getIssueAttachmentV1(ctx, "missing-or-tombstoned", projectID.String(), uuid.NewString(), assetID.String())
			if test.wantSigned {
				if err != nil || location == nil {
					t.Fatalf("tombstoned workspace download = %v, %v; want signed redirect", location, err)
				}
			} else {
				var httpErr *djangoapi.HTTPError
				if err == nil || !errors.As(err, &httpErr) || httpErr.Status != test.wantStatus || !reflect.DeepEqual(httpErr.Body, test.wantBody) {
					t.Fatalf("workspace-boundary GET error = %v, want HTTP %d body %#v", err, test.wantStatus, test.wantBody)
				}
			}
			if !reflect.DeepEqual(repository.calls, test.wantCalls) {
				t.Fatalf("repository calls = %v, want %v", repository.calls, test.wantCalls)
			}
		})
	}
}

func TestAPIIssueAttachmentMutationsMapMissingIssueToSourceNotFound(t *testing.T) {
	for _, test := range []struct {
		name string
		call func(*LifecycleService, context.Context, string, string, string) error
	}{
		{
			name: "PATCH confirmation",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID string) error {
				return service.confirmIssueAttachmentV1(ctx, slug, projectID, issueID, uuid.NewString())
			},
		},
		{
			name: "DELETE",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID string) error {
				return service.deleteIssueAttachmentV1(ctx, slug, projectID, issueID, uuid.NewString())
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			userID, projectID, issueID := uuid.New(), uuid.New(), uuid.New()
			repository := &apiAttachmentGetRepository{workspaceID: uuid.New(), issueFound: false}
			service := &LifecycleService{assets: &Service{repository: repository}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

			err := test.call(service, ctx, "workspace", projectID.String(), issueID.String())
			var httpErr *djangoapi.HTTPError
			if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 || httpErr.Body["error"] != "The requested resource does not exist." {
				t.Fatalf("missing issue mutation error = %v, want source 404", err)
			}
			wantCalls := []string{"RelatedWorkspaceIDBySlug", "IssuePermission"}
			if !reflect.DeepEqual(repository.calls, wantCalls) {
				t.Fatalf("repository calls = %v, want no asset effects and calls %v", repository.calls, wantCalls)
			}
		})
	}
}

func TestAPIIssueAttachmentCollectionGETRequiresIssuePermission(t *testing.T) {
	for _, test := range []struct {
		name         string
		workspaceErr error
		issueFound   bool
		issueMember  bool
		rows         []map[string]any
		wantIDs      []string
		wantStatus   int
		wantCalls    []string
	}{
		{
			name:         "missing workspace is not found",
			workspaceErr: &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}},
			wantStatus:   404,
			wantCalls:    []string{"RelatedWorkspaceIDBySlug"},
		},
		{
			name:       "nonmember cannot list attachment metadata",
			issueFound: true,
			wantStatus: 403,
			wantCalls:  []string{"RelatedWorkspaceIDBySlug", "IssuePermission"},
		},
		{
			name:        "authorized issue returns its uploaded attachments",
			issueFound:  true,
			issueMember: true,
			rows: []map[string]any{
				{"id": "asset-one", "asset": "objects/one.png", "entity_type": "ISSUE_ATTACHMENT", "is_uploaded": true},
				{"id": "asset-two", "asset": "objects/two.png", "entity_type": "ISSUE_ATTACHMENT", "is_uploaded": true},
			},
			wantIDs:   []string{"asset-one", "asset-two"},
			wantCalls: []string{"RelatedWorkspaceIDBySlug", "IssuePermission", "IssueAttachmentsBySlug"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiAttachmentGetRepository{
				workspaceID: uuid.New(), apiWorkspaceLookupErr: test.workspaceErr,
				issueFound: test.issueFound, issueMember: test.issueMember, issueRows: test.rows,
			}
			service := &LifecycleService{assets: &Service{repository: repository}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())

			rows, err := service.listIssueAttachmentsAPI(ctx, "missing-or-tombstoned", uuid.NewString(), uuid.NewString())
			if test.wantStatus == 0 {
				if err != nil {
					t.Fatalf("API attachment collection GET error = %v", err)
				}
			} else {
				var httpErr *djangoapi.HTTPError
				if err == nil || !errors.As(err, &httpErr) || httpErr.Status != test.wantStatus {
					t.Fatalf("API attachment collection GET error = %v, want HTTP %d", err, test.wantStatus)
				}
			}
			if len(rows) != len(test.wantIDs) {
				t.Fatalf("API attachment count = %d, want %d", len(rows), len(test.wantIDs))
			}
			for index, wantID := range test.wantIDs {
				if rows[index].ID == nil || *rows[index].ID != wantID {
					t.Fatalf("API attachment[%d] id = %v, want %q", index, rows[index].ID, wantID)
				}
			}
			if !reflect.DeepEqual(repository.calls, test.wantCalls) {
				t.Fatalf("repository calls = %v, want %v", repository.calls, test.wantCalls)
			}
		})
	}
}

func TestAPIssueAttachmentUploadMapsMissingIssueToSourceNotFound(t *testing.T) {
	userID, projectID, issueID := uuid.New(), uuid.New(), uuid.New()
	repository := &apiAttachmentGetRepository{workspaceID: uuid.New(), issueFound: false}
	service := &LifecycleService{assets: &Service{repository: repository}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

	_, err := service.createIssueAttachmentAPI(ctx, "workspace", projectID.String(), issueID.String(), io.NopCloser(strings.NewReader(`{}`)), true)
	var httpErr *djangoapi.HTTPError
	if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 {
		t.Fatalf("soft-deleted issue upload error = %v, want HTTP 404", err)
	}
	wantBody := map[string]any{"error": "The requested resource does not exist."}
	if !reflect.DeepEqual(httpErr.Body, wantBody) {
		t.Fatalf("soft-deleted issue upload body = %#v, want %#v", httpErr.Body, wantBody)
	}
	wantCalls := []string{"RelatedWorkspaceIDBySlug", "IssuePermission"}
	if !reflect.DeepEqual(repository.calls, wantCalls) {
		t.Fatalf("repository calls = %v, want %v", repository.calls, wantCalls)
	}
}

func TestAPIIssueAttachmentOperationsMapMissingWorkspaceToSourceNotFound(t *testing.T) {
	for _, test := range []struct {
		name string
		call func(*LifecycleService, context.Context, string, string, string) error
	}{
		{
			name: "POST",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID string) error {
				_, err := service.createIssueAttachmentAPI(ctx, slug, projectID, issueID, io.NopCloser(strings.NewReader(`{}`)), true)
				return err
			},
		},
		{
			name: "PATCH",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID string) error {
				return service.confirmIssueAttachmentV1(ctx, slug, projectID, issueID, uuid.NewString())
			},
		},
		{
			name: "DELETE",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID string) error {
				return service.deleteIssueAttachmentV1(ctx, slug, projectID, issueID, uuid.NewString())
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiAttachmentGetRepository{
				apiWorkspaceLookupErr: &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}},
			}
			service := &LifecycleService{assets: &Service{repository: repository}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())
			err := test.call(service, ctx, "missing-workspace", uuid.NewString(), uuid.NewString())
			var httpErr *djangoapi.HTTPError
			if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 || httpErr.Body["error"] != "The requested resource does not exist." {
				t.Fatalf("missing workspace %s error = %v, want source 404", test.name, err)
			}
			if !reflect.DeepEqual(repository.calls, []string{"RelatedWorkspaceIDBySlug"}) {
				t.Fatalf("missing workspace %s repository calls = %v", test.name, repository.calls)
			}
		})
	}
}

func TestAPIIssueAttachmentPostKeepsTombstonedWorkspaceValidationOrder(t *testing.T) {
	for _, test := range []struct {
		name       string
		member     bool
		body       string
		wantStatus int
		wantBody   map[string]any
		wantCalls  []string
	}{
		{
			name:       "unauthorized before active workspace lookup",
			member:     false,
			body:       `{"name":"file.png","type":"image/png","size":68}`,
			wantStatus: 403,
			wantBody:   map[string]any{"error": "You are not allowed to upload this attachment"},
			wantCalls:  []string{"RelatedWorkspaceIDBySlug", "IssuePermission"},
		},
		{
			name:       "invalid MIME before active workspace lookup",
			member:     true,
			body:       `{"type":"image/png"}`,
			wantStatus: 400,
			wantBody:   map[string]any{"error": "Invalid request.", "status": false},
			wantCalls:  []string{"RelatedWorkspaceIDBySlug", "IssuePermission"},
		},
		{
			name:       "valid request checks active workspace after validation",
			member:     true,
			body:       `{"name":"file.png","type":"image/png","size":68}`,
			wantStatus: 404,
			wantBody:   map[string]any{"error": "The requested resource does not exist."},
			wantCalls:  []string{"RelatedWorkspaceIDBySlug", "IssuePermission", "WorkspaceIDBySlug"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &apiAttachmentGetRepository{
				workspaceID: uuid.New(), issueFound: true, issueMember: test.member,
				activeWorkspaceErr: &djangoapi.HTTPError{Status: 404, Body: map[string]any{"error": "The required object does not exist."}},
			}
			service := &LifecycleService{assets: &Service{repository: repository, fileLimit: 5 << 20}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())
			_, err := service.createIssueAttachmentAPI(ctx, "tombstoned-workspace", uuid.NewString(), uuid.NewString(), io.NopCloser(strings.NewReader(test.body)), true)
			var httpErr *djangoapi.HTTPError
			if err == nil || !errors.As(err, &httpErr) || httpErr.Status != test.wantStatus || !reflect.DeepEqual(httpErr.Body, test.wantBody) {
				t.Fatalf("tombstoned workspace POST error = %v, want HTTP %d body %#v", err, test.wantStatus, test.wantBody)
			}
			if !reflect.DeepEqual(repository.calls, test.wantCalls) {
				t.Fatalf("tombstoned workspace POST calls = %v, want %v", repository.calls, test.wantCalls)
			}
		})
	}
}

func TestAPIIssueAttachmentPatchDeleteUseIssueScopedRepository(t *testing.T) {
	for _, test := range []struct {
		name          string
		call          func(*LifecycleService, context.Context, string, string, string, string) error
		asset         FileAsset
		wantUpdated   bool
		wantDeleted   bool
		wantTaskNames []string
		wantCalls     []string
	}{
		{
			name: "PATCH confirmation",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID, assetID string) error {
				return service.confirmIssueAttachmentV1(ctx, slug, projectID, issueID, assetID)
			},
			asset:       FileAsset{ID: uuid.NewString(), Asset: "workspace/object.png", IsUploaded: false},
			wantUpdated: true,
			wantTaskNames: []string{
				"plane.bgtasks.issue_activities_task.issue_activity",
				"plane.bgtasks.storage_metadata_task.get_asset_object_metadata",
			},
			wantCalls: []string{"RelatedWorkspaceIDBySlug", "IssuePermission", "IssueAttachment", "UpdateIssueAttachmentUploaded"},
		},
		{
			name: "DELETE",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID, assetID string) error {
				return service.deleteIssueAttachmentV1(ctx, slug, projectID, issueID, assetID)
			},
			asset:         FileAsset{ID: uuid.NewString(), Asset: "workspace/object.png", IsUploaded: true, StorageMetadata: map[string]any{"ContentLength": 68}},
			wantDeleted:   true,
			wantTaskNames: []string{"plane.bgtasks.issue_activities_task.issue_activity"},
			wantCalls:     []string{"RelatedWorkspaceIDBySlug", "IssuePermission", "IssueAttachment", "DeleteIssueAttachment"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			userID, projectID, issueID, assetID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			repository := &apiAttachmentGetRepository{workspaceID: uuid.New(), asset: test.asset, issueFound: true, issueMember: true}
			tasks := &apiAttachmentTaskRecorder{}
			service := &LifecycleService{assets: &Service{repository: repository, tasks: tasks}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
			if err := test.call(service, ctx, "tombstoned-workspace", projectID.String(), issueID.String(), assetID.String()); err != nil {
				t.Fatalf("mutation against surviving tombstoned-workspace rows: %v", err)
			}
			if repository.issueAssetUpdated != test.wantUpdated || repository.issueAssetDeleted != test.wantDeleted {
				t.Fatalf("issue asset updated=%t deleted=%t, want updated=%t deleted=%t", repository.issueAssetUpdated, repository.issueAssetDeleted, test.wantUpdated, test.wantDeleted)
			}
			if !reflect.DeepEqual(repository.calls, test.wantCalls) {
				t.Fatalf("repository calls = %v, want %v", repository.calls, test.wantCalls)
			}
			var gotTaskNames []string
			for _, task := range tasks.tasks {
				name, _ := task["name"].(string)
				gotTaskNames = append(gotTaskNames, name)
			}
			if !reflect.DeepEqual(gotTaskNames, test.wantTaskNames) {
				t.Fatalf("published task names = %v, want %v", gotTaskNames, test.wantTaskNames)
			}
		})
	}
}

func TestAPIssueAttachmentConfirmationQueuesTheCorrectIssueScopedAsset(t *testing.T) {
	userID, workspaceID, projectID, pathIssueID, assetID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	workspaceCopy, projectCopy, issueCopy := workspaceID, projectID, pathIssueID
	asset := FileAsset{ID: assetID.String(), IsUploaded: false, StorageMetadata: map[string]any{"ContentLength": 68}, WorkspaceID: &workspaceCopy, ProjectID: &projectCopy, IssueID: &issueCopy}
	repository := &apiAttachmentGetRepository{workspaceID: workspaceID, asset: asset, issueFound: true, issueMember: true}
	tasks := &apiAttachmentTaskRecorder{}
	service := &LifecycleService{assets: &Service{repository: repository, tasks: tasks}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

	err := service.confirmIssueAttachmentV1(ctx, "workspace", projectID.String(), pathIssueID.String(), assetID.String())
	if err != nil {
		t.Fatalf("confirm issue-scoped attachment: %v", err)
	}
	if len(tasks.tasks) != 1 {
		t.Fatalf("published tasks = %v, want one activity task", tasks.tasks)
	}
	kwargs, ok := tasks.tasks[0]["kwargs"].(map[string]any)
	if !ok {
		t.Fatalf("activity task kwargs = %#v", tasks.tasks[0]["kwargs"])
	}
	currentJSON, ok := kwargs["current_instance"].(string)
	if !ok {
		t.Fatalf("current_instance = %#v, want serialized asset", kwargs["current_instance"])
	}
	var current map[string]any
	if err := json.Unmarshal([]byte(currentJSON), &current); err != nil {
		t.Fatalf("decode current_instance: %v", err)
	}
	if current["id"] != assetID.String() {
		t.Fatalf("current_instance id = %#v, want %s", current["id"], assetID)
	}
	if current["issue"] != pathIssueID.String() {
		t.Fatalf("current_instance issue = %#v, want path issue %s", current["issue"], pathIssueID)
	}
	if repository.calls[len(repository.calls)-2] != "IssueAttachment" || repository.calls[len(repository.calls)-1] != "UpdateIssueAttachmentUploaded" {
		t.Fatalf("repository calls = %v, want issue-scoped lookup before upload update", repository.calls)
	}
}

func TestAPIIssueAttachmentListDeniesNonmemberBeforeQuery(t *testing.T) {
	repository := &apiAttachmentGetRepository{
		workspaceID: uuid.New(), issueFound: true, issueMember: false,
		issueRows: []map[string]any{{"id": uuid.NewString(), "entity_type": "ISSUE_ATTACHMENT", "is_uploaded": true}},
	}
	service := &LifecycleService{assets: &Service{repository: repository}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())

	_, err := service.listIssueAttachmentsAPI(ctx, "workspace", uuid.NewString(), uuid.NewString())
	var httpErr *djangoapi.HTTPError
	if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 403 {
		t.Fatalf("nonmember attachment list error = %v, want 403", err)
	}
	if !reflect.DeepEqual(repository.calls, []string{"RelatedWorkspaceIDBySlug", "IssuePermission"}) {
		t.Fatalf("nonmember attachment list calls = %v, want permission check before attachment query", repository.calls)
	}
}

func TestAPIIssueAttachmentGetRejectsAssetFromDifferentIssue(t *testing.T) {
	userID, workspaceID, projectID, pathIssueID, assetIssueID, assetID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	workspaceCopy, projectCopy, issueCopy := workspaceID, projectID, assetIssueID
	repository := &apiAttachmentGetRepository{
		workspaceID: workspaceID,
		asset: FileAsset{
			ID: assetID.String(), Asset: "workspace/other-issue.png", IsUploaded: true,
			Attributes: map[string]any{"name": "other-issue.png"}, WorkspaceID: &workspaceCopy,
			ProjectID: &projectCopy, IssueID: &issueCopy,
		},
		issueFound: true, issueMember: true,
	}
	storage, err := NewStorage(StorageConfig{AccessKeyID: "test-access-key", SecretAccessKey: "test-secret-key", Bucket: "assets-test", Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	service := &LifecycleService{assets: &Service{repository: repository, storage: storage}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

	location, err := service.getIssueAttachmentV1(ctx, "workspace", projectID.String(), pathIssueID.String(), assetID.String())
	var httpErr *djangoapi.HTTPError
	if err == nil || location != nil || !errors.As(err, &httpErr) || httpErr.Status != 404 {
		t.Fatalf("cross-issue attachment GET = (%v, %v), want no signed location and 404", location, err)
	}
	if !reflect.DeepEqual(repository.calls, []string{"RelatedWorkspaceIDBySlug", "IssuePermission", "IssueAttachment"}) {
		t.Fatalf("cross-issue attachment GET calls = %v, want issue-scoped lookup", repository.calls)
	}
}

func TestAPIIssueAttachmentMutationsRejectAssetFromDifferentIssue(t *testing.T) {
	for _, test := range []struct {
		name string
		call func(*LifecycleService, context.Context, string, string, string, string) error
	}{
		{
			name: "confirmation",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID, assetID string) error {
				return service.confirmIssueAttachmentV1(ctx, slug, projectID, issueID, assetID)
			},
		},
		{
			name: "deletion",
			call: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID, assetID string) error {
				return service.deleteIssueAttachmentV1(ctx, slug, projectID, issueID, assetID)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			userID, workspaceID, projectID, pathIssueID, assetIssueID, assetID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
			workspaceCopy, projectCopy, issueCopy := workspaceID, projectID, assetIssueID
			repository := &apiAttachmentGetRepository{
				workspaceID: workspaceID,
				asset:       FileAsset{ID: assetID.String(), Asset: "workspace/other-issue.png", IsUploaded: false, WorkspaceID: &workspaceCopy, ProjectID: &projectCopy, IssueID: &issueCopy},
				issueFound:  true, issueMember: true,
			}
			tasks := &apiAttachmentTaskRecorder{}
			service := &LifecycleService{assets: &Service{repository: repository, tasks: tasks}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)

			err := test.call(service, ctx, "workspace", projectID.String(), pathIssueID.String(), assetID.String())
			var httpErr *djangoapi.HTTPError
			if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 {
				t.Fatalf("cross-issue %s error = %v, want 404", test.name, err)
			}
			if repository.issueAssetUpdated || repository.issueAssetDeleted || repository.assetUpdated || repository.assetDeleted {
				t.Fatalf("cross-issue %s mutated asset: %#v", test.name, repository)
			}
			if len(tasks.tasks) != 0 {
				t.Fatalf("cross-issue %s published tasks = %#v, want none", test.name, tasks.tasks)
			}
			if !reflect.DeepEqual(repository.calls, []string{"RelatedWorkspaceIDBySlug", "IssuePermission", "IssueAttachment"}) {
				t.Fatalf("cross-issue %s calls = %v, want issue-scoped lookup before mutation", test.name, repository.calls)
			}
		})
	}
}
