//go:build integration

package assets

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublicAssetAuthorizationPostgres(t *testing.T) {
	databaseURL := postgresfixture.EmptyDatabase(t)
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse isolated public assets database URL: %v", err)
	}
	config.MaxConns = 1 // Keep PostgreSQL TEMP tables visible to each repository call.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect to isolated public assets database: %v", err)
	}
	defer database.Close()
	if _, execErr := database.Exec(ctx, `
CREATE TEMP TABLE workspaces (id uuid PRIMARY KEY, slug text, deleted_at timestamptz);
CREATE TEMP TABLE projects (id uuid PRIMARY KEY, workspace_id uuid, deleted_at timestamptz, archived_at timestamptz);
CREATE TEMP TABLE deploy_boards (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, entity_name text, entity_identifier uuid, anchor text, deleted_at timestamptz, is_disabled boolean, created_at timestamptz);
CREATE TEMP TABLE issue_comments (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, created_by_id uuid, deleted_at timestamptz);
CREATE TEMP TABLE issues (id uuid PRIMARY KEY, workspace_id uuid, project_id uuid, deleted_at timestamptz);
CREATE TEMP TABLE file_assets (
 id uuid PRIMARY KEY, created_at timestamptz, updated_at timestamptz, attributes jsonb, asset text,
 workspace_id uuid, project_id uuid, comment_id uuid, issue_id uuid, draft_issue_id uuid, entity_type text, size double precision,
 is_uploaded boolean, is_deleted boolean, is_archived boolean, storage_metadata jsonb,
	 user_id uuid, created_by_id uuid, deleted_at timestamptz
)`); execErr != nil {
		t.Fatalf("create isolated public assets TEMP tables: %v", execErr)
	}
	repository, err := NewPostgresRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	actorID, foreignID := uuid.New(), uuid.New()
	workspaceID, projectID, foreignProjectID := uuid.New(), uuid.New(), uuid.New()
	boardID, anchor := uuid.New(), "public-assets-"+uuid.NewString()
	commentID, foreignCommentID, otherUserCommentID := uuid.New(), uuid.New(), uuid.New()
	issueID := uuid.New()
	newCommentID := uuid.New()
	for _, seed := range []struct {
		query string
		args  pgx.StrictNamedArgs
	}{
		{`INSERT INTO workspaces(id,slug) VALUES (@id,'public-assets-test')`, pgx.StrictNamedArgs{"id": workspaceID}},
		{`INSERT INTO projects(id,workspace_id) VALUES (@project,@workspace),(@foreign_project,@workspace)`, pgx.StrictNamedArgs{"workspace": workspaceID, "project": projectID, "foreign_project": foreignProjectID}},
		{`INSERT INTO deploy_boards(id,workspace_id,project_id,entity_name,entity_identifier,anchor,is_disabled,created_at) VALUES (@board,@workspace,@project,'project',@project,@anchor,false,NOW())`, pgx.StrictNamedArgs{"board": boardID, "workspace": workspaceID, "project": projectID, "anchor": anchor}},
		{`INSERT INTO issue_comments(id,workspace_id,project_id,created_by_id) VALUES (@comment,@workspace,@project,@actor),(@foreign_project_comment,@workspace,@foreign_project,@actor),(@foreign_user_comment,@workspace,@project,@foreign_user),(@second_comment,@workspace,@project,@actor)`, pgx.StrictNamedArgs{"workspace": workspaceID, "project": projectID, "foreign_project": foreignProjectID, "actor": actorID, "comment": commentID, "foreign_project_comment": foreignCommentID, "foreign_user_comment": otherUserCommentID, "foreign_user": foreignID, "second_comment": newCommentID}},
		{`INSERT INTO issues(id,workspace_id,project_id) VALUES (@issue,@workspace,@project)`, pgx.StrictNamedArgs{"issue": issueID, "workspace": workspaceID, "project": projectID}},
	} {
		if _, execErr := database.Exec(ctx, seed.query, seed.args); execErr != nil {
			t.Fatalf("seed public assets authorization TEMP rows: %v", execErr)
		}
	}
	gotWorkspaceID, gotProjectID, _, found, err := repository.PublicProjectBoard(ctx, anchor, false)
	if err != nil || !found || gotWorkspaceID != workspaceID || gotProjectID != projectID {
		t.Fatalf("resolve live enabled board = %s/%s found=%t err=%v", gotWorkspaceID, gotProjectID, found, err)
	}
	for _, disabled := range []struct {
		name  string
		setup string
		clear string
	}{
		{"disabled board", `UPDATE deploy_boards SET is_disabled=true WHERE id=$1`, `UPDATE deploy_boards SET is_disabled=false WHERE id=$1`},
		{"deleted board", `UPDATE deploy_boards SET deleted_at=NOW() WHERE id=$1`, `UPDATE deploy_boards SET deleted_at=NULL WHERE id=$1`},
		{"mismatched board project", `UPDATE deploy_boards SET project_id=$2 WHERE id=$1`, `UPDATE deploy_boards SET project_id=$2 WHERE id=$1`},
		{"deleted workspace", `UPDATE workspaces SET deleted_at=NOW() WHERE id=$1`, `UPDATE workspaces SET deleted_at=NULL WHERE id=$1`},
		{"deleted project", `UPDATE projects SET deleted_at=NOW() WHERE id=$1`, `UPDATE projects SET deleted_at=NULL WHERE id=$1`},
		{"archived project", `UPDATE projects SET archived_at=NOW() WHERE id=$1`, `UPDATE projects SET archived_at=NULL WHERE id=$1`},
	} {
		t.Run(disabled.name, func(t *testing.T) {
			var args []any
			if disabled.name == "mismatched board project" {
				args = []any{boardID, foreignProjectID}
			} else if strings.Contains(disabled.setup, "workspaces") {
				args = []any{workspaceID}
			} else if strings.Contains(disabled.setup, "projects") {
				args = []any{projectID}
			} else {
				args = []any{boardID}
			}
			if _, execErr := database.Exec(ctx, disabled.setup, args...); execErr != nil {
				t.Fatal(execErr)
			}
			if _, _, _, found, publicProjectBoardErr := repository.PublicProjectBoard(ctx, anchor, false); publicProjectBoardErr != nil || found {
				t.Fatalf("unusable board found=%t err=%v, want unavailable", found, publicProjectBoardErr)
			}
			clearArgs := args
			if disabled.name == "mismatched board project" {
				clearArgs = []any{boardID, projectID}
			}
			if _, execErr := database.Exec(ctx, disabled.clear, clearArgs...); execErr != nil {
				t.Fatal(execErr)
			}
		})
	}

	create := func(assetID, userID uuid.UUID, entityType string, entityID string) error {
		return repository.CreatePublicUpload(ctx, assetID, userID, workspaceID, projectID, "asset-"+assetID.String(), []byte(`{"name":"image.png"}`), 10, entityType, entityID)
	}
	ownedAssetID, mixedAssetID, foreignAssetID, pendingAssetID, issueAssetID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	if createErr := create(ownedAssetID, actorID, "COMMENT_DESCRIPTION", commentID.String()); createErr != nil {
		t.Fatalf("create owner comment upload: %v", createErr)
	}
	if createErr := create(mixedAssetID, actorID, "COMMENT_DESCRIPTION", ""); createErr != nil {
		t.Fatalf("create pre-comment upload: %v", createErr)
	}
	if createErr := create(pendingAssetID, actorID, "COMMENT_DESCRIPTION", ""); createErr != nil {
		t.Fatalf("create second pre-comment upload: %v", createErr)
	}
	if createErr := create(issueAssetID, actorID, "ISSUE_DESCRIPTION", issueID.String()); createErr != nil {
		t.Fatalf("create issue description upload: %v", createErr)
	}
	var linkedIssueID, linkedCommentID *uuid.UUID
	if scanErr := database.QueryRow(ctx, `SELECT issue_id,comment_id FROM file_assets WHERE id=$1`, issueAssetID).Scan(&linkedIssueID, &linkedCommentID); scanErr != nil {
		t.Fatal(scanErr)
	}
	if linkedIssueID == nil || *linkedIssueID != issueID || linkedCommentID != nil {
		t.Fatalf("issue description relations issue=%v comment=%v, want issue %s and no comment", linkedIssueID, linkedCommentID, issueID)
	}
	if scanErr := database.QueryRow(ctx, `SELECT comment_id FROM file_assets WHERE id=$1`, mixedAssetID).Scan(&linkedCommentID); scanErr != nil {
		t.Fatal(scanErr)
	}
	if linkedCommentID != nil {
		t.Fatalf("pre-comment upload unexpectedly linked comment %s", *linkedCommentID)
	}
	if createErr := create(uuid.New(), actorID, "COMMENT_DESCRIPTION", foreignCommentID.String()); !isPublicNotFound(createErr) {
		t.Fatalf("foreign-project comment upload error=%v, want not-found", createErr)
	}
	if createErr := create(uuid.New(), actorID, "COMMENT_DESCRIPTION", otherUserCommentID.String()); !isPublicNotFound(createErr) {
		t.Fatalf("other-user comment upload error=%v, want not-found", createErr)
	}
	if bulkAssociateErr := repository.BulkAssociatePublicAssetComments(ctx, actorID, workspaceID, projectID, otherUserCommentID, []uuid.UUID{mixedAssetID}); !isPublicNotFound(bulkAssociateErr) {
		t.Fatalf("other-user target comment bulk association error=%v, want not-found", bulkAssociateErr)
	}
	if createErr := create(foreignAssetID, foreignID, "COMMENT_DESCRIPTION", otherUserCommentID.String()); createErr != nil {
		t.Fatalf("create foreign owner's upload: %v", createErr)
	}
	nullOwnerAssetID := uuid.New()
	if _, execErr := database.Exec(ctx, `INSERT INTO file_assets(id,created_at,updated_at,attributes,asset,workspace_id,project_id,entity_type,size,is_uploaded,is_deleted,is_archived,storage_metadata)
		VALUES (@id,NOW(),NOW(),'{}'::jsonb,'legacy-null-owner',@workspace,@project,'COMMENT_DESCRIPTION',1,false,false,false,'{}'::jsonb)`, pgx.StrictNamedArgs{"id": nullOwnerAssetID, "workspace": workspaceID, "project": projectID}); execErr != nil {
		t.Fatalf("seed legacy asset with nullable creator: %v", execErr)
	}

	asset, err := repository.PublicOwnedProjectAsset(ctx, actorID, workspaceID, projectID, ownedAssetID, false)
	if err != nil || asset.ID != ownedAssetID.String() {
		t.Fatalf("owner lookup = %#v err=%v", asset, err)
	}
	if _, err := repository.PublicOwnedProjectAsset(ctx, foreignID, workspaceID, projectID, ownedAssetID, false); !isPublicNotFound(err) {
		t.Fatalf("foreign owner lookup error=%v, want not-found", err)
	}
	if _, err := repository.PublicOwnedProjectAsset(ctx, actorID, workspaceID, projectID, nullOwnerAssetID, false); !isPublicNotFound(err) {
		t.Fatalf("nullable creator lookup error=%v, want not-found", err)
	}
	if err := repository.UpdatePublicAsset(ctx, foreignID, workspaceID, projectID, ownedAssetID, []byte(`{}`)); !isPublicNotFound(err) {
		t.Fatalf("foreign confirm error=%v, want not-found", err)
	}
	if err := repository.UpdatePublicAsset(ctx, actorID, workspaceID, projectID, ownedAssetID, []byte(`{"confirmed":true}`)); err != nil {
		t.Fatalf("owner confirm: %v", err)
	}
	if err := repository.DeletePublicAsset(ctx, foreignID, workspaceID, projectID, ownedAssetID); !isPublicNotFound(err) {
		t.Fatalf("foreign delete error=%v, want not-found", err)
	}
	if err := repository.DeletePublicAsset(ctx, actorID, workspaceID, projectID, ownedAssetID); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if err := repository.RestorePublicAsset(ctx, foreignID, workspaceID, projectID, ownedAssetID); !isPublicNotFound(err) {
		t.Fatalf("foreign restore error=%v, want not-found", err)
	}
	if err := repository.RestorePublicAsset(ctx, actorID, workspaceID, projectID, ownedAssetID); err != nil {
		t.Fatalf("owner restore: %v", err)
	}

	if _, _, err := repository.PublicBulkAssetRows(ctx, actorID, workspaceID, projectID, []uuid.UUID{ownedAssetID, foreignAssetID}); !isPublicNotFound(err) {
		t.Fatalf("mixed-owner bulk selection error=%v, want not-found", err)
	}
	if err := repository.BulkAssociatePublicAssetComments(ctx, actorID, workspaceID, projectID, newCommentID, []uuid.UUID{ownedAssetID, foreignAssetID}); !isPublicNotFound(err) {
		t.Fatalf("mixed-owner bulk mutation error=%v, want not-found", err)
	}
	var commentAfterMixed *uuid.UUID
	if err := database.QueryRow(ctx, `SELECT comment_id FROM file_assets WHERE id=$1`, ownedAssetID).Scan(&commentAfterMixed); err != nil {
		t.Fatal(err)
	}
	if commentAfterMixed == nil || *commentAfterMixed != commentID {
		t.Fatalf("mixed request changed pre-associated asset comment_id=%v, want original %s", commentAfterMixed, commentID)
	}
	if err := repository.BulkAssociatePublicAssetComments(ctx, actorID, workspaceID, projectID, otherUserCommentID, []uuid.UUID{mixedAssetID, pendingAssetID}); !isPublicNotFound(err) {
		t.Fatalf("foreign target comment error=%v, want not-found", err)
	}
	if err := database.QueryRow(ctx, `SELECT comment_id FROM file_assets WHERE id=$1`, mixedAssetID).Scan(&commentAfterMixed); err != nil {
		t.Fatal(err)
	}
	if commentAfterMixed != nil {
		t.Fatalf("foreign target comment partially associated asset to %s", *commentAfterMixed)
	}
	if err := repository.BulkAssociatePublicAssetComments(ctx, actorID, workspaceID, projectID, commentID, []uuid.UUID{mixedAssetID, pendingAssetID}); err != nil {
		t.Fatalf("nonmember owner bulk association to own comment: %v", err)
	}
	if err := database.QueryRow(ctx, `SELECT comment_id FROM file_assets WHERE id=$1`, mixedAssetID).Scan(&commentAfterMixed); err != nil {
		t.Fatal(err)
	}
	if commentAfterMixed == nil || *commentAfterMixed != commentID {
		t.Fatalf("valid owner comment association = %v, want %s", commentAfterMixed, commentID)
	}
}

func isPublicNotFound(err error) bool {
	var httpErr *djangoapi.HTTPError
	return errors.As(err, &httpErr) && httpErr.Status == 404
}
