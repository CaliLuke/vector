//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
)

func TestMCPWorkLogWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, action := range []string{"update", "delete"} {
		t.Run(action, func(t *testing.T) {
			runMutationAuthorizationWait(t, mutationWaitScenario{
				tool: "work_log", fixture: newMilestoneFixture,
				prepare: func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					item := f.item(t)
					scoped := catalogueScoped(call, map[string]any{"workspace_slug": "workspace-one", "project_id": testsupport.LabelProject, "workitem_id": item})
					created := milestoneObject(t, scoped("work_log", map[string]any{"action": "create", "duration": 30}))
					input := map[string]any{"action": action, "workspace_slug": "workspace-one", "project_id": testsupport.LabelProject, "workitem_id": item, "work_log_id": created["id"], "request_id": uuid.NewString()}
					if action == "update" {
						input["duration"] = 45
					}
					return input
				},
				snapshot: workLogStorageSnapshot,
			})
		})
	}
}
