package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	store "github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
)

type accountAuthenticationMiddlewareRepository struct {
	accountAuthenticationHTTPRepository
	userLookups int
	user        store.User
	userErr     error
}

func (repository *accountAuthenticationMiddlewareRepository) UserByID(context.Context, string) (store.User, error) {
	repository.userLookups++
	return repository.user, repository.userErr
}

type accountAuthenticationMiddlewareSessions struct {
	*cycleMiddlewareSessions
	inspections int
	userID      string
}

func (sessions *accountAuthenticationMiddlewareSessions) InspectReadServiceSessionWithoutTimezone(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	sessions.inspections++
	return false, false, nil, djangosession.Principal{UserID: sessions.userID, UserTimezone: "Not/AZone"}, nil
}

func TestAccountAuthenticationAPIViewUsesOptionalSessionCSRFWithoutTimezone(t *testing.T) {
	repository := &accountAuthenticationMiddlewareRepository{}
	recorder := &cycleMiddlewareRecorder{}
	sessions := &accountAuthenticationMiddlewareSessions{
		cycleMiddlewareSessions: &cycleMiddlewareSessions{recorder: recorder},
		userID:                  "user-1",
	}
	service, err := systemapi.NewAccountAuthentication(
		repository,
		accountAuthenticationHTTPThrottle{},
		accountAuthenticationHTTPCache{},
		accountAuthenticationHTTPTasks{},
		accountAuthenticationHTTPMagicIssuer{},
		accountAuthenticationHTTPResetIssuer{},
		password.Default(),
		&djangosession.Validator{},
		"https://app.example",
		"https://space.example",
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{
		accountAuthentication:         api.NewEndpoints(service),
		accountAuthenticationMetadata: service.(systemapi.SessionProtocolAuthorizer),
		readServicesAuthSession:       sessions,
		readServicesAuthCookie:        djangoSessionCookieConfig{Name: "session-id", Configured: true},
	})
	request := httptest.NewRequest(http.MethodPost, "https://app.example/auth/email-check/", strings.NewReader(`{"email":"alice@example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://app.example")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "CSRF cookie not set") {
		t.Fatalf("authenticated public APIView status=%d body=%s, want CSRF 403", response.Code, response.Body.String())
	}
	if sessions.inspections != 1 || repository.userLookups != 0 {
		t.Fatalf("session inspections=%d user lookups=%d, want one session inspection before blocked dispatch", sessions.inspections, repository.userLookups)
	}
}

func TestAccountAuthenticationProductionMiddlewareOrderAndTimezoneProfile(t *testing.T) {
	repository := &accountAuthenticationMiddlewareRepository{}
	recorder := &cycleMiddlewareRecorder{}
	sessions := &accountAuthenticationMiddlewareSessions{cycleMiddlewareSessions: &cycleMiddlewareSessions{recorder: recorder}}
	activity := &cycleMiddlewareActivity{recorder: recorder}
	service, err := systemapi.NewAccountAuthentication(
		repository,
		accountAuthenticationHTTPThrottle{},
		accountAuthenticationHTTPCache{},
		accountAuthenticationHTTPTasks{},
		accountAuthenticationHTTPMagicIssuer{},
		accountAuthenticationHTTPResetIssuer{},
		password.Default(),
		&djangosession.Validator{},
		"https://app.example",
		"https://space.example",
	)
	if err != nil {
		t.Fatal(err)
	}
	endpoints := api.NewEndpoints(service)
	handler := extensionHTTPHandler(pendingReplacementEndpoints{
		accountAuthentication:         endpoints,
		accountAuthenticationMetadata: service.(systemapi.SessionProtocolAuthorizer),
		readServicesAuthSession:       sessions,
		readServicesAuthCookie:        djangoSessionCookieConfig{Name: "session-id", Configured: true},
		activityReadsActivity:         activity,
		activityReadsSecret:           "account-authentication-test-secret",
		activityReadsBodyLimit:        64,
	})

	// Public APIViews allow anonymous requests while applying optional session
	// authentication without initializing the user's timezone.
	public := httptest.NewRequest(http.MethodPost, "/auth/email-check/", strings.NewReader(`{"email":"alice@example.com"}`))
	public.Header.Set("Content-Type", "application/json")
	public.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	publicResponse := httptest.NewRecorder()
	handler.ServeHTTP(publicResponse, public)
	if publicResponse.Code != http.StatusOK {
		t.Fatalf("public endpoint status=%d body=%s", publicResponse.Code, publicResponse.Body.String())
	}
	if sessions.inspections != 1 {
		t.Fatalf("public endpoint inspected %d sessions, want 1", sessions.inspections)
	}

	// Password endpoints require the session, but their Django source is a plain
	// APIView (without TimezoneMixin). The production adapter therefore inspects
	// the session without rejecting the source user's invalid timezone.
	passwordRequest := httptest.NewRequest(http.MethodPost, "/auth/change-password/", strings.NewReader(`{"new_password":"StrongPassword123!"}`))
	passwordRequest.Header.Set("Content-Type", "application/json")
	passwordRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	passwordResponse := httptest.NewRecorder()
	handler.ServeHTTP(passwordResponse, passwordRequest)
	if passwordResponse.Code != http.StatusUnauthorized {
		t.Fatalf("password endpoint treated timezone as invalid: status=%d body=%s", passwordResponse.Code, passwordResponse.Body.String())
	}
	if sessions.inspections != 2 {
		t.Fatalf("password endpoint session inspections=%d, want 2", sessions.inspections)
	}
	if repository.userLookups != 0 {
		t.Fatalf("password endpoint user lookups=%d, want 0 before authorization", repository.userLookups)
	}

	var body json.RawMessage
	if err := json.Unmarshal(passwordResponse.Body.Bytes(), &body); err != nil {
		t.Fatalf("password response is not JSON: status=%d body=%s", passwordResponse.Code, passwordResponse.Body.String())
	}

	// Request-size rejection is outside session inspection and dispatch.
	oversized := httptest.NewRequest(http.MethodPost, "/auth/email-check/", strings.NewReader(strings.Repeat("x", 65)))
	oversizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(oversizedResponse, oversized)
	if oversizedResponse.Code != http.StatusRequestEntityTooLarge || sessions.inspections != 2 || repository.userLookups != 0 {
		t.Fatalf("oversized request status=%d session inspections=%d user lookups=%d", oversizedResponse.Code, sessions.inspections, repository.userLookups)
	}
	if len(activity.records) != 0 {
		t.Fatalf("oversized request reached audit: %#v", activity.records)
	}

	// The composed API-key audit fallback records the response after dispatch.
	audited := httptest.NewRequest(http.MethodPost, "/auth/email-check/", strings.NewReader(`{"email":"audit@example.com"}`))
	audited.Header.Set("Content-Type", "application/json")
	audited.Header.Set("X-Api-Key", "account-key")
	auditedResponse := httptest.NewRecorder()
	handler.ServeHTTP(auditedResponse, audited)
	if auditedResponse.Code != http.StatusOK || len(activity.records) != 1 {
		t.Fatalf("audited request status=%d audit records=%d body=%s", auditedResponse.Code, len(activity.records), auditedResponse.Body.String())
	}
	if record := activity.records[0]; record.Path != "/auth/email-check/" || record.Method != http.MethodPost || record.ResponseCode != http.StatusOK {
		t.Fatalf("audit record=%#v", record)
	}
}

func TestAccountAuthenticationProductionMountSeparatesCSRFTrustFromCORS(t *testing.T) {
	repository := &accountAuthenticationMiddlewareRepository{}
	recorder := &cycleMiddlewareRecorder{}
	sessions := &accountAuthenticationMiddlewareSessions{
		cycleMiddlewareSessions: &cycleMiddlewareSessions{recorder: recorder},
		userID:                  "user-1",
	}
	service, err := systemapi.NewAccountAuthentication(
		repository,
		accountAuthenticationHTTPThrottle{},
		accountAuthenticationHTTPCache{},
		accountAuthenticationHTTPTasks{},
		accountAuthenticationHTTPMagicIssuer{},
		accountAuthenticationHTTPResetIssuer{},
		password.Default(),
		&djangosession.Validator{},
		"https://app.example",
		"https://space.example",
	)
	if err != nil {
		t.Fatal(err)
	}
	secret := "abcdefghijklmnopqrstuvwxyzABCDEF"
	token, err := systemapi.NewCSRFToken().CsrfToken(context.Background(), &csrftoken.CsrfTokenPayload{CsrfCookie: &secret})
	if err != nil {
		t.Fatal(err)
	}
	trustedOrigin := "https://csrf-trusted.example.test"
	handler := extensionHTTPHandler(pendingReplacementEndpoints{
		accountAuthentication:                   api.NewEndpoints(service),
		accountAuthenticationMetadata:           service.(systemapi.SessionProtocolAuthorizer),
		accountAuthenticationCSRFTrustedOrigins: []string{trustedOrigin},
		readServicesCORS:                        []string{"https://cors-only.example.test"},
		readServicesAuthSession:                 sessions,
		readServicesAuthCookie:                  djangoSessionCookieConfig{Name: "session-id", Configured: true},
	})
	request := httptest.NewRequest(http.MethodPost, "https://api.example.test/auth/email-check/", strings.NewReader(`{"email":"alice@example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", trustedOrigin)
	request.Header.Set("X-CSRFToken", token.CsrfToken)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	request.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: secret})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("trusted CSRF origin was rejected: status=%d body=%s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("CSRF trusted origin was incorrectly granted CORS: ACAO=%q headers=%v", got, response.Header())
	}
}

func TestAccountAuthenticationUnsupportedMediaTypeUses415ThroughProductionMount(t *testing.T) {
	repository := &accountAuthenticationMiddlewareRepository{}
	recorder := &cycleMiddlewareRecorder{}
	sessions := &accountAuthenticationMiddlewareSessions{cycleMiddlewareSessions: &cycleMiddlewareSessions{recorder: recorder}}
	service, err := systemapi.NewAccountAuthentication(
		repository,
		accountAuthenticationHTTPThrottle{},
		accountAuthenticationHTTPCache{},
		accountAuthenticationHTTPTasks{},
		accountAuthenticationHTTPMagicIssuer{},
		accountAuthenticationHTTPResetIssuer{},
		password.Default(),
		&djangosession.Validator{},
		"https://app.example",
		"https://space.example",
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{
		accountAuthentication:         api.NewEndpoints(service),
		accountAuthenticationMetadata: service.(systemapi.SessionProtocolAuthorizer),
		readServicesAuthSession:       sessions,
		readServicesAuthCookie:        djangoSessionCookieConfig{Name: "session-id", Configured: true},
	})
	request := httptest.NewRequest(http.MethodPost, "/auth/email-check/", strings.NewReader("email=alice%40example.com"))
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unsupported content type status=%d body=%s, want 415", response.Code, response.Body.String())
	}
	if got, want := strings.TrimSpace(response.Body.String()), `{"detail":"Unsupported media type \"text/plain\" in request."}`; got != want {
		t.Fatalf("unsupported content type body=%s, want %s", got, want)
	}
}
