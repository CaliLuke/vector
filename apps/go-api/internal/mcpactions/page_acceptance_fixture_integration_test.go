//go:build integration

package mcpactions_test

import (
	"context"
	"fmt"
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type pageMutationCase struct{ tool, action string }

var pageMutationCases = []pageMutationCase{
	{"page", "create"}, {"page", "update"}, {"page", "archive"}, {"page", "delete"}, {"page", "set_collection"}, {"page", "attach_to_workitem"}, {"page", "detach_from_workitem"},
	{"collection", "create"}, {"collection", "update"}, {"collection", "delete"}, {"collection", "add_member"}, {"collection", "update_member"}, {"collection", "remove_member"}, {"collection", "add_pages"}, {"collection", "remove_page"},
}

func pageSnapshot(t *testing.T, f milestoneFixture) []byte {
	t.Helper()
	return pageSnapshotQuery(t, t.Context(), f.pool.QueryRow)
}
func pageSnapshotQuery(t *testing.T, ctx context.Context, query func(context.Context, string, ...any) pgx.Row) []byte {
	t.Helper()
	var raw string
	require.NoError(t, query(ctx, `SELECT jsonb_build_object(
 'pages',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM pages t),
 'projects',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM project_pages t),
 'collections',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM page_collections t),
 'members',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM page_collection_members t),
 'filing',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM page_collection_pages t),
 'links',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM work_item_pages t),
 'events',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM page_api_events t),
 'revisions',(SELECT jsonb_agg(to_jsonb(t) ORDER BY workspace_id) FROM page_api_revisions t),
 'receipts',(SELECT jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text) FROM mutation_receipts t),
 'effects',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM work_item_effects t))::text`).Scan(&raw))
	return []byte(raw)
}
func pageNativeCommand(t *testing.T, f milestoneFixture, tc pageMutationCase) pagereads.PageAPICommand {
	t.Helper()
	repo := pagereads.NewPageAPIRepository(f.pool)
	base := pagereads.PageAPICommand{PageAPIQuery: pagereads.PageAPIQuery{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "workspace-one"}}
	seed := func(tool string, fields map[string]any) uuid.UUID {
		t.Helper()
		in := base
		in.Tool = tool
		in.Action = "create"
		in.Fields = fields
		out, err := repo.MutatePageAPI(t.Context(), in)
		require.NoError(t, err)
		return uuid.MustParse(out.ID)
	}
	cmd := base
	cmd.Tool = tc.tool
	cmd.Action = tc.action
	if tc.tool == "collection" || tc.action == "set_collection" {
		cmd.CollectionID = seed("collection", map[string]any{"name": uuid.NewString()})
	}
	if tc.tool == "page" && tc.action != "create" || tc.action == "add_pages" || tc.action == "remove_page" {
		cmd.PageID = seed("page", map[string]any{"name": "Original", "description_html": "<p>Original</p>"})
	}
	switch tc.action {
	case "create", "update":
		cmd.Fields = map[string]any{"name": uuid.NewString()}
		if tc.tool == "page" {
			cmd.Fields["description_html"] = "<p>Accepted body</p>"
		}
	case "delete":
		if tc.tool == "page" {
			archive := cmd
			archive.Action = "archive"
			_, err := repo.MutatePageAPI(t.Context(), archive)
			require.NoError(t, err)
		}
	case "add_member", "update_member", "remove_member":
		user, _ := customerReferenceTarget(t, f, "USER")
		cmd.UserID = uuid.MustParse(user)
		cmd.Fields = map[string]any{"member_access": 2}
		if tc.action != "add_member" {
			add := cmd
			add.Action = "add_member"
			member, err := repo.MutatePageAPI(t.Context(), add)
			require.NoError(t, err)
			cmd.MemberID = uuid.MustParse(member.ID)
			cmd.UserID = uuid.Nil
			if tc.action == "remove_member" {
				cmd.Fields = nil
			} else {
				cmd.Fields["member_access"] = 0
			}
		}
	case "add_pages":
		cmd.PageIDs = []uuid.UUID{cmd.PageID}
		cmd.PageID = uuid.Nil
	case "remove_page":
		file := cmd
		file.Tool = "page"
		file.Action = "set_collection"
		_, err := repo.MutatePageAPI(t.Context(), file)
		require.NoError(t, err)
		require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT id FROM page_collection_pages WHERE page_id=$1`, cmd.PageID).Scan(&cmd.LinkID))
		cmd.PageID = uuid.Nil
	case "attach_to_workitem", "detach_from_workitem":
		cmd.ProjectID = uuid.MustParse(testsupport.LabelProject)
		cmd.ItemID = uuid.MustParse(f.item(t))
		if tc.action == "detach_from_workitem" {
			attach := cmd
			attach.Action = "attach_to_workitem"
			link, err := repo.MutatePageAPI(t.Context(), attach)
			require.NoError(t, err)
			cmd.LinkID = uuid.MustParse(link.ID)
			cmd.PageID = uuid.Nil
		}
	}
	if tc.action == "create" {
		cmd.CollectionID = uuid.Nil
	}
	var rev int64
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM page_api_revisions WHERE workspace_id=$1`, cmd.WorkspaceID).Scan(&rev))
	revision := fmt.Sprintf("page:%s:%d", cmd.WorkspaceID, rev)
	cmd.ExpectedRevision = &revision
	request := uuid.New()
	cmd.RequestID = &request
	return cmd
}
func pageWriteInput(t *testing.T, f milestoneFixture, tc pageMutationCase) map[string]any {
	t.Helper()
	cmd := pageNativeCommand(t, f, tc)
	input := maps.Clone(cmd.Fields)
	if input == nil {
		input = map[string]any{}
	}
	input["action"] = cmd.Action
	input["workspace_slug"] = cmd.WorkspaceSlug
	input["request_id"] = cmd.RequestID.String()
	// Lock-wait cases change catalogue revisions independently of this request.
	for key, id := range map[string]uuid.UUID{"project_id": cmd.ProjectID, "page_id": cmd.PageID, "collection_id": cmd.CollectionID, "workitem_id": cmd.ItemID, "collection_member_id": cmd.MemberID, "user_id": cmd.UserID} {
		if id != uuid.Nil {
			input[key] = id.String()
		}
	}
	if cmd.LinkID != uuid.Nil {
		key := "page_collection_id"
		if cmd.Action == "detach_from_workitem" {
			key = "workitem_page_id"
		}
		input[key] = cmd.LinkID.String()
	}
	if len(cmd.PageIDs) > 0 {
		input["page_ids"] = cmd.PageIDs[0].String()
	}
	if access, ok := input["member_access"].(int); ok {
		input["member_access"] = []string{"view", "comment", "edit"}[access]
	}
	return input
}
