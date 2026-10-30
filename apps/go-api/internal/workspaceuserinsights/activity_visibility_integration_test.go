//go:build integration

package workspaceuserinsights

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
)

func activityPageQuery() djangoapi.Query {
	return djangoapi.Query{Params: map[string]string{"per_page": "10"}, Present: map[string]bool{"per_page": true}}
}

func TestPropertyActivityFeedsEnforceOwningItemVisibility(t *testing.T) {
	db, ctx := startWorkspaceUserInsightsDB(t)
	if _, err := db.Exec(ctx, `UPDATE workspace_members SET role=15; UPDATE issue_activities SET field='custom_property',old_value='Private old value',new_value='Private new value',comment='Secret property'`); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{
		`UPDATE project_members SET role=5; UPDATE projects SET guest_view_all_features=false; UPDATE issues SET created_by_id=NULL`,
		`UPDATE issues SET is_draft=true`,
		`UPDATE issues SET deleted_at=now()`,
		`UPDATE projects SET deleted_at=now()`,
		`UPDATE workspaces SET deleted_at=now()`,
		`UPDATE workspace_members SET deleted_at=now()`,
		`UPDATE project_members SET deleted_at=now()`,
		`UPDATE project_members SET is_active=false`,
		`UPDATE issue_activities SET deleted_at=now()`,
	} {
		t.Run(mutation, func(t *testing.T) {
			tx, err := db.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { testsupport.Rollback(t, tx, ctx) }()
			if _, err = tx.Exec(ctx, mutation); err != nil {
				t.Fatal(err)
			}
			repo, err := NewRepository(tx)
			if err != nil {
				t.Fatal(err)
			}
			body, err := repo.Activity(ctx, insightsRequesterID, "demo", insightsTargetID, nil, activityPageQuery())
			if err != nil {
				t.Fatal(err)
			}
			var page struct {
				Total   int               `json:"total_count"`
				Results []json.RawMessage `json:"results"`
			}
			if err = json.Unmarshal(body, &page); err != nil || page.Total != 0 || len(page.Results) != 0 || strings.Contains(string(body), "Private") {
				t.Fatalf("hidden activity leaked: %s %v", body, err)
			}
			rows, err := repo.ExportActivity(ctx, insightsRequesterID, "demo", insightsTargetID, "2026-01-01")
			if err != nil || len(rows) != 0 {
				t.Fatalf("hidden export leaked: %+v %v", rows, err)
			}
		})
	}
}

func TestActivityExportUsesRequesterMembershipInsteadOfActor(t *testing.T) {
	db, ctx := startWorkspaceUserInsightsDB(t)
	if _, err := db.Exec(ctx, `UPDATE workspace_members SET role=15; UPDATE project_members SET is_active=false`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO project_members(workspace_id,project_id,member_id,role,is_active) VALUES($1,$2,$3,20,true)`, insightsWorkspaceID, insightsProject1ID, insightsTargetID); err != nil {
		t.Fatal(err)
	}
	repo, err := NewRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := repo.ExportActivity(ctx, insightsRequesterID, "demo", insightsTargetID, "2026-01-01")
	if err != nil || len(rows) != 0 {
		t.Fatalf("actor access authorized requester: %+v %v", rows, err)
	}
	if _, err = db.Exec(ctx, `UPDATE project_members SET is_active=member_id=$1`, insightsRequesterID); err != nil {
		t.Fatal(err)
	}
	rows, err = repo.ExportActivity(ctx, insightsRequesterID, "demo", insightsTargetID, "2026-01-01")
	if err != nil || len(rows) != 1 {
		t.Fatalf("requester could not export visible history of former actor: %+v %v", rows, err)
	}
}

type revokeActivityAccess struct {
	*pgx.Conn
	preflight bool
}

func (db *revokeActivityAccess) revoke(ctx context.Context) {
	if db.preflight {
		_, _ = db.Exec(ctx, `UPDATE workspace_members SET is_active=false`)
	}
	db.preflight = true
}
func (db *revokeActivityAccess) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	db.revoke(ctx)
	return db.Conn.QueryRow(ctx, sql, args...)
}
func (db *revokeActivityAccess) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	db.revoke(ctx)
	return db.Conn.Query(ctx, sql, args...)
}

func TestActivityResultQueriesRecheckRevokedWorkspaceMembership(t *testing.T) {
	for _, export := range []bool{false, true} {
		t.Run(map[bool]string{false: "feed", true: "export"}[export], func(t *testing.T) {
			db, ctx := startWorkspaceUserInsightsDB(t)
			if _, err := db.Exec(ctx, `UPDATE workspace_members SET role=15`); err != nil {
				t.Fatal(err)
			}
			repo, err := NewRepository(&revokeActivityAccess{Conn: db})
			if err != nil {
				t.Fatal(err)
			}
			if export {
				rows, readErr := repo.ExportActivity(ctx, insightsRequesterID, "demo", insightsTargetID, "2026-01-01")
				if readErr != nil || len(rows) != 0 {
					t.Fatalf("revoked export: %+v %v", rows, readErr)
				}
			} else {
				body, readErr := repo.Activity(ctx, insightsRequesterID, "demo", insightsTargetID, nil, activityPageQuery())
				if readErr != nil {
					t.Fatal(readErr)
				}
				var page struct{ Results []json.RawMessage }
				if err = json.Unmarshal(body, &page); err != nil || len(page.Results) != 0 {
					t.Fatalf("revoked feed: %s %v", body, err)
				}
			}
		})
	}
}
