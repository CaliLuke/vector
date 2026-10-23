package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

func TestAuthSessionMiddlewareUsesExplicitCookieWithoutSavingOldSession(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		status       int
		incoming     bool
		cookie       string
		saves        int
	}{
		{"login cycles", "set", 302, true, "new-session", 0},
		{"new login", "set", 302, false, "new-session", 0},
		{"logout", "expire", 302, true, "", 0},
		{"logout on failure", "expire", 500, true, "", 0},
		{"failed login", "set", 500, true, "absent", 0},
		{"normal refresh", "none", 200, true, "old-session", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{}
			if tc.incoming {
				sessions.rotated = &djangosession.SessionRefresh{SessionKey: "old-session"}
			}
			handler := djangoAuthSessionMiddleware(sessions, djangoSessionCookieConfig{Name: "admin-session-id", Configured: true}, func(*http.Request) bool { return true })(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				current, found := djangosession.CurrentSessionFromContext(r.Context())
				if found != tc.incoming || found && current.SessionKey != "old-session" {
					t.Fatalf("wrong current session: %v %v", current, found)
				}
				switch tc.action {
				case "set":
					if !djangosession.SetResponseSession(r.Context(), djangosession.SessionRefresh{SessionKey: "new-session", MaxAgeSeconds: 3600, ExpiresAt: time.Now().Add(time.Hour)}) {
						t.Fatal("missing response action")
					}
				case "expire":
					if !djangosession.ExpireResponseSession(r.Context()) {
						t.Fatal("missing response action")
					}
				}
				w.WriteHeader(tc.status)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest("POST", "/auth/sign-in/", nil))
			if response.Code != tc.status || sessions.refreshCalls != tc.saves {
				t.Fatalf("status=%d saves=%d", response.Code, sessions.refreshCalls)
			}
			if sessions.deferredTimezoneCalls != 1 || sessions.timezoneCalls != 0 {
				t.Fatalf("auth inspector timezone=%d deferred=%d", sessions.timezoneCalls, sessions.deferredTimezoneCalls)
			}
			cookies := response.Result().Cookies()
			if tc.cookie == "absent" {
				if len(cookies) != 0 {
					t.Fatalf("unexpected cookies: %v", cookies)
				}
				return
			}
			if len(cookies) != 1 || cookies[0].Name != "admin-session-id" || cookies[0].Value != tc.cookie {
				t.Fatalf("wrong cookies: %v", cookies)
			}
			if tc.action == "expire" && cookies[0].MaxAge != -1 {
				t.Fatalf("logout did not expire cookie: %v", cookies[0])
			}
		})
	}
}

func TestAuthSessionIsAvailableWithoutAutomaticSaving(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(fmt.Sprintf("expire=%v", expire), func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{userID: "user-id", sessionKey: "authenticated-session"}
			handler := djangoAuthSessionMiddleware(sessions, djangoSessionCookieConfig{Name: "session-id", Configured: true}, authRedirectSessionPath)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				current, found := djangosession.CurrentSessionFromContext(r.Context())
				if !found || current.SessionKey != sessions.sessionKey {
					t.Fatal("authenticated session is unavailable to the sign-out handler")
				}
				if expire {
					djangosession.ExpireResponseSession(r.Context())
				}
				w.WriteHeader(http.StatusFound)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/sign-out/", nil))
			if sessions.refreshCalls != 0 {
				t.Fatal("session was saved despite automatic saving being disabled")
			}
			cookies := response.Result().Cookies()
			if expire {
				if len(cookies) != 1 || cookies[0].Name != "session-id" || cookies[0].MaxAge != -1 {
					t.Fatalf("sign-out did not expire the session cookie: %v", cookies)
				}
			} else if len(cookies) != 0 {
				t.Fatalf("unchanged session emitted cookies: %v", cookies)
			}
		})
	}
}

func TestPasswordLoginRotatesDjangoCSRFCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/auth/set-password/", nil)
	ctx, _ := djangosession.WithResponseSessionAction(request.Context())
	if !djangosession.SetResponseSession(ctx, djangosession.SessionRefresh{SessionKey: "session-key"}) {
		t.Fatal("missing session response action")
	}
	request = request.WithContext(ctx)
	policy := djangoCSRFCookieConfig{Domain: ".example.test", Secure: true}
	handler := passwordLoginCSRFCookieMiddleware(policy)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies=%v, want one rotated CSRF cookie", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != defaultCSRFCookieName || cookie.Value == "" || cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure || cookie.Domain != "example.test" || cookie.MaxAge != 31449600 {
		t.Fatalf("CSRF cookie=%+v", cookie)
	}
}

func TestPasswordLoginDoesNotRotateCSRFCookieAfterFailedResponse(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/auth/change-password/", nil)
	ctx, _ := djangosession.WithResponseSessionAction(request.Context())
	request = request.WithContext(ctx)
	handler := passwordLoginCSRFCookieMiddleware(djangoCSRFCookieConfig{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if cookies := response.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("failed response cookies=%v", cookies)
	}
}

func TestAuthRedirectSecurityMiddlewareSetsDjangoHeadersWithoutChangingAllow(t *testing.T) {
	handler := authRedirectSecurityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Location", "https://app.example.test/")
		w.WriteHeader(http.StatusFound)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/google/", nil))
	for name, want := range map[string]string{
		"Cross-Origin-Opener-Policy": "same-origin",
		"Referrer-Policy":            "same-origin",
		"Vary":                       "Accept-Encoding, Cookie, origin",
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Location":                   "https://app.example.test/",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s=%q, want %q", name, got, want)
		}
	}
	if response.Header().Get("Allow") != "" {
		t.Fatalf("unexpected Allow header on redirect: %q", response.Header().Get("Allow"))
	}
}

func TestReadServicesGzipMiddlewareSkipsPlainAuthenticationViews(t *testing.T) {
	for _, test := range []struct {
		path     string
		wantVary bool
	}{
		{path: "/auth/email-check/"},
		{path: "/auth/reset-password/dXNlcg/token/"},
		{path: "/api/users/me/", wantVary: true},
	} {
		t.Run(test.path, func(t *testing.T) {
			handler := readServicesGzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(strings.Repeat("x", 256)))
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, test.path, nil))
			got := strings.Contains(response.Header().Get("Vary"), "Accept-Encoding")
			if got != test.wantVary {
				t.Fatalf("Vary=%q, want Accept-Encoding=%t", response.Header().Get("Vary"), test.wantVary)
			}
		})
	}
}

func TestAccountAuthenticationGzipUsesDjangoResponseSizeThreshold(t *testing.T) {
	for _, test := range []struct {
		name string
		size int
		want bool
	}{
		{name: "small DRF response", size: 43, want: false},
		{name: "password response over GZip threshold", size: 256, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := accountAuthenticationGzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(strings.Repeat("x", test.size)))
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auth/set-password/", nil))
			got := strings.Contains(response.Header().Get("Vary"), "Accept-Encoding")
			if got != test.want {
				t.Fatalf("Vary=%q, want Accept-Encoding=%t", response.Header().Get("Vary"), test.want)
			}
		})
	}
}
