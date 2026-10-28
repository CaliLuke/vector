package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/notifications"
	api "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
)

type notificationRepositoryStub struct {
	authorizeErr, detailErr error
	record                  map[string]any
	list                    any
	inherited               bool
}

func (s *notificationRepositoryStub) Authorize(context.Context, store.Principal, string) error {
	return s.authorizeErr
}
func (s *notificationRepositoryStub) List(context.Context, store.Principal, string, url.Values) (any, error) {
	return s.list, nil
}
func (s *notificationRepositoryStub) Counts(context.Context, store.Principal, string) (map[string]int64, error) {
	return map[string]int64{"total_unread_notifications_count": 2, "mention_unread_notifications_count": 1}, nil
}
func (s *notificationRepositoryStub) Detail(_ context.Context, _ store.Principal, _, _ string, inherited bool) (map[string]any, error) {
	s.inherited = inherited
	return s.record, s.detailErr
}
func notificationAdapter(t *testing.T, repository *notificationRepositoryStub) (*notificationsService, context.Context) {
	t.Helper()
	value, err := NewNotificationReads(repository, repository, &activitySessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001", UserTimezone: "UTC"}})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*notificationsService)
	ctx, err := service.APIKeyAuth(t.Context(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}
func notificationBody() map[string]any {
	return map[string]any{
		"id": "00000000-0000-0000-0000-000000000002", "workspace": "00000000-0000-0000-0000-000000000003", "receiver": "00000000-0000-0000-0000-000000000001",
		"created_at": "2026-01-02T03:04:05Z", "updated_at": "2026-01-02T03:04:05Z", "created_by": nil, "updated_by": nil, "deleted_at": nil, "archived_at": nil, "read_at": nil, "snoozed_till": nil,
		"project": nil, "entity_identifier": nil, "entity_name": "issue", "sender": "Issue changed", "title": "", "message_html": "", "message_stripped": nil,
		"triggered_by": nil, "triggered_by_details": nil, "data": nil, "message": nil,
	}
}

func TestNotificationsGeneratedResultsPreserveArbitraryJSONAndNulls(t *testing.T) {
	for _, data := range []any{nil, false, "text", json.Number("9007199254740993"), []any{nil, "2025-03-04T05:06:07Z"}, map[string]any{"large": json.Number("9007199254740993")}} {
		t.Run(fmt.Sprintf("%T", data), func(t *testing.T) {
			record := notificationBody()
			record["data"] = data
			record["message"] = data
			record["triggered_by_details"] = map[string]any{"id": "00000000-0000-0000-0000-000000000004", "first_name": "", "last_name": "", "display_name": "Actor", "avatar": "", "is_bot": false}
			repository := &notificationRepositoryStub{record: record, list: []any{record}}
			service, ctx := notificationAdapter(t, repository)
			detail, err := service.Get(ctx, &api.GetPayload{Slug: "demo", Pk: "target"})
			if err != nil {
				t.Fatal(err)
			}
			if validateNotificationErr := api.ValidateNotification(detail); validateNotificationErr != nil {
				t.Fatal(validateNotificationErr)
			}
			assertNotificationJSON(t, detail, record)
			list, err := service.List(ctx, &api.ListPayload{Slug: "demo"})
			if err != nil {
				t.Fatal(err)
			}
			assertNotificationJSON(t, list, []any{record})
			page := map[string]any{"count": 1, "total_count": 1, "total_results": 1, "total_pages": 1, "next_cursor": "2:1:0", "prev_cursor": "2:-1:1", "next_page_results": false, "prev_page_results": false, "extra_stats": nil, "grouped_by": nil, "sub_grouped_by": nil, "results": []any{record}}
			repository.list = page
			list, err = service.List(ctx, &api.ListPayload{Slug: "demo"})
			if err != nil {
				t.Fatal(err)
			}
			assertNotificationJSON(t, list, page)
		})
	}
}
func assertNotificationJSON(t *testing.T, value, want any) {
	t.Helper()
	gotBytes, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	decode := func(raw []byte) any {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var result any
		if err := decoder.Decode(&result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	if !reflect.DeepEqual(decode(gotBytes), decode(wantBytes)) {
		t.Fatalf("encoded notification=%s\nwant=%s", gotBytes, wantBytes)
	}
}

func TestNotificationReadInheritedDetailDoesNotRequireMembership(t *testing.T) {
	repository := &notificationRepositoryStub{record: notificationBody(), authorizeErr: store.Failure(403, "error", "not a member")}
	service, ctx := notificationAdapter(t, repository)
	if _, err := service.Get(ctx, &api.GetPayload{Slug: "demo", Pk: "target"}); err != nil {
		t.Fatal(err)
	}
	if !repository.inherited {
		t.Fatal("detail lost inherited source lookup contract")
	}
}
