package publicprojectmetadatareads

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

func TestReadServiceSessionInspectionDefersTimezoneAfterAuthentication(t *testing.T) {
	sessions := &throttleSessionsStub{
		valid: true, timezone: "Mars/Olympus",
		prepared: djangosession.SessionRefresh{SessionKey: "session-key"},
	}
	throttle, err := NewAnonymousThrottle(&throttleCacheStub{data: map[string][]byte{}}, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, true, 3600)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/test/deferred/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})

	_, _, _, _, ordinaryErr := throttle.InspectReadServiceSession(context.Background(), request)
	expire, refresh, prepared, principal, deferredErr := throttle.InspectReadServiceSessionWithoutTimezone(context.Background(), request)

	if !errors.Is(ordinaryErr, djangosession.ErrTimezoneNotFound) {
		t.Fatalf("ordinary inspection error=%v", ordinaryErr)
	}
	if deferredErr != nil || expire || refresh || prepared == nil || prepared.SessionKey != "session-key" {
		t.Fatalf("deferred inspection: expire=%v refresh=%v prepared=%+v err=%v", expire, refresh, prepared, deferredErr)
	}
	if principal.UserID != "user-id" || principal.UserTimezone != "Mars/Olympus" || sessions.validatedKey != "session-key" {
		t.Fatalf("principal=%+v validated key=%q", principal, sessions.validatedKey)
	}
}

func TestReadServiceDeferredTimezoneRetainsFallbackRotation(t *testing.T) {
	sessions := &throttleSessionsStub{
		valid: true, usedFallback: true, timezone: "Mars/Olympus",
		prepared: djangosession.SessionRefresh{SessionKey: "old-session"},
		refresh:  djangosession.SessionRefresh{SessionKey: "rotated-session"},
	}
	throttle, err := NewAnonymousThrottle(&throttleCacheStub{data: map[string][]byte{}}, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, true, 3600)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPatch, "/test/deferred/", nil)
	request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "old-session"})

	expire, refresh, prepared, principal, err := throttle.InspectReadServiceSessionWithoutTimezone(context.Background(), request)

	if err != nil || expire || refresh || prepared == nil || prepared.SessionKey != "rotated-session" {
		t.Fatalf("expire=%v refresh=%v prepared=%+v err=%v", expire, refresh, prepared, err)
	}
	if !sessions.rotated || sessions.refreshedKey != "old-session" || sessions.refreshedAge != 3600 || principal.UserID != "user-id" {
		t.Fatalf("rotation=%v refreshed key=%q age=%d principal=%+v", sessions.rotated, sessions.refreshedKey, sessions.refreshedAge, principal)
	}
}

func TestReadServiceDeferredTimezonePreservesSaveEverySetting(t *testing.T) {
	for _, scenario := range []struct {
		name                    string
		saveEvery, wantPrepared bool
	}{
		{"disabled", false, false},
		{"enabled", true, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			sessions := &throttleSessionsStub{valid: true, timezone: "Mars/Olympus"}
			throttle, err := NewAnonymousThrottle(&throttleCacheStub{data: map[string][]byte{}}, sessions, &throttleInvalidatorStub{}, djangosession.DefaultCookieName, scenario.saveEvery, 3600)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPut, "/test/deferred/", nil)
			request.AddCookie(&http.Cookie{Name: djangosession.DefaultCookieName, Value: "session-key"})

			_, _, prepared, principal, err := throttle.InspectReadServiceSessionWithoutTimezone(context.Background(), request)

			if err != nil || (prepared != nil) != scenario.wantPrepared || principal.UserID != "user-id" {
				t.Fatalf("prepared=%+v principal=%+v err=%v", prepared, principal, err)
			}
			if prepared != nil && prepared.SessionKey != "session-key" {
				t.Fatalf("prepared key=%q", prepared.SessionKey)
			}
			if sessions.rotated {
				t.Fatal("unexpected fallback rotation")
			}
		})
	}
}
