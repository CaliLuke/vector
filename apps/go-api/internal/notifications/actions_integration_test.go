//go:build integration

package notifications

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type notificationDeletionFailure struct {
	beforePublish func()
	err           error
	calls         []string
}

type notificationNoopPublisher struct{}

func (notificationNoopPublisher) SoftDelete(context.Context, string, string, string) error {
	return nil
}

func (s *notificationDeletionFailure) SoftDelete(_ context.Context, app, model, id string) error {
	s.calls = append(s.calls, app+":"+model+":"+id)
	s.beforePublish()
	return s.err
}

func TestNotificationActionPublicationFailurePreservesSave(t *testing.T) {
	tx := notificationTestDB(t)
	insertNotification(t, tx, 11, "delete target")
	assertSaved := func() {
		t.Helper()
		var saved bool
		err := tx.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL AND updated_by_id=$2 AND updated_at>'2026-01-01Z' FROM notifications WHERE id=$1`, notificationID(11), notificationID(3)).Scan(&saved)
		if err != nil || !saved {
			t.Fatalf("soft deletion must be saved before publication and survive its failure: saved=%v err=%v", saved, err)
		}
	}
	publisher := &notificationDeletionFailure{beforePublish: assertSaved, err: errors.New("broker rejected task")}
	repository, err := NewActionRepository(tx, publisher)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Set(t.Context(), notificationPrincipal, "test", notificationID(11), "deleted_at", time.Now().UTC())
	if !errors.Is(err, publisher.err) {
		t.Fatalf("publication error=%v", err)
	}
	assertSaved()
	if len(publisher.calls) != 1 || publisher.calls[0] != "db:notification:"+notificationID(11) {
		t.Fatalf("task=%v", publisher.calls)
	}
}

func TestNotificationMarkAllReadMatchesPersistedListSelection(t *testing.T) {
	tx := notificationTestDB(t)
	for i, title := range []string{
		"assigned mention", "subscribed mention", "assigned inbox", "subscribed inbox",
		"created subscription", "other entity", "future snooze", "past snooze", "archived",
		"deleted", "other workspace", "other recipient", "unrelated", "already read",
		"archived future snooze",
	} {
		insertNotification(t, tx, 21+i, title)
	}
	execNotificationSQL(t, tx, `
		UPDATE notifications SET sender='In-app mentioned' WHERE title IN ('assigned mention','subscribed mention');
		UPDATE notifications SET entity_name='project' WHERE title='other entity';
		UPDATE notifications SET snoozed_till='2099-01-01Z' WHERE title IN ('future snooze','archived future snooze');
		UPDATE notifications SET snoozed_till='2000-01-01Z' WHERE title='past snooze';
		UPDATE notifications SET archived_at=now() WHERE title IN ('archived','archived future snooze');
		UPDATE notifications SET deleted_at=now() WHERE title='deleted';
		UPDATE notifications SET workspace_id='00000000-0000-0000-0000-000000000002' WHERE title='other workspace';
		UPDATE notifications SET receiver_id='00000000-0000-0000-0000-000000000099' WHERE title='other recipient';
		INSERT INTO issue_subscribers(issue_id,workspace_id,subscriber_id)
		SELECT entity_identifier,'00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000003'
		FROM notifications WHERE title NOT IN ('assigned mention','assigned inbox','unrelated');
		INSERT INTO issue_assignees(id,issue_id,workspace_id,assignee_id)
		SELECT id,entity_identifier,workspace_id,receiver_id FROM notifications WHERE title LIKE 'assigned%';
		INSERT INTO issues(id,workspace_id,created_by_id)
		SELECT entity_identifier,workspace_id,receiver_id FROM notifications WHERE title='created subscription';
	`)
	repository, err := NewActionRepository(tx, notificationNoopPublisher{})
	require.NoError(t, err)
	for _, check := range []struct {
		name  string
		query url.Values
		body  map[string]any
		want  []string
	}{
		{"mentions and multiple types", url.Values{"mentioned": {"true"}, "type": {"assigned,subscribed"}}, map[string]any{"mentioned": true, "type": "assigned,subscribed"}, []string{"assigned mention", "subscribed mention"}},
		{"inbox and subscribed alias", url.Values{"type": {"subscribed"}, "read": {"false"}}, map[string]any{"type": "watching", "read": false}, []string{"subscribed inbox"}},
		{"snoozed", url.Values{"snoozed": {"true"}}, map[string]any{"snoozed": true}, []string{"past snooze", "future snooze"}},
		{"archived", url.Values{"archived": {"true"}}, map[string]any{"archived": true}, []string{"archived", "archived future snooze"}},
	} {
		t.Run(check.name, func(t *testing.T) {
			execNotificationSQL(t, tx, `UPDATE notifications SET read_at=CASE WHEN title='already read' THEN '2020-01-01Z'::timestamptz ELSE NULL END`)
			selected, err := repository.List(t.Context(), notificationPrincipal, "test", check.query)
			require.NoError(t, err)
			require.ElementsMatch(t, check.want, notificationTitles(t, selected))
			selection, err := ParseBulkSelection(check.body)
			require.NoError(t, err)
			require.NoError(t, repository.MarkAllRead(t.Context(), notificationPrincipal, "test", selection))
			var updated []string
			err = tx.QueryRow(t.Context(), `SELECT array_agg(title) FROM notifications WHERE read_at=CURRENT_TIMESTAMP`).Scan(&updated)
			require.NoError(t, err)
			require.ElementsMatch(t, check.want, updated)
			var unchanged bool
			err = tx.QueryRow(t.Context(), `SELECT read_at='2020-01-01Z' FROM notifications WHERE title='already read'`).Scan(&unchanged)
			require.NoError(t, err)
			require.True(t, unchanged)
		})
	}
}
