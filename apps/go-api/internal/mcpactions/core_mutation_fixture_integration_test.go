//go:build integration

package mcpactions_test

import (
	"context"
	"fmt"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type coreMutationCase struct{ tool, action string }

var coreMutationCases = []coreMutationCase{
	{"workitem", "create"}, {"workitem", "update"}, {"workitem", "delete"}, {"workitem", "archive"}, {"workitem", "manage_assignee"}, {"workitem", "manage_label"},
	{"workitem_comment", "create"}, {"workitem_comment", "update"}, {"workitem_comment", "delete"},
	{"workitem_link", "create"}, {"workitem_link", "update"}, {"workitem_link", "delete"},
	{"intake", "create"}, {"intake", "update"}, {"intake", "delete"},
}

func coreFixture(t *testing.T) milestoneFixture {
	t.Helper()
	f := templateFixture(t)
	_, err := f.pool.Exec(t.Context(), `UPDATE projects SET intake_view=true WHERE id=$1`, testsupport.LabelProject)
	require.NoError(t, err)
	_, err = f.pool.Exec(t.Context(), `INSERT INTO intakes(id,name,description,is_default,view_props,logo_props,project_id,workspace_id,created_at,updated_at) VALUES($1,'Intake','',true,'{}','{}',$2,$3,now(),now())`, uuid.New(), testsupport.LabelProject, testsupport.LabelWorkspace)
	require.NoError(t, err)
	return f
}
func coreSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	return coreSnapshotQuery(t, t.Context(), f.pool.QueryRow)
}
func coreSnapshotQuery(t *testing.T, ctx context.Context, query func(context.Context, string, ...any) pgx.Row) []byte {
	t.Helper()
	var raw string
	require.NoError(t, query(ctx, `SELECT jsonb_build_object(
 'items',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM issues t),
 'assignees',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM issue_assignees t),
 'labels',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM issue_labels t),
 'intake',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM intake_issues t),
 'comments',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM issue_comments t),
 'descriptions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM descriptions t),
 'links',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM issue_links t),
 'dependencies',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM issue_relations t),
 'custom',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM work_item_custom_relations t),
 'definitions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM work_item_relation_definitions t),
 'item_revisions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY issue_id) FROM issue_edit_revisions t),
 'comment_revisions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY comment_id) FROM issue_comment_edit_revisions t),
 'link_revisions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY link_id) FROM issue_link_edit_revisions t),
 'relation_revisions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY workspace_id) FROM work_item_relation_revisions t),
 'assets',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM file_assets t),
 'subscribers',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM issue_subscribers t),
 'receipts',(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM mutation_receipts t),
 'effects',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM work_item_effects t))::text`).Scan(&raw))
	return []byte(raw)
}
func coreNativeCommand(t *testing.T, f milestoneFixture, tc coreMutationCase) (projectmutations.Command, func(context.Context, projectmutations.Command) (projectmutations.Result, error)) {
	t.Helper()
	repo, err := workitems.NewRepository(f.pool, rosterPublisher{}, "https://vector.example")
	require.NoError(t, err)
	mutate := repo.MutateWorkitem
	cmd := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "workspace-one", Action: "create", Fields: map[string]any{"name": uuid.NewString()}}
	switch tc.tool {
	case "workitem_comment":
		mutate = repo.MutateComment
		cmd.ParentID = uuid.MustParse(f.item(t))
		cmd.Fields = map[string]any{"comment_html": "<p>Original</p>"}
	case "workitem_link":
		mutate = repo.MutateLink
		cmd.ParentID = uuid.MustParse(f.item(t))
		cmd.Fields = map[string]any{"url": "https://example.test/" + uuid.NewString(), "title": "Original"}
	case "intake":
		mutate = repo.MutateIntake
	}
	if tc.action != "create" {
		created, createErr := mutate(t.Context(), cmd)
		require.NoError(t, createErr)
		cmd.ResourceID = uuid.MustParse(created.ID)
	}
	cmd.Action = tc.action
	switch tc.action {
	case "update":
		switch tc.tool {
		case "workitem":
			cmd.Fields = map[string]any{"name": "Accepted edit"}
		case "workitem_comment":
			cmd.Fields = map[string]any{"comment_html": "<p>Accepted edit</p>"}
		case "workitem_link":
			cmd.Fields = map[string]any{"title": "Accepted edit"}
		case "intake":
			cmd.Fields = map[string]any{"source": "api"}
		}
	case "delete":
		cmd.Fields = nil
	case "archive":
		var state uuid.UUID
		require.NoError(t, f.pool.QueryRow(t.Context(), `INSERT INTO states(id,workspace_id,project_id,name,description,color,slug,"group",sequence,"default",is_triage,created_at,updated_at) VALUES($1,$2,$3,$1::uuid::text,'','#123456','completed','completed',0,false,false,now(),now()) RETURNING id`, uuid.New(), cmd.WorkspaceID, cmd.ProjectID).Scan(&state))
		_, err = f.pool.Exec(t.Context(), `UPDATE issues SET state_id=$2 WHERE id=$1`, cmd.ResourceID, state)
		require.NoError(t, err)
		cmd.Fields = map[string]any{"archive": true}
	case "manage_assignee":
		cmd.Fields = map[string]any{"add_ids": []string{testsupport.LabelActor}}
	case "manage_label":
		label := uuid.NewString()
		_, err = f.pool.Exec(t.Context(), `INSERT INTO labels(id,workspace_id,project_id,name,description,color,sort_order,created_at,updated_at) VALUES($1,$2,$3,$1::uuid::text,'','#123456',0,now(),now())`, label, cmd.WorkspaceID, cmd.ProjectID)
		require.NoError(t, err)
		cmd.Fields = map[string]any{"add_ids": []string{label}}
	}
	if tc.action != "create" {
		table, column := "issue_edit_revisions", "issue_id"
		if tc.tool == "workitem_comment" {
			table, column = "issue_comment_edit_revisions", "comment_id"
		}
		if tc.tool == "workitem_link" {
			table, column = "issue_link_edit_revisions", "link_id"
		}
		var rev int64
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM `+table+` WHERE `+column+`=$1`, cmd.ResourceID).Scan(&rev))
		token := fmt.Sprintf("%s:%s:%d", tc.tool, cmd.ResourceID, rev)
		cmd.ExpectedRevision = &token
	}
	request := uuid.New()
	cmd.RequestID = &request
	return cmd, mutate
}
func coreWriteInput(t *testing.T, f milestoneFixture, tc coreMutationCase) map[string]any {
	t.Helper()
	cmd, _ := coreNativeCommand(t, f, tc)
	input := maps.Clone(cmd.Fields)
	if input == nil {
		input = map[string]any{}
	}
	input["action"] = tc.action
	input["workspace_slug"] = cmd.WorkspaceSlug
	input["project_id"] = cmd.ProjectID.String()
	input["request_id"] = cmd.RequestID.String()
	if cmd.ParentID != uuid.Nil {
		input["workitem_id"] = cmd.ParentID.String()
	}
	if cmd.ResourceID != uuid.Nil {
		key := "workitem_id"
		if tc.tool == "workitem_comment" {
			key = "comment_id"
		}
		if tc.tool == "workitem_link" {
			key = "link_id"
		}
		input[key] = cmd.ResourceID.String()
	}
	if tc.action == "manage_assignee" || tc.action == "manage_label" {
		key := "add_label_id"
		if tc.action == "manage_assignee" {
			key = "add_user_id"
		}
		input[key] = input["add_ids"].([]string)[0]
		delete(input, "add_ids")
	}
	return input
}
