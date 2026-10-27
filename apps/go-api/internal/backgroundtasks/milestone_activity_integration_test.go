//go:build integration

package backgroundtasks

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMilestoneActivityRetainsDeletedTitleAndNotifiesOncePerEvent(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	cache := startMilestoneActivityRedis(t)
	item, milestone := f.activityItem(t), f.create(t)
	receiver := f.subscriber(t, item)
	_, err := f.repo.ManageWorkItems(f.ctx, f.opts, milestone, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	addEvent, addition := f.activityEffect(t)
	_, err = f.repo.Delete(f.ctx, f.opts, milestone)
	require.NoError(t, err)
	removeEvent, removal := f.activityEffect(t)
	store := f.activityStore(t)
	app, dispatcher := f.activityRuntime(t, store)
	workflowID, err := dispatcher.Admit(f.ctx, addEvent, addition)
	require.NoError(t, err)
	waitVersionWorkflow(t, f.ctx, app, workflowID)
	_, err = dispatcher.Admit(f.ctx, addEvent, addition)
	require.NoError(t, err)
	ids := f.recordAndNotify(t, store, addEvent, addition)
	require.Equal(t, ids, f.recordAndNotify(t, store, addEvent, addition))
	f.recordAndNotify(t, store, removeEvent, removal)
	var oldID, newID, oldTitle, newTitle *string
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT old_identifier::text,new_identifier::text,old_value,new_value FROM issue_activities WHERE id=$1`, ids[0]).Scan(&oldID, &newID, &oldTitle, &newTitle))
	require.Nil(t, oldID)
	require.Nil(t, oldTitle)
	require.Equal(t, milestone.String(), *newID)
	require.Equal(t, "Release <one>", *newTitle)
	var histories, notifications, emails int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM issue_activities WHERE field='milestone'),(SELECT count(*) FROM notifications WHERE receiver_id=$1),(SELECT count(*) FROM email_notification_logs WHERE receiver_id=$1)`, receiver).Scan(&histories, &notifications, &emails))
	require.Equal(t, 2, histories)
	require.Equal(t, 2, notifications)
	require.Equal(t, 2, emails)
	value, err := cache.Get(f.ctx, item.String()).Result()
	require.NoError(t, err)
	require.Equal(t, f.opts.Origin, value)
	sender := &recordingIssueEmailSender{failRemaining: 1}
	mail, err := NewPostgresIssueEmailStore(f.pool, fixedIssueEmailConfig{}, sender)
	require.NoError(t, err)
	closeMilestoneActivityStore(t, mail.Close)
	batches, err := mail.pendingBatches(f.ctx)
	require.NoError(t, err)
	require.Len(t, batches, 1)
	require.ErrorContains(t, mail.sendBatch(f.ctx, batches[0]), "temporary SMTP failure")
	require.NoError(t, mail.sendBatch(f.ctx, batches[0]))
	require.NoError(t, mail.sendBatch(f.ctx, batches[0]))
	require.Equal(t, 2, sender.attempts())
	require.Equal(t, 1, sender.count())
	require.True(t, sender.contains("set the milestone"))
	require.True(t, sender.contains("removed the milestone"))
	require.True(t, sender.contains("Release &lt;one&gt;"))
}

func TestMilestoneActivityRejectsForeignScopeBeforeWritingOriginOrHistory(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	cache := startMilestoneActivityRedis(t)
	item, milestone := f.activityItem(t), f.create(t)
	_, err := f.repo.ManageWorkItems(f.ctx, f.opts, milestone, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	event, input := f.activityEffect(t)
	input.WorkspaceID = uuid.NewString()
	_, err = f.activityStore(t).RecordIssueActivity(f.ctx, event, input)
	require.ErrorIs(t, err, ErrPermanentIssueActivity)
	count, err := cache.Exists(f.ctx, item.String()).Result()
	require.NoError(t, err)
	require.Zero(t, count)
	var histories int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM issue_activities`).Scan(&histories))
	require.Zero(t, histories)
}

func TestMilestoneNotificationDropsRevokedRecipientsAndQueuedEmails(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	startMilestoneActivityRedis(t)
	item, milestone := f.activityItem(t), f.create(t)
	receiver := f.subscriber(t, item)
	_, err := f.repo.ManageWorkItems(f.ctx, f.opts, milestone, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	event, input := f.activityEffect(t)
	store := f.activityStore(t)
	f.recordAndNotify(t, store, event, input)
	sender := &recordingIssueEmailSender{}
	mail, err := NewPostgresIssueEmailStore(f.pool, fixedIssueEmailConfig{}, sender)
	require.NoError(t, err)
	closeMilestoneActivityStore(t, mail.Close)
	batches, err := mail.pendingBatches(f.ctx)
	require.NoError(t, err)
	require.Len(t, batches, 1)
	f.exec(t, `UPDATE workspace_members SET is_active=false WHERE member_id=$1`, receiver)
	require.NoError(t, mail.sendBatch(f.ctx, batches[0]))
	require.Zero(t, sender.count())
	var processed, sent bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT processed_at IS NOT NULL,sent_at IS NOT NULL FROM email_notification_logs WHERE receiver_id=$1`, receiver).Scan(&processed, &sent))
	require.True(t, processed)
	require.False(t, sent)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, milestone, nil, []uuid.UUID{item})
	require.NoError(t, err)
	event, input = f.activityEffect(t)
	f.recordAndNotify(t, store, event, input)
	var notifications int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM notifications WHERE receiver_id=$1`, receiver).Scan(&notifications))
	require.Equal(t, 1, notifications)
	f.exec(t, `UPDATE workspace_members SET is_active=true WHERE member_id=$1`, receiver)
	require.NoError(t, mail.sendBatch(f.ctx, batches[0]))
	require.Zero(t, sender.count(), "a discarded queued email must not revive after membership is restored")
}

func TestMilestoneActivityValidationRejectsMissingScopeAndMalformedSnapshots(t *testing.T) {
	id := uuid.NewString()
	for _, tc := range []struct{ name, workspace, requested, current string }{
		{"missing workspace", "", `{"milestone":null,"milestone_title":null}`, `{"milestone":null,"milestone_title":null}`},
		{"missing title", testsupport.LabelWorkspace, `{"milestone":"` + id + `"}`, `{"milestone":null,"milestone_title":null}`},
		{"title without ID", testsupport.LabelWorkspace, `{"milestone":null,"milestone_title":"Title"}`, `{"milestone":null,"milestone_title":null}`},
		{"invalid ID", testsupport.LabelWorkspace, `{"milestone":"bad","milestone_title":"Title"}`, `{"milestone":null,"milestone_title":null}`},
		{"missing old snapshot", testsupport.LabelWorkspace, `{"milestone":"` + id + `","milestone_title":"Title"}`, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := IssueActivityInput{Type: "issue.activity.updated", ProjectID: testsupport.LabelProject, WorkspaceID: tc.workspace, IssueID: &id, RequestedData: json.RawMessage(tc.requested), CurrentInstance: json.RawMessage(tc.current)}
			_, err := (&PostgresIssueActivityStore{}).RecordIssueActivity(t.Context(), uuid.NewString(), input)
			require.ErrorIs(t, err, ErrPermanentIssueActivity)
		})
	}
}
