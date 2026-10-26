//go:build integration

package assets

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestIssueAttachmentRepositoryBindsReadAndWritesToPathIssue(t *testing.T) {
	f, owningIssue, asset := newPublishedAttachmentFixture(t)
	ctx, repository, workspace, project, actor := f.ctx, f.repository, f.workspace, f.project, f.actor
	pathIssue := uuid.New()
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,created_by_id) VALUES($1,$2,$3,$4)`, pathIssue, workspace, project, actor)
	f.exec(`UPDATE file_assets SET is_uploaded=false WHERE id=$1`, asset)
	connection, assetID := f.db, asset.String()

	_, _, err := repository.IssueAttachment(ctx, actor, workspace, project, pathIssue, asset)
	assertScopeNotFound(t, err, "read attachment under another issue")
	if _, _, err := repository.IssueAttachment(ctx, actor, workspace, project, owningIssue, asset); err != nil {
		t.Fatalf("read attachment under owning issue: %v", err)
	}

	assertScopeNotFound(t, repository.UpdateIssueAttachmentUploaded(ctx, actor, workspace, project, pathIssue, asset, nil, false), "confirm attachment under another issue")
	assertScopeNotFound(t, repository.DeleteIssueAttachment(ctx, actor, workspace, project, pathIssue, asset, nil), "delete attachment under another issue")
	assertAttachmentState(t, connection, assetID, false, false)

	if err := repository.UpdateIssueAttachmentUploaded(ctx, actor, workspace, project, owningIssue, asset, nil, false); err != nil {
		t.Fatalf("confirm attachment under owning issue: %v", err)
	}
	if err := repository.DeleteIssueAttachment(ctx, actor, workspace, project, owningIssue, asset, nil); err != nil {
		t.Fatalf("delete attachment under owning issue: %v", err)
	}
	assertAttachmentState(t, connection, assetID, true, true)
}

func assertScopeNotFound(t *testing.T, err error, operation string) {
	t.Helper()
	var httpErr *djangoapi.HTTPError
	if err == nil || !errors.As(err, &httpErr) || httpErr.Status != 404 {
		t.Fatalf("%s error = %v, want HTTP 404", operation, err)
	}
}

func assertAttachmentState(t *testing.T, connection *pgx.Conn, assetID string, uploaded, deleted bool) {
	t.Helper()
	var gotUploaded, gotDeleted, hasDeletedAt bool
	if err := connection.QueryRow(context.Background(), `SELECT is_uploaded,is_deleted,deleted_at IS NOT NULL FROM file_assets WHERE id=$1::uuid`, assetID).Scan(&gotUploaded, &gotDeleted, &hasDeletedAt); err != nil {
		t.Fatal(err)
	}
	if gotUploaded != uploaded || gotDeleted != deleted || hasDeletedAt != deleted {
		t.Fatalf("attachment state uploaded=%t deleted=%t has_deleted_at=%t, want uploaded=%t deleted=%t", gotUploaded, gotDeleted, hasDeletedAt, uploaded, deleted)
	}
}
