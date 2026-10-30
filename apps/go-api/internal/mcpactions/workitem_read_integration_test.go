//go:build integration

package mcpactions_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/activityreads"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const childReadSchema = `
ALTER TABLE issues ADD COLUMN created_by_id uuid;
CREATE TABLE issue_comments(id uuid PRIMARY KEY, created_at timestamptz DEFAULT '2024-01-02T00:00:00Z', updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, project_id uuid, workspace_id uuid, comment_stripped text, comment_json jsonb,
 comment_html text, attachments jsonb DEFAULT '[]', access text DEFAULT 'INTERNAL', external_source text, external_id text,
 edited_at timestamptz, description_id uuid, issue_id uuid, actor_id uuid, parent_id uuid);
CREATE TABLE descriptions(id uuid PRIMARY KEY, project_id uuid, workspace_id uuid, deleted_at timestamptz);
CREATE TABLE issue_links(id uuid PRIMARY KEY, created_at timestamptz DEFAULT '2024-01-02T00:00:00Z', updated_at timestamptz, deleted_at timestamptz,
 created_by_id uuid, updated_by_id uuid, project_id uuid, workspace_id uuid, title text, url text, metadata jsonb, issue_id uuid);
CREATE TABLE issue_activities(id uuid PRIMARY KEY, created_at timestamptz DEFAULT '2024-01-02T00:00:00Z', updated_at timestamptz, deleted_at timestamptz,
 project_id uuid, workspace_id uuid, issue_id uuid, actor_id uuid, issue_comment_id uuid, verb text, field text,
 old_value text, new_value text, comment text, attachments jsonb DEFAULT '[]', old_identifier uuid, new_identifier uuid, epoch double precision DEFAULT 1704153600);
INSERT INTO issue_comments(id,project_id,workspace_id,issue_id,comment_html)
 SELECT '90000000-0000-4000-8000-000000000001',project_id,workspace_id,id,'<p>Hello <mention-component entity_name="user_mention" entity_identifier="10000000-0000-4000-8000-000000000001"></mention-component></p>' FROM issues;
INSERT INTO issue_comments(id,project_id,workspace_id,issue_id,comment_html,created_at)
 SELECT '90000000-0000-4000-8000-000000000002',project_id,workspace_id,id,'older comment','2024-01-01' FROM issues;
INSERT INTO issue_links(id,project_id,workspace_id,issue_id,title,url,metadata)
 SELECT '90000000-0000-4000-8000-000000000001',project_id,workspace_id,id,'A link','https://example.test/a','{"created_at":"2024-01-02T00:00:00Z"}' FROM issues;
INSERT INTO issue_links(id,project_id,workspace_id,issue_id,title,url,created_at)
 SELECT '90000000-0000-4000-8000-000000000002',project_id,workspace_id,id,'B link','https://example.test/b','2024-01-01' FROM issues;
INSERT INTO issue_activities(id,project_id,workspace_id,issue_id,verb,field)
 SELECT '90000000-0000-4000-8000-000000000001',project_id,workspace_id,id,'updated','name' FROM issues;
INSERT INTO issue_activities(id,project_id,workspace_id,issue_id,verb,field,created_at)
 SELECT '90000000-0000-4000-8000-000000000002',project_id,workspace_id,id,'updated','priority','2024-01-03' FROM issues;
INSERT INTO issue_activities(id,project_id,workspace_id,issue_id,field)
 SELECT gen_random_uuid(),project_id,workspace_id,id,field FROM issues CROSS JOIN unnest(ARRAY['comment','vote','reaction','draft']) field;
`

func addChildReadFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	addCycleFixture(t, pool)
	_, err := pool.Exec(t.Context(), childReadSchema)
	require.NoError(t, err)
}

func TestWorkitemChildReadsMatchPinnedContracts(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addChildReadFixture(t, pool)
	pat, oauth := f.handler(t, nil), oauthHandler(t, f, auth.Authenticate)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"PAT", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := func(tool, input string) rpcResult {
					if mode == "OAuth" {
						return oauthToolCall(t, oauth, version, "oauth-user", "workspace-one", tool, input, 200)
					}
					return rpcVersion(t, pat, "test-one", "tools/call", resourceCall(tool, input), version)
				}
				for _, tc := range []struct {
					tool                     string
					pageFields, recordFields int
				}{
					{"workitem_comment", 12, 20}, {"workitem_link", 12, 12}, {"workitem_activity", 7, 18},
				} {
					t.Run(tc.tool, func(t *testing.T) {
						page := cyclePageWire(t, call(tc.tool, strings.TrimSuffix(itemReadInput(tc.tool, "list"), "}")+`,"per_page":1}`))
						require.Len(t, page, tc.pageFields)
						require.Equal(t, float64(2), page["total_count"])
						require.Equal(t, "1:1:0", page["next_cursor"])
						require.Equal(t, true, page["next_page_results"])
						row := page["results"].([]any)[0].(map[string]any)
						require.Len(t, row, tc.recordFields)
						require.Equal(t, readChild, row["id"])
						detail := cyclePageWire(t, call(tc.tool, itemReadInput(tc.tool, "retrieve")))
						require.Equal(t, row, detail)
						next := cyclePageWire(t, call(tc.tool, strings.TrimSuffix(itemReadInput(tc.tool, "list"), "}")+`,"per_page":1,"cursor":"1:1:0"}`))
						require.Equal(t, "90000000-0000-4000-8000-000000000002", next["results"].([]any)[0].(map[string]any)["id"])
						require.Equal(t, true, next["prev_page_results"])
						zero := cyclePageWire(t, call(tc.tool, strings.TrimSuffix(itemReadInput(tc.tool, "list"), "}")+`,"per_page":0,"cursor":""}`))
						require.Equal(t, "1000:1:0", zero["next_cursor"])
					})
				}
				comment := cyclePageWire(t, call("workitem_comment", itemReadInput("workitem_comment", "retrieve")))
				require.Equal(t, "<p>Hello @[10000000-0000-4000-8000-000000000001]</p>", comment["comment_html"])
				require.Nil(t, comment["comment_stripped"])
				require.Equal(t, true, comment["is_member"])
				activity := cyclePageWire(t, call("workitem_activity", itemReadInput("workitem_activity", "retrieve")))
				require.Equal(t, float64(1704153600), activity["epoch"])
			})
		}
	}
}

type changingCollaborationReader struct {
	*workitems.Repository
	change func()
}

func (r changingCollaborationReader) ExecuteCollaboration(ctx context.Context, q workitems.Request, body io.Reader, contentType string) (workitems.Result, error) {
	r.change()
	return r.Repository.ExecuteCollaboration(ctx, q, body, contentType)
}

type changingActivityReader struct {
	*activityreads.Repository
	change func()
}

func (r changingActivityReader) Read(ctx context.Context, p activityreads.Principal, q activityreads.Request) (activityreads.Result, error) {
	r.change()
	return r.Repository.Read(ctx, p, q)
}

func TestWorkitemChildReadsRecheckVisibilityInDataSnapshot(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addChildReadFixture(t, pool)
	for _, mutation := range []string{
		`UPDATE users SET is_active=false`, `UPDATE workspaces SET deleted_at=now()`,
		`UPDATE workspace_members SET is_active=false`, `UPDATE workspace_members SET role=0`,
		`UPDATE project_members SET is_active=false`, `UPDATE project_members SET role=5`,
		`UPDATE project_members SET workspace_id='20000000-0000-4000-8000-000000000002'`,
		`UPDATE projects SET archived_at=now()`, `UPDATE projects SET deleted_at=now()`,
		`UPDATE issues SET deleted_at=now()`, `UPDATE issues SET is_draft=true`,
		`UPDATE workspaces SET slug='renamed' WHERE slug='workspace-one'; UPDATE workspaces SET slug='workspace-one' WHERE slug='workspace-two'`,
	} {
		t.Run(mutation, func(t *testing.T) {
			for _, action := range []string{"list", "retrieve"} {
				for _, tool := range []string{"workitem_comment", "workitem_link", "workitem_activity"} {
					tx, err := pool.Begin(t.Context())
					require.NoError(t, err)
					t.Cleanup(func() { testsupport.Rollback(t, tx, context.Background()) })
					collaboration, err := workitems.NewRepository(tx, rosterPublisher{}, "https://vector.example")
					require.NoError(t, err)
					activities, err := activityreads.NewRepository(tx)
					require.NoError(t, err)
					change := func() { _, changeErr := tx.Exec(t.Context(), mutation); require.NoError(t, changeErr) }
					apps := databaseApplications(t, pool)
					apps.Collaboration = changingCollaborationReader{collaboration, change}
					apps.Activities = changingActivityReader{activities, change}
					f.service, err = mcpactions.New(apps)
					require.NoError(t, err)
					h := oauthHandler(t, f, auth.Authenticate)
					result := oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", tool, itemReadInput(tool, action), 200)
					require.True(t, result.IsError, result.Content)
					require.Empty(t, result.StructuredContent)
					require.NotContains(t, result.Content[0].Text, "[internal_error]")
					require.NoError(t, tx.Rollback(t.Context()))
				}
			}
		})
	}
}

func TestWorkitemChildReadsHideForeignRelationsAndPreserveOpaqueMetadata(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addChildReadFixture(t, pool)
	_, err := pool.Exec(t.Context(), `
 UPDATE users SET user_timezone='America/Los_Angeles';
 INSERT INTO descriptions VALUES('70000000-0000-4000-8000-000000000009','30000000-0000-4000-8000-000000000003','20000000-0000-4000-8000-000000000002',NULL);
 UPDATE issue_comments SET description_id='70000000-0000-4000-8000-000000000009',parent_id='90000000-0000-4000-8000-000000000002' WHERE id='90000000-0000-4000-8000-000000000001';
 UPDATE issue_comments SET workspace_id='20000000-0000-4000-8000-000000000002' WHERE id='90000000-0000-4000-8000-000000000002';
 UPDATE issue_activities SET issue_comment_id='90000000-0000-4000-8000-000000000002';
 UPDATE issue_links SET workspace_id='20000000-0000-4000-8000-000000000002' WHERE id='90000000-0000-4000-8000-000000000002';
 `)
	require.NoError(t, err)
	h := oauthHandler(t, f, auth.Authenticate)
	call := func(tool, input string) rpcResult {
		return oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", tool, input, 200)
	}
	comment := cyclePageWire(t, call("workitem_comment", itemReadInput("workitem_comment", "retrieve")))
	require.Nil(t, comment["parent"])
	require.Nil(t, comment["description"])
	require.Equal(t, "2024-01-01T16:00:00-08:00", comment["created_at"])
	link := cyclePageWire(t, call("workitem_link", itemReadInput("workitem_link", "retrieve")))
	require.Equal(t, "2024-01-01T16:00:00-08:00", link["created_at"])
	require.Equal(t, "2024-01-02T00:00:00Z", link["metadata"].(map[string]any)["created_at"])
	activity := cyclePageWire(t, call("workitem_activity", itemReadInput("workitem_activity", "retrieve")))
	require.Nil(t, activity["issue_comment"])
	for _, tool := range []string{"workitem_comment", "workitem_link"} {
		page := cyclePageWire(t, call(tool, itemReadInput(tool, "list")))
		require.Equal(t, float64(1), page["total_count"])
		hidden := call(tool, strings.ReplaceAll(itemReadInput(tool, "retrieve"), readChild, "90000000-0000-4000-8000-000000000002"))
		require.True(t, hidden.IsError)
		require.Contains(t, hidden.Content[0].Text, "[not_found]")
	}
}

func TestWorkitemChildReadsAllowEligibleGuestsAndDistinguishEmptyParents(t *testing.T) {
	f, pool, auth, _ := oauthFixture(t)
	addChildReadFixture(t, pool)
	h := oauthHandler(t, f, auth.Authenticate)
	call := func(tool, input string) rpcResult {
		return oauthToolCall(t, h, "2026-07-28", "oauth-user", "workspace-one", tool, input, 200)
	}
	for _, mutation := range []string{
		`UPDATE project_members SET role=5; UPDATE issues SET created_by_id='10000000-0000-4000-8000-000000000001'`,
		`UPDATE issues SET created_by_id=NULL; UPDATE projects SET guest_view_all_features=true`,
	} {
		_, err := pool.Exec(t.Context(), mutation)
		require.NoError(t, err)
		for _, tool := range []string{"workitem_comment", "workitem_link", "workitem_activity"} {
			require.Equal(t, float64(2), cyclePageWire(t, call(tool, itemReadInput(tool, "list")))["total_count"])
		}
	}
	_, err := pool.Exec(t.Context(), `UPDATE issue_comments SET deleted_at=now(); UPDATE issue_links SET deleted_at=now(); UPDATE issue_activities SET deleted_at=now()`)
	require.NoError(t, err)
	for _, tool := range []string{"workitem_comment", "workitem_link", "workitem_activity"} {
		require.Empty(t, cyclePageWire(t, call(tool, itemReadInput(tool, "list")))["results"])
		missing := call(tool, itemReadInput(tool, "retrieve"))
		require.True(t, missing.IsError)
		require.Contains(t, missing.Content[0].Text, "[not_found]")
		invalid := call(tool, strings.TrimSuffix(itemReadInput(tool, "list"), "}")+`,"cursor":"invalid"}`)
		require.True(t, invalid.IsError)
		require.Contains(t, invalid.Content[0].Text, "[invalid_params]")
		absent := call(tool, strings.ReplaceAll(itemReadInput(tool, "list"), readIssue, "80000000-0000-4000-8000-000000000099"))
		require.True(t, absent.IsError)
		require.Contains(t, absent.Content[0].Text, "[not_found]")
	}
}
