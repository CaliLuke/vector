package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	api "github.com/CaliLuke/go-plane/projectconfiguration/gen/project_configuration"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type configurationMiddlewareDB struct {
	*pgxpool.Pool
	recorder *cycleMiddlewareRecorder
}

type configurationMiddlewareRow func(...any) error

// Label writes now enter a transaction before parsing their body. This fixture
// exercises middleware ordering; malformed input never reaches a write query.
type configurationMiddlewareTx struct {
	pgx.Tx
	db configurationMiddlewareDB
}

func (db configurationMiddlewareDB) Begin(context.Context) (pgx.Tx, error) {
	return configurationMiddlewareTx{db: db}, nil
}

func (tx configurationMiddlewareTx) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return tx.db.QueryRow(ctx, query, args...)
}

func (configurationMiddlewareTx) Commit(context.Context) error   { return nil }
func (configurationMiddlewareTx) Rollback(context.Context) error { return nil }

func (row configurationMiddlewareRow) Scan(values ...any) error { return row(values...) }

func (db configurationMiddlewareDB) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	return configurationMiddlewareRow(func(values ...any) error {
		if strings.Contains(query, "FROM project_members") {
			db.recorder.add("authorization")
			*values[0].(*bool) = !db.recorder.denied
		} else {
			db.recorder.add("dispatch")
			*values[0].(*[]byte) = []byte(`[]`)
		}
		return nil
	})
}

type configurationMiddlewareEffects struct{ t *testing.T }

type configurationMiddlewareSessions struct {
	*cycleMiddlewareSessions
	expired bool
}

// The middleware discovers this optional contract through a type assertion.
var _ sessionPrincipalInspector = configurationMiddlewareSessions{}

func (sessions configurationMiddlewareSessions) InspectReadServiceSessionWithoutTimezone(_ context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	sessions.recorder.add("session-inspection")
	if sessions.expired {
		return true, false, nil, djangosession.Principal{}, nil
	}
	if _, err := request.Cookie("session-id"); err != nil {
		return false, false, nil, djangosession.Principal{}, nil
	}
	zone := sessions.userTimezone
	if sessions.invalidZone {
		zone = "Not/AZone"
	}
	return false, false, nil, djangosession.Principal{UserID: cycleMiddlewareActor, UserTimezone: zone}, nil
}

func (effects configurationMiddlewareEffects) Task(context.Context, string, []any, map[string]any) error {
	effects.t.Fatal("unexpected durable task")
	return nil
}

func (effects configurationMiddlewareEffects) SoftDelete(context.Context, string, string, string) error {
	effects.t.Fatal("unexpected soft-delete task")
	return nil
}

func TestProjectConfigurationMiddlewareRequestSizeAuthenticationTimezoneAuthorizationThrottlingDispatchAudit(t *testing.T) {
	const base = "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/"
	for _, tc := range []struct {
		name, method, suffix, body             string
		apiKey, anonymous, denied, invalidZone bool
		expired                                bool
		status                                 int
		events                                 []string
	}{
		{name: "request-size precedes authentication", method: "POST", suffix: "estimates/", body: strings.Repeat("x", 65), status: 413},
		{name: "anonymous write", method: "POST", suffix: "estimates/", body: `{`, anonymous: true, status: 401, events: []string{"session-inspection"}},
		{name: "expired session clears cookie before rejecting write", method: "POST", suffix: "estimates/", body: `{`, expired: true, status: 401, events: []string{"session-inspection"}},
		{name: "permission-classes precede timezone", method: "POST", suffix: "estimates/", body: `{`, denied: true, invalidZone: true, status: 403, events: []string{"session-inspection", "authorization"}},
		{name: "allowed class permission reaches timezone", method: "POST", suffix: "estimates/", body: `{`, invalidZone: true, status: 400, events: []string{"session-inspection", "authorization"}},
		{name: "method permission follows timezone", method: "POST", suffix: "bulk-create-labels/", body: `{`, denied: true, invalidZone: true, status: 400, events: []string{"session-inspection", "timezone"}},
		{name: "method permission precedes body parsing", method: "POST", suffix: "bulk-create-labels/", body: `{`, denied: true, status: 403, events: []string{"session-inspection", "timezone", "authorization"}},
		{name: "read dispatch", method: "GET", suffix: "project-estimates/", status: 200, events: []string{"session-inspection", "timezone", "authorization", "dispatch"}},
		{name: "metadata timezone", method: "OPTIONS", suffix: "bulk-create-labels/", invalidZone: true, status: 400, events: []string{"session-inspection", "timezone"}},
		{name: "API authorization precedes throttling", method: "POST", suffix: "labels/", body: `{`, apiKey: true, denied: true, status: 403, events: []string{"api-key-authentication", "authorization", "audit"}},
		{name: "API throttling precedes body parsing and audit", method: "POST", suffix: "labels/", body: `{`, apiKey: true, status: 400, events: []string{"api-key-authentication", "authorization", "throttling", "audit"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &cycleMiddlewareRecorder{denied: tc.denied}
			sessions := configurationMiddlewareSessions{cycleMiddlewareSessions: &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC", invalidZone: tc.invalidZone}, expired: tc.expired}
			audit := &cycleMiddlewareActivity{recorder: recorder}
			repository, err := store.NewRepository(configurationMiddlewareDB{recorder: recorder})
			if err != nil {
				t.Fatal(err)
			}
			service, err := systemapi.NewProjectConfiguration(repository, sessions, cycleMiddlewareAPIKeys{recorder: recorder, timezone: "UTC"}, cycleMiddlewareThrottle{recorder: recorder}, configurationMiddlewareEffects{t})
			if err != nil {
				t.Fatal(err)
			}
			handler := newHTTPHandlerWithPendingReplacements(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				pendingReplacementEndpoints{projectConfiguration: api.NewEndpoints(service), projectConfigurationMetadata: service.(projectConfigurationProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsActivity: audit, activityReadsSecret: "configuration-test-secret", activityReadsBodyLimit: 64})
			path := base + tc.suffix
			if tc.apiKey {
				path = "/api/v1" + path[len("/api"):]
			}
			request := httptest.NewRequest(tc.method, path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if tc.apiKey {
				request.Header.Set("X-Api-Key", "valid-api-key")
			} else if !tc.anonymous {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status || !slices.Equal(recorder.events, tc.events) {
				t.Fatalf("status=%d events=%v body=%s; want status=%d events=%v", response.Code, recorder.events, response.Body.String(), tc.status, tc.events)
			}
			if tc.expired {
				cookies := (&http.Response{Header: response.Header()}).Cookies()
				if len(cookies) != 1 || cookies[0].Name != "session-id" || cookies[0].MaxAge != -1 {
					t.Fatalf("expired session cookies=%v", cookies)
				}
			}
			wantAudits := 0
			if tc.apiKey {
				wantAudits = 1
			}
			if len(audit.records) != wantAudits {
				t.Fatalf("audit records=%d want=%d", len(audit.records), wantAudits)
			}
		})
	}
}
