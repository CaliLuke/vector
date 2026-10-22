package systemapi

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/notifications"
	api "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
)

type notificationUnreadBody struct{ reads int }

func (body *notificationUnreadBody) Read([]byte) (int, error) {
	body.reads++
	return 0, io.ErrUnexpectedEOF
}

func (*notificationUnreadBody) Close() error { return nil }

type notificationActionStub struct {
	notificationRepositoryStub
	authorizations, lookups, writes int
	markAllCalls                    int
	markAllSnoozed, markAllArchived bool
	markAllType                     string
	field                           string
	value                           any
	writeError                      error
}

func (s *notificationActionStub) Authorize(ctx context.Context, p store.Principal, slug string) error {
	s.authorizations++
	return s.notificationRepositoryStub.Authorize(ctx, p, slug)
}
func (s *notificationActionStub) Detail(ctx context.Context, p store.Principal, slug, id string, inherited bool) (map[string]any, error) {
	s.lookups++
	return s.notificationRepositoryStub.Detail(ctx, p, slug, id, inherited)
}
func (s *notificationActionStub) Set(_ context.Context, _ store.Principal, _, _, field string, value any) (map[string]any, error) {
	s.writes++
	s.field = field
	s.value = value
	return s.record, s.writeError
}
func (s *notificationActionStub) MarkAllRead(_ context.Context, _ store.Principal, _ string, snoozed, archived bool, kind string) error {
	s.markAllCalls++
	s.markAllSnoozed, s.markAllArchived, s.markAllType = snoozed, archived, kind
	return s.writeError
}
func actionService(t *testing.T, repository *notificationActionStub) (*notificationActionsService, context.Context) {
	t.Helper()
	value, err := NewNotificationActions(repository, &activitySessionStub{principal: djangosession.Principal{UserID: "00000000-0000-0000-0000-000000000001", UserTimezone: "UTC"}})
	if err != nil {
		t.Fatal(err)
	}
	service := value.(*notificationActionsService)
	ctx, err := service.APIKeyAuth(t.Context(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}
func requireActionError(t *testing.T, err error, name string) {
	t.Helper()
	var failure *api.NotificationError
	if !errors.As(err, &failure) || failure.Name != name {
		t.Fatalf("want generated action error %s, got %v", name, err)
	}
}
func TestNotificationActionAuthorizationPrecedesTargetAndWrite(t *testing.T) {
	repository := &notificationActionStub{notificationRepositoryStub: notificationRepositoryStub{authorizeErr: store.Failure(403, "error", "You don't have the required permissions.")}}
	service, ctx := actionService(t, repository)
	_, err := service.MarkUnread(ctx, &api.MarkUnreadPayload{Slug: "demo", Pk: "id"})
	requireActionError(t, err, "status_403")
	if repository.authorizations != 1 || repository.lookups != 0 || repository.writes != 0 {
		t.Fatalf("authorization/lookup/write=%d/%d/%d", repository.authorizations, repository.lookups, repository.writes)
	}
}
func TestNotificationActionInheritedDeleteAndPublicationFailure(t *testing.T) {
	repository := &notificationActionStub{notificationRepositoryStub: notificationRepositoryStub{authorizeErr: store.Failure(403, "error", "no membership"), record: notificationBody()}, writeError: errors.New("publisher failed after save")}
	service, ctx := actionService(t, repository)
	err := service.Delete(ctx, &api.DeletePayload{Slug: "demo", Pk: "id"})
	requireActionError(t, err, "status_500")
	if repository.authorizations != 0 || repository.lookups != 1 || repository.writes != 1 || !repository.inherited || repository.field != "deleted_at" {
		t.Fatalf("inherited delete boundaries: %#v", repository)
	}
}
func TestNotificationActionMissingTargetDoesNotWrite(t *testing.T) {
	repository := &notificationActionStub{notificationRepositoryStub: notificationRepositoryStub{detailErr: store.Failure(404, "error", "The required object does not exist.")}}
	service, ctx := actionService(t, repository)
	_, err := service.Unarchive(ctx, &api.UnarchivePayload{Slug: "demo", Pk: "id"})
	requireActionError(t, err, "status_404")
	if repository.authorizations != 1 || repository.lookups != 1 || repository.writes != 0 {
		t.Fatalf("authorization/lookup/write=%d/%d/%d", repository.authorizations, repository.lookups, repository.writes)
	}
}

func TestNotificationMarkAllReadAuthorizesBeforeBodyAndPreservesDefaults(t *testing.T) {
	repository := &notificationActionStub{}
	service, ctx := actionService(t, repository)
	result, err := service.MarkAllRead(ctx, &api.MarkAllReadPayload{Slug: "demo"}, io.NopCloser(strings.NewReader(`{"type":"watching","snoozed":true}`)))
	if err != nil || result == nil || result.Message != "Successful" || repository.authorizations != 1 || repository.markAllCalls != 1 || !repository.markAllSnoozed || repository.markAllArchived || repository.markAllType != "watching" {
		t.Fatalf("result=%#v error=%v authorization=%d calls=%d", result, err, repository.authorizations, repository.markAllCalls)
	}
}

func TestNotificationMarkAllReadRejectsUnauthorizedBeforeReadingBody(t *testing.T) {
	repository := &notificationActionStub{notificationRepositoryStub: notificationRepositoryStub{authorizeErr: store.Failure(403, "error", "denied")}}
	service, ctx := actionService(t, repository)
	body := &notificationUnreadBody{}
	_, err := service.MarkAllRead(ctx, &api.MarkAllReadPayload{Slug: "demo"}, body)
	requireActionError(t, err, "status_403")
	if repository.authorizations != 1 || repository.markAllCalls != 0 || body.reads != 0 {
		t.Fatalf("authorization=%d writes=%d body reads=%d", repository.authorizations, repository.markAllCalls, body.reads)
	}
}

func TestNotificationSnoozeUpdateLooksUpBeforeSerializerValidation(t *testing.T) {
	repository := &notificationActionStub{notificationRepositoryStub: notificationRepositoryStub{record: notificationBody()}}
	service, ctx := actionService(t, repository)
	_, err := service.Update(ctx, &api.UpdatePayload{Slug: "demo", Pk: "id"}, io.NopCloser(strings.NewReader(`{"snoozed_till":"not-a-datetime"}`)))
	requireActionError(t, err, "status_400")
	if repository.authorizations != 1 || repository.lookups != 1 || repository.writes != 0 {
		t.Fatalf("authorization/lookup/write=%d/%d/%d", repository.authorizations, repository.lookups, repository.writes)
	}
}

func TestNotificationSnoozeUpdateClearsOmittedValue(t *testing.T) {
	repository := &notificationActionStub{notificationRepositoryStub: notificationRepositoryStub{record: notificationBody()}}
	service, ctx := actionService(t, repository)
	result, err := service.Update(ctx, &api.UpdatePayload{Slug: "demo", Pk: "id"}, io.NopCloser(strings.NewReader(`{}`)))
	if err != nil || result == nil || repository.field != "snoozed_till" || repository.value != nil || repository.authorizations != 1 || repository.lookups != 1 || repository.writes != 1 {
		t.Fatalf("result=%#v error=%v field=%q value=%#v authorization/lookup/write=%d/%d/%d", result, err, repository.field, repository.value, repository.authorizations, repository.lookups, repository.writes)
	}
}

func TestNotificationMarkAllReadRejectsNonObjectWithoutWrites(t *testing.T) {
	for _, body := range []string{`[]`, `null`, `true`, `3`, `"text"`} {
		repository := &notificationActionStub{}
		service, ctx := actionService(t, repository)
		_, err := service.MarkAllRead(ctx, &api.MarkAllReadPayload{Slug: "demo"}, io.NopCloser(strings.NewReader(body)))
		requireActionError(t, err, "status_400")
		if repository.markAllCalls != 0 {
			t.Fatalf("invalid %s mutated notifications", body)
		}
	}
}
