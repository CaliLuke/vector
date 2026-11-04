//go:build integration

package mcpactions_test

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCPCoreReceiptReplayRechecksCurrentReferences(t *testing.T) {
	runCatalogueCredentials(t, templateFixture(t), func(t *testing.T, f milestoneFixture, call catalogueCall) {
		item := f.item(t)
		for _, kind := range []string{"label", "assignee", "state", "mention", "definition"} {
			t.Run(kind, func(t *testing.T) {
				tool := "workitem"
				input := map[string]any{"action": "update", "project_id": testsupport.LabelProject, "workitem_id": item, "request_id": uuid.NewString()}
				var target, retire, restore string
				code := "invalid_params"
				switch kind {
				case "label":
					target = milestoneObject(t, call("label", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": uuid.NewString(), "color": "#123456"}))["id"].(string)
					input["action"] = "manage_label"
					input["add_label_id"] = target
					retire = `UPDATE labels SET deleted_at=now() WHERE id=$1`
					restore = `UPDATE labels SET deleted_at=NULL WHERE id=$1`
				case "state":
					target = milestoneObject(t, call("state", map[string]any{"action": "create", "project_id": testsupport.LabelProject, "name": uuid.NewString(), "color": "#123456", "group": "started"}))["id"].(string)
					input["state"] = target
					retire = `UPDATE states SET deleted_at=now() WHERE id=$1`
					restore = `UPDATE states SET deleted_at=NULL WHERE id=$1`
				case "assignee", "mention":
					target, _ = customerReferenceTarget(t, f, "USER")
					_, err := f.pool.Exec(t.Context(), `INSERT INTO project_members SELECT (jsonb_populate_record(NULL::project_members,to_jsonb(pm)||jsonb_build_object('id',gen_random_uuid(),'member_id',$1::text,'role',15))).* FROM project_members pm WHERE project_id=$2 AND member_id=$3`, target, testsupport.LabelProject, testsupport.LabelActor)
					require.NoError(t, err)
					retire = `UPDATE users SET is_active=false WHERE id=$1`
					restore = `UPDATE users SET is_active=true WHERE id=$1`
					if kind == "assignee" {
						input["action"] = "manage_assignee"
						input["add_user_id"] = target
					} else {
						tool = "workitem_comment"
						input["action"] = "create"
						input["comment_html"] = "<p>Hello @[" + target + "]</p>"
					}
				case "definition":
					tool = "workitem_relation"
					target = milestoneObject(t, call(tool, map[string]any{"action": "create_definition", "name": uuid.NewString(), "outward": "supports", "inward": "supported by"}))["id"].(string)
					input["action"] = "create"
					input["relation_definition_id"] = target
					input["relation_definition_label"] = "supports"
					input["workitem_ids"] = []string{f.item(t)}
					retire = `UPDATE work_item_relation_definitions SET is_active=false WHERE id=$1`
					restore = `UPDATE work_item_relation_definitions SET is_active=true WHERE id=$1`
					code = "not_found"
				}
				accepted := milestoneObject(t, call(tool, input))
				require.Equal(t, false, accepted["replayed"])
				if kind == "mention" {
					var stored string
					require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT comment_html FROM issue_comments WHERE id=$1`, accepted["comment_id"]).Scan(&stored))
					require.Contains(t, stored, `entity_name="user_mention"`)
					require.Contains(t, stored, target)
					var notify bool
					require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (payload->>'notification')::boolean FROM work_item_effects WHERE kind='activity' AND payload->>'type'='comment.activity.created' AND payload->>'issue_id'=$1 ORDER BY created_at DESC LIMIT 1`, item).Scan(&notify))
					require.True(t, notify, "MCP mentions enter the native notification workflow")
				}
				_, err := f.pool.Exec(t.Context(), retire, target)
				require.NoError(t, err)
				// Register restoration before asserting rejection so every case stays independent.
				t.Cleanup(func() {
					_, err := f.pool.Exec(context.WithoutCancel(t.Context()), restore, target)
					require.NoError(t, err)
				})
				before := coreSnapshot(t, f)
				milestoneFailure(t, call(tool, input), code)
				require.Equal(t, before, coreSnapshot(t, f))
			})
		}
	})
}
