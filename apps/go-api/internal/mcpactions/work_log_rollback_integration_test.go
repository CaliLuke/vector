//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
)

func TestMCPWorkLogStorageFailureRollsBack(t *testing.T) {
	runCatalogueMatrix(t, func(t *testing.T, f milestoneFixture, root catalogueCall) {
		for _, action := range []string{"create", "update", "delete"} {
			for _, table := range []string{"mutation_receipts", "work_item_effects"} {
				t.Run(action+"/"+table, func(t *testing.T) {
					item := f.item(t)
					call := catalogueScoped(root, map[string]any{"project_id": testsupport.LabelProject, "workitem_id": item})
					input := map[string]any{"action": action, "request_id": uuid.NewString()}
					if action == "create" {
						input["duration"] = 15
					} else {
						created := milestoneObject(t, call("work_log", map[string]any{"action": "create", "duration": 30}))
						input["work_log_id"] = created["id"]
						if action == "update" {
							input["duration"] = 45
						}
					}
					assertCatalogueStorageFailure(t, f, call, "work_log", input, table, func() []byte { return workLogStorageSnapshot(t, f) })
				})
			}
		}
	})
}
