package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	intakeapi "github.com/CaliLuke/go-plane/projectintakewrites/gen/project_intake_writes"
)

func TestProjectIntakePatchRejectsUnauthorizedBeforeMutation(t *testing.T) {
	repository := &extensionIntakeRepository{denied: true}
	sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
	service, err := systemapi.NewProjectIntakeWrites(repository, sessions)
	if err != nil {
		t.Fatal(err)
	}
	pending := pendingReplacementEndpoints{
		projectIntakeWrites:    intakeapi.NewEndpoints(service),
		readServicesSession:    sessions,
		readServicesCookie:     djangoSessionCookieConfig{Name: "session-id", Configured: true},
		activityReadsBodyLimit: 1 << 20,
	}
	handler := extensionHTTPHandler(pending)
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/workspaces/demo/projects/11111111-1111-4111-8111-111111111111/intakes/22222222-2222-4222-8222-222222222222/",
		strings.NewReader(`{"description":"guest update"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("PATCH status=%d want=%d body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
	if repository.authorizations != 1 || repository.calls != 0 {
		t.Fatalf("authorization calls=%d mutation calls=%d, want 1 and 0", repository.authorizations, repository.calls)
	}
}
