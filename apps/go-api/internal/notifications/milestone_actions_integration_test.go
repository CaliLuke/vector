//go:build integration

package notifications

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMilestoneNotificationActionsRecheckVisibilityAfterLookup(t *testing.T) {
	tx := notificationTestDB(t)
	insertNotification(t, tx, 11, "Milestone removed")
	insertNotification(t, tx, 12, "Priority changed")
	execNotificationSQL(t, tx, `INSERT INTO projects(id,workspace_id) VALUES($1,$2);
 `, notificationID(20), notificationID(1))
	execNotificationSQL(t, tx, `INSERT INTO project_members(project_id,workspace_id,member_id,role,is_active) VALUES($1,$2,$3,15,true)`, notificationID(20), notificationID(1), notificationID(3))
	execNotificationSQL(t, tx, `INSERT INTO issues(id,project_id,workspace_id) VALUES($1,$2,$3)`, notificationID(111), notificationID(20), notificationID(1))
	execNotificationSQL(t, tx, `UPDATE notifications SET project_id=$1,data='{"issue_activity":{"field":"milestone","old_value":"Secret"}}' WHERE id=$2`, notificationID(20), notificationID(11))
	publisher := &notificationDeletionFailure{beforePublish: func() { t.Fatal("hidden notification was published") }}
	repo, err := NewActionRepository(tx, publisher)
	require.NoError(t, err)
	_, err = repo.Detail(t.Context(), notificationPrincipal, "test", notificationID(11), false)
	require.NoError(t, err)
	// Access changes after the service lookup but before the repository write.
	execNotificationSQL(t, tx, `UPDATE project_members SET role=5`)
	for _, field := range []string{"read_at", "archived_at", "snoozed_till", "deleted_at"} {
		_, err = repo.Set(t.Context(), notificationPrincipal, "test", notificationID(11), field, time.Now())
		assertNotificationStatus(t, err, 404)
	}
	require.NoError(t, repo.MarkAllRead(t.Context(), notificationPrincipal, "test", false, false, "all"))
	var hiddenUnchanged, ordinaryRead bool
	require.NoError(t, tx.QueryRow(t.Context(), `SELECT read_at IS NULL AND archived_at IS NULL AND snoozed_till IS NULL AND deleted_at IS NULL FROM notifications WHERE id=$1`, notificationID(11)).Scan(&hiddenUnchanged))
	require.True(t, hiddenUnchanged)
	require.NoError(t, tx.QueryRow(t.Context(), `SELECT read_at IS NOT NULL FROM notifications WHERE id=$1`, notificationID(12)).Scan(&ordinaryRead))
	require.True(t, ordinaryRead)
	require.Empty(t, publisher.calls)
	execNotificationSQL(t, tx, `UPDATE projects SET guest_view_all_features=true`)
	updated, err := repo.Set(t.Context(), notificationPrincipal, "test", notificationID(11), "read_at", time.Now())
	require.NoError(t, err)
	require.NotNil(t, updated["read_at"])
}
