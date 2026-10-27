//go:build integration

package backgroundtasks

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const archiveMilestoneProject = `UPDATE projects SET archived_at=now() WHERE id IN(SELECT project_id FROM project_members WHERE member_id=$1)`

const disableMilestoneEmailPreference = `INSERT INTO user_notification_preferences(id,user_id,created_at,updated_at,property_change,state_change,comment,mention,issue_completed) VALUES(gen_random_uuid(),$1,now(),now(),false,true,true,true,true)`

func TestMilestoneNotificationRecipientPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, change          string
		notifications, emails int
	}{
		{"inactive project member", `UPDATE project_members SET is_active=false WHERE member_id=$1`, 0, 0},
		{"deleted project member", `UPDATE project_members SET deleted_at=now() WHERE member_id=$1`, 0, 0},
		{"restricted guest", `UPDATE project_members SET role=5 WHERE member_id=$1`, 0, 0},
		{"inactive user", `UPDATE users SET is_active=false WHERE id=$1`, 0, 0},
		{"archived project", archiveMilestoneProject, 0, 0},
		{"property email disabled", disableMilestoneEmailPreference, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMilestoneWebhookFixture(t)
			startMilestoneActivityRedis(t)
			item, milestone := f.activityItem(t), f.create(t)
			receiver := f.subscriber(t, item)
			_, err := f.repo.ManageWorkItems(f.ctx, f.opts, milestone, []uuid.UUID{item}, nil)
			require.NoError(t, err)
			f.exec(t, tc.change, receiver)
			event, input := f.activityEffect(t)
			f.recordAndNotify(t, f.activityStore(t), event, input)
			var notifications, emails int
			require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM notifications WHERE receiver_id=$1),(SELECT count(*) FROM email_notification_logs WHERE receiver_id=$1)`, receiver).Scan(&notifications, &emails))
			require.Equal(t, tc.notifications, notifications)
			require.Equal(t, tc.emails, emails)
		})
	}
}

func TestMilestoneEmailRechecksPreferenceAndRefreshesDeliveryAddress(t *testing.T) {
	for _, tc := range []struct {
		name, change string
		sent         int
	}{
		{"preference changed after queueing", disableMilestoneEmailPreference, 0},
		{"project archived after queueing", archiveMilestoneProject, 0},
		{"workspace and email renamed", "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newMilestoneWebhookFixture(t)
			startMilestoneActivityRedis(t)
			item, milestone := f.activityItem(t), f.create(t)
			receiver := f.subscriber(t, item)
			_, err := f.repo.ManageWorkItems(f.ctx, f.opts, milestone, []uuid.UUID{item}, nil)
			require.NoError(t, err)
			event, input := f.activityEffect(t)
			f.recordAndNotify(t, f.activityStore(t), event, input)
			sender := &recordingIssueEmailSender{}
			mail, err := NewPostgresIssueEmailStore(f.pool, fixedIssueEmailConfig{}, sender)
			require.NoError(t, err)
			closeMilestoneActivityStore(t, mail.Close)
			batches, err := mail.pendingBatches(f.ctx)
			require.NoError(t, err)
			require.Len(t, batches, 1)
			f.exec(t, `UPDATE workspaces SET slug='renamed' WHERE id=$1`, testsupport.LabelWorkspace)
			f.exec(t, `UPDATE users SET email='new-address@example.test' WHERE id=$1`, receiver)
			if tc.change != "" {
				f.exec(t, tc.change, receiver)
			}
			require.NoError(t, mail.sendBatch(f.ctx, batches[0]))
			require.Equal(t, tc.sent, sender.count())
			if tc.sent > 0 {
				require.True(t, sender.contains("new-address@example.test"))
				require.True(t, sender.contains("renamed"))
			}
		})
	}
}
