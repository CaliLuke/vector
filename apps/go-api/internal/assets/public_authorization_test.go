package assets

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

type publicOwnershipRepository struct {
	assetRepository
	userID, workspaceID, projectID, assetID, commentID uuid.UUID
	asset                                              FileAsset
	updated, deleted, restored, associated             int
	lastActor                                          uuid.UUID
}

func (r *publicOwnershipRepository) PublicProjectBoard(context.Context, string, bool) (uuid.UUID, uuid.UUID, string, bool, error) {
	return r.workspaceID, r.projectID, "published", true, nil
}

func (r *publicOwnershipRepository) PublicOwnedProjectAsset(_ context.Context, actorID, workspaceID, projectID, assetID uuid.UUID, _ bool) (FileAsset, error) {
	if actorID != r.userID || workspaceID != r.workspaceID || projectID != r.projectID || assetID != r.assetID {
		return FileAsset{}, &djangoapi.HTTPError{Status: 404}
	}
	return r.asset, nil
}

func (r *publicOwnershipRepository) UpdatePublicAsset(_ context.Context, actorID, workspaceID, projectID, assetID uuid.UUID, _ []byte) error {
	r.lastActor = actorID
	if actorID != r.userID || workspaceID != r.workspaceID || projectID != r.projectID || assetID != r.assetID {
		return &djangoapi.HTTPError{Status: 404}
	}
	r.updated++
	return nil
}

func (r *publicOwnershipRepository) DeletePublicAsset(_ context.Context, actorID, workspaceID, projectID, assetID uuid.UUID) error {
	r.lastActor = actorID
	if actorID != r.userID || workspaceID != r.workspaceID || projectID != r.projectID || assetID != r.assetID {
		return &djangoapi.HTTPError{Status: 404}
	}
	r.deleted++
	return nil
}

func (r *publicOwnershipRepository) RestorePublicAsset(_ context.Context, actorID, workspaceID, projectID, assetID uuid.UUID) error {
	r.lastActor = actorID
	if actorID != r.userID || workspaceID != r.workspaceID || projectID != r.projectID || assetID != r.assetID {
		return &djangoapi.HTTPError{Status: 404}
	}
	r.restored++
	return nil
}

func (r *publicOwnershipRepository) PublicBulkAssetRows(_ context.Context, actorID, workspaceID, projectID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, string, error) {
	r.lastActor = actorID
	if actorID != r.userID || workspaceID != r.workspaceID || projectID != r.projectID || len(ids) != 1 || ids[0] != r.assetID {
		return nil, "", &djangoapi.HTTPError{Status: 404}
	}
	return ids, "COMMENT_DESCRIPTION", nil
}

func (r *publicOwnershipRepository) BulkAssociatePublicAssetComments(_ context.Context, actorID, workspaceID, projectID, commentID uuid.UUID, ids []uuid.UUID) error {
	r.lastActor = actorID
	if actorID != r.userID || workspaceID != r.workspaceID || projectID != r.projectID || commentID != r.commentID || len(ids) != 1 || ids[0] != r.assetID {
		return &djangoapi.HTTPError{Status: 404}
	}
	r.associated++
	return nil
}

type publicTaskRecorder struct{ calls int }

func (r *publicTaskRecorder) Task(context.Context, string, []any, map[string]any) error {
	r.calls++
	return nil
}

func TestPublicAssetOwnerKeepsNonmemberEditSequenceAndBlocksForeignWrites(t *testing.T) {
	actorID, foreignID := uuid.New(), uuid.New()
	workspaceID, projectID, assetID, commentID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := &publicOwnershipRepository{
		userID: actorID, workspaceID: workspaceID, projectID: projectID, assetID: assetID, commentID: commentID,
		asset: FileAsset{ID: assetID.String(), Attributes: map[string]any{"name": "upload.png"}},
	}
	tasks := &publicTaskRecorder{}
	service := &LifecycleService{assets: &Service{repository: repository, tasks: tasks}}
	ctx := context.WithValue(context.Background(), assetUserKey{}, actorID)

	if err := service.confirmPublicAsset(ctx, "published", assetID.String(), io.NopCloser(strings.NewReader(`{"attributes":{"name":"upload.png"}}`))); err != nil {
		t.Fatalf("owner confirm: %v", err)
	}
	if repository.updated != 1 || tasks.calls != 1 {
		t.Fatalf("owner confirm effects updated=%d tasks=%d, want one update and metadata task", repository.updated, tasks.calls)
	}
	if err := service.bulkAssociatePublicAssets(ctx, "published", commentID.String(), io.NopCloser(strings.NewReader(`{"asset_ids":["`+assetID.String()+`"]}`))); err != nil {
		t.Fatalf("owner associate upload with own comment: %v", err)
	}
	if repository.associated != 1 {
		t.Fatalf("owner comment association count=%d, want 1", repository.associated)
	}
	if err := service.deletePublicAsset(ctx, "published", assetID.String()); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if err := service.restorePublicAsset(ctx, "published", assetID.String()); err != nil {
		t.Fatalf("owner restore: %v", err)
	}
	if repository.deleted != 1 || repository.restored != 1 {
		t.Fatalf("owner delete/restore = %d/%d, want one each", repository.deleted, repository.restored)
	}

	foreignCtx := context.WithValue(context.Background(), assetUserKey{}, foreignID)
	beforeTasks, beforeUpdates := tasks.calls, repository.updated
	if err := service.confirmPublicAsset(foreignCtx, "published", assetID.String(), io.NopCloser(strings.NewReader(`{"attributes":{}}`))); err == nil {
		t.Fatal("foreign actor confirmed another user's asset")
	}
	if err := service.deletePublicAsset(foreignCtx, "published", assetID.String()); err == nil {
		t.Fatal("foreign actor deleted another user's asset")
	}
	if err := service.restorePublicAsset(foreignCtx, "published", assetID.String()); err == nil {
		t.Fatal("foreign actor restored another user's asset")
	}
	if err := service.bulkAssociatePublicAssets(foreignCtx, "published", commentID.String(), io.NopCloser(strings.NewReader(`{"asset_ids":["`+assetID.String()+`"]}`))); err == nil {
		t.Fatal("foreign actor associated another user's asset")
	}
	if tasks.calls != beforeTasks || repository.updated != beforeUpdates || repository.deleted != 1 || repository.restored != 1 || repository.associated != 1 {
		t.Fatalf("foreign requests changed effects: tasks=%d updates=%d deletes=%d restores=%d associations=%d", tasks.calls, repository.updated, repository.deleted, repository.restored, repository.associated)
	}
}
