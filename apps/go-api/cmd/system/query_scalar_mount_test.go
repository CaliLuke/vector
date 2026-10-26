package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	activityapi "github.com/CaliLuke/go-plane/activityreads/gen/plane_internal_django_migration_api"
	projectstatereadsapi "github.com/CaliLuke/go-plane/gen/project_state_reads"
	publicprojectmetadatareadsapi "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	system "github.com/CaliLuke/go-plane/gen/system"
	workspacerecentvisitsapi "github.com/CaliLuke/go-plane/gen/workspace_recent_visits"
	workspaceapi "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
)

type recentVisitQueryContractService struct {
	workspaceRecentVisitsHTTPStub
	entityName *string
}

func (service *recentVisitQueryContractService) List(_ context.Context, payload *workspacerecentvisitsapi.ListPayload) ([]*workspacerecentvisitsapi.WorkspaceRecentVisit, error) {
	service.entityName = payload.EntityName
	return []*workspacerecentvisitsapi.WorkspaceRecentVisit{}, nil
}

func TestRecentVisitMountSelectsLastDecodedEntityName(t *testing.T) {
	service := &recentVisitQueryContractService{}
	handler := buildHTTPHandlerWithReplacementServices(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		workspacerecentvisitsapi.NewEndpoints(service), nil, nil,
	)
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/recent-visits/?entity_name=issue&entity%5Fname=project&ignored=%ZZ", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if service.entityName == nil {
		t.Fatalf("response = %d %q; entity_name was omitted", response.Code, response.Body.String())
	}
	if response.Code != http.StatusOK || *service.entityName != "project" {
		t.Fatalf("response = %d %q; entity_name = %q, want project", response.Code, response.Body.String(), *service.entityName)
	}
}

func TestProjectStateMountSelectsLastDecodedGrouped(t *testing.T) {
	service := projectStateReadsHTTPStub{}
	handler := queryContractPendingHandler(pendingReplacementEndpoints{
		projectStateReads: projectstatereadsapi.NewEndpoints(service), projectStateMetadata: service,
	})
	for _, test := range []struct{ query, prefix string }{
		{"grouped=false&gr%6Fuped=true&ignored=%ZZ", `{"backlog":`},
		{"grouped=true&grouped=&ignored=%ZZ", `[`},
	} {
		t.Run(test.query, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000002/states/?"+test.query, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || !strings.HasPrefix(response.Body.String(), test.prefix) {
				t.Fatalf("response = %d %q, want shape %q", response.Code, response.Body.String(), test.prefix)
			}
		})
	}
}

type activityExpandContractService struct {
	activityHTTPService
	expand *string
}

func (service *activityExpandContractService) GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivities(_ context.Context, payload *activityapi.GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesPayload) (*activityapi.ActivityRawJSONResponseMetadata, io.ReadCloser, error) {
	service.expand = payload.Expand
	return activityHTTPRaw(`{"results":[]}`)
}

func TestActivityMountSelectsLastDecodedExpand(t *testing.T) {
	service := &activityExpandContractService{}
	handler := queryContractPendingHandler(pendingReplacementEndpoints{
		activityReads: activityapi.NewEndpoints(service), activityReadsMetadata: service, activityReadsBodyLimit: defaultFileSizeLimit,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/acme/projects/00000000-0000-4000-8000-000000000002/issues/00000000-0000-4000-8000-000000000003/activities/?expand=actor&exp%61nd=project,workspace&ignored=%ZZ", nil)
	request.Header.Set("X-Api-Key", "valid-key")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if service.expand == nil {
		t.Fatalf("response = %d %q; expand was omitted", response.Code, response.Body.String())
	}
	if response.Code != http.StatusOK || *service.expand != "project,workspace" {
		t.Fatalf("response = %d %q; expand = %q, want project,workspace", response.Code, response.Body.String(), *service.expand)
	}
}

func TestPublicMetadataMountSelectsLastDecodedFormat(t *testing.T) {
	handler := queryContractPendingHandler(pendingReplacementEndpoints{
		publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
		publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
	})
	for _, test := range []struct {
		query  string
		status int
	}{
		{"format=xml&for%6Dat=json&ignored=%ZZ", http.StatusOK},
		{"format=json&format=%ZZ", http.StatusNotFound},
		{"format=xml&format=&ignored=%ZZ", http.StatusOK},
	} {
		t.Run(test.query, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/labels/?"+test.query, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("response = %d %q, want %d", response.Code, response.Body.String(), test.status)
			}
		})
	}
}

type slugQueryContractService struct {
	workspaceSlugHTTPStub
	slug *string
}

func (service *slugQueryContractService) Check(ctx context.Context, payload *workspaceapi.CheckPayload) (*workspaceapi.WorkspaceSlugAvailability, error) {
	service.slug = payload.Slug
	return service.workspaceSlugHTTPStub.Check(ctx, payload)
}

func TestWorkspaceSlugMountSelectsLastDecodedSlug(t *testing.T) {
	for _, test := range []struct {
		query, slug string
		status      int
	}{
		{"slug=first&sl%75g=last&ignored=%ZZ", "last", http.StatusOK},
		{"slug=first&slug=%ZZ", "%ZZ", http.StatusOK},
		{"slug=first&slug=", "", http.StatusBadRequest},
	} {
		t.Run(test.query, func(t *testing.T) {
			service := &slugQueryContractService{}
			handler := newHTTPHandlerWithStickiesAndWorkspaceSlugCheck(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, nil, nil, workspaceapi.NewEndpoints(service))
			request := httptest.NewRequest(http.MethodGet, "/api/workspace-slug-check/?"+test.query, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			actual := ""
			if service.slug != nil {
				actual = *service.slug
			}
			if response.Code != test.status || actual != test.slug {
				t.Fatalf("response = %d %q; slug = %q, want %d %q", response.Code, response.Body.String(), actual, test.status, test.slug)
			}
		})
	}
}

func queryContractPendingHandler(pending pendingReplacementEndpoints) http.Handler {
	return newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pending,
	)
}
