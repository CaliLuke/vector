package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	cycleapi "github.com/CaliLuke/go-plane/cyclelifecycle/gen/cycle_lifecycle_api"
	"github.com/CaliLuke/go-plane/gen/system"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	cyclereads "github.com/CaliLuke/go-plane/internal/cyclereads"
	publicprojectmetadatareads "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/google/uuid"
)

const cycleMiddlewareActor = "11111111-1111-4111-8111-111111111111"

type cycleMiddlewareRecorder struct {
	events []string
	denied bool
}

func (recorder *cycleMiddlewareRecorder) add(event string) {
	recorder.events = append(recorder.events, event)
}

type cycleMiddlewareRepository struct{ recorder *cycleMiddlewareRecorder }

func (repository cycleMiddlewareRepository) Create(_ context.Context, _, _, _ string, _ cyclereads.CycleCreate, _ bool, _ map[string]any) (map[string]any, error) {
	repository.recorder.add("dispatch")
	return map[string]any{"id": "22222222-2222-4222-8222-222222222222", "name": "cycle", "description": "", "assignee_ids": []string{}}, nil
}
func (repository cycleMiddlewareRepository) AuthorizeProject(context.Context, string, string, string) error {
	repository.recorder.add("authorization")
	if repository.recorder.denied {
		return cyclereads.ErrForbidden
	}
	return nil
}
func (repository cycleMiddlewareRepository) AuthorizeAppProject(context.Context, string, string, string) error {
	repository.recorder.add("authorization")
	if repository.recorder.denied {
		return cyclereads.ErrForbidden
	}
	return nil
}
func (repository cycleMiddlewareRepository) AuthorizeAppDelete(context.Context, string, string, string, string) error {
	repository.recorder.add("authorization")
	if repository.recorder.denied {
		return cyclereads.ErrForbidden
	}
	return nil
}
func (repository cycleMiddlewareRepository) AuthorizeReadProject(context.Context, string, string, string) error {
	repository.recorder.add("read-authorization")
	return nil
}
func (cycleMiddlewareRepository) DateCheck(context.Context, string, string, *string, *string, *string) (bool, error) {
	return true, nil
}
func (cycleMiddlewareRepository) SetArchived(context.Context, string, string, string, string, bool) (time.Time, error) {
	return time.Time{}, nil
}
func (cycleMiddlewareRepository) AuthorizeDelete(context.Context, string, string, string, string, bool) error {
	return nil
}
func (cycleMiddlewareRepository) Delete(context.Context, string, string, string, string, bool) error {
	return nil
}
func (cycleMiddlewareRepository) Cycle(context.Context, string, string, string) (cyclereads.CycleLifecycleResource, error) {
	return cyclereads.CycleLifecycleResource{}, nil
}
func (cycleMiddlewareRepository) Patch(context.Context, string, string, string, string, bool, map[string]any, map[string]any) (map[string]any, error) {
	return nil, nil
}
func (cycleMiddlewareRepository) PutNoop(context.Context, string, string, string, string) (map[string]any, error) {
	return nil, nil
}

type cycleMiddlewareSessions struct {
	recorder     *cycleMiddlewareRecorder
	invalidZone  bool
	userTimezone string
}

func (sessions *cycleMiddlewareSessions) Validate(_ context.Context, key string) (djangosession.Principal, error) {
	sessions.recorder.add("session-authentication")
	if key != "valid-session" {
		return djangosession.Principal{}, djangosession.ErrSessionNotFound
	}
	return djangosession.Principal{UserID: cycleMiddlewareActor, UserTimezone: sessions.userTimezone}, nil
}
func (sessions *cycleMiddlewareSessions) InspectReadServiceSession(_ context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	sessions.recorder.add("session-inspection")
	if _, err := request.Cookie("session-id"); err != nil {
		return false, false, nil, djangosession.Principal{}, nil
	}
	if sessions.invalidZone {
		sessions.recorder.add("timezone")
		return false, false, nil, djangosession.Principal{}, djangosession.ErrTimezoneNotFound
	}
	sessions.recorder.add("timezone")
	return false, false, nil, djangosession.Principal{UserID: cycleMiddlewareActor, UserTimezone: sessions.userTimezone}, nil
}
func (sessions *cycleMiddlewareSessions) InspectReadServiceSessionWithoutTimezone(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	return sessions.InspectReadServiceSession(ctx, request)
}
func (sessions *cycleMiddlewareSessions) InspectSession(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error) {
	return false, false, nil, cycleMiddlewareActor, nil
}
func (*cycleMiddlewareSessions) InspectSessionWithTimezone(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error) {
	return false, false, nil, cycleMiddlewareActor, nil
}
func (*cycleMiddlewareSessions) RefreshSession(_ context.Context, _ *http.Request, refresh djangosession.SessionRefresh) (djangosession.SessionRefresh, error) {
	return refresh, nil
}
func (*cycleMiddlewareSessions) AllowRequest(context.Context, *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	return true, 0, false, false, nil, cycleMiddlewareActor, nil
}
func (*cycleMiddlewareSessions) AllowRequestWithoutTimezone(context.Context, *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	return true, 0, false, false, nil, cycleMiddlewareActor, nil
}
func (*cycleMiddlewareSessions) FinalizeUnaccessedSession(context.Context, *http.Request, int) (bool, *djangosession.SessionRefresh, error) {
	return false, nil, nil
}

type cycleMiddlewareAPIKeys struct {
	recorder *cycleMiddlewareRecorder
	timezone string
}

func (keys cycleMiddlewareAPIKeys) Validate(_ context.Context, key string) (apikey.Principal, error) {
	keys.recorder.add("api-key-authentication")
	if key != "valid-api-key" {
		return apikey.Principal{}, apikey.ErrInvalid
	}
	return apikey.Principal{UserID: uuid.MustParse(cycleMiddlewareActor), Timezone: keys.timezone}, nil
}

type cycleMiddlewareThrottle struct{ recorder *cycleMiddlewareRecorder }

func (throttle cycleMiddlewareThrottle) Allow(context.Context, string) (apikey.Rate, error) {
	throttle.recorder.add("throttling")
	return apikey.Rate{Allowed: true, Remaining: "9", Reset: 1}, nil
}

type cycleMiddlewareActivity struct {
	recorder *cycleMiddlewareRecorder
	records  []publicprojectmetadatareads.APIActivityLog
}

func (activity *cycleMiddlewareActivity) Log(_ context.Context, record publicprojectmetadatareads.APIActivityLog) error {
	activity.recorder.add("audit")
	activity.records = append(activity.records, record)
	return nil
}

func newCycleMiddlewareTestHandler(t *testing.T, recorder *cycleMiddlewareRecorder, sessions *cycleMiddlewareSessions, keys cycleMiddlewareAPIKeys, throttle cycleMiddlewareThrottle, activity *cycleMiddlewareActivity, bodyLimit int64) http.Handler {
	t.Helper()
	service, err := systemapi.NewCycleLifecycle(cycleMiddlewareRepository{recorder: recorder}, sessions, keys, throttle)
	if err != nil {
		t.Fatal(err)
	}
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false),
		system.NewEndpoints(systemapi.NewSystem()), false, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			cycleLifecycle: cycleapi.NewEndpoints(service), cycleLifecycleMetadata: service,
			readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true},
			activityReadsActivity: activity, activityReadsSecret: "cycle-test-secret", activityReadsBodyLimit: bodyLimit,
		},
	)
}

func TestCycleLifecycleProductionMiddlewareOrderRequestSizeAuthenticationTimezoneAuthorizationThrottlingDispatchAudit(t *testing.T) {
	for _, authMode := range []string{"django-session", "api-key"} {
		t.Run(authMode, func(t *testing.T) {
			path := "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/cycles/"
			if authMode == "api-key" {
				path = "/api/v1" + path[len("/api"):]
			}
			makeRequest := func(body string, valid bool) *http.Request {
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				if authMode == "api-key" {
					if valid {
						request.Header.Set("X-Api-Key", "valid-api-key")
					} else {
						request.Header.Set("X-Api-Key", "invalid-api-key")
					}
				} else if valid {
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				}
				return request
			}
			run := func(request *http.Request, denied, invalidZone bool) (int, []string, []publicprojectmetadatareads.APIActivityLog) {
				recorder := &cycleMiddlewareRecorder{denied: denied}
				sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC", invalidZone: invalidZone}
				keyZone := "UTC"
				if invalidZone && authMode == "api-key" {
					keyZone = "Not/AZone"
				}
				activity := &cycleMiddlewareActivity{recorder: recorder}
				handler := newCycleMiddlewareTestHandler(t, recorder, sessions, cycleMiddlewareAPIKeys{recorder: recorder, timezone: keyZone}, cycleMiddlewareThrottle{recorder: recorder}, activity, 32)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				return response.Code, recorder.events, activity.records
			}
			assert := func(label string, request *http.Request, denied, invalidZone bool, wantStatus int, wantEvents []string) {
				t.Helper()
				status, events, audits := run(request, denied, invalidZone)
				if status != wantStatus || !reflect.DeepEqual(events, wantEvents) {
					t.Fatalf("%s status=%d events=%v audit=%d; want status=%d events=%v", label, status, events, len(audits), wantStatus, wantEvents)
				}
				if authMode == "api-key" && status != http.StatusRequestEntityTooLarge && len(audits) != 1 {
					t.Fatalf("%s API-key audit records=%d, want exactly one", label, len(audits))
				}
				if (authMode == "django-session" || status == http.StatusRequestEntityTooLarge) && len(audits) != 0 {
					t.Fatalf("%s unexpected API-key audit records=%d", label, len(audits))
				}
			}

			assert("request-size precedes auth and effects", makeRequest(strings.Repeat("x", 64), true), false, false, http.StatusRequestEntityTooLarge, nil)
			if authMode == "django-session" {
				assert("unauthenticated session stops before authorization", makeRequest(`{"name":"cycle"}`, false), false, false, http.StatusUnauthorized, []string{"session-inspection"})
				assert("session timezone precedes authorization", makeRequest(`{"name":"cycle"}`, true), false, true, http.StatusBadRequest, []string{"session-inspection", "timezone"})
				assert("session authorization precedes dispatch", makeRequest(`{"name":"cycle"}`, true), true, false, http.StatusForbidden, []string{"session-inspection", "timezone", "authorization"})
				assert("session dispatch follows authorization", makeRequest(`{"name":"cycle"}`, true), false, false, http.StatusCreated, []string{"session-inspection", "timezone", "authorization", "dispatch"})
				return
			}

			assert("invalid API key stops before authorization", makeRequest(`{"name":"cycle"}`, false), false, false, http.StatusForbidden, []string{"api-key-authentication", "audit"})
			assert("API authorization precedes throttle", makeRequest(`{"name":"cycle"}`, true), true, false, http.StatusForbidden, []string{"api-key-authentication", "authorization", "audit"})
			assert("API timezone follows permission and throttle", makeRequest(`{"name":"cycle"}`, true), false, true, http.StatusBadRequest, []string{"api-key-authentication", "authorization", "throttling", "audit"})
			assert("API audit follows successful dispatch", makeRequest(`{"name":"cycle"}`, true), false, false, http.StatusCreated, []string{"api-key-authentication", "authorization", "throttling", "dispatch", "audit"})
		})
	}
}
