package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	intakeapi "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
)

func TestProjectIntakeWritesProductionTransportCompatibility(t *testing.T) {
	repository := &extensionIntakeRepository{}
	sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
	service, err := systemapi.NewProjectIntakeWrites(repository, sessions)
	if err != nil {
		t.Fatal(err)
	}
	pending := pendingReplacementEndpoints{
		projectIntakeWrites:         intakeapi.NewEndpoints(service),
		projectIntakeWritesMetadata: service.(systemapi.SessionResourceProtocolAuthorizer),
		readServicesSession:         sessions,
		readServicesCookie:          djangoSessionCookieConfig{Name: "session-id", Configured: true},
		activityReadsBodyLimit:      1 << 20,
	}
	handler := extensionHTTPHandler(pending)
	path := "/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/inboxes/"

	t.Run("canonical OPTIONS returns authenticated read metadata", func(t *testing.T) {
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
		var metadata any
		if err := json.Unmarshal(response.Body.Bytes(), &metadata); err != nil {
			t.Fatal(err)
		}
		canonical, err := json.Marshal(metadata)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(canonical)); got != "d9dd5ab123a1d29bd5de928fc5873781b700ffa484c5ae8542b3e913a34ca513" {
			t.Fatalf("Intake OPTIONS metadata SHA-256=%s, want pinned IntakeSerializer metadata", got)
		}
		if got := response.Header().Get("Vary"); !strings.Contains(strings.ToLower(got), "accept-encoding") {
			t.Fatalf("large OPTIONS representation Vary=%q, want accept-encoding", got)
		}
	})

	t.Run("anonymous canonical OPTIONS remains rejected", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d want=401 body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("slashless writes redirect before reaching the mutation", func(t *testing.T) {
		before := repository.calls
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
		if repository.calls != before {
			t.Fatalf("slashless request reached mutation: calls=%d before=%d", repository.calls, before)
		}
	})

	t.Run("slashless preflight returns the accepted options status", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, path[:len(path)-1], nil)
		request.Header.Set("Origin", "https://app.plane.so")
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d want=200 body=%s", response.Code, response.Body.String())
		}
	})
}
