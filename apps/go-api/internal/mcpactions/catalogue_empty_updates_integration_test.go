//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCatalogueUpdatesWithoutFieldsPreserveRecords(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, _ milestoneFixture, call catalogueCall) {
		milestoneObject(t, call("workspace", map[string]any{"action": "update_features", "initiatives": true}))
		for _, tc := range []struct{ tool, selector, name string }{
			{"initiative", "initiative_id", "name"},
			{"release", "release_id", "name"},
			{"release_label", "label_id", "name"},
			{"release_tag", "tag_id", "version"},
		} {
			t.Run(tc.tool, func(t *testing.T) {
				created := milestoneObject(t, call(tc.tool, map[string]any{"action": "create", tc.name: uuid.NewString()}))
				updated := milestoneObject(t, call(tc.tool, map[string]any{"action": "update", tc.selector: created["id"]}))
				delete(created, "updated_at")
				delete(updated, "updated_at")
				require.Equal(t, created, updated)
				catalogueNull(t, call(tc.tool, map[string]any{"action": "delete", tc.selector: created["id"]}))
			})
		}
	})
}
