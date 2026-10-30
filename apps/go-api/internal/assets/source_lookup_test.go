package assets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

type sourceLookupRepository struct {
	assetRepository
	workspaceID         uuid.UUID
	lookupErr           error
	relatedWorkspaceErr error
	assetIDErr          error
	userAssetErr        error
	issueAttachmentErr  error
	deleteAttachmentErr error
	assetKeyErr         error
	restoreErr          error
	asset               FileAsset
	deleteAllowed       bool
	calls               []string
}

func (repository *sourceLookupRepository) AuthorizedWorkspaceID(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return repository.workspaceID, nil
}

func (repository *sourceLookupRepository) WorkspacePermissionAllowed(context.Context, uuid.UUID, string) (bool, error) {
	return true, nil
}

func (repository *sourceLookupRepository) WorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	return repository.workspaceID, nil
}

func (repository *sourceLookupRepository) RelatedWorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	repository.calls = append(repository.calls, "RelatedWorkspaceIDBySlug")
	return repository.workspaceID, repository.relatedWorkspaceErr
}

func (repository *sourceLookupRepository) ProjectPermissionAllowed(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return true, nil
}

func (repository *sourceLookupRepository) AssetByWorkspaceAndID(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error) {
	repository.calls = append(repository.calls, "AssetByWorkspaceAndID")
	return FileAsset{}, repository.lookupErr
}

func (repository *sourceLookupRepository) AssetByWorkspaceAndIDIncludingDeleted(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error) {
	return repository.asset, repository.lookupErr
}

func (repository *sourceLookupRepository) AssetForDuplicate(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, error) {
	repository.calls = append(repository.calls, "AssetForDuplicate")
	return FileAsset{}, repository.lookupErr
}

func (repository *sourceLookupRepository) AssetByProjectAndID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, error) {
	repository.calls = append(repository.calls, "AssetByProjectAndID")
	return FileAsset{}, repository.lookupErr
}

func (repository *sourceLookupRepository) AssetByID(context.Context, uuid.UUID) (FileAsset, error) {
	repository.calls = append(repository.calls, "AssetByID")
	return FileAsset{}, repository.assetIDErr
}

func (repository *sourceLookupRepository) UserAssetByID(context.Context, uuid.UUID, uuid.UUID) (FileAsset, error) {
	repository.calls = append(repository.calls, "UserAssetByID")
	if repository.userAssetErr != nil {
		return FileAsset{}, repository.userAssetErr
	}
	return FileAsset{}, repository.lookupErr
}

func (repository *sourceLookupRepository) IssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (FileAsset, map[string]any, error) {
	repository.calls = append(repository.calls, "IssueAttachment")
	return FileAsset{}, nil, repository.issueAttachmentErr
}

func (repository *sourceLookupRepository) DeleteIssueAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, map[string]any) error {
	repository.calls = append(repository.calls, "DeleteIssueAttachment")
	return repository.deleteAttachmentErr
}

func (repository *sourceLookupRepository) UpdateAssetDeletedByKey(context.Context, string, *uuid.UUID, *uuid.UUID, uuid.UUID, bool) error {
	repository.calls = append(repository.calls, "UpdateAssetDeletedByKey")
	return repository.assetKeyErr
}

func (repository *sourceLookupRepository) RestoreWorkspaceAsset(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	repository.calls = append(repository.calls, "RestoreWorkspaceAsset")
	return repository.restoreErr
}

func (repository *sourceLookupRepository) AssetParentVisible(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

func (repository *sourceLookupRepository) WorkspaceMemberByID(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	repository.calls = append(repository.calls, "WorkspaceMemberByID")
	return true, nil
}

func (repository *sourceLookupRepository) IssueAccess(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (bool, bool, error) {
	repository.calls = append(repository.calls, "IssueAccess")
	return true, true, nil
}

func (repository *sourceLookupRepository) IssueAttachmentDeleteAllowed(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) (bool, error) {
	repository.calls = append(repository.calls, "IssueAttachmentDeleteAllowed")
	return repository.deleteAllowed, nil
}

func TestSourceLookupErrorsAreMappedAtServiceBoundaries(t *testing.T) {
	userID, workspaceID, projectID, assetID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	internalErr := errors.New("database connection lost")

	for _, test := range []struct {
		name        string
		lookup      string
		wantLookups []string
		missingBody map[string]any
		call        func(*LifecycleService) error
	}{
		{
			name:        "workspace upload confirmation",
			lookup:      "AssetByWorkspaceAndID",
			wantLookups: []string{"AssetByWorkspaceAndID"},
			missingBody: map[string]any{"error": "Asset not found"},
			call: func(service *LifecycleService) error {
				return service.confirmLegacyWorkspaceAssetUpload(ctx, "assets", assetID.String(), nil)
			},
		},
		{
			name:        "duplicate source",
			lookup:      "AssetForDuplicate",
			wantLookups: []string{"AssetForDuplicate"},
			missingBody: map[string]any{"error": "Asset not found"},
			call: func(service *LifecycleService) error {
				body := io.NopCloser(strings.NewReader(`{"entity_type":"ISSUE_DESCRIPTION","entity_id":"` + uuid.NewString() + `"}`))
				_, err := service.duplicateWorkspaceAsset(ctx, "assets", assetID.String(), body)
				return err
			},
		},
		{
			name:        "project-scoped mutation",
			lookup:      "AssetByProjectAndID",
			wantLookups: []string{"RelatedWorkspaceIDBySlug", "AssetByProjectAndID"},
			missingBody: map[string]any{"error": "The required object does not exist."},
			call: func(service *LifecycleService) error {
				return service.deleteProjectAsset(ctx, "assets", projectID.String(), assetID.String())
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failure := range []struct {
				name string
				err  error
			}{
				{name: "internal repository failure", err: internalErr},
				{
					name: "missing row",
					err: &djangoapi.HTTPError{
						Status: 404,
						Body:   map[string]any{"detail": "repository not found"},
					},
				},
			} {
				t.Run(failure.name, func(t *testing.T) {
					repository := &sourceLookupRepository{workspaceID: workspaceID, lookupErr: failure.err}
					service := &LifecycleService{assets: &Service{repository: repository}}
					err := test.call(service)
					if len(repository.calls) != len(test.wantLookups) {
						t.Fatalf("repository lookups = %v, want %v", repository.calls, test.wantLookups)
					}
					for index, want := range test.wantLookups {
						if repository.calls[index] != want {
							t.Fatalf("repository lookups = %v, want %v", repository.calls, test.wantLookups)
						}
					}

					if errors.Is(failure.err, internalErr) {
						require.Same(t, internalErr, err, "preserve the original repository error")
						return
					}

					var httpErr *djangoapi.HTTPError
					if !errors.As(err, &httpErr) || httpErr.Status != 404 || !reflect.DeepEqual(httpErr.Body, test.missingBody) {
						t.Fatalf("service missing-row error = %#v, want source 404 %#v", err, test.missingBody)
					}
				})
			}
		})
	}
}

type missingAssetCase struct {
	name       string
	want       map[string]any
	call       func(*LifecycleService) error
	repository func() *sourceLookupRepository
}

func runMissingAssetCases(t *testing.T, workspaceID uuid.UUID, cases []missingAssetCase) {
	t.Helper()
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			repository := test.repository()
			if repository.workspaceID == uuid.Nil {
				repository.workspaceID = workspaceID
			}
			service := &LifecycleService{assets: &Service{repository: repository}}
			assertAssetErrorBody(t, test.call(service), test.want)
		})
	}
}

func assertAssetErrorBody(t *testing.T, err error, want map[string]any) {
	t.Helper()
	if err == nil {
		t.Fatal("operation succeeded for a missing asset")
	}
	encoded, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("encode operation error: %v", marshalErr)
	}
	var got map[string]any
	if unmarshalErr := json.Unmarshal(encoded, &got); unmarshalErr != nil {
		t.Fatalf("decode operation error %s: %v", encoded, unmarshalErr)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("operation body = %#v, want %#v", got, want)
	}
}

func TestV2AssetsMissingRowsUseSourceBodies(t *testing.T) {
	userID, workspaceID := uuid.New(), uuid.New()
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	missing := &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	assetID, projectID, issueID := uuid.NewString(), uuid.NewString(), uuid.NewString()

	runMissingAssetCases(t, workspaceID, []missingAssetCase{
		{
			name:       "v2 user asset delete",
			want:       map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{userAssetErr: missing} },
			call: func(service *LifecycleService) error {
				return service.DeleteAPIAssetsV2UserAssetsByAssetID(ctx, &api.DeleteAPIAssetsV2UserAssetsByAssetIDPayload{AssetID: assetID})
			},
		},
		{
			name:       "v2 user asset confirmation",
			want:       map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{userAssetErr: missing} },
			call: func(service *LifecycleService) error {
				return service.PatchAPIAssetsV2UserAssetsByAssetID(ctx, &api.PatchAPIAssetsV2UserAssetsByAssetIDPayload{AssetID: assetID}, nil)
			},
		},
		{
			name:       "v2 static asset lookup",
			want:       map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{assetIDErr: missing} },
			call: func(service *LifecycleService) error {
				_, err := service.GetAPIAssetsV2StaticByAssetID(ctx, &api.GetAPIAssetsV2StaticByAssetIDPayload{AssetID: assetID})
				return err
			},
		},
		{
			name: "v2 workspace asset mutation lookup",
			want: map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{workspaceID: workspaceID, lookupErr: missing}
			},
			call: func(service *LifecycleService) error {
				return service.PatchAPIAssetsV2WorkspacesBySlugByAssetID(ctx, &api.PatchAPIAssetsV2WorkspacesBySlugByAssetIDPayload{Slug: "workspace", AssetID: assetID}, nil)
			},
		},
		{
			name: "v2 workspace asset delete lookup",
			want: map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{workspaceID: workspaceID, lookupErr: missing}
			},
			call: func(service *LifecycleService) error {
				return service.DeleteAPIAssetsV2WorkspacesBySlugByAssetID(ctx, &api.DeleteAPIAssetsV2WorkspacesBySlugByAssetIDPayload{Slug: "workspace", AssetID: assetID})
			},
		},
		{
			name: "v2 project asset download lookup",
			want: map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{
					workspaceID: workspaceID,
					relatedWorkspaceErr: &djangoapi.HTTPError{
						Status: 404,
						Body:   map[string]any{"error": "The required object does not exist."},
					},
				}
			},
			call: func(service *LifecycleService) error {
				_, err := service.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPk(ctx, &api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDByPkPayload{Slug: "workspace", ProjectID: projectID, Pk: assetID})
				return err
			},
		},
		{
			name: "v2 workspace asset restore",
			want: map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{
					workspaceID: workspaceID,
					asset: FileAsset{
						ID:          assetID,
						WorkspaceID: &workspaceID,
					},
					restoreErr: missing,
				}
			},
			call: func(service *LifecycleService) error {
				return service.PostAPIAssetsV2WorkspacesBySlugRestoreByAssetID(ctx, &api.PostAPIAssetsV2WorkspacesBySlugRestoreByAssetIDPayload{Slug: "workspace", AssetID: assetID})
			},
		},
		{
			name: "v2 issue attachment detail lookup",
			want: map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{workspaceID: workspaceID, issueAttachmentErr: missing}
			},
			call: func(service *LifecycleService) error {
				_, err := service.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPk(ctx, &api.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkPayload{Slug: "workspace", ProjectID: projectID, IssueID: issueID, Pk: assetID})
				return err
			},
		},
		{
			name: "v2 issue attachment confirmation",
			want: map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{workspaceID: workspaceID, issueAttachmentErr: missing}
			},
			call: func(service *LifecycleService) error {
				return service.PatchAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPk(ctx, &api.PatchAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkPayload{Slug: "workspace", ProjectID: projectID, IssueID: issueID, Pk: assetID}, nil)
			},
		},
		{
			name: "v2 issue attachment delete",
			want: map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{workspaceID: workspaceID, deleteAttachmentErr: missing, deleteAllowed: true}
			},
			call: func(service *LifecycleService) error {
				return service.DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPk(ctx, &api.DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsByPkPayload{Slug: "workspace", ProjectID: projectID, IssueID: issueID, Pk: assetID})
			},
		},
	})
}

func TestV1AssetsMissingRowsUseSourceBodies(t *testing.T) {
	userID, workspaceID := uuid.New(), uuid.New()
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	missing := &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
	assetID, projectID, issueID := uuid.NewString(), uuid.NewString(), uuid.NewString()

	runMissingAssetCases(t, workspaceID, []missingAssetCase{
		{
			name:       "v1 user asset delete",
			want:       map[string]any{"error": "The requested resource does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{userAssetErr: missing} },
			call: func(service *LifecycleService) error {
				return service.DeleteAPIV1AssetsUserAssetsByAssetID(ctx, &api.DeleteAPIV1AssetsUserAssetsByAssetIDPayload{AssetID: assetID})
			},
		},
		{
			name:       "v1 user asset confirmation",
			want:       map[string]any{"error": "The requested resource does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{userAssetErr: missing} },
			call: func(service *LifecycleService) error {
				return service.PatchAPIV1AssetsUserAssetsByAssetID(ctx, &api.PatchAPIV1AssetsUserAssetsByAssetIDPayload{AssetID: assetID}, nil)
			},
		},
		{
			name:       "v1 server user asset delete alias",
			want:       map[string]any{"error": "The requested resource does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{userAssetErr: missing} },
			call: func(service *LifecycleService) error {
				return service.DeleteAPIV1AssetsUserAssetsByAssetIDServer(ctx, &api.DeleteAPIV1AssetsUserAssetsByAssetIDServerPayload{AssetID: assetID})
			},
		},
		{
			name: "v1 issue attachment mutation lookup",
			want: map[string]any{"error": "The requested resource does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{workspaceID: workspaceID, relatedWorkspaceErr: missing}
			},
			call: func(service *LifecycleService) error {
				return service.PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPk(ctx, &api.PatchAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkPayload{Slug: "workspace", ProjectID: projectID, IssueID: issueID, Pk: assetID}, nil)
			},
		},
		{
			name: "v1 issue attachment delete lookup",
			want: map[string]any{"error": "The requested resource does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{workspaceID: workspaceID, relatedWorkspaceErr: missing}
			},
			call: func(service *LifecycleService) error {
				return service.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPk(ctx, &api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkPayload{Slug: "workspace", ProjectID: projectID, IssueID: issueID, Pk: assetID})
			},
		},
		{
			name: "v1 work item attachment delete alias",
			want: map[string]any{"error": "The requested resource does not exist."},
			repository: func() *sourceLookupRepository {
				return &sourceLookupRepository{workspaceID: workspaceID, relatedWorkspaceErr: missing}
			},
			call: func(service *LifecycleService) error {
				return service.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPk(ctx, &api.DeleteAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDAttachmentsByPkPayload{Slug: "workspace", ProjectID: projectID, IssueID: issueID, Pk: assetID})
			},
		},
	})
}

func TestLegacyAssetsMissingRowsUseSourceBodies(t *testing.T) {
	userID, workspaceID := uuid.New(), uuid.New()
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	missing := &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}

	runMissingAssetCases(t, workspaceID, []missingAssetCase{
		{
			name:       "legacy user key delete",
			want:       map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{assetKeyErr: missing} },
			call: func(service *LifecycleService) error {
				return service.DeleteAPIUsersFileAssetsByAssetKey(ctx, &api.DeleteAPIUsersFileAssetsByAssetKeyPayload{AssetKey: "missing"})
			},
		},
		{
			name:       "legacy workspace key restore",
			want:       map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{assetKeyErr: missing} },
			call: func(service *LifecycleService) error {
				return service.PostAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyRestore(ctx, &api.PostAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyRestorePayload{WorkspaceID: workspaceID.String(), AssetKey: "missing"})
			},
		},
		{
			name:       "legacy workspace key delete",
			want:       map[string]any{"error": "The required object does not exist."},
			repository: func() *sourceLookupRepository { return &sourceLookupRepository{assetKeyErr: missing} },
			call: func(service *LifecycleService) error {
				return service.DeleteAPIWorkspacesFileAssetsByWorkspaceIDByAssetKey(ctx, &api.DeleteAPIWorkspacesFileAssetsByWorkspaceIDByAssetKeyPayload{WorkspaceID: workspaceID.String(), AssetKey: "missing"})
			},
		},
	})
}

func TestAssetLookupInternalFailureRemains500(t *testing.T) {
	ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())
	assetID := uuid.NewString()
	repository := &sourceLookupRepository{assetIDErr: errors.New("database unavailable")}
	service := &LifecycleService{assets: &Service{repository: repository}}
	_, err := service.GetAPIAssetsV2StaticByAssetID(ctx, &api.GetAPIAssetsV2StaticByAssetIDPayload{AssetID: assetID})
	if err == nil || !strings.Contains(fmt.Sprintf("%T", err), "Status500") {
		t.Fatalf("static lookup error = %T (%v), want generated 500 response", err, err)
	}
}

func TestLegacyAttachmentDenialShortCircuitsBeforeAssetEffects(t *testing.T) {
	ctx := context.WithValue(context.Background(), assetUserKey{}, uuid.New())
	workspaceID := uuid.New()
	assetID, projectID, issueID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	repository := &sourceLookupRepository{workspaceID: workspaceID, deleteAllowed: false}
	service := &LifecycleService{assets: &Service{repository: repository}}
	err := service.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPk(ctx, &api.DeleteAPIWorkspacesBySlugProjectsByProjectIDIssuesByIssueIDIssueAttachmentsByPkPayload{
		Slug: "workspace", ProjectID: projectID, IssueID: issueID, Pk: assetID,
	})
	if err == nil || !strings.Contains(fmt.Sprintf("%T", err), "Status403") {
		t.Fatalf("denial response = %T (%v), want generated 403 response", err, err)
	}
	assertAssetErrorBody(t, err, map[string]any{"error": "You don't have the required permissions."})
	if !reflect.DeepEqual(repository.calls, []string{"IssueAttachmentDeleteAllowed"}) {
		t.Fatalf("repository calls = %v, want only permission check before any asset mutation or effect", repository.calls)
	}
}
