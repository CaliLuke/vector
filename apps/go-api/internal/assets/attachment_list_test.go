package assets

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

type issueAttachmentListRepository struct {
	assetRepository
	workspaceID  uuid.UUID
	calls        []string
	projectID    uuid.UUID
	issueID      uuid.UUID
	uploadedOnly bool
}

func (repository *issueAttachmentListRepository) ProjectPermissionAllowed(context.Context, uuid.UUID, string, uuid.UUID) (bool, error) {
	repository.calls = append(repository.calls, "ProjectPermissionAllowed")
	return true, nil
}

func (repository *issueAttachmentListRepository) RelatedWorkspaceIDBySlug(context.Context, string) (uuid.UUID, error) {
	repository.calls = append(repository.calls, "RelatedWorkspaceIDBySlug")
	return repository.workspaceID, nil
}

func (repository *issueAttachmentListRepository) IssueAttachments(_ context.Context, _, projectID, issueID uuid.UUID, uploadedOnly bool) ([]map[string]any, error) {
	repository.calls = append(repository.calls, "IssueAttachments")
	repository.projectID = projectID
	repository.issueID = issueID
	repository.uploadedOnly = uploadedOnly
	return nil, nil
}

func TestIssueAttachmentListWrappersPreserveUploadFilterAndScopeOrder(t *testing.T) {
	userID, workspaceID := uuid.New(), uuid.New()
	projectID, issueID := uuid.New(), uuid.New()
	for _, test := range []struct {
		name         string
		list         func(*LifecycleService, context.Context, string, string, string) (any, error)
		uploadedOnly bool
	}{
		{
			name: "legacy includes all issue attachments",
			list: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID string) (any, error) {
				return service.listLegacyIssueAttachments(ctx, slug, projectID, issueID)
			},
			uploadedOnly: false,
		},
		{
			name: "current collection includes uploaded attachments only",
			list: func(service *LifecycleService, ctx context.Context, slug, projectID, issueID string) (any, error) {
				return service.listIssueAttachments(ctx, slug, projectID, issueID)
			},
			uploadedOnly: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &issueAttachmentListRepository{workspaceID: workspaceID}
			service := &LifecycleService{assets: &Service{repository: repository}}
			ctx := context.WithValue(context.Background(), assetUserKey{}, userID)
			if _, err := test.list(service, ctx, "workspace", projectID.String(), issueID.String()); err != nil {
				t.Fatalf("list attachments: %v", err)
			}
			if !reflect.DeepEqual(repository.calls, []string{"ProjectPermissionAllowed", "RelatedWorkspaceIDBySlug", "IssueAttachments"}) {
				t.Fatalf("repository calls=%v", repository.calls)
			}
			if repository.projectID != projectID || repository.issueID != issueID || repository.uploadedOnly != test.uploadedOnly {
				t.Fatalf("attachment query project=%s issue=%s uploadedOnly=%t", repository.projectID, repository.issueID, repository.uploadedOnly)
			}
		})
	}
}
