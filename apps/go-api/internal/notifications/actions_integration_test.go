//go:build integration

package notifications

import (
	"context"
	"errors"
	"testing"
	"time"
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

func TestNotificationMarkAllReadUsesSourceFilters(t *testing.T) {
	tx := notificationTestDB(t)
	now := time.Now().UTC()
	execNotificationSQL(t, tx, `
 INSERT INTO notifications(id,workspace_id,receiver_id,entity_identifier,title,snoozed_till,archived_at)
 VALUES
  ($1,$2,$3,$4,'future snooze',$5,NULL),
  ($6,$2,$3,$7,'no snooze',NULL,NULL),
  ($8,$2,$3,$9,'past snooze',$10,NULL),
  ($11,$2,$3,$12,'archived',NULL,$13)`,
		notificationID(21), notificationID(1), notificationID(3), notificationID(101), now.Add(time.Hour),
		notificationID(22), notificationID(102), notificationID(23), notificationID(103), now.Add(-time.Hour),
		notificationID(24), notificationID(104), now)
	execNotificationSQL(t, tx, `INSERT INTO issue_subscribers(issue_id,workspace_id,subscriber_id,deleted_at) VALUES($1,$2,$3,NULL),($4,$2,$3,NULL),($5,$2,$3,NULL),($6,$2,$3,NULL)`,
		notificationID(101), notificationID(1), notificationID(3), notificationID(102), notificationID(103), notificationID(104))
	repository, err := NewActionRepository(tx, notificationNoopPublisher{})
	if err != nil {
		t.Fatal(err)
	}
	if markAllReadErr := repository.MarkAllRead(t.Context(), notificationPrincipal, "test", false, false, "watching"); markAllReadErr != nil {
		t.Fatal(markAllReadErr)
	}
	var read []int
	rows, err := tx.Query(t.Context(), `SELECT CASE id WHEN $1 THEN 21 WHEN $2 THEN 22 WHEN $3 THEN 23 WHEN $4 THEN 24 END FROM notifications WHERE read_at IS NOT NULL ORDER BY id`, notificationID(21), notificationID(22), notificationID(23), notificationID(24))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		read = append(read, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(read) != 2 || read[0] != 21 || read[1] != 22 {
		t.Fatalf("source snooze/archive/subscribed filters marked notifications %v read", read)
	}
}
