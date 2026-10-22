//go:build integration

package assets

import (
	"context"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestIssueAttachmentRepositoryBindsReadAndWritesToPathIssue(t *testing.T) {
	ctx := context.Background()
	const (
		workspaceID = "00000000-0000-4000-8000-000000000001"
		projectID   = "00000000-0000-4000-8000-000000000002"
		issueA      = "00000000-0000-4000-8000-000000000003"
		issueB      = "00000000-0000-4000-8000-000000000004"
		userID      = "00000000-0000-4000-8000-000000000005"
		assetID     = "00000000-0000-4000-8000-000000000006"
	)
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer connection.Close(context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE file_assets (
  id uuid PRIMARY KEY, asset text NOT NULL, entity_type text NOT NULL, attributes jsonb NOT NULL,
  workspace_id uuid NOT NULL, project_id uuid NOT NULL, issue_id uuid, user_id uuid,
  is_uploaded boolean NOT NULL, is_deleted boolean NOT NULL, size double precision NOT NULL,
  storage_metadata jsonb NOT NULL, comment_id uuid, created_at timestamptz NOT NULL,
  created_by_id uuid, deleted_at timestamptz, draft_issue_id uuid, entity_identifier text,
  external_id text, external_source text, is_archived boolean NOT NULL, page_id uuid,
  updated_at timestamptz NOT NULL, updated_by_id uuid
);
`)
		if err != nil {
			return err
		}
		_, err = connection.Exec(ctx, `
INSERT INTO file_assets (
  id,asset,entity_type,attributes,workspace_id,project_id,issue_id,user_id,is_uploaded,is_deleted,
  size,storage_metadata,created_at,deleted_at,is_archived,updated_at
) VALUES (
  $1,'objects/issue-b.png','ISSUE_ATTACHMENT','{"name":"issue-b.png"}'::jsonb,$2,$3,$4,$5,false,false,
  12,'{}'::jsonb,NOW(),NULL,false,NOW()
);
`, assetID, workspaceID, projectID, issueB, userID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Close()
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.WithoutCancel(ctx))
	repository, err := NewPostgresRepository(connection)
	if err != nil {
		t.Fatal(err)
	}
	workspace, project, pathIssue, owningIssue, actor, asset := parseScopeTestUUID(t, workspaceID), parseScopeTestUUID(t, projectID), parseScopeTestUUID(t, issueA), parseScopeTestUUID(t, issueB), parseScopeTestUUID(t, userID), parseScopeTestUUID(t, assetID)

	_, _, err = repository.IssueAttachment(ctx, workspace, project, pathIssue, asset)
	assertScopeNotFound(t, err, "read attachment under another issue")
	if _, _, err := repository.IssueAttachment(ctx, workspace, project, owningIssue, asset); err != nil {
		t.Fatalf("read attachment under owning issue: %v", err)
	}

	assertScopeNotFound(t, repository.UpdateIssueAttachmentUploaded(ctx, actor, workspace, project, pathIssue, asset), "confirm attachment under another issue")
	assertScopeNotFound(t, repository.DeleteIssueAttachment(ctx, actor, workspace, project, pathIssue, asset), "delete attachment under another issue")
	assertAttachmentState(t, connection, assetID, false, false)

	if err := repository.UpdateIssueAttachmentUploaded(ctx, actor, workspace, project, owningIssue, asset); err != nil {
		t.Fatalf("confirm attachment under owning issue: %v", err)
	}
	if err := repository.DeleteIssueAttachment(ctx, actor, workspace, project, owningIssue, asset); err != nil {
		t.Fatalf("delete attachment under owning issue: %v", err)
	}
	assertAttachmentState(t, connection, assetID, true, true)
}

func parseScopeTestUUID(t *testing.T, value string) uuid.UUID {
	t.Helper()
	parsed, err := uuid.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
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
