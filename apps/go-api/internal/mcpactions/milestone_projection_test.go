package mcpactions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Pinned toolkit envelope: count, total_count, cursors, page-presence flags,
// results. The native HTTP envelope has five additional fields.
func TestMilestonePinnedPageEnvelopeAndEntryShapes(t *testing.T) {
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		f := newFixture(t)
		check := func(scope milestones.Scope, q djangoapi.Query) {
			require.Equal(t, readProject, scope.ProjectID.String())
			require.Equal(t, "20000000-0000-4000-8000-000000000001", scope.WorkspaceID.String())
			require.Equal(t, "2", q.Params["per_page"])
			require.Equal(t, "2:1:0", q.Params["cursor"])
			require.True(t, q.Present["cursor"])
		}
		f.milestones.list = func(scope milestones.Scope, q djangoapi.Query) (milestones.Page[milestones.Record], error) {
			check(scope, q)
			return milestones.Page[milestones.Record]{Results: []milestones.Record{{ID: milestoneTestID, Title: "Ship", CreatedAt: "2026-01-02T03:04:05Z", UpdatedAt: "2026-01-02T03:04:05Z"}}, Total: 3, Limit: 2, Index: 1}, nil
		}
		f.milestones.links = func(scope milestones.Scope, id uuid.UUID, q djangoapi.Query) (milestones.Page[milestones.Link], error) {
			check(scope, q)
			require.Equal(t, milestoneTestID, id.String())
			return milestones.Page[milestones.Link]{Results: []milestones.Link{{ID: milestoneTestID, Issue: readProject, Milestone: milestoneTestID}}, Total: 3, Limit: 2, Index: 1}, nil
		}
		h := f.handler(t, nil)
		for _, tc := range []struct{ action, entry string }{
			{"list", `{"id":"` + milestoneTestID + `","title":"Ship","target_date":null,"external_source":null,"external_id":null,"created_at":"2026-01-02T03:04:05Z","updated_at":"2026-01-02T03:04:05Z"}`},
			{"list_workitems", `{"id":"` + milestoneTestID + `","issue":"` + readProject + `","milestone":"` + milestoneTestID + `"}`},
		} {
			input := map[string]any{"action": tc.action, "project_id": readProject, "per_page": 2, "cursor": "2:1:0"}
			if tc.action == "list_workitems" {
				input["milestone_id"] = milestoneTestID
			}
			body, err := json.Marshal(input)
			require.NoError(t, err)
			result := rpcVersion(t, h, "test-one", "tools/call", milestoneCall(string(body)), version)
			require.False(t, result.IsError, result.Content)
			require.JSONEq(t, `{"results":[`+tc.entry+`],"count":1,"total_count":3,"next_cursor":"2:2:0","prev_cursor":"2:0:1","next_page_results":false,"prev_page_results":true}`, string(result.StructuredContent))
		}
	}
}

func TestMilestoneMCPFieldsPreservePinnedOmissionAndEffectOrigin(t *testing.T) {
	f := newFixture(t, mcpactions.WithOrigin("https://vector.example"))
	f.milestones.write = func(ctx context.Context, action string, opts milestones.WriteOptions, id uuid.UUID, fields milestones.Fields, _, _ []uuid.UUID) (milestones.MutationResult, error) {
		require.Equal(t, "update", action)
		require.Equal(t, milestoneTestID, id.String())
		require.Equal(t, "https://vector.example", opts.Origin)
		require.False(t, fields.Title.Present)
		require.False(t, fields.TargetDate.Present)
		require.False(t, fields.ExternalID.Present)
		require.True(t, fields.ExternalSource.Present)
		require.Equal(t, " source ", *fields.ExternalSource.Value)
		require.NoError(t, opts.Revalidate(ctx))
		return milestones.MutationResult{MilestoneID: id.String(), Revision: "revision", Result: &milestones.Record{ID: id.String(), Title: "Kept", CreatedAt: "2026-01-02T03:04:05Z", UpdatedAt: "2026-01-02T03:04:05Z"}}, nil
	}
	result := rpc(t, f.handler(t, nil), "test-one", "tools/call", milestoneCall(fmt.Sprintf(`{"action":"update","project_id":%q,"milestone_id":%q,"title":"","target_date":"","external_id":"","external_source":" source ","expected_revision":"revision"}`, readProject, milestoneTestID)))
	require.False(t, result.IsError, result.Content)
	var receipt map[string]any
	require.NoError(t, json.Unmarshal(result.StructuredContent, &receipt))
	require.Len(t, receipt, 3)
	require.Equal(t, "Kept", receipt["result"].(map[string]any)["title"])
}
