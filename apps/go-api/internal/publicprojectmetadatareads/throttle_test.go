package publicprojectmetadatareads

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/jackc/pgx/v5/pgconn"
)

type throttleCacheStub struct {
	data map[string][]byte
	ttl  time.Duration
}

func (cache *throttleCacheStub) Get(_ context.Context, key string) ([]byte, bool, error) {
	data, found := cache.data[key]
	return data, found, nil
}

func (cache *throttleCacheStub) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	cache.data[key] = value
	cache.ttl = ttl
	return nil
}

type throttleSessionsStub struct {
	valid        bool
	usedFallback bool
	timezone     string
	err          error
	refresh      djangosession.SessionRefresh
	refreshErr   error
	refreshedKey string
	refreshedAge int64
	validatedKey string
	principalKey string
	prepared     djangosession.SessionRefresh
	rotated      bool
}

type throttleInvalidatorStub struct {
	keys []string
	err  error
}

func (invalidator *throttleInvalidatorStub) InvalidateSession(_ context.Context, key string) error {
	invalidator.keys = append(invalidator.keys, key)
	return invalidator.err
}

func (sessions *throttleSessionsStub) Inspect(_ context.Context, key string) (djangosession.Principal, djangosession.SessionRefresh, error) {
	sessions.validatedKey = key
	prepared := sessions.prepared
	if prepared.SessionKey == "" {
		prepared.SessionKey = key
	}
	if sessions.err != nil {
		return djangosession.Principal{}, prepared, sessions.err
	}
	if sessions.valid {
		sessions.principalKey = key
		userTimezone := sessions.timezone
		if userTimezone == "" {
			userTimezone = "UTC"
		}
		return djangosession.Principal{UserID: "user-id", SessionKey: key, UsedFallback: sessions.usedFallback, AuthHashUsedFallback: sessions.usedFallback, UserTimezone: userTimezone}, prepared, nil
	}
	return djangosession.Principal{}, prepared, errors.New("invalid session")
}

func (sessions *throttleSessionsStub) RefreshPrepared(_ context.Context, prepared djangosession.SessionRefresh, age int64, _ time.Time) (djangosession.SessionRefresh, error) {
	sessions.refreshedKey = prepared.SessionKey
	sessions.refreshedAge = age
	return sessions.refresh, sessions.refreshErr
}

func (sessions *throttleSessionsStub) RefreshUnaccessed(_ context.Context, key string, age int64, _ time.Time) (djangosession.SessionRefresh, error) {
	sessions.refreshedKey = key
	sessions.refreshedAge = age
	return sessions.refresh, sessions.refreshErr
}

func (sessions *throttleSessionsStub) RotateFallbackPrepared(_ context.Context, prepared djangosession.SessionRefresh, age int64, _ time.Time) (djangosession.SessionRefresh, error) {
	sessions.rotated = true
	sessions.refreshedKey = prepared.SessionKey
	sessions.refreshedAge = age
	return sessions.refresh, sessions.refreshErr
}

func TestAnonymousThrottleMatchesDRFWindowAndDjangoCacheFormat(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	throttle, _ := NewAnonymousThrottle(cache, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0)
	now := time.Unix(1_725_220_000, 250_000_000)
	throttle.now = func() time.Time { return now }
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	for index := 0; index < anonThrottleLimit; index++ {
		allowed, _, expire, refresh, _, _, err := throttle.AllowRequest(context.Background(), request)
		if err != nil || !allowed || expire || refresh {
			t.Fatalf("request %d allowed=%v expire=%v refresh=%v err=%v", index+1, allowed, expire, refresh, err)
		}
	}
	allowed, wait, expire, refresh, _, _, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || allowed || wait != time.Minute || expire || refresh {
		t.Fatalf("throttled allowed=%v wait=%v expire=%v refresh=%v err=%v", allowed, wait, expire, refresh, err)
	}
	key := ":1:throttle_anon_192.0.2.10"
	data := cache.data[key]
	history, err := decodeFloatListPickle(data)
	wantTimestamp := float64(now.UnixNano()) / float64(time.Second)
	if err != nil || len(history) != anonThrottleLimit || history[0] != wantTimestamp {
		t.Fatalf("history=%v err=%v pickle=%s", history, err, hex.EncodeToString(data))
	}
	if cache.ttl != time.Minute {
		t.Fatalf("ttl=%v", cache.ttl)
	}
	throttle.now = func() time.Time { return now.Add(time.Minute) }
	if allowed, _, expire, refresh, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil || !allowed || expire || refresh {
		t.Fatalf("expired window allowed=%v expire=%v refresh=%v err=%v", allowed, expire, refresh, err)
	}
}

func TestConfiguredAnonymousThrottleUsesAuthenticationScopeAndRate(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	throttle, err := NewConfiguredAnonymousThrottle(
		cache, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0,
		AnonymousThrottleConfig{Scope: "authentication", Rate: "2/hour"},
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_725_220_000, 0)
	throttle.now = func() time.Time { return now }
	request := httptest.NewRequest(http.MethodPost, "/auth/email-check/", nil)
	request.RemoteAddr = "192.0.2.44:1234"
	for index := 0; index < 2; index++ {
		allowed, _, _, _, _, _, err := throttle.AllowRequestWithoutTimezone(context.Background(), request)
		if err != nil || !allowed {
			t.Fatalf("request %d allowed=%v err=%v", index+1, allowed, err)
		}
	}
	allowed, wait, _, _, _, _, err := throttle.AllowRequestWithoutTimezone(context.Background(), request)
	if err != nil || allowed || wait != time.Hour {
		t.Fatalf("throttled allowed=%v wait=%v err=%v", allowed, wait, err)
	}
	if _, found := cache.data[":1:throttle_authentication_192.0.2.44"]; !found || cache.ttl != time.Hour {
		t.Fatalf("cache=%v ttl=%v", cache.data, cache.ttl)
	}
}

func TestConfiguredAnonymousThrottleRejectsInvalidRate(t *testing.T) {
	for _, rate := range []string{"", "ten/minute", "1/Weeks"} {
		_, err := NewConfiguredAnonymousThrottle(
			&throttleCacheStub{}, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0,
			AnonymousThrottleConfig{Scope: "authentication", Rate: rate},
		)
		if err == nil {
			t.Fatalf("rate %q was accepted", rate)
		}
	}
}

func TestConfiguredAnonymousThrottleAcceptsNegativeRateAndOmitsWait(t *testing.T) {
	throttle, err := NewConfiguredAnonymousThrottle(
		&throttleCacheStub{}, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0,
		AnonymousThrottleConfig{Scope: "authentication", Rate: "-1/minute"},
	)
	if err != nil {
		t.Fatal(err)
	}
	allowed, wait, _, _, _, _, err := throttle.AllowRequestWithoutTimezone(
		context.Background(), httptest.NewRequest(http.MethodPost, "/auth/email-check/", nil),
	)
	if err != nil || allowed || wait != 0 {
		t.Fatalf("allowed=%v wait=%v err=%v", allowed, wait, err)
	}
}

func TestConfiguredAnonymousThrottleOmitsWaitForOverfullHistory(t *testing.T) {
	now := time.Unix(1_725_220_000, 0)
	key := ":1:throttle_authentication_192.0.2.44"
	cache := &throttleCacheStub{data: map[string][]byte{
		key: encodeFloatListPickle([]float64{float64(now.Unix()), float64(now.Unix() - 1), float64(now.Unix() - 2)}),
	}}
	throttle, err := NewConfiguredAnonymousThrottle(
		cache, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0,
		AnonymousThrottleConfig{Scope: "authentication", Rate: "2/hour"},
	)
	if err != nil {
		t.Fatal(err)
	}
	throttle.now = func() time.Time { return now }
	request := httptest.NewRequest(http.MethodPost, "/auth/email-check/", nil)
	request.RemoteAddr = "192.0.2.44:1234"
	allowed, wait, _, _, _, _, err := throttle.AllowRequestWithoutTimezone(context.Background(), request)
	if err != nil || allowed || wait != 0 {
		t.Fatalf("allowed=%v wait=%v err=%v", allowed, wait, err)
	}
}

func TestConfiguredAnonymousThrottleExtremeNegativeRateCannotOverflowWait(t *testing.T) {
	now := time.Unix(1_725_220_000, 0)
	key := ":1:throttle_authentication_192.0.2.44"
	cache := &throttleCacheStub{data: map[string][]byte{
		key: encodeFloatListPickle([]float64{float64(now.Unix()), float64(now.Unix() - 1)}),
	}}
	throttle, err := NewConfiguredAnonymousThrottle(
		cache, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0,
		AnonymousThrottleConfig{Scope: "authentication", Rate: "-9223372036854775808/hour"},
	)
	if err != nil {
		t.Fatal(err)
	}
	throttle.now = func() time.Time { return now }
	request := httptest.NewRequest(http.MethodPost, "/auth/email-check/", nil)
	request.RemoteAddr = "192.0.2.44:1234"
	allowed, wait, _, _, _, _, err := throttle.AllowRequestWithoutTimezone(context.Background(), request)
	if err != nil || allowed || wait != 0 {
		t.Fatalf("allowed=%v wait=%v err=%v", allowed, wait, err)
	}
}

func TestConfiguredAnonymousThrottleZeroRateWaitMatchesDRF(t *testing.T) {
	now := time.Unix(1_725_220_000, 0)
	key := ":1:throttle_authentication_192.0.2.44"
	cache := &throttleCacheStub{data: map[string][]byte{}}
	throttle, err := NewConfiguredAnonymousThrottle(
		cache, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0,
		AnonymousThrottleConfig{Scope: "authentication", Rate: "0/minute"},
	)
	if err != nil {
		t.Fatal(err)
	}
	throttle.now = func() time.Time { return now }
	request := httptest.NewRequest(http.MethodPost, "/auth/email-check/", nil)
	request.RemoteAddr = "192.0.2.44:1234"
	allowed, wait, _, _, _, _, err := throttle.AllowRequestWithoutTimezone(context.Background(), request)
	if err != nil || allowed || wait != time.Minute {
		t.Fatalf("empty history allowed=%v wait=%v err=%v", allowed, wait, err)
	}
	cache.data[key] = encodeFloatListPickle([]float64{float64(now.Unix())})
	allowed, wait, _, _, _, _, err = throttle.AllowRequestWithoutTimezone(context.Background(), request)
	if err != nil || allowed || wait != 0 {
		t.Fatalf("non-empty history allowed=%v wait=%v err=%v", allowed, wait, err)
	}
}

func TestAnonymousThrottleUsesDjangoForwardedIdentifierAndExemptsSessions(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{valid: true}
	throttle, _ := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, "custom-session", false, 0)
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("X-Forwarded-For", " 192.0.2.1, 198.51.100.2 ")
	request.AddCookie(&http.Cookie{Name: "custom-session", Value: "valid"})
	if allowed, _, expire, refresh, _, userID, err := throttle.AllowRequest(context.Background(), request); err != nil || !allowed || expire || refresh || userID != "user-id" || len(cache.data) != 0 {
		t.Fatalf("authenticated request allowed=%v expire=%v userID=%q cache=%v err=%v", allowed, expire, userID, cache.data, err)
	}
	request = httptest.NewRequest("GET", "/", nil)
	request.Header.Set("X-Forwarded-For", " 192.0.2.1, 198.51.100.2 ")
	throttle.sessions = &throttleSessionsStub{}
	if allowed, _, expire, refresh, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil || !allowed || expire || refresh {
		t.Fatalf("anonymous request allowed=%v expire=%v err=%v", allowed, expire, err)
	}
	if _, found := cache.data[":1:throttle_anon_192.0.2.1,198.51.100.2"]; !found {
		t.Fatalf("cache keys=%v", cache.data)
	}
}

func TestAnonymousThrottleValidatesTimezoneAfterAuthentication(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{
		valid: true, usedFallback: true, timezone: "Mars/Olympus",
		prepared: djangosession.SessionRefresh{SessionKey: "old-session"},
		refresh:  djangosession.SessionRefresh{SessionKey: "rotated-session"},
	}
	throttle, _ := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, true, 3600)
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "valid-session-key"})
	allowed, _, _, _, rotated, userID, err := throttle.AllowRequest(context.Background(), request)
	if allowed || !errors.Is(err, djangosession.ErrTimezoneNotFound) || rotated == nil || rotated.SessionKey != "rotated-session" || !sessions.rotated || userID != "user-id" {
		t.Fatalf("allowed=%v rotated=%#v userID=%q session rotated=%v err=%v", allowed, rotated, userID, sessions.rotated, err)
	}
	if len(cache.data) != 0 {
		t.Fatalf("authenticated timezone failure consumed throttle capacity: %v", cache.data)
	}
	if _, _, _, _, err := throttle.InspectSession(context.Background(), request); err != nil {
		t.Fatalf("request-logger-only inspection activated timezone: %v", err)
	}
	if _, _, _, _, err := throttle.InspectSessionWithTimezone(context.Background(), request); !errors.Is(err, djangosession.ErrTimezoneNotFound) {
		t.Fatalf("view authentication inspection timezone error=%v", err)
	}
	allowed, _, _, _, _, userID, err = throttle.AllowRequestWithoutTimezone(context.Background(), request)
	if err != nil || !allowed || userID != "user-id" {
		t.Fatalf("non-timezone view allowed=%v userID=%q err=%v", allowed, userID, err)
	}
}

func TestAnonymousThrottleUsesLastDuplicateSessionCookie(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{valid: true}
	throttle, _ := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, "session-id", false, 3600)
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Cookie", "session-id=first-session; session-id=last-session")
	if _, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if sessions.validatedKey != "last-session" {
		t.Fatalf("validated session=%q", sessions.validatedKey)
	}
}

func TestAnonymousThrottlePreservesExplicitEmptyCookieName(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{valid: true}
	throttle, err := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, "", false, 3600)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "valid-session"})
	allowed, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || !allowed || sessions.validatedKey != "" {
		t.Fatalf("allowed=%v validated=%q err=%v", allowed, sessions.validatedKey, err)
	}
	request.Header.Set("Cookie", "; ;")
	if cookie := lastCookie(request, ""); cookie != nil {
		t.Fatalf("empty cookie chunks produced cookie=%#v", cookie)
	}
}

func TestAnonymousThrottleJoinsRepeatedCookieHeadersLikeDjango(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{valid: true}
	throttle, _ := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, "session-id", false, 3600)
	request := httptest.NewRequest("GET", "/", nil)
	request.Header["Cookie"] = []string{"session-id=first-session", "session-id=second-session"}
	if _, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if sessions.validatedKey != "first-session,session-id=second-session" {
		t.Fatalf("validated session=%q", sessions.validatedKey)
	}
}

func TestAnonymousThrottleUnquotesCookieLikePython(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{valid: true}
	throttle, _ := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, "session-id", false, 3600)
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Cookie", `session-id="quoted\054value\x"`)
	if _, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if sessions.validatedKey != "quoted,valuex" {
		t.Fatalf("validated session=%q", sessions.validatedKey)
	}
	request.Header.Set("Cookie", `session-id="\303\251abcdef"`)
	if _, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if sessions.validatedKey != "\u00c3\u00a9abcdef" {
		t.Fatalf("octal-unquoted session=%q", sessions.validatedKey)
	}
	request.Header.Set("Cookie", `session-id="éabcdef"`)
	if _, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if sessions.validatedKey != "\u00c3\u00a9abcdef" {
		t.Fatalf("Latin-1 raw-byte session=%q", sessions.validatedKey)
	}
	request.Header.Set("Cookie", "session-id=\""+string([]byte{0xff})+"abcdefg\"")
	if _, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if sessions.validatedKey != "\u00ffabcdefg" {
		t.Fatalf("isolated Latin-1 session=%q", sessions.validatedKey)
	}
}

func TestAnonymousThrottleUsesAdminSessionForInstanceRoute(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{valid: true, refresh: djangosession.SessionRefresh{SessionKey: "admin-session-key"}}
	throttle, err := NewAnonymousThrottle(
		cache, sessions, &throttleInvalidatorStub{}, "session-id", true, 604800,
		AdminSessionConfig{CookieName: "admin-session-id", CookieAge: 3600},
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	throttle.now = func() time.Time { return now }
	request := httptest.NewRequest("GET", "/api/instances/configurations/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "normal-session-key"})
	request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "admin-session-key"})
	allowed, _, _, _, prepared, _, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || !allowed || prepared == nil || sessions.validatedKey != "admin-session-key" {
		t.Fatalf("allowed=%v prepared=%#v validated=%q err=%v", allowed, prepared, sessions.validatedKey, err)
	}
	result, err := throttle.RefreshSession(context.Background(), request, *prepared)
	if err != nil || sessions.refreshedAge != 604800 || result.MaxAgeSeconds != 3600 || !result.CookieExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("refresh=%#v database age=%v err=%v", result, sessions.refreshedAge, err)
	}
}

func TestAnonymousThrottleUsesPublicSessionRegardlessOfAnchorText(t *testing.T) {
	for _, route := range []string{
		"/api/public/anchor/contains-instances-value/labels/",
		"/api/public/anchor/instances/labels/",
		"/api/workspaces/instances/projects/",
		"/api/instances-example/",
	} {
		t.Run(route, func(t *testing.T) {
			sessions := &throttleSessionsStub{valid: true, refresh: djangosession.SessionRefresh{SessionKey: "normal-session-key", MaxAgeSeconds: 604800}}
			throttle, err := NewAnonymousThrottle(&throttleCacheStub{data: map[string][]byte{}}, sessions, &throttleInvalidatorStub{}, "session-id", true, 604800)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, route, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "normal-session-key"})
			request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "admin-session-key"})
			allowed, _, _, _, prepared, _, err := throttle.AllowRequest(context.Background(), request)
			if err != nil || !allowed || prepared == nil || sessions.validatedKey != "normal-session-key" {
				t.Fatalf("allowed=%v prepared=%#v validated=%q err=%v", allowed, prepared, sessions.validatedKey, err)
			}
			result, err := throttle.RefreshSession(context.Background(), request, *prepared)
			if err != nil || result.MaxAgeSeconds != 604800 || sessions.refreshedKey != "normal-session-key" {
				t.Fatalf("refresh=%#v refreshed key=%q err=%v", result, sessions.refreshedKey, err)
			}
		})
	}
}

func TestAnonymousThrottleDoesNotAuthenticatePublicRequestWithOnlyAdminCookie(t *testing.T) {
	sessions := &throttleSessionsStub{valid: true}
	throttle, err := NewAnonymousThrottle(&throttleCacheStub{data: map[string][]byte{}}, sessions, &throttleInvalidatorStub{}, "session-id", true, 604800)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/instances/labels/", nil)
	request.AddCookie(&http.Cookie{Name: "admin-session-id", Value: "admin-session-key"})
	_, _, _, _, prepared, userID, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || prepared != nil || userID != "" || sessions.validatedKey != "" {
		t.Fatalf("prepared=%#v user=%q inspected=%q err=%v", prepared, userID, sessions.validatedKey, err)
	}
}

func TestAnonymousThrottleFinalizesUnaccessedSessionsLikeDjango(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	for _, testCase := range []struct {
		name        string
		path        string
		cookieName  string
		cookieValue string
		status      int
		saveEvery   bool
		wantExpire  bool
		wantRefresh bool
		wantMaxAge  int64
	}{
		{name: "invalid key expires on 500", path: "/api/public/anchor/x/labels/", cookieName: "session-id", cookieValue: "short", status: 500, wantExpire: true},
		{name: "valid key retained without save every", path: "/api/public/anchor/x/labels/", cookieName: "session-id", cookieValue: "valid-session", status: 413},
		{name: "valid key retained on 500", path: "/api/public/anchor/x/labels/", cookieName: "session-id", cookieValue: "valid-session", status: 500, saveEvery: true},
		{name: "standard key refreshed on 413", path: "/api/public/anchor/x/labels/", cookieName: "session-id", cookieValue: "valid-session", status: 413, saveEvery: true, wantRefresh: true, wantMaxAge: 604800},
		{name: "anchor text does not select admin cookie", path: "/api/public/anchor/instances-x/labels/", cookieName: "admin-session-id", cookieValue: "valid-session", status: 413, saveEvery: true},
		{name: "anchor text retains standard cookie age", path: "/api/public/anchor/instances-x/labels/", cookieName: "session-id", cookieValue: "valid-session", status: 413, saveEvery: true, wantRefresh: true, wantMaxAge: 604800},
		{name: "admin key gets admin cookie age", path: "/api/instances/", cookieName: "admin-session-id", cookieValue: "valid-session", status: 413, saveEvery: true, wantRefresh: true, wantMaxAge: 3600},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			databaseExpiry := now.Add(7 * 24 * time.Hour)
			sessions := &throttleSessionsStub{refresh: djangosession.SessionRefresh{
				SessionKey: "refreshed", MaxAgeSeconds: 604800, ExpiresAt: databaseExpiry, CookieExpiresAt: databaseExpiry,
			}}
			throttle, err := NewAnonymousThrottle(
				&throttleCacheStub{data: map[string][]byte{}}, sessions, &throttleInvalidatorStub{},
				"session-id", testCase.saveEvery, 604800,
				AdminSessionConfig{CookieName: "admin-session-id", CookieAge: 3600},
			)
			if err != nil {
				t.Fatal(err)
			}
			throttle.now = func() time.Time { return now }
			request := httptest.NewRequest("GET", testCase.path, nil)
			request.AddCookie(&http.Cookie{Name: testCase.cookieName, Value: testCase.cookieValue})
			expire, refresh, err := throttle.FinalizeUnaccessedSession(context.Background(), request, testCase.status)
			if err != nil || expire != testCase.wantExpire || (refresh != nil) != testCase.wantRefresh {
				t.Fatalf("expire=%v refresh=%#v err=%v", expire, refresh, err)
			}
			if refresh != nil && (refresh.MaxAgeSeconds != float64(testCase.wantMaxAge) ||
				!refresh.CookieExpiresAt.Equal(now.Add(time.Duration(testCase.wantMaxAge)*time.Second)) ||
				!refresh.ExpiresAt.Equal(databaseExpiry) || sessions.refreshedAge != 604800) {
				t.Fatalf("refresh=%#v database age=%v", refresh, sessions.refreshedAge)
			}
		})
	}
}

func TestAnonymousThrottleJoinsRepeatedForwardedForHeaders(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	throttle, _ := NewAnonymousThrottle(cache, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0)
	request := httptest.NewRequest("GET", "/", nil)
	request.Header["X-Forwarded-For"] = []string{"192.0.2.1", "198.51.100.2"}
	if allowed, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil || !allowed {
		t.Fatalf("allowed=%v err=%v", allowed, err)
	}
	if _, found := cache.data[":1:throttle_anon_192.0.2.1,198.51.100.2"]; !found {
		t.Fatalf("cache keys=%v", cache.data)
	}
}

func TestAnonymousThrottleDecodesForwardedForAsASGILatin1BeforeRemovingWhitespace(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	throttle, _ := NewAnonymousThrottle(cache, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0)
	request := httptest.NewRequest("GET", "/", nil)
	request.Header["X-Forwarded-For"] = []string{"192.0.2.1" + string([]byte{0xa0}) + ", 198.51.100.2"}
	if allowed, _, _, _, _, _, err := throttle.AllowRequest(context.Background(), request); err != nil || !allowed {
		t.Fatalf("allowed=%v err=%v", allowed, err)
	}
	if _, found := cache.data[":1:throttle_anon_192.0.2.1,198.51.100.2"]; !found {
		t.Fatalf("cache keys=%v", cache.data)
	}
}

func TestAnonymousThrottleRefreshesFallbackSessionsWithoutSaveEveryRequest(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{
		valid: true, usedFallback: true,
		refresh: djangosession.SessionRefresh{SessionKey: "rotated-session-key"},
	}
	throttle, _ := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0)
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "fallback-session-key"})
	allowed, _, expire, refresh, rotated, _, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || !allowed || expire || refresh || rotated == nil || rotated.SessionKey != "rotated-session-key" || !sessions.rotated || sessions.principalKey != "fallback-session-key" || sessions.refreshedKey != "fallback-session-key" || len(cache.data) != 0 {
		t.Fatalf("allowed=%v expire=%v refresh=%v rotated=%#v cache=%v err=%v", allowed, expire, refresh, rotated, cache.data, err)
	}
}

func TestAnonymousThrottleDoesNotTreatXAPIKeyAsSpaceSessionAuthentication(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	throttle, err := NewAnonymousThrottle(cache, &throttleSessionsStub{}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "192.0.2.20:4321"
	request.Header.Set("X-Api-Key", "ignored-by-django-space-view")
	allowed, _, expire, refresh, _, _, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || !allowed || expire || refresh {
		t.Fatalf("allowed=%v expire=%v refresh=%v err=%v", allowed, expire, refresh, err)
	}
	if _, found := cache.data[":1:throttle_anon_192.0.2.20"]; !found {
		t.Fatalf("X-Api-Key request did not consume anonymous throttle capacity: %v", cache.data)
	}
}

func TestAllowAnonymousRequestUsesSharedBucketWithoutInspectingSession(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{}
	throttle, err := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.30:4321"
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "already-inspected"})
	allowed, wait, err := throttle.AllowAnonymousRequest(context.Background(), request)
	if err != nil || !allowed || wait != 0 {
		t.Fatalf("allowed=%v wait=%v err=%v", allowed, wait, err)
	}
	if sessions.validatedKey != "" {
		t.Fatalf("anonymous throttle re-inspected session %q", sessions.validatedKey)
	}
	if _, found := cache.data[":1:throttle_anon_192.0.2.30"]; !found {
		t.Fatalf("shared anonymous bucket missing: %v", cache.data)
	}
}

func TestAnonymousThrottlePropagatesSessionStoreFailures(t *testing.T) {
	for _, sessionErr := range []error{
		fmt.Errorf("database unavailable: %w", djangosession.ErrStore),
		djangosession.ErrInvalidSessionShape,
	} {
		cache := &throttleCacheStub{data: map[string][]byte{}}
		throttle, _ := NewAnonymousThrottle(cache, &throttleSessionsStub{err: sessionErr}, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, false, 0)
		request := httptest.NewRequest("GET", "/", nil)
		request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})
		if allowed, _, expire, refresh, _, _, err := throttle.AllowRequest(context.Background(), request); allowed || expire || refresh || !errors.Is(err, sessionErr) {
			t.Fatalf("session error=%v allowed=%v expire=%v err=%v", sessionErr, allowed, expire, err)
		}
		if len(cache.data) != 0 {
			t.Fatalf("session error must not consume throttle capacity: %v", cache.data)
		}
	}
}

func TestAnonymousThrottleExpiresEmptySessionsAndFlushesAuthHashMismatch(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		err         error
		expires     bool
		invalidates bool
	}{
		{name: "invalid key", err: djangosession.ErrInvalidKey, expires: true},
		{name: "missing or expired", err: djangosession.ErrSessionNotFound, expires: true},
		{name: "corrupt stored data", err: djangosession.ErrInvalidData},
		{name: "authentication hash mismatch", err: djangosession.ErrInvalidAuthHash, expires: true, invalidates: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cache := &throttleCacheStub{data: map[string][]byte{}}
			invalidator := &throttleInvalidatorStub{}
			throttle, _ := NewAnonymousThrottle(cache, &throttleSessionsStub{err: testCase.err}, invalidator, djangosession.DefaultCookieName, false, 0)
			request := httptest.NewRequest("GET", "/", nil)
			request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})
			allowed, _, expire, refresh, _, _, err := throttle.AllowRequest(context.Background(), request)
			if err != nil || !allowed || expire != testCase.expires || refresh {
				t.Fatalf("allowed=%v expire=%v err=%v", allowed, expire, err)
			}
			if got := len(invalidator.keys); got != btoi(testCase.invalidates) {
				t.Fatalf("invalidated keys=%v", invalidator.keys)
			}
		})
	}
}

func TestAnonymousThrottlePropagatesSessionInvalidationFailure(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	invalidator := &throttleInvalidatorStub{err: errors.New("database unavailable")}
	throttle, _ := NewAnonymousThrottle(cache, &throttleSessionsStub{err: djangosession.ErrInvalidAuthHash}, invalidator, djangosession.DefaultCookieName, false, 0)
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})
	allowed, _, expire, refresh, _, _, err := throttle.AllowRequest(context.Background(), request)
	if allowed || expire || refresh || err == nil || !errors.Is(err, ErrSessionInvalidation) || !strings.Contains(err.Error(), "invalidate Django session") {
		t.Fatalf("allowed=%v expire=%v err=%v", allowed, expire, err)
	}
	if len(cache.data) != 0 {
		t.Fatalf("invalidation failure must not consume throttle capacity: %v", cache.data)
	}
}

func TestAnonymousThrottleRefreshesValidSessionOnlyWhenConfigured(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{valid: true, refresh: djangosession.SessionRefresh{SessionKey: "session-key"}}
	throttle, err := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, true, 604800)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})
	allowed, _, expire, refresh, prepared, _, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || !allowed || expire || refresh || prepared == nil || prepared.SessionKey != "session-key" {
		t.Fatalf("allowed=%v expire=%v refresh=%v prepared=%#v err=%v", allowed, expire, refresh, prepared, err)
	}
	result, err := throttle.RefreshSession(context.Background(), request, *prepared)
	if err != nil || result.SessionKey != "session-key" || sessions.refreshedKey != "session-key" {
		t.Fatalf("refresh=%#v refreshedKey=%q err=%v", result, sessions.refreshedKey, err)
	}
}

func TestAnonymousThrottleRefreshesCorruptStoredSessionWhenConfigured(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{err: djangosession.ErrInvalidData}
	throttle, err := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, true, 604800)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "corrupt-session-key"})
	allowed, _, expire, refresh, prepared, _, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || !allowed || expire || refresh || prepared == nil {
		t.Fatalf("allowed=%v expire=%v refresh=%v err=%v", allowed, expire, refresh, err)
	}
}

func TestAnonymousThrottleRetainsPreparedAnonymousSessionWhenConfigured(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{
		err:      djangosession.ErrBackend,
		prepared: djangosession.SessionRefresh{SessionKey: "session-key"},
		refresh:  djangosession.SessionRefresh{SessionKey: "session-key"},
	}
	throttle, err := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, true, 3600)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})
	allowed, _, expire, refresh, prepared, userID, err := throttle.AllowRequest(context.Background(), request)
	if err != nil || !allowed || expire || refresh || prepared == nil || userID != "" {
		t.Fatalf("allowed=%v expire=%v refresh=%v prepared=%#v userID=%q err=%v", allowed, expire, refresh, prepared, userID, err)
	}
	if _, err := throttle.RefreshSession(context.Background(), request, *prepared); err != nil || sessions.refreshedKey != "session-key" {
		t.Fatalf("refreshedKey=%q err=%v", sessions.refreshedKey, err)
	}
}

func TestAnonymousThrottleInspectsSessionLikeRequestLoggerAuthentication(t *testing.T) {
	cache := &throttleCacheStub{data: map[string][]byte{}}
	sessions := &throttleSessionsStub{valid: true}
	throttle, err := NewAnonymousThrottle(cache, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, true, 3600)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})
	expire, refresh, rotated, _, err := throttle.InspectSession(context.Background(), request)
	if err != nil || expire || refresh || rotated == nil || rotated.SessionKey != "session-key" || sessions.validatedKey != "session-key" {
		t.Fatalf("expire=%v refresh=%v rotated=%#v validated=%q err=%v", expire, refresh, rotated, sessions.validatedKey, err)
	}
	sessions.err = djangosession.ErrInvalidData
	expire, refresh, prepared, _, err := throttle.InspectSession(context.Background(), request)
	if err != nil || expire || refresh || prepared == nil {
		t.Fatalf("corrupt session expire=%v refresh=%v prepared=%#v err=%v", expire, refresh, prepared, err)
	}
	throttle.saveEvery = false
	expire, refresh, _, _, err = throttle.InspectSession(context.Background(), request)
	if err != nil || expire || refresh {
		t.Fatalf("corrupt session without save-every expire=%v refresh=%v err=%v", expire, refresh, err)
	}
	sessions.err = fmt.Errorf("database unavailable: %w", djangosession.ErrSessionStore)
	if _, _, _, _, err = throttle.InspectSession(context.Background(), request); !errors.Is(err, djangosession.ErrSessionStore) {
		t.Fatalf("session store error=%v", err)
	}
}

func btoi(value bool) int {
	if value {
		return 1
	}
	return 0
}

type sessionInvalidationDatabaseStub struct {
	query string
	args  []any
	err   error
}

func (database *sessionInvalidationDatabaseStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	database.query = query
	database.args = args
	return pgconn.NewCommandTag("DELETE 1"), database.err
}

func TestPostgreSQLSessionInvalidatorDeletesDjangoSession(t *testing.T) {
	database := &sessionInvalidationDatabaseStub{}
	invalidator, err := NewPostgreSQLSessionInvalidator(database)
	if err != nil {
		t.Fatal(err)
	}
	if err := invalidator.InvalidateSession(context.Background(), "session-key"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(database.query, "DELETE FROM sessions") || len(database.args) != 1 || database.args[0] != "session-key" {
		t.Fatalf("query=%q args=%v", database.query, database.args)
	}
}

func TestRedisThrottleOptionsMatchDjangoTLSVerificationPolicy(t *testing.T) {
	options, err := redisThrottleOptions("rediss://cache.example.test:6380/1")
	if err != nil {
		t.Fatal(err)
	}
	if options.TLSConfig == nil || !options.TLSConfig.InsecureSkipVerify {
		t.Fatalf("TLSConfig=%+v", options.TLSConfig)
	}
	plainOptions, err := redisThrottleOptions("redis://cache.example.test:6379/1")
	if err != nil {
		t.Fatal(err)
	}
	if plainOptions.TLSConfig != nil {
		t.Fatalf("plain Redis TLSConfig=%+v", plainOptions.TLSConfig)
	}
}

func TestRedisThrottleCacheDefersConfigurationUntilAThrottleRequest(t *testing.T) {
	cache, err := NewRedisThrottleCache("")
	if err != nil {
		t.Fatalf("startup configured Redis eagerly: %v", err)
	}
	if _, _, err := cache.Get(context.Background(), "key"); err == nil || !strings.Contains(err.Error(), "parse Redis URL") {
		t.Fatalf("lazy cache Get() error = %v", err)
	}
	if err := cache.Close(); err != nil {
		t.Fatalf("Close() after lazy configuration failure = %v", err)
	}
}

func TestPickleCodecReadsPythonProtocolFour(t *testing.T) {
	const fixtureHex = "80049520000000000000005d9428474008000000000000474000000000000000473ff0000000000000652e"
	fixture, err := hex.DecodeString(fixtureHex)
	if err != nil {
		t.Fatal(err)
	}
	values, err := decodeFloatListPickle(fixture)
	if err != nil || len(values) != 3 || values[0] != 3 || values[2] != 1 {
		t.Fatalf("values=%v err=%v", values, err)
	}
	if got := hex.EncodeToString(encodeFloatListPickle(values)); got != fixtureHex {
		t.Fatalf("encoded pickle=%s want=%s", got, fixtureHex)
	}
}
