package assets

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/assetslifecycle/gen/plane_internal_django_migration_api"
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
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
	service := &LifecycleService{assets: &Service{repository: invalidMethodRepository{}}}

	_, err := service.invoke(ctx, "DeleteAPIAssetsV2UserAssets", &planeinternaldjangomigrationapi.DeleteAPIAssetsV2UserAssetsPayload{}, nil)
	var httpErr *djangoapi.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != 405 {
		t.Fatalf("invalid user collection method error = %#v, want HTTP 405", err)
	}
	if got := httpErr.Body["detail"]; got != `Method "DELETE" not allowed.` {
		t.Fatalf("405 detail = %#v", got)
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
