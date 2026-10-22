package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestIntakeMixedMethodAllow(t *testing.T) {
	for _, alias := range []string{"intakes", "inboxes"} {
		base := "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/" + alias + "/"
		for _, tc := range []struct{ suffix, method, allow string }{
			{"", http.MethodPost, "POST, OPTIONS"},
			{"", http.MethodOptions, "GET, POST, HEAD, OPTIONS"},
			{"", http.MethodGet, "GET, POST, HEAD, OPTIONS"},
			{"", http.MethodHead, "GET, POST, HEAD, OPTIONS"},
			{"22222222-2222-4222-8222-222222222222/", http.MethodPatch, "PATCH, DELETE, OPTIONS"},
			{"22222222-2222-4222-8222-222222222222/", http.MethodDelete, "PATCH, DELETE, OPTIONS"},
			{"22222222-2222-4222-8222-222222222222/", http.MethodOptions, "GET, PATCH, DELETE, HEAD, OPTIONS"},
			{"22222222-2222-4222-8222-222222222222/", http.MethodGet, "GET, PATCH, DELETE, HEAD, OPTIONS"},
		} {
			protocol, ok := resolveReadServiceProtocolForMethod(base+tc.suffix, tc.method)
			if !ok || protocol.allow != tc.allow {
				t.Fatalf("%s %s: found=%v Allow=%q want=%q", tc.method, base+tc.suffix, ok, protocol.allow, tc.allow)
			}
			if !protocol.authenticateMethod {
				t.Fatalf("%s %s bypasses the session protocol transport", tc.method, base+tc.suffix)
			}
		}
	}
	if _, ok := resolveReadServiceProtocolForMethod("/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/intakes/unknown/action/", http.MethodPost); ok {
		t.Fatal("nonexistent intake action resolved")
	}
}

type intakeProtocolAuthorizer struct{}

func (intakeProtocolAuthorizer) AuthorizeSessionProtocol(_ context.Context, credential string) (int, map[string]any) {
	if credential != "valid-session" {
		return http.StatusUnauthorized, map[string]any{"detail": "Authentication credentials were not provided."}
	}
	return 0, nil
}

func (intakeProtocolAuthorizer) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, _ systemapi.SessionProtocolResource) (int, map[string]any) {
	return intakeProtocolAuthorizer{}.AuthorizeSessionProtocol(ctx, credential)
}

func TestIntakeMixedProtocolAppliesSlashlessAndOptionsCompatibility(t *testing.T) {
	protocol := newIntakeMixedProtocol(
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/",
		"GET, POST, HEAD, OPTIONS",
		"Intake",
		http.MethodPost,
		true,
	)
	if !protocol.authenticateMethod {
		t.Fatal("intake transport protocol must handle slashless and OPTIONS requests")
	}
	for _, pattern := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/",
	} {
		protocol := newIntakeMixedProtocol(pattern, "GET, POST, HEAD, OPTIONS", "Intake", http.MethodPost, true)
		_, metadata := notificationProtocolMetadata(protocol)
		encoded, err := json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(encoded, &value); err != nil {
			t.Fatal(err)
		}
		canonical, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(canonical)); got != "d9dd5ab123a1d29bd5de928fc5873781b700ffa484c5ae8542b3e913a34ca513" {
			t.Fatalf("%s OPTIONS metadata SHA-256=%s, want pinned IntakeSerializer metadata", pattern, got)
		}
	}
	path := "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/inboxes/"
	var handler http.Handler = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == path {
			response.WriteHeader(http.StatusCreated)
			return
		}
		http.NotFound(response, request)
	})
	handler = newSessionServiceProtocolMiddleware(
		notificationProtocolMetadata,
		intakeProtocolAuthorizer{},
		loomhttp.RuntimeCORSPolicy{},
		func(next http.Handler) http.Handler { return next },
		djangoSessionCookieConfig{Name: "session-id", Configured: true},
		false,
		apiV1CompatibilityOptions{},
	)(handler)

	t.Run("canonical options keeps the read metadata Allow", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, path, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d want=200 body=%s", response.Code, response.Body.String())
		}
		if got, want := response.Header().Get("Allow"), "GET, POST, HEAD, OPTIONS"; got != want {
			t.Fatalf("Allow=%q want=%q", got, want)
		}
	})
	t.Run("slashless write redirects to canonical path", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, path[:len(path)-1], nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusMovedPermanently {
			t.Fatalf("status=%d want=301 body=%s", response.Code, response.Body.String())
		}
		if got := response.Header().Get("Location"); got != path {
			t.Fatalf("Location=%q want=%q", got, path)
		}
	})
	t.Run("slashless preflight retains the accepted options response", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, path[:len(path)-1], nil)
		request.Header.Set("Origin", "https://app.plane.so")
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d want=200 body=%s", response.Code, response.Body.String())
		}
	})
}
