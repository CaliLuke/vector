//go:build integration

package mcpactions_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
)

func TestMCPAttachmentWaitRechecksAuthorizationBeforeWriteAndReplay(t *testing.T) {
	for _, action := range []string{"upload_from_url", "delete"} {
		t.Run(action, func(t *testing.T) {
			runProjectMutationAuthorizationWait(t, projectMutationWaitScenario{
				tool: "workitem_attachment",
				fixture: func(t *testing.T) milestoneFixture {
					return attachmentRecoveryFixture(t).milestoneFixture
				},
				prepare: func(t *testing.T, f milestoneFixture, call catalogueCall) map[string]any {
					input := map[string]any{"action": "upload_from_url", "workspace_slug": "workspace-one", "project_id": testsupport.LabelProject, "workitem_id": f.item(t), "url": "https://files.example.test/authorized.txt", "request_id": uuid.NewString()}
					if action == "delete" {
						created := milestoneObject(t, call("workitem_attachment", input))
						input["action"], input["attachment_id"], input["request_id"] = action, created["attachment_id"], uuid.NewString()
						delete(input, "url")
					}
					return input
				},
				snapshot: attachmentStorageSnapshot,
			})
		})
	}
}
