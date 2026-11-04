//go:build integration

package mcpactions_test

import (
	"maps"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPReleaseContractsPreserveFalseFieldsLatestSelectionAndAtomicBatches(t *testing.T) {
	runCatalogue(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		_, foreignProject := addPrivateProjectInOtherWorkspace(t, f)
		tag := milestoneObject(t, call("release_tag", map[string]any{"action": "create", "version": " v1 ", "description": "Tag description", "commit_hash": "abc", "git_tag": "v1"}))
		require.Equal(t, "v1", tag["version"])
		require.Equal(t, "Tag description", tag["description"])
		require.Equal(t, "abc", tag["commit_hash"])
		require.Equal(t, "v1", tag["git_tag"])
		input := map[string]any{"action": "create", "name": " Release ", "tag_id": tag["id"], "lead_id": testsupport.LabelActor, "is_prerelease": true, "status": "released", "release_date": "2026-11-02", "target_date": "2026-11-01", "external_source": "deploy", "external_id": "build-1"}
		created := milestoneObject(t, call("release", input))
		require.Equal(t, "Release", created["name"])
		require.Equal(t, false, created["is_latest"])
		require.Equal(t, tag["id"], created["tag"])
		require.Equal(t, testsupport.LabelActor, created["lead"])
		require.Equal(t, "build-1", created["external_id"])
		updated := milestoneObject(t, call("release", map[string]any{"action": "update", "release_id": created["id"], "is_prerelease": false}))
		require.Equal(t, false, updated["is_prerelease"])
		require.Equal(t, true, updated["is_latest"])
		for _, key := range []string{"name", "tag", "lead", "target_date", "release_date", "external_source", "external_id"} {
			require.Equal(t, created[key], updated[key], key)
		}
		later := milestoneObject(t, call("release", map[string]any{"action": "create", "name": "Later", "status": "released", "release_date": "2026-11-03"}))
		require.Equal(t, true, later["is_latest"])
		require.Equal(t, false, milestoneObject(t, call("release", map[string]any{"action": "retrieve", "release_id": created["id"]}))["is_latest"])
		notes := milestoneObject(t, call("release", map[string]any{"action": "update_changelog", "release_id": created["id"], "description_html": "<p>HTML wins</p><script>bad()</script>", "description_stripped": "Ignored text"}))
		body := notes["changelog"].(map[string]any)
		require.Contains(t, body["description_html"], "HTML wins")
		require.NotContains(t, body["description_html"], "<script")
		require.NotContains(t, body["description_stripped"], "Ignored text")
		require.Empty(t, body["description_json"])
		valid := map[string]any{"action": "update", "release_id": created["id"], "name": "Valid"}
		for _, fields := range []map[string]any{
			{"name": "Later"}, {"target_date": "2026-02-30"}, {"tag_id": uuid.NewString()},
			{"action": "manage_workitems", "name": nil, "add_ids": f.item(t) + "," + uuid.NewString()},
			{"release_id": foreignProject},
		} {
			next := maps.Clone(valid)
			maps.Copy(next, fields)
			if next["name"] == nil {
				delete(next, "name")
			}
			next["request_id"] = uuid.NewString()
			before := releaseSnapshot(t, f)
			require.True(t, call("release", next).IsError)
			require.Equal(t, before, releaseSnapshot(t, f))
			retry := maps.Clone(valid)
			retry["request_id"] = next["request_id"]
			require.Equal(t, false, milestoneObject(t, call("release", retry))["replayed"])
		}
	})
}
