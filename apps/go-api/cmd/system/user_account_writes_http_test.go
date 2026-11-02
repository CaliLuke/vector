package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"

	systemapi "github.com/CaliLuke/go-plane"
	store "github.com/CaliLuke/go-plane/internal/useraccountwrites"
	api "github.com/CaliLuke/go-plane/user_account_writes/gen/user_account_writes"
)

type accountMountRepository struct {
	emailError             error
	writes, reads, logouts int
	deactivationChecks     int
	emails                 int
	actor                  uuid.UUID
}

func (r *accountMountRepository) UpdateMe(_ context.Context, actor uuid.UUID, fields map[string]json.RawMessage) (json.RawMessage, error) {
	r.writes++
	r.actor = actor
	return json.Marshal(fields)
}
func (r *accountMountRepository) Me(context.Context, uuid.UUID) (json.RawMessage, error) {
	r.reads++
	return json.RawMessage(`{"email":"original@example.com"}`), nil
}
func (r *accountMountRepository) QueueEmailCode(context.Context, uuid.UUID, string, string, string) error {
	if r.emailError != nil {
		return r.emailError
	}
	r.emails++
	return nil
}
func (r *accountMountRepository) ChangeEmail(_ context.Context, actor uuid.UUID, _, email, _ string) (json.RawMessage, error) {
	if r.emailError != nil {
		return nil, r.emailError
	}
	r.actor = actor
	r.writes++
	r.logouts++
	r.emails += 2
	return json.Marshal(map[string]string{"email": email})
}
func (r *accountMountRepository) DeactivationBlocker(context.Context, uuid.UUID) (string, error) {
	r.deactivationChecks++
	return "", nil
}
func (r *accountMountRepository) Deactivate(_ context.Context, actor uuid.UUID, hash, _ string) error {
	r.actor = actor
	r.writes++
	if !strings.HasPrefix(hash, "pbkdf2_sha256$") {
		panic("deactivation password was not hashed")
	}
	return nil
}

type accountMountTasks struct{ names []string }

func (q *accountMountTasks) Task(_ context.Context, name string, _ []any, _ map[string]any) error {
	q.names = append(q.names, name)
	return nil
}

func TestUserAccountWritesProductionMounts(t *testing.T) {
	for _, tc := range []struct {
		method, path, body    string
		status, writes, tasks int
		contentType           string
	}{
		{method: "PATCH", path: "/api/users/me/", body: `{"display_name":"Renamed user"}`, status: 200, writes: 1},
		{method: "PATCH", path: "/api/users/me/", body: `{"display_name":"Renamed user"}`, contentType: "text/plain", status: 415},
		{method: "PATCH", path: "/api/users/me/", body: `[]`, status: 400},
		{method: "PATCH", path: "/api/users/me/", body: `{"first_name":"` + strings.Repeat("é", 256) + `"}`, status: 400},
		{method: "PATCH", path: "/api/users/me/", body: `{"first_name":"` + strings.Repeat("é", 255) + `"}`, status: 200, writes: 1},
		{method: "DELETE", path: "/api/users/me/", body: `{`, status: 204, writes: 1, tasks: 1},
		{method: "PATCH", path: "/api/users/me/email/", body: `[]`, status: 400},
		{method: "PATCH", path: "/api/users/me/email/", body: `{"email":"new@example.com","code":"123456"}`, status: 200, writes: 1, tasks: 2},
		{method: "PATCH", path: "/api/users/me/email/", body: `{"email":"new@example.com","code":"123456"}`, contentType: "text/plain", status: 415},
		{method: "POST", path: "/api/users/me/email/generate-code/", body: `[]`, status: 400},
		{method: "POST", path: "/api/users/me/email/generate-code/", body: `{"email":"` + strings.Repeat("a", 244) + `@example.com"}`, status: 400},
		{method: "POST", path: "/api/users/me/email/generate-code/", body: `{"email":"new@example.com"}`, status: 200, tasks: 1},
		{method: "POST", path: "/api/users/me/email/generate-code/", body: `{"email":"new@example.com"}`, contentType: "text/plain", status: 415},
	} {
		t.Run(tc.method+tc.path+"/"+tc.contentType, func(t *testing.T) {
			for _, authenticated := range []bool{false, true} {
				repository := &accountMountRepository{}
				quota := &testsupport.RateStoreFake{}
				tasks := &accountMountTasks{}
				sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
				handler := newAccountWriteHandler(t, repository, sessions, quota, tasks)
				request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				contentType := tc.contentType
				if contentType == "" {
					contentType = "application/json"
				}
				request.Header.Set("Content-Type", contentType)
				if authenticated {
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				want := 401
				if authenticated {
					want = tc.status
				}
				if response.Code != want {
					t.Fatalf("authenticated=%v status=%d want=%d body=%s", authenticated, response.Code, want, response.Body.String())
				}
				if authenticated && tc.contentType == "text/plain" {
					var errorBody map[string]string
					if err := json.Unmarshal(response.Body.Bytes(), &errorBody); err != nil || errorBody["detail"] != `Unsupported media type "text/plain" in request.` {
						t.Fatalf("unsupported media type response=%s", response.Body.String())
					}
				}
				if authenticated && tc.contentType == "text/plain" {
					if repository.writes+repository.reads+len(tasks.names) != 0 {
						t.Fatalf("unsupported media type reached domain I/O: writes=%d reads=%d tasks=%v", repository.writes, repository.reads, tasks.names)
					}
					if tc.path == "/api/users/me/email/generate-code/" {
						// Charge the native request limit before parsing its body.
						if len(quota.Calls) != 1 {
							t.Fatalf("unsupported media type effects=%d, want one quota admission", len(quota.Calls))
						}
						if !strings.HasPrefix(quota.Calls[0].Key, "vector:rate:v1:email-verification:") {
							t.Fatal("unsupported media type reached another quota scope")
						}
					} else if len(quota.Calls) != 0 {
						t.Fatalf("unsupported media type reached quota: accesses=%d", len(quota.Calls))
					}
				}
				if authenticated && tc.body == `[]` {
					wantBody := ""
					switch tc.path {
					case "/api/users/me/email/":
						wantBody = "Invalid email format"
					case "/api/users/me/email/generate-code/":
						wantBody = "Email is required"
					}
					if wantBody != "" && !strings.Contains(response.Body.String(), wantBody) {
						t.Fatalf("array validation response=%s want to contain %q", response.Body.String(), wantBody)
					}
				}
				if authenticated && tc.path == "/api/users/me/" && response.Header().Get("Allow") != "PATCH, DELETE, OPTIONS" {
					t.Fatalf("user write Allow=%q want=PATCH, DELETE, OPTIONS", response.Header().Get("Allow"))
				}
				if authenticated && strings.Contains(tc.body, strings.Repeat("é", 256)) && !strings.Contains(response.Body.String(), `"first_name":["Ensure this field has no more than 255 characters."]`) {
					t.Fatalf("profile validation body=%s", response.Body.String())
				}
				if !authenticated {
					if repository.writes+repository.reads+len(quota.Calls)+len(tasks.names) != 0 {
						t.Fatal("anonymous request reached domain I/O")
					}
					continue
				}
				if repository.writes != tc.writes || len(tasks.names)+repository.emails != tc.tasks {
					t.Fatalf("writes=%d tasks=%v want=%d/%d", repository.writes, tasks.names, tc.writes, tc.tasks)
				}
				if tc.writes != 0 && repository.actor == uuid.Nil() {
					t.Fatal("write lost authenticated principal")
				}
				if tc.path == "/api/users/me/email/" && tc.writes != 0 && repository.logouts != 1 {
					t.Fatal("email update did not invalidate session")
				}
				if authenticated && tc.path == "/api/users/me/email/" && tc.writes != 0 {
					cookies := response.Result().Cookies()
					if len(cookies) != 1 || cookies[0].Name != "session-id" || cookies[0].MaxAge >= 0 {
						t.Fatalf("email update response cookies=%#v, want expired session-id", cookies)
					}
				}
				if authenticated && tc.method == "DELETE" && tc.path == "/api/users/me/" {
					cookies := response.Result().Cookies()
					if len(cookies) != 1 || cookies[0].Name != "session-id" || cookies[0].MaxAge >= 0 {
						t.Fatalf("deactivation response cookies=%#v, want expired session-id", cookies)
					}
				}
			}
		})
	}
}

// This test exercises request-size, authentication, authorization, dispatch,
// and audit middleware at the production mount. Session requests intentionally
// pass through the API-key activity-audit fallback without producing an API-key
// audit record.
func TestUserAccountWritesProductionMiddlewareEvidence(t *testing.T) {
	repository := &accountMountRepository{}
	quota := &testsupport.RateStoreFake{}
	tasks := &accountMountTasks{}
	recorder := &cycleMiddlewareRecorder{}
	sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
	activity := &cycleMiddlewareActivity{recorder: recorder}
	service, err := systemapi.NewUserAccountWrites(repository, sessions, quota, tasks, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{
		userAccountWrites: api.NewEndpoints(service), userAccountWritesMetadata: service.(sessionResourceProtocolAuthorizer),
		readServicesSession: sessions, readServicesCookie: browserSessionCookieConfig{Name: "session-id", Configured: true},
		activityReadsActivity: activity, activityReadsSecret: "user-account-test-secret", activityReadsBodyLimit: 32,
	})

	oversized := httptest.NewRequest(http.MethodPatch, "/api/users/me/", strings.NewReader(`{"first_name":"`+strings.Repeat("x", 64)+`"}`))
	oversized.Header.Set("Content-Type", "application/json")
	oversized.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	tooLarge := httptest.NewRecorder()
	handler.ServeHTTP(tooLarge, oversized)
	if tooLarge.Code != http.StatusRequestEntityTooLarge || len(recorder.events) != 0 {
		t.Fatalf("request-size stage status=%d session events=%v", tooLarge.Code, recorder.events)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/users/me/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("authenticated dispatch status=%d body=%s", response.Code, response.Body.String())
	}
	if len(recorder.events) == 0 || !strings.Contains(strings.Join(recorder.events, ","), "session-inspection") {
		t.Fatalf("authentication middleware did not inspect session: %v", recorder.events)
	}
	if repository.deactivationChecks != 1 || repository.writes != 1 || len(tasks.names) != 1 {
		t.Fatalf("authorization/dispatch effects checks=%d writes=%d tasks=%v", repository.deactivationChecks, repository.writes, tasks.names)
	}
	if len(activity.records) != 0 {
		t.Fatalf("browser session request unexpectedly entered API-key audit fallback: %#v", activity.records)
	}
}

func TestEmailVerificationThrottlePrecedesTimezone(t *testing.T) {
	repository := &accountMountRepository{}
	quota := &testsupport.RateStoreFake{}
	tasks := &accountMountTasks{}
	sessions := searchMountSessions{&cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, invalidZone: true}}
	service, err := systemapi.NewUserAccountWrites(repository, sessions, quota, tasks, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{userAccountWrites: api.NewEndpoints(service), userAccountWritesMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: browserSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20})
	for attempt := 1; attempt <= 4; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/users/me/email/generate-code/", strings.NewReader(`{`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if attempt < 4 {
			if response.Code != 400 || !strings.Contains(response.Body.String(), "The required key does not exist.") {
				t.Fatalf("attempt %d: status=%d body=%s", attempt, response.Code, response.Body.String())
			}
		} else if response.Code != 429 || !strings.Contains(response.Body.String(), "RATE_LIMIT_EXCEEDED") || response.Header().Get("Retry-After") == "" {
			t.Fatalf("attempt %d: status=%d body=%s", attempt, response.Code, response.Body.String())
		}
	}
	if repository.writes+repository.reads+len(tasks.names) != 0 || len(quota.Calls) == 0 {
		t.Fatalf("writes=%d reads=%d tasks=%v quota accesses=%d", repository.writes, repository.reads, tasks.names, len(quota.Calls))
	}
	accesses := len(quota.Calls)
	request := httptest.NewRequest(http.MethodOptions, "/api/users/me/email/generate-code/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 400 || len(quota.Calls) != accesses {
		t.Fatalf("OPTIONS must validate timezone without action throttle: status=%d accesses=%d want=%d", response.Code, len(quota.Calls), accesses)
	}
}

func newAccountWriteHandler(t *testing.T, repository *accountMountRepository, sessions *cycleMiddlewareSessions, quota *testsupport.RateStoreFake, tasks *accountMountTasks) http.Handler {
	t.Helper()
	service, err := systemapi.NewUserAccountWrites(repository, sessions, quota, tasks, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	return extensionHTTPHandler(pendingReplacementEndpoints{userAccountWrites: api.NewEndpoints(service), userAccountWritesMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: browserSessionCookieConfig{Name: "session-id", Configured: true}, activityReadsBodyLimit: 1 << 20})
}

func TestEmailChangeDomainFailuresUseSafeHTTPResponses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"taken", store.ErrEmailTaken, 400, "An account with this email already exists"},
		{"unchanged", store.ErrEmailUnchanged, 400, "New email must be different from current email"},
		{"invalid code", store.ErrCodeInvalid, 400, "Invalid verification code"},
		{"expired code", store.ErrCodeExpired, 400, "Verification code has expired or is invalid"},
		{"revoked session", store.ErrUnauthenticated, 401, "Authentication credentials were not provided."},
		{"storage failure", errors.New("private database detail"), 500, "Something went wrong please try again later"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &accountMountRepository{emailError: tc.err}
			sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
			handler := newAccountWriteHandler(t, repository, sessions, &testsupport.RateStoreFake{}, &accountMountTasks{})
			request := httptest.NewRequest("PATCH", "/api/users/me/email/", strings.NewReader(`{"email":"new@example.com","code":"123456"}`))
			request.Header.Set("Content-Type", "application/json")
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status || !strings.Contains(response.Body.String(), tc.message) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
			if strings.Contains(response.Body.String(), "private database detail") {
				t.Fatal("private storage error was exposed")
			}
			if repository.writes != 0 || repository.logouts != 0 || len(response.Header().Values("Set-Cookie")) != 0 {
				t.Fatal("failed change admitted writes or expired the cookie")
			}
		})
	}
}
