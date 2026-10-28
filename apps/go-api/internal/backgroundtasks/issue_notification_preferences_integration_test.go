//go:build integration

package backgroundtasks

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIssueNotificationsHonorCompletionPreferenceAndReplay(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	item := f.activityItem(t)
	enabled, disabled := f.subscriber(t, item), f.subscriber(t, item)
	state := uuid.NewString()
	f.exec(t, `INSERT INTO states(id,project_id,workspace_id,name,description,color,slug,sequence,"group","default",is_triage,created_at,updated_at) VALUES($1,$2,$3,'Done','','','',1,'completed',false,false,now(),now())`, state, testsupport.LabelProject, testsupport.LabelWorkspace)
	f.exec(t, `INSERT INTO user_notification_preferences(id,created_at,updated_at,user_id,property_change,state_change,comment,mention,issue_completed)
 VALUES(gen_random_uuid(),now(),now(),$1,false,false,false,false,true),(gen_random_uuid(),now(),now(),$2,false,false,false,false,false)`, enabled, disabled)
	itemID, actorID := item.String(), testsupport.LabelActor
	input := IssueActivityInput{
		Type: "issue.activity.updated", ProjectID: testsupport.LabelProject, WorkspaceID: testsupport.LabelWorkspace,
		IssueID: &itemID, ActorID: &actorID,
		RequestedData: json.RawMessage(`{"state_id":"` + state + `"}`), CurrentInstance: json.RawMessage(`{}`),
	}
	store := f.activityStore(t)
	event := uuid.NewString()
	ids := f.recordAndNotify(t, store, event, input)
	require.NoError(t, store.ProcessIssueNotifications(f.ctx, event, input, ids))
	var notifications, emails int
	var recipient string
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM notifications WHERE entity_identifier=$1`, item).Scan(&notifications))
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*),min(receiver_id::text) FROM email_notification_logs WHERE entity_identifier=$1`, item).Scan(&emails, &recipient))
	require.Equal(t, 2, notifications, "email preferences must not suppress in-app notifications")
	require.Equal(t, 1, emails, "replay must not duplicate completion emails")
	require.Equal(t, enabled, recipient)
}
