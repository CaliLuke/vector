package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	projectpublication "github.com/CaliLuke/go-plane/projectpublication/gen/project_publication"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

type projectPublicationHTTPService struct {
	createCalls int
	updateCalls int
	deleteCalls int
}

func (s *projectPublicationHTTPService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, nil
}
func (s *projectPublicationHTTPService) Create(context.Context, *projectpublication.CreatePayload, io.ReadCloser) (*projectpublication.ProjectPublicationRawJSON, io.ReadCloser, error) {
	s.createCalls++
	return &projectpublication.ProjectPublicationRawJSON{ContentType: "application/json"}, io.NopCloser(strings.NewReader(`{"id":"board"}`)), nil
}
func (s *projectPublicationHTTPService) Update(context.Context, *projectpublication.UpdatePayload, io.ReadCloser) (*projectpublication.ProjectPublicationRawJSON, io.ReadCloser, error) {
	s.updateCalls++
	return &projectpublication.ProjectPublicationRawJSON{ContentType: "application/json"}, io.NopCloser(strings.NewReader(`{"id":"board"}`)), nil
}
func (s *projectPublicationHTTPService) Delete(context.Context, *projectpublication.DeletePayload) error {
	s.deleteCalls++
	return nil
}

type projectPublicationHTTPAuthorizer struct {
	status   int
	resource []struct {
		method, pattern, slug, projectID, id string
	}
}

func (a *projectPublicationHTTPAuthorizer) AuthorizeSessionProtocol(context.Context, string) (int, map[string]any) {
	return a.status, map[string]any{"error": "Forbidden"}
}
func (a *projectPublicationHTTPAuthorizer) AuthorizeSessionResourceProtocol(_ context.Context, _ string, resource systemapi.SessionProtocolResource) (int, map[string]any) {
	a.resource = append(a.resource, struct {
		method, pattern, slug, projectID, id string
	}{resource.Method, resource.Pattern, resource.Slug, resource.ProjectID, resource.ID})
	if a.status != 0 {
		return a.status, map[string]any{"error": "Forbidden"}
	}
	return 0, nil
}

func TestProjectPublicationGeneratedHTTPMountsAndAuthorizesWrites(t *testing.T) {
	mux := loomhttp.NewMuxer()
	service := &projectPublicationHTTPService{}
	authorizer := &projectPublicationHTTPAuthorizer{}
	endpoints := projectpublication.NewEndpoints(service)
	mountProjectPublication(mux, endpoints, authorizer, loomhttp.RuntimeCORSPolicy{}, apiV1CompatibilityOptions{}, nil, djangoSessionCookieConfig{})
	handler := http.HandlerFunc(mux.ServeHTTP)

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/project-deploy-boards/", `{}`},
		{http.MethodPatch, "/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/project-deploy-boards/00000000-0000-4000-8000-000000000002/", `{}`},
		{http.MethodDelete, "/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/project-deploy-boards/00000000-0000-4000-8000-000000000002/", ""},
	}
	for _, tc := range requests {
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		wantStatus := http.StatusOK
		if tc.method == http.MethodDelete {
			wantStatus = http.StatusNoContent
		}
		if response.Code != wantStatus {
			t.Errorf("%s %s status = %d, want %d; body=%s", tc.method, tc.path, response.Code, wantStatus, response.Body.String())
		}
	}
	if service.createCalls != 1 || service.updateCalls != 1 || service.deleteCalls != 1 {
		t.Fatalf("service calls = create %d, update %d, delete %d; want 1 each", service.createCalls, service.updateCalls, service.deleteCalls)
	}
	if len(authorizer.resource) != 3 {
		t.Fatalf("authorization calls = %d, want 3", len(authorizer.resource))
	}
	wantMethods := []string{http.MethodPost, http.MethodPatch, http.MethodDelete}
	for index, resource := range authorizer.resource {
		if resource.method != wantMethods[index] || resource.slug != "acme" || resource.projectID != "00000000-0000-4000-8000-000000000001" {
			t.Errorf("authorization resource[%d] = %+v", index, resource)
		}
		if index > 0 && resource.id != "00000000-0000-4000-8000-000000000002" {
			t.Errorf("authorization resource[%d] id = %q", index, resource.id)
		}
	}
}

func TestProjectPublicationGeneratedHTTPMountStopsForbiddenWrite(t *testing.T) {
	mux := loomhttp.NewMuxer()
	service := &projectPublicationHTTPService{}
	authorizer := &projectPublicationHTTPAuthorizer{status: http.StatusForbidden}
	mountProjectPublication(mux, projectpublication.NewEndpoints(service), authorizer, loomhttp.RuntimeCORSPolicy{}, apiV1CompatibilityOptions{}, nil, djangoSessionCookieConfig{})
	request := httptest.NewRequest(http.MethodPost, "/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/project-deploy-boards/", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	http.HandlerFunc(mux.ServeHTTP).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("forbidden create status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
	if service.createCalls != 0 {
		t.Fatalf("forbidden create reached service %d times", service.createCalls)
	}
}
