package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	redirectapi "github.com/CaliLuke/go-plane/authredirects/gen/auth_redirects"
	redirectserver "github.com/CaliLuke/go-plane/authredirects/gen/http/auth_redirects/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestAuthRedirectPlainViewCSRFProductionMount(t *testing.T) {
	t.Setenv("WEB_URL", "https://web.example.test")
	var effects int
	mux := loomhttp.NewMuxer()
	endpoints := &redirectapi.Endpoints{}
	endpoints.EmailSignIn = loom.Endpoint(func(ctx context.Context, req any) (any, error) {
		effects++
		body, _ := io.ReadAll(req.(*redirectapi.EmailSignInRequestData).Body)
		if string(body) != "email=alice%40example.com&csrfmiddlewaretoken="+strings.Repeat("a", 32) {
			t.Errorf("downstream request body=%q", body)
		}
		return &redirectapi.EmailSignInResponseData{
			Result: &redirectapi.AuthRedirectResponse{Location: "/app/", ContentType: "text/html"},
			Body:   io.NopCloser(strings.NewReader("")),
		}, nil
	})
	server := redirectserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	server.Use(authRedirectSecurityMiddleware)
	redirectserver.Mount(mux, server)
	productionMount := djangoPlainViewCSRFMiddleware(nil, defaultFileSizeLimit)(mux)
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, token := range []string{"missing", "invalid"} {
			t.Run(method+"/"+token, func(t *testing.T) {
				request := httptest.NewRequest(method, "https://app.example.test/auth/sign-in/", strings.NewReader("email=alice%40example.com"))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				if token == "invalid" {
					request.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: strings.Repeat("a", 32)})
					request.Header.Set("X-CSRFToken", strings.Repeat("b", 32))
				}
				response := httptest.NewRecorder()
				productionMount.ServeHTTP(response, request)
				if response.Code != http.StatusForbidden || effects != 0 {
					t.Fatalf("status=%d effects=%d, want 403 and no dispatch", response.Code, effects)
				}
				if response.Header().Get("Content-Type") != "text/html; charset=utf-8" || !strings.Contains(response.Body.String(), "CSRF Verification Failed") {
					t.Fatalf("csrf failure response headers=%v body=%q", response.Header(), response.Body.String())
				}
				if !strings.Contains(response.Body.String(), "href=\"https://web.example.test\"") {
					t.Fatalf("csrf failure home URL missing: %q", response.Body.String())
				}
				bodyDigest := sha256.Sum256(response.Body.Bytes())
				if got := hex.EncodeToString(bodyDigest[:]); got != "5881d2511efaa72ac17186d63a879166bb171961fe60cae321625b75fd150313" {
					t.Fatalf("csrf failure body sha256=%s, want source template digest", got)
				}
			})
		}
	}
	secret := strings.Repeat("a", 32)
	valid := httptest.NewRequest(http.MethodPost, "http://app.example.test/auth/sign-in/", strings.NewReader("email=alice%40example.com&csrfmiddlewaretoken="+secret))
	valid.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	valid.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: secret})
	response := httptest.NewRecorder()
	productionMount.ServeHTTP(response, valid)
	if response.Code != http.StatusFound || effects != 1 {
		t.Fatalf("valid form csrf status=%d effects=%d, want dispatch", response.Code, effects)
	}
	oversized := httptest.NewRequest(http.MethodPost, "http://app.example.test/auth/sign-in/", strings.NewReader("csrfmiddlewaretoken="+secret))
	oversized.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response = httptest.NewRecorder()
	djangoPlainViewCSRFMiddleware(nil, 16)(mux).ServeHTTP(response, oversized)
	if response.Code != http.StatusRequestEntityTooLarge || effects != 1 {
		t.Fatalf("oversized form status=%d effects=%d, want 413 without dispatch", response.Code, effects)
	}
}

func TestAuthRedirectCSRFMatchesOnlyResolvedResetRoutes(t *testing.T) {
	for _, test := range []struct {
		path string
		want bool
	}{
		{"/auth/reset-password/dXNlcg/token/", true},
		{"/auth/spaces/reset-password/dXNlcg/token/", true},
		{"/auth/reset-password/dXNlcg/token/extra/", false},
		{"/auth/reset-password/dXNlcg/", false},
		{"/auth/reset-password//token/", false},
		{"/auth/reset-password/dXNlcg/token", false},
	} {
		t.Run(test.path, func(t *testing.T) {
			if got := authRedirectPlainViewPath(test.path); got != test.want {
				t.Fatalf("authRedirectPlainViewPath(%q)=%t want %t", test.path, got, test.want)
			}
		})
	}
}

func TestAccountCSRFSourceDiagnosticsAndOriginPrecedence(t *testing.T) {
	secret := strings.Repeat("a", 32)
	for _, test := range []struct{ name, cookie, token, origin, want string }{
		{"short token", secret, "short", "", "CSRF token from the 'X-Csrftoken' HTTP header has incorrect length."},
		{"invalid alphabet", secret, strings.Repeat("!", 32), "", "CSRF token from the 'X-Csrftoken' HTTP header has invalid characters."},
		{"unicode alphabet", secret, strings.Repeat("é", 32), "", "CSRF token from the 'X-Csrftoken' HTTP header has invalid characters."},
		{"different secret", secret, strings.Repeat("b", 32), "", "CSRF token from the 'X-Csrftoken' HTTP header incorrect."},
		{"invalid cookie", "short", secret, "", "CSRF cookie has incorrect length."},
		{"missing token", secret, "", "", "CSRF token missing."},
		{"origin before missing cookie", "", "", "https://denied.example.test", "Origin checking failed - https://denied.example.test does not match any trusted origins."},
		{"valid origin still requires token", secret, "", "http://app.example.test", "CSRF token missing."},
		{"valid origin and token", secret, secret, "http://app.example.test", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://app.example.test/auth/email-check/", nil)
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: test.cookie})
			}
			if test.token != "" {
				request.Header.Set("X-CSRFToken", test.token)
			}
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if got := accountAuthenticationCSRFReason(request, nil); got != test.want {
				t.Fatalf("reason=%q, want %q", got, test.want)
			}
		})
	}
}

func TestAccountCSRFFormTokenPreservesBodyAndPrecedesHeader(t *testing.T) {
	secret := strings.Repeat("a", 32)
	body := "csrfmiddlewaretoken=" + secret + "&email=alice%40example.com"
	request := httptest.NewRequest(http.MethodPost, "http://app.example.test/auth/email-check/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-CSRFToken", "invalid-header")
	request.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: secret})
	if reason := accountAuthenticationCSRFReason(request, nil); reason != "" {
		t.Fatalf("valid POST token rejected: %s", reason)
	}
	got, err := io.ReadAll(request.Body)
	if err != nil || string(got) != body {
		t.Fatalf("downstream body=%q err=%v", got, err)
	}
}

func TestAccountCSRFPresentEmptyHeaderIsMalformed(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://app.example.test/auth/email-check/", nil)
	request.AddCookie(&http.Cookie{Name: defaultCSRFCookieName, Value: strings.Repeat("a", 32)})
	request.Header.Set("X-CSRFToken", "")
	if got := accountAuthenticationCSRFReason(request, nil); got != "CSRF token from the 'X-Csrftoken' HTTP header has incorrect length." {
		t.Fatalf("empty header diagnostic=%q", got)
	}
}
