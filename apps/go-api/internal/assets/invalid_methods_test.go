package assets

import (
	"context"
	"errors"
	"testing"

	planeinternaldjangomigrationapi "github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

type invalidMethodRepository struct {
	assetRepository
	workspaceErr     error
	projectOK        bool
	workspaceAllowed bool
}

func (repository invalidMethodRepository) AuthorizedWorkspaceID(context.Context, uuid.UUID, string) (uuid.UUID, error) {
	return uuid.New(), repository.workspaceErr
}

func (repository invalidMethodRepository) WorkspacePermissionAllowed(context.Context, uuid.UUID, string) (bool, error) {
	return repository.workspaceAllowed, nil
}

func (repository invalidMethodRepository) RelatedWorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	if !repository.workspaceAllowed {
		panic("workspace lookup before permission")
	}
	return uuid.New(), nil
}

func (repository invalidMethodRepository) ProjectPermissionAllowed(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	return repository.projectOK, nil
}

func TestV2WorkspaceDecoratorPermissionUsesRoleBeforeLookup(t *testing.T) {
	userID := uuid.New()
	service := &LifecycleService{assets: &Service{repository: invalidMethodRepository{workspaceAllowed: false}}}
	_, err := service.v2WorkspaceID(context.Background(), userID, "missing-or-denied")
	var httpErr *djangoapi.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != 403 {
		t.Fatalf("V2 workspace permission error = %#v, want HTTP 403", err)
	}
	if got := httpErr.Body["error"]; got != "You don't have the required permissions." {
		t.Fatalf("permission body = %#v", got)
	}
}

func (repository invalidMethodRepository) ProjectMemberActive(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return repository.projectOK, nil
}

func (invalidMethodRepository) WorkspaceMemberByID(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func (invalidMethodRepository) WorkspaceMemberBySlug(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}

func TestInvalidAssetMethodsReturn405AfterPermission(t *testing.T) {
	if len(invalidAssetMethods) != 23 {
		t.Fatalf("invalid method contract count = %d, want 23", len(invalidAssetMethods))
	}
	userID, projectID := uuid.New(), uuid.NewString()
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	for _, tc := range []struct {
		name, operation string
		payload         any
		repository      invalidMethodRepository
		status          int
		field, message  string
	}{
		{
			name: "user collection write", operation: "DeleteAPIAssetsV2UserAssets",
			payload: &planeinternaldjangomigrationapi.DeleteAPIAssetsV2UserAssetsPayload{},
			status:  405, field: "detail", message: `Method "DELETE" not allowed.`,
		},
		{
			name: "legacy user collection read", operation: "GetAPIUsersFileAssets",
			payload: &planeinternaldjangomigrationapi.GetAPIUsersFileAssetsPayload{},
			status:  405, field: "detail", message: `Method "GET" not allowed.`,
		},
		{
			name: "allowed workspace read", operation: "GetAPIAssetsV2WorkspacesBySlug",
			payload:    &planeinternaldjangomigrationapi.GetAPIAssetsV2WorkspacesBySlugPayload{Slug: "assets"},
			repository: invalidMethodRepository{workspaceAllowed: true},
			status:     405, field: "detail", message: `Method "GET" not allowed.`,
		},
		{
			name: "denied workspace read", operation: "GetAPIAssetsV2WorkspacesBySlug",
			payload: &planeinternaldjangomigrationapi.GetAPIAssetsV2WorkspacesBySlugPayload{Slug: "assets"},
			status:  403, field: "error", message: "You don't have the required permissions.",
		},
		{
			name: "allowed project read", operation: "GetAPIAssetsV2WorkspacesBySlugProjectsByProjectID",
			payload:    &planeinternaldjangomigrationapi.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDPayload{Slug: "assets", ProjectID: projectID},
			repository: invalidMethodRepository{projectOK: true},
			status:     405, field: "detail", message: `Method "GET" not allowed.`,
		},
		{
			name: "denied project read", operation: "GetAPIAssetsV2WorkspacesBySlugProjectsByProjectID",
			payload: &planeinternaldjangomigrationapi.GetAPIAssetsV2WorkspacesBySlugProjectsByProjectIDPayload{Slug: "assets", ProjectID: projectID},
			status:  403, field: "error", message: "You don't have the required permissions.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &LifecycleService{assets: &Service{repository: tc.repository}}
			_, err := service.invoke(ctx, tc.operation, tc.payload, nil)
			var httpErr *djangoapi.HTTPError
			if !errors.As(err, &httpErr) || httpErr.Status != tc.status {
				t.Fatalf("unsupported operation error = %#v, want HTTP %d", err, tc.status)
			}
			if got := httpErr.Body[tc.field]; got != tc.message {
				t.Fatalf("unsupported operation response = %#v, want %s=%q", httpErr.Body, tc.field, tc.message)
			}
		})
	}
}

func TestInvalidAssetAttachmentDeleteChecksWorkspaceBefore405(t *testing.T) {
	userID, projectID, issueID := uuid.New(), uuid.New(), uuid.New()
	payload := &planeinternaldjangomigrationapi.DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachmentsPayload{
		Slug: "private", ProjectID: projectID.String(), IssueID: issueID.String(),
	}
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	service := &LifecycleService{assets: &Service{repository: invalidMethodRepository{
		workspaceErr: &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "membership"}},
	}}}
	_, err := service.invoke(ctx, "DeleteAPIAssetsV2WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDAttachments", payload, nil)
	var httpErr *djangoapi.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != 403 {
		t.Fatalf("nonmember invalid DELETE error = %#v, want HTTP 403", err)
	}
	if got := httpErr.Body["error"]; got != "You don't have the required permissions." {
		t.Fatalf("permission body = %#v", got)
	}
}
