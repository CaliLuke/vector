package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/notifications"
	notificationssvr "github.com/CaliLuke/go-plane/notificationreads/gen/http/notifications/server"
	notificationsapi "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	loomhttp "github.com/CaliLuke/loom/http"
)

type notificationMiddlewareRepository struct {
	authorizeErr               error
	authorizations, dispatches int
}

func (s *notificationMiddlewareRepository) Authorize(context.Context, store.Principal, string) error {
	s.authorizations++
	return s.authorizeErr
}
func (s *notificationMiddlewareRepository) Counts(context.Context, store.Principal, string) (map[string]int64, error) {
	s.dispatches++
	return map[string]int64{"total_unread_notifications_count": 2, "mention_unread_notifications_count": 1}, nil
}
func (*notificationMiddlewareRepository) List(context.Context, store.Principal, string, url.Values) (any, error) {
	panic("unexpected List")
}
func (*notificationMiddlewareRepository) Detail(context.Context, store.Principal, string, string, bool) (map[string]any, error) {
	panic("unexpected Detail")
}

type notificationMiddlewareSessionValidator struct{}

func (notificationMiddlewareSessionValidator) Validate(context.Context, string) (djangosession.Principal, error) {
	panic("session middleware must provide the inspected principal")
}

// Notifications use Django-session authentication. API-key throttling remains
// covered by TestActivityMiddlewareOrderRequestSizeAuthenticationTimezoneAuthorizationThrottlingDispatchAudit;
// X-Api-Key on these session routes triggers only the source's audit effect.
func TestNotificationMiddlewareOrderRequestSizeAuthenticationTimezoneAuthorizationDispatchAudit(t *testing.T) {
	for _, test := range []struct {
		name, user, timezone, body                              string
		authorizeErr                                            error
		status, inspections, authorizations, dispatches, audits int
	}{
		{name: "request-size before authentication", user: "receiver", timezone: "UTC", body: "oversized", status: 413},
		{name: "authentication before timezone", timezone: "Not/AZone", status: 401, inspections: 1, audits: 1},
		{name: "timezone before authorization", user: "receiver", timezone: "Not/AZone", authorizeErr: store.Failure(403, "error", "denied"), status: 400, inspections: 1, audits: 1},
		{name: "authorization before dispatch", user: "receiver", timezone: "UTC", authorizeErr: store.Failure(403, "error", "denied"), status: 403, inspections: 1, authorizations: 1, audits: 1},
		{name: "dispatch before audit", user: "receiver", timezone: "UTC", status: 200, inspections: 1, authorizations: 1, dispatches: 1, audits: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &notificationMiddlewareRepository{authorizeErr: test.authorizeErr}
			sessions := &publicProjectMetadataThrottleStub{userID: test.user, userTimezone: test.timezone}
			logger := &apiV1ActivityLoggerStub{beforeLog: func() {
				if repository.authorizations != test.authorizations || repository.dispatches != test.dispatches {
					t.Errorf("audit ran too early: authorization=%d dispatch=%d", repository.authorizations, repository.dispatches)
				}
			}}
			service, err := systemapi.NewNotificationReads(repository, repository, notificationMiddlewareSessionValidator{})
			if err != nil {
				t.Fatal(err)
			}
			policy, err := djangoRuntimeCORSPolicy(nil)
			if err != nil {
				t.Fatal(err)
			}
			mux := loomhttp.NewMuxer()
			server := notificationssvr.New(notificationsapi.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, policy)
			server.Use(sessionServiceMiddlewareStack(apiV1CompatibilityOptions{bodySizeLimit: 1, activityLogger: logger, activitySecret: "secret"}, sessions, djangoSessionCookieConfig{}))
			notificationssvr.Mount(mux, server)
			request := httptest.NewRequest(http.MethodGet, "/api/workspaces/demo/users/notifications/unread/", strings.NewReader(test.body))
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			request.Header.Set("X-Api-Key", "audit-only")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != test.status || sessions.calls != test.inspections || repository.authorizations != test.authorizations || repository.dispatches != test.dispatches || len(logger.records) != test.audits {
				t.Fatalf("status=%d authentication=%d authorization=%d dispatch=%d audit=%d body=%s", response.Code, sessions.calls, repository.authorizations, repository.dispatches, len(logger.records), response.Body.String())
			}
			if len(logger.records) > 0 && logger.records[0].ResponseCode != test.status {
				t.Fatalf("audit status=%d, want %d", logger.records[0].ResponseCode, test.status)
			}
		})
	}
}
