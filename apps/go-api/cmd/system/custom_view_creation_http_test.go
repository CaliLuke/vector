package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	customviewcreationapi "github.com/CaliLuke/go-plane/customviewcreation/gen/custom_view_creation"
	customviewcreation "github.com/CaliLuke/go-plane/internal/customviewcreation"
	loomhttp "github.com/CaliLuke/loom/http"
)

type customViewCreationHTTPRepository struct {
	projectTargetErr, workspaceTargetErr error
	projectTargets, workspaceTargets     int
	projectUpdates, workspaceUpdates     int
	projectUser, workspaceUser           string
	projectSlug, workspaceSlug           string
	projectID                            string
	projectChanges                       customviewcreation.ProjectViewChanges
	workspaceProps                       any
}

func (repository *customViewCreationHTTPRepository) ProjectTarget(_ context.Context, user, slug, projectID string) error {
	repository.projectTargets++
	repository.projectUser, repository.projectSlug, repository.projectID = user, slug, projectID
	return repository.projectTargetErr
}

func (repository *customViewCreationHTTPRepository) WorkspaceTarget(_ context.Context, user, slug string) error {
	repository.workspaceTargets++
	repository.workspaceUser, repository.workspaceSlug = user, slug
	return repository.workspaceTargetErr
}

func (repository *customViewCreationHTTPRepository) UpdateWorkspace(_ context.Context, user, slug string, props any) error {
	repository.workspaceUpdates++
	repository.workspaceUser, repository.workspaceSlug, repository.workspaceProps = user, slug, props
	return nil
}

func (repository *customViewCreationHTTPRepository) UpdateProject(_ context.Context, user, slug, projectID string, changes customviewcreation.ProjectViewChanges) error {
	repository.projectUpdates++
	repository.projectUser, repository.projectSlug, repository.projectID, repository.projectChanges = user, slug, projectID, changes
	return nil
}

func TestCustomViewCreationProductionMounts(t *testing.T) {
	const (
		projectID = "11111111-1111-4111-8111-111111111111"
		actorID   = cycleMiddlewareActor
	)
	for _, tc := range []struct {
		name, method, path, body string
		anonymous                bool
		status                   int
	}{
		{name: "project preferences", method: http.MethodPost, path: "/api/workspaces/demo/projects/" + projectID + "/project-views/", body: `{"view_props":{"layout":"kanban"}}`, status: http.StatusNoContent},
		{name: "workspace preferences", method: http.MethodPost, path: "/api/workspaces/demo/workspace-views/", body: `{"view_props":"compact"}`, status: http.StatusNoContent},
		{name: "anonymous", method: http.MethodPost, path: "/api/workspaces/demo/workspace-views/", body: `{}`, anonymous: true, status: http.StatusUnauthorized},
		{name: "non-object project", method: http.MethodPost, path: "/api/workspaces/demo/projects/" + projectID + "/project-views/", body: `[]`, status: http.StatusBadRequest},
		{name: "non-object workspace", method: http.MethodPost, path: "/api/workspaces/demo/workspace-views/", body: `[]`, status: http.StatusBadRequest},
		{name: "invalid JSON after project lookup", method: http.MethodPost, path: "/api/workspaces/demo/projects/" + projectID + "/project-views/", body: `{`, status: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &customViewCreationHTTPRepository{}
			sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
			service, err := systemapi.NewCustomViewCreation(repository, sessions)
			if err != nil {
				t.Fatal(err)
			}
			policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.com"})
			if err != nil {
				t.Fatal(err)
			}
			mux := loomhttp.NewMuxer()
			endpoints := customviewcreationapi.NewEndpoints(service)
			mountCustomViewCreation(mux, endpoints, apiV1CompatibilityOptions{bodySizeLimit: 1 << 20}, sessions,
				djangoSessionCookieConfig{Name: "session-id", Configured: true}, policy)
			handler := loomhttp.RequestMetadataMiddleware(requestMetadataPolicy)(mux)
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if !tc.anonymous {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, tc.status, response.Body.String())
			}
			switch tc.name {
			case "project preferences":
				if repository.projectTargets != 1 || repository.projectUpdates != 1 || repository.projectUser != actorID || repository.projectSlug != "demo" || repository.projectID != projectID || !repository.projectChanges.HasViewProps {
					t.Fatalf("project target/update mismatch: %+v", repository)
				}
			case "workspace preferences":
				if repository.workspaceTargets != 1 || repository.workspaceUpdates != 1 || repository.workspaceUser != actorID || repository.workspaceSlug != "demo" || repository.workspaceProps != "compact" {
					t.Fatalf("workspace target/update mismatch: %+v", repository)
				}
			case "anonymous":
				if repository.workspaceTargets != 0 || repository.workspaceUpdates != 0 {
					t.Fatalf("anonymous request reached repository: %+v", repository)
				}
			case "non-object project", "non-object workspace":
				if repository.projectUpdates != 0 || repository.workspaceUpdates != 0 || repository.projectTargets+repository.workspaceTargets != 1 {
					t.Fatalf("non-object request target/update order mismatch: %+v", repository)
				}
			case "invalid JSON after project lookup":
				if repository.projectTargets != 1 || repository.projectUpdates != 0 {
					t.Fatalf("invalid JSON target/update order mismatch: %+v", repository)
				}
			}
		})
	}
}

func TestCustomViewCreationProtocolRoutes(t *testing.T) {
	protocols := customViewCreationProtocols()
	if len(protocols) != 2 {
		t.Fatalf("custom view protocols=%d, want 2", len(protocols))
	}
	want := map[string]string{
		"/api/workspaces/{slug}/projects/{project_id}/project-views/": "POST, OPTIONS",
		"/api/workspaces/{slug}/workspace-views/":                     "POST, OPTIONS",
	}
	for _, protocol := range protocols {
		if protocol.allow != want[protocol.pattern] || !protocol.authenticateMethod || !isCustomViewCreationProtocol(protocol.pattern) {
			t.Errorf("protocol=%+v", protocol)
		}
		delete(want, protocol.pattern)
	}
	if len(want) != 0 {
		t.Fatalf("missing protocol paths: %v", want)
	}
}
