//go:build integration

package notifications

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"testing"

	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

// Temporary tables shadow the real tables on this connection only, so this
// suite can reuse the parity database without changing another test's seeds.
func notificationTestDB(t *testing.T) pgx.Tx {
	t.Helper()
	ctx := t.Context()
	databaseURL := os.Getenv("NOTIFICATIONS_TEST_DATABASE_URL")
	if databaseURL == "" {
		pair, err := postgresfixture.StartUncloned(ctx, func(context.Context, string) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := pair.Close(); err != nil {
				t.Error(err)
			}
		})
		databaseURL = pair.SeedURL()
	}
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	_, err = tx.Exec(ctx, `
 CREATE TEMP TABLE workspaces(id uuid PRIMARY KEY,slug text,deleted_at timestamptz);
 CREATE TEMP TABLE workspace_members(workspace_id uuid,member_id uuid,role integer,is_active boolean,deleted_at timestamptz);
 CREATE TEMP TABLE users(id uuid PRIMARY KEY,is_active boolean DEFAULT true,first_name text DEFAULT '',last_name text DEFAULT '',avatar text DEFAULT '',display_name text DEFAULT '',is_bot boolean DEFAULT false,avatar_asset_id uuid);
 CREATE TEMP TABLE file_assets(id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,issue_id uuid,entity_type text);
 CREATE TEMP TABLE projects(id uuid PRIMARY KEY,workspace_id uuid,deleted_at timestamptz,archived_at timestamptz,guest_view_all_features boolean DEFAULT false);
 CREATE TEMP TABLE project_members(project_id uuid,workspace_id uuid,member_id uuid,role integer,is_active boolean,deleted_at timestamptz);
 CREATE TEMP TABLE issues(id uuid PRIMARY KEY,project_id uuid,is_draft boolean DEFAULT false,workspace_id uuid,created_by_id uuid,deleted_at timestamptz);
 CREATE TEMP TABLE intake_issues(issue_id uuid,status integer,deleted_at timestamptz);
 CREATE TEMP TABLE issue_assignees(id uuid PRIMARY KEY,issue_id uuid,workspace_id uuid,assignee_id uuid,deleted_at timestamptz);
 CREATE TEMP TABLE issue_subscribers(issue_id uuid,workspace_id uuid,subscriber_id uuid,deleted_at timestamptz);
 CREATE TEMP TABLE notifications(
 id uuid PRIMARY KEY,workspace_id uuid,project_id uuid,receiver_id uuid,triggered_by_id uuid,
 created_by_id uuid,updated_by_id uuid,created_at timestamptz DEFAULT '2026-01-01Z',updated_at timestamptz DEFAULT '2026-01-01Z',
 deleted_at timestamptz,read_at timestamptz,snoozed_till timestamptz,archived_at timestamptz,
 entity_identifier uuid,entity_name text DEFAULT 'issue',title text,message jsonb,data jsonb DEFAULT '{}',
 message_html text DEFAULT '',message_stripped text,sender text DEFAULT 'Issue changed');
 INSERT INTO workspaces VALUES ('00000000-0000-0000-0000-000000000001','test',NULL),('00000000-0000-0000-0000-000000000002','other',NULL);
 INSERT INTO workspace_members VALUES ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000003',15,true,NULL);
 INSERT INTO users(id) VALUES ('00000000-0000-0000-0000-000000000003');
 `)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}
func notificationID(n int) string { return fmt.Sprintf("00000000-0000-0000-0000-%012d", n) }

var notificationPrincipal = Principal{UserID: notificationID(3), Timezone: "UTC"}

func execNotificationSQL(t *testing.T, tx pgx.Tx, query string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(t.Context(), query, args...); err != nil {
		t.Fatal(err)
	}
}
func insertNotification(t *testing.T, tx pgx.Tx, id int, title string) {
	t.Helper()
	execNotificationSQL(t, tx, `INSERT INTO notifications(id,workspace_id,receiver_id,entity_identifier,title)VALUES($1,$2,$3,$4,$5)`, notificationID(id), notificationID(1), notificationID(3), notificationID(id+100), title)
}
func notificationTitles(t *testing.T, result any) []string {
	t.Helper()
	values, ok := result.([]any)
	if !ok {
		t.Fatalf("expected notification array, got %T", result)
	}
	titles := []string{}
	for _, item := range values {
		titles = append(titles, item.(map[string]any)["title"].(string))
	}
	return titles
}
func assertNotificationStatus(t *testing.T, err error, status int) {
	t.Helper()
	var e *HTTPError
	if !errors.As(err, &e) || e.Status != status {
		t.Fatalf("want HTTP %d, got %v", status, err)
	}
}

func TestNotificationRepositoryPostgresScopeAndSelection(t *testing.T) {
	tx := notificationTestDB(t)
	r, _ := NewRepository(tx)
	ctx := t.Context()
	for i, title := range []string{"older", "newer", "mentioned", "other entity", "deleted", "other receiver", "other workspace", "past snooze", "future snooze", "archived"} {
		insertNotification(t, tx, 10+i, title)
	}
	execNotificationSQL(t, tx, `UPDATE notifications SET created_at='2026-01-02Z' WHERE title='newer';
 UPDATE notifications SET sender='User MENTIONED you' WHERE title='mentioned';
 UPDATE notifications SET entity_name='project' WHERE title='other entity';
 UPDATE notifications SET deleted_at=now() WHERE title='deleted';
 UPDATE notifications SET receiver_id='00000000-0000-0000-0000-000000000099' WHERE title='other receiver';
 UPDATE notifications SET workspace_id='00000000-0000-0000-0000-000000000002' WHERE title='other workspace';
 UPDATE notifications SET snoozed_till='2000-01-01Z' WHERE title='past snooze';
 UPDATE notifications SET snoozed_till='2099-01-01Z' WHERE title='future snooze';
 UPDATE notifications SET archived_at=now() WHERE title='archived'`)
	result, err := r.List(ctx, notificationPrincipal, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := notificationTitles(t, result); !reflect.DeepEqual(got, []string{"future snooze", "newer", "older"}) {
		t.Fatalf("default list=%v", got)
	}
	mentions, err := r.List(ctx, notificationPrincipal, "test", url.Values{"mentioned": {"false"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := notificationTitles(t, mentions); !reflect.DeepEqual(got, []string{"mentioned"}) {
		t.Fatalf("nonempty mentioned query=%v", got)
	}
	counts, err := r.Counts(ctx, notificationPrincipal, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(counts, map[string]int64{"total_unread_notifications_count": 3, "mention_unread_notifications_count": 1}) {
		t.Fatalf("counts=%v", counts)
	}
	for _, id := range []int{14, 15, 16} {
		_, err := r.Detail(ctx, notificationPrincipal, "test", notificationID(id), true)
		assertNotificationStatus(t, err, 404)
	}
	// Removing membership denies decorated actions, but inherited retrieve still
	// resolves the receiver's notification exactly as the source viewset does.
	execNotificationSQL(t, tx, `UPDATE workspace_members SET is_active=false`)
	assertNotificationStatus(t, r.Authorize(ctx, notificationPrincipal, "test"), 403)
	if _, err := r.Detail(ctx, notificationPrincipal, "test", notificationID(10), true); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationRepositoryPostgresSubscribedExclusions(t *testing.T) {
	tx := notificationTestDB(t)
	r, _ := NewRepository(tx)
	ctx := t.Context()
	for i, title := range []string{"ordinary assigned subscription", "created subscription", "assignee primary key subscription"} {
		insertNotification(t, tx, 10+i, title)
		execNotificationSQL(t, tx, `INSERT INTO issues(id,workspace_id)VALUES($1,$2)`, notificationID(110+i), notificationID(1))
		execNotificationSQL(t, tx, `INSERT INTO issue_subscribers(issue_id,workspace_id,subscriber_id)VALUES($1,$2,$3)`, notificationID(110+i), notificationID(1), notificationID(3))
	}
	execNotificationSQL(t, tx, `UPDATE issues SET created_by_id=$1 WHERE id=$2`, notificationID(3), notificationID(111))
	execNotificationSQL(t, tx, `INSERT INTO issue_assignees(id,issue_id,workspace_id,assignee_id)VALUES($1,$2,$3,$4),($5,$5,$3,$4)`, notificationID(500), notificationID(110), notificationID(1), notificationID(3), notificationID(112))
	result, err := r.List(ctx, notificationPrincipal, "test", url.Values{"type": {"subscribed"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := notificationTitles(t, result); !reflect.DeepEqual(got, []string{"ordinary assigned subscription"}) {
		t.Fatalf("subscribed=%v", got)
	}
	execNotificationSQL(t, tx, `UPDATE workspace_members SET role=5`)
	result, err = r.List(ctx, notificationPrincipal, "test", url.Values{"type": {"assigned,created"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := notificationTitles(t, result); len(got) != 0 {
		t.Fatalf("guest created union must be empty: %v", got)
	}
}

func TestNotificationRepositoryPostgresPaginationOrdering(t *testing.T) {
	tx := notificationTestDB(t)
	r, _ := NewRepository(tx)
	ctx := t.Context()
	insertNotification(t, tx, 10, "old creation tie")
	insertNotification(t, tx, 11, "new creation tie")
	insertNotification(t, tx, 12, "latest update")
	execNotificationSQL(t, tx, `UPDATE notifications SET updated_at='2026-01-02Z';
 UPDATE notifications SET created_at='2026-01-03Z' WHERE title='new creation tie';
 UPDATE notifications SET updated_at='2026-01-04Z' WHERE title='latest update'`)
	result, err := r.List(ctx, notificationPrincipal, "test", url.Values{"per_page": {"2"}, "cursor": {"2:0:0"}, "order_by": {"updated_at"}})
	if err != nil {
		t.Fatal(err)
	}
	page := result.(map[string]any)
	if got := notificationTitles(t, page["results"]); !reflect.DeepEqual(got, []string{"new creation tie", "old creation tie"}) {
		t.Fatalf("first page=%v", got)
	}
	for key, want := range map[string]any{"count": 2, "total_count": int64(3), "total_results": int64(3), "total_pages": int64(2), "next_cursor": "2:1:0", "prev_cursor": "2:-1:1", "next_page_results": true, "prev_page_results": false} {
		if !reflect.DeepEqual(page[key], want) {
			t.Errorf("%s=%v, want %v", key, page[key], want)
		}
	}
	result, err = r.List(ctx, notificationPrincipal, "test", url.Values{"per_page": {"2"}, "cursor": {"2:1:0"}, "order_by": {"updated_at"}})
	if err != nil {
		t.Fatal(err)
	}
	page = result.(map[string]any)
	if got := notificationTitles(t, page["results"]); !reflect.DeepEqual(got, []string{"latest update"}) {
		t.Fatalf("second page=%v", got)
	}
	if page["next_page_results"] != false || page["prev_page_results"] != true {
		t.Fatalf("second page cursors=%v", page)
	}
}

func TestNotificationRepositoryOversizedCursorMatchesDjango(t *testing.T) {
	tx := notificationTestDB(t)
	repository, err := NewRepository(tx)
	if err != nil {
		t.Fatal(err)
	}
	for _, cursor := range []string{"2:4611686018427387904:0", "2:100000000000000000000:0"} {
		_, err := repository.List(t.Context(), notificationPrincipal, "test", url.Values{"per_page": {"2"}, "cursor": {cursor}})
		assertNotificationStatus(t, err, 500)
	}
	result, err := repository.List(t.Context(), notificationPrincipal, "test", url.Values{"per_page": {"1"}, "cursor": {"1:9223372036854775807:0"}})
	if err != nil || result.(map[string]any)["count"] != 0 {
		t.Fatalf("valid bigint boundary = %v, %v", result, err)
	}
	execNotificationSQL(t, tx, `UPDATE workspace_members SET role=5`)
	result, err = repository.List(t.Context(), notificationPrincipal, "test", url.Values{"per_page": {"2"}, "cursor": {"2:100000000000000000000:0"}, "type": {"created,assigned"}})
	if err != nil || result.(map[string]any)["count"] != 0 {
		t.Fatalf("guest .none() boundary = %v, %v", result, err)
	}
}
