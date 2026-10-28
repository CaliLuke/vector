package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	cyclereads "github.com/CaliLuke/go-plane/gen/cycle_reads"
	deployboardreads "github.com/CaliLuke/go-plane/gen/deploy_board_reads"
	estimatereads "github.com/CaliLuke/go-plane/gen/estimate_reads"
	cyclereadssvr "github.com/CaliLuke/go-plane/gen/http/cycle_reads/server"
	deployboardreadssvr "github.com/CaliLuke/go-plane/gen/http/deploy_board_reads/server"
	estimatereadssvr "github.com/CaliLuke/go-plane/gen/http/estimate_reads/server"
	intakeinboxreadssvr "github.com/CaliLuke/go-plane/gen/http/intake_inbox_reads/server"
	modulereadssvr "github.com/CaliLuke/go-plane/gen/http/module_reads/server"
	pagereadssvr "github.com/CaliLuke/go-plane/gen/http/page_reads/server"
	workspaceprojectreadssvr "github.com/CaliLuke/go-plane/gen/http/workspace_project_reads/server"
	intakeinboxreads "github.com/CaliLuke/go-plane/gen/intake_inbox_reads"
	modulereads "github.com/CaliLuke/go-plane/gen/module_reads"
	pagereads "github.com/CaliLuke/go-plane/gen/page_reads"
	workspaceprojectreads "github.com/CaliLuke/go-plane/gen/workspace_project_reads"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	pagereadsstore "github.com/CaliLuke/go-plane/internal/pagereads"
	workitemhistorysvr "github.com/CaliLuke/go-plane/workitemhistory/gen/http/work_item_history/server"
	workitemhistoryapi "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

type readServiceMount struct {
	service string
	method  string
	verb    string
	pattern string
}

func TestReadServiceGeneratedMountsAccountForSelectedFiftyNineRoutes(t *testing.T) {
	mux := loomhttp.NewMuxer()
	var mounts []readServiceMount

	workspaceProjectServer := workspaceprojectreadssvr.New(&workspaceprojectreads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	workspaceprojectreadssvr.Mount(mux, workspaceProjectServer)
	for _, mount := range workspaceProjectServer.Mounts {
		mounts = append(mounts, readServiceMount{"workspace_project_reads", mount.Method, mount.Verb, mount.Pattern})
	}
	cycleServer := cyclereadssvr.New(&cyclereads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	cyclereadssvr.Mount(mux, cycleServer)
	for _, mount := range cycleServer.Mounts {
		mounts = append(mounts, readServiceMount{"cycle_reads", mount.Method, mount.Verb, mount.Pattern})
	}
	moduleServer := modulereadssvr.New(&modulereads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	modulereadssvr.Mount(mux, moduleServer)
	for _, mount := range moduleServer.Mounts {
		mounts = append(mounts, readServiceMount{"module_reads", mount.Method, mount.Verb, mount.Pattern})
	}
	estimateServer := estimatereadssvr.New(&estimatereads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	estimatereadssvr.Mount(mux, estimateServer)
	for _, mount := range estimateServer.Mounts {
		mounts = append(mounts, readServiceMount{"estimate_reads", mount.Method, mount.Verb, mount.Pattern})
	}
	pageServer := pagereadssvr.New(&pagereads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	pagereadssvr.Mount(mux, pageServer)
	for _, mount := range pageServer.Mounts {
		mounts = append(mounts, readServiceMount{"page_reads", mount.Method, mount.Verb, mount.Pattern})
	}
	intakeServer := intakeinboxreadssvr.New(&intakeinboxreads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	intakeinboxreadssvr.Mount(mux, intakeServer)
	for _, mount := range intakeServer.Mounts {
		mounts = append(mounts, readServiceMount{"intake_inbox_reads", mount.Method, mount.Verb, mount.Pattern})
	}
	deployBoardServer := deployboardreadssvr.New(&deployboardreads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	deployboardreadssvr.Mount(mux, deployBoardServer)
	for _, mount := range deployBoardServer.Mounts {
		mounts = append(mounts, readServiceMount{"deploy_board_reads", mount.Method, mount.Verb, mount.Pattern})
	}
	historyServer := workitemhistorysvr.New(&workitemhistoryapi.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	workitemhistorysvr.Mount(mux, historyServer)
	for _, mount := range historyServer.Mounts {
		mounts = append(mounts, readServiceMount{"work_item_history", mount.Method, mount.Verb, mount.Pattern})
	}

	want := map[string]string{
		"/api/users/me/workspaces/":                                                                                      "workspace_project_reads",
		"/api/workspaces/":                                                                                               "workspace_project_reads",
		"/api/workspaces/{slug}/":                                                                                        "workspace_project_reads",
		"/api/workspaces/{slug}/project-stats/":                                                                          "workspace_project_reads",
		"/api/workspaces/{slug}/projects/":                                                                               "workspace_project_reads",
		"/api/workspaces/{slug}/projects/details/":                                                                       "workspace_project_reads",
		"/api/workspaces/{slug}/projects/{pk}/":                                                                          "workspace_project_reads",
		"/api/workspaces/{slug}/cycles/":                                                                                 "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/":                                                  "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/":                                             "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/":                                                           "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/":                                      "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/":                                        "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/":                                   "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/":                        "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/":                                       "cycle_reads",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/":                                                      "cycle_reads",
		"/api/workspaces/{slug}/modules/":                                                                                "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/archived-modules/":                                                 "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/":                                            "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/modules/":                                                          "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/":                                      "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/":                                       "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/":                            "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/":                                 "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/":                            "module_reads",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/":                                                     "module_reads",
		"/api/workspaces/{slug}/estimates/":                                                                              "estimate_reads",
		"/api/workspaces/{slug}/projects/{project_id}/estimates/":                                                        "estimate_reads",
		"/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/":                                          "estimate_reads",
		"/api/workspaces/{slug}/projects/{project_id}/pages-summary/":                                                    "page_reads",
		"/api/workspaces/{slug}/projects/{project_id}/pages/":                                                            "page_reads",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/":                                                  "page_reads",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/":                                      "page_reads",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/":                                         "page_reads",
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/":                                    "page_reads",
		"/api/workspaces/{slug}/draft-issues/":                                                                           "intake_inbox_reads",
		"/api/workspaces/{slug}/draft-issues/{pk}/":                                                                      "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/":                                                     "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/":                                                "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/":                                                          "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/":                                                     "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/intake-issues/":                                                    "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/":                                               "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/":            "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/":       "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/":                                                          "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/":                                                     "intake_inbox_reads",
		"/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/":                                            "deploy_board_reads",
		"/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/":                                       "deploy_board_reads",
		"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/history/":                                        "work_item_history",
		"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/meta/":                                           "work_item_history",
		"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/":                                       "work_item_history",
		"/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/{pk}/":                                  "work_item_history",
		"/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/":                   "work_item_history",
		"/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/{pk}/":              "work_item_history",
		"/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/property-history/":                       "work_item_history",
		"/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/properties/{property_id}/versions/":      "work_item_history",
		"/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/properties/{property_id}/versions/{pk}/": "work_item_history",
	}
	if len(want) != 59 {
		t.Fatalf("route accounting fixture has %d GETs, want 59", len(want))
	}
	getMounts := make([]readServiceMount, 0, len(mounts))
	explicitHeads := 0
	authoredOptionsMounts := 0
	corsPreflightMounts := 0
	for _, mount := range mounts {
		switch mount.verb {
		case http.MethodGet:
			getMounts = append(getMounts, mount)
		case http.MethodHead:
			explicitHeads++
		case http.MethodOptions:
			if mount.method == "CORS preflight" {
				corsPreflightMounts++
			} else {
				authoredOptionsMounts++
			}
		}
	}
	if len(getMounts) != 59 {
		t.Fatalf("generated GET accounting = %d, want 59", len(getMounts))
	}
	if explicitHeads != 1 {
		t.Fatalf("generated explicit HEAD accounting = %d, want page FileResponse HEAD only", explicitHeads)
	}
	if authoredOptionsMounts != 20 || corsPreflightMounts != 43 {
		t.Fatalf("generated OPTIONS accounting = %d authored + %d CORS preflight, want 20 + 43", authoredOptionsMounts, corsPreflightMounts)
	}
	seen := make(map[string]bool, len(getMounts))
	for _, mount := range getMounts {
		service, exists := want[mount.pattern]
		if !exists {
			t.Errorf("unexpected generated GET %s", mount.pattern)
			continue
		}
		if service != mount.service {
			t.Errorf("%s belongs to %s, want %s", mount.pattern, mount.service, service)
		}
		if seen[mount.pattern] {
			t.Errorf("duplicate generated GET %s", mount.pattern)
		}
		seen[mount.pattern] = true
	}
	for pattern := range want {
		if !seen[pattern] {
			t.Errorf("missing generated GET %s", pattern)
		}
	}
}

const readServicesAuthenticationDetail = "Authentication credentials were not provided."

type workspaceProjectUnauthorizedService struct{ workspaceprojectreads.Service }

func (workspaceProjectUnauthorizedService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, &workspaceprojectreads.WorkspaceProjectUnauthenticated{Detail: readServicesAuthenticationDetail}
}

type cycleUnauthorizedService struct{ cyclereads.Service }

func (cycleUnauthorizedService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, &cyclereads.CycleReadsUnauthenticated{Detail: readServicesAuthenticationDetail}
}

type moduleUnauthorizedService struct{ modulereads.Service }

func (moduleUnauthorizedService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, &modulereads.ModuleReadsUnauthenticated{Detail: readServicesAuthenticationDetail}
}

type estimateUnauthorizedService struct{ estimatereads.Service }

func (estimateUnauthorizedService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, &estimatereads.EstimateReadsUnauthenticated{Detail: readServicesAuthenticationDetail}
}

type pageUnauthorizedService struct{ pagereads.Service }

func (pageUnauthorizedService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, &pagereads.PageReadsUnauthenticated{Detail: readServicesAuthenticationDetail}
}

type intakeUnauthorizedService struct{ intakeinboxreads.Service }

func (intakeUnauthorizedService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, &intakeinboxreads.IntakeInboxReadsUnauthenticated{Detail: readServicesAuthenticationDetail}
}

type deployBoardUnauthorizedService struct{ deployboardreads.Service }

func (deployBoardUnauthorizedService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	return ctx, &deployboardreads.DeployBoardReadsUnauthenticated{Detail: readServicesAuthenticationDetail}
}

func TestReadServiceGeneratedTransportsEnforceAuthentication(t *testing.T) {
	mux := loomhttp.NewMuxer()
	workspaceProjectServer := workspaceprojectreadssvr.New(workspaceprojectreads.NewEndpoints(workspaceProjectUnauthorizedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	workspaceprojectreadssvr.Mount(mux, workspaceProjectServer)
	cycleServer := cyclereadssvr.New(cyclereads.NewEndpoints(cycleUnauthorizedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	cyclereadssvr.Mount(mux, cycleServer)
	moduleServer := modulereadssvr.New(modulereads.NewEndpoints(moduleUnauthorizedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	modulereadssvr.Mount(mux, moduleServer)
	estimateServer := estimatereadssvr.New(estimatereads.NewEndpoints(estimateUnauthorizedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	estimatereadssvr.Mount(mux, estimateServer)
	pageServer := pagereadssvr.New(pagereads.NewEndpoints(pageUnauthorizedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	pagereadssvr.Mount(mux, pageServer)
	intakeServer := intakeinboxreadssvr.New(intakeinboxreads.NewEndpoints(intakeUnauthorizedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	intakeinboxreadssvr.Mount(mux, intakeServer)
	deployBoardServer := deployboardreadssvr.New(deployboardreads.NewEndpoints(deployBoardUnauthorizedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	deployboardreadssvr.Mount(mux, deployBoardServer)

	id := "00000000-0000-4000-8000-000000000001"
	for _, path := range []string{
		"/api/workspaces/",
		"/api/workspaces/acme/projects/" + id + "/cycles/",
		"/api/workspaces/acme/projects/" + id + "/modules/",
		"/api/workspaces/acme/projects/" + id + "/estimates/",
		"/api/workspaces/acme/projects/" + id + "/pages/",
		"/api/workspaces/acme/projects/" + id + "/intakes/",
		"/api/workspaces/acme/projects/" + id + "/project-deploy-boards/",
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("GET %s status=%d body=%q, want 401", path, response.Code, response.Body.String())
		}
		if got := strings.TrimSpace(response.Body.String()); got != `{"detail":"Authentication credentials were not provided."}` {
			t.Errorf("GET %s body=%q", path, got)
		}
		if got := response.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("GET %s Content-Type=%q, want application/json", path, got)
		}
		if got := response.Header().Get("loom-error"); got != "unauthenticated" {
			t.Errorf("GET %s loom-error=%q, want unauthenticated", path, got)
		}
	}
}

type workspaceProjectResponseService struct {
	workspaceprojectreads.Service
}

func (workspaceProjectResponseService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	if key == "valid-session" {
		return ctx, nil
	}
	return ctx, &workspaceprojectreads.WorkspaceProjectUnauthenticated{Detail: readServicesAuthenticationDetail}
}

func (workspaceProjectResponseService) Workspaces(context.Context, *workspaceprojectreads.WorkspacesPayload) (*workspaceprojectreads.RawJSONResponseMetadata, io.ReadCloser, error) {
	return &workspaceprojectreads.RawJSONResponseMetadata{ContentType: "application/json"}, io.NopCloser(strings.NewReader("[]")), nil
}

func (workspaceProjectResponseService) UserWorkspacesOptions(context.Context, *workspaceprojectreads.UserWorkspacesOptionsPayload) (*workspaceprojectreads.WorkspaceProjectOptions, error) {
	return &workspaceprojectreads.WorkspaceProjectOptions{
		Name:        "User Work Spaces Endpoint",
		Description: "",
		Renders:     []string{"application/json"},
		Parses:      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	}, nil
}

func TestReadServiceProtocolCompanionsMatchDjango(t *testing.T) {
	mux := loomhttp.NewMuxer()
	corsPolicy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	server := workspaceprojectreadssvr.New(
		workspaceprojectreads.NewEndpoints(workspaceProjectResponseService{}),
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		nil,
		nil,
		corsPolicy,
	)
	workspaceprojectreadssvr.Mount(mux, server)
	mountWorkspaceProjectReadsProtocolRoutes(mux, server, corsPolicy)
	handler := readServicesCompatibilityMiddleware(mux)

	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, request)
	if get.Code != http.StatusOK || get.Body.String() != "[]" {
		t.Fatalf("GET = %d %q", get.Code, get.Body.String())
	}
	if get.Header().Get("Content-Type") != "application/json" {
		t.Errorf("GET Content-Type = %q", get.Header().Get("Content-Type"))
	}
	assertReadServiceProtocolHeaders(t, get, "GET, POST, HEAD, OPTIONS")

	request = httptest.NewRequest(http.MethodHead, "/api/workspaces/", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	request.Header.Set("Origin", "https://app.example.test")
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, request)
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD = %d %q", head.Code, head.Body.String())
	}
	if head.Header().Get("Content-Type") != "application/json" {
		t.Errorf("HEAD Content-Type = %q", head.Header().Get("Content-Type"))
	}
	assertReadServiceProtocolHeaders(t, head, "GET, POST, HEAD, OPTIONS")
	if head.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" {
		t.Errorf("HEAD CORS headers = %#v", head.Header())
	}

	optionsRequest := httptest.NewRequest(http.MethodOptions, "/api/users/me/workspaces/", nil)
	optionsRequest.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	options := httptest.NewRecorder()
	handler.ServeHTTP(options, optionsRequest)
	if options.Code != http.StatusOK {
		t.Fatalf("OPTIONS = %d %q", options.Code, options.Body.String())
	}
	assertReadServiceProtocolHeaders(t, options, "GET, HEAD, OPTIONS")
	var metadata workspaceprojectreads.WorkspaceProjectOptions
	if err := json.Unmarshal(options.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Name != "User Work Spaces Endpoint" || len(metadata.Renders) != 1 || len(metadata.Parses) != 3 {
		t.Errorf("OPTIONS metadata = %#v", metadata)
	}

	mixedOptions := httptest.NewRecorder()
	handler.ServeHTTP(mixedOptions, httptest.NewRequest(http.MethodOptions, "/api/workspaces/", nil))
	if mixedOptions.Code == http.StatusOK {
		t.Errorf("mixed-writer ordinary OPTIONS was served by Go: %d %q", mixedOptions.Code, mixedOptions.Body.String())
	}

	preflightRequest := httptest.NewRequest(http.MethodOptions, "/api/workspaces/", nil)
	preflightRequest.Header.Set("Origin", "https://app.example.test")
	preflightRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight := httptest.NewRecorder()
	handler.ServeHTTP(preflight, preflightRequest)
	if preflight.Code != http.StatusNoContent || preflight.Body.Len() != 0 ||
		preflight.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" ||
		preflight.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("CORS preflight = %d %q headers=%#v", preflight.Code, preflight.Body.String(), preflight.Header())
	}
}

func TestReadServicesUseDjangoLastScalarQueryValue(t *testing.T) {
	var got map[string][]string
	handler := readServicesCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		got = request.URL.Query()
		response.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/modules/00000000-0000-4000-8000-000000000002/issues/?priority=high&priority=low&filters=A&filters=B&multi=one&multi=two",
		nil,
	)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if values := got["priority"]; len(values) != 1 || values[0] != "low" {
		t.Errorf("scalar priority values = %v, want Django's last value", values)
	}
	if values := got["filters"]; len(values) != 1 || values[0] != "B" {
		t.Errorf("scalar filters values = %v, want Django's last value", values)
	}
	if values := got["multi"]; len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Errorf("unclassified multi-value query changed: %v", values)
	}
}
func TestReadServicesRewriteSemicolonsBeforeDjangoScalarSelection(t *testing.T) {
	var got []string
	handler := readServicesCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		got = request.URL.Query()["created_at"]
		response.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/workspaces/acme/draft-issues/?created_at=2026-01-01;after&created_at=2026-02-02;before",
		nil,
	)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if len(got) != 1 || got[0] != "2026-02-02;before" {
		t.Fatalf("created_at values = %v, want Django's final semicolon-bearing value", got)
	}
}

func TestReadServicesQueryPresenceMiddlewarePreservesExplicitEmptyKeys(t *testing.T) {
	var present map[string]bool
	handler := readServicesQueryPresenceMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		present = map[string]bool{
			"per_page":    systemapi.ReadServiceQueryPresent(request.Context(), "per_page"),
			"cursor":      systemapi.ReadServiceQueryPresent(request.Context(), "cursor"),
			"order_by":    systemapi.ReadServiceQueryPresent(request.Context(), "order_by"),
			"owner":       systemapi.ReadServiceQueryPresent(request.Context(), "owner"),
			"status":      systemapi.ReadServiceQueryPresent(request.Context(), "status"),
			"type":        systemapi.ReadServiceQueryPresent(request.Context(), "type"),
			"track_visit": systemapi.ReadServiceQueryPresent(request.Context(), "track_visit"),
			"missing":     systemapi.ReadServiceQueryPresent(request.Context(), "missing"),
		}
		response.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/?per_page=&cursor=&order_by=&owner=&status=&type=&track_visit=", nil)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	for _, key := range []string{"per_page", "cursor", "order_by", "owner", "status", "type", "track_visit"} {
		if !present[key] {
			t.Errorf("%s presence was lost", key)
		}
	}
	if present["missing"] {
		t.Error("absent query key was reported present")
	}
}
func TestReadServicesNegotiateJSONBeforeDispatch(t *testing.T) {

	nextCalls := 0
	handler := readServicesContentNegotiationMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		nextCalls++
		response.WriteHeader(http.StatusNoContent)
	}))
	for _, testCase := range []struct {
		name       string
		method     string
		target     string
		accept     string
		wantCode   int
		wantBody   string
		wantLength int
		wantCall   bool
	}{
		{name: "JSON", method: http.MethodGet, target: "/", accept: "application/json", wantCode: http.StatusNoContent, wantCall: true},
		{name: "wildcard", method: http.MethodGet, target: "/", accept: "*/*", wantCode: http.StatusNoContent, wantCall: true},
		{name: "JSON format", method: http.MethodGet, target: "/?format=json", wantCode: http.StatusNoContent, wantCall: true},
		{name: "last repeated format", method: http.MethodGet, target: "/?format=xml&format=json", wantCode: http.StatusNoContent, wantCall: true},
		{name: "unsupported format", method: http.MethodGet, target: "/?format=xml", wantCode: http.StatusNotFound, wantBody: string(readServicesFormatNotFoundBody), wantLength: len(readServicesFormatNotFoundBody)},
		{name: "unsupported", method: http.MethodGet, target: "/", accept: "application/xml", wantCode: http.StatusNotAcceptable, wantBody: string(readServicesNotAcceptableBody), wantLength: len(readServicesNotAcceptableBody)},
		{name: "unsupported HEAD", method: http.MethodHead, target: "/", accept: "application/xml", wantCode: http.StatusNotAcceptable, wantLength: len(readServicesNotAcceptableBody)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			before := nextCalls
			request := httptest.NewRequest(testCase.method, testCase.target, nil)
			if testCase.accept != "" {
				request.Header.Set("Accept", testCase.accept)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.wantCode || response.Body.String() != testCase.wantBody {
				t.Errorf("response = %d %q, want %d %q", response.Code, response.Body.String(), testCase.wantCode, testCase.wantBody)
			}
			if got := nextCalls > before; got != testCase.wantCall {
				t.Errorf("downstream called = %v, want %v", got, testCase.wantCall)
			}
			if testCase.wantLength > 0 && response.Header().Get("Content-Length") != fmt.Sprint(testCase.wantLength) {
				t.Errorf("Content-Length = %q, want %d", response.Header().Get("Content-Length"), testCase.wantLength)
			}
		})
	}
}

func TestReadServicesHonorJSONRendererIndent(t *testing.T) {
	handler := readServicesContentNegotiationMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"items":[{"id":1}]}`))
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "application/json; indent=2")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	want := "{\n  \"items\": [\n    {\n      \"id\": 1\n    }\n  ]\n}"
	if response.Code != http.StatusOK || response.Body.String() != want {
		t.Errorf("indented response = %d %q, want %q", response.Code, response.Body.String(), want)
	}
	if response.Header().Get("Content-Length") != fmt.Sprint(len(want)) {
		t.Errorf("Content-Length = %q, want %d", response.Header().Get("Content-Length"), len(want))
	}
}

func TestReadServicesRejectMalformedDjangoUUIDPathsBeforeAuthentication(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	nextCalls := 0
	next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		nextCalls++
		response.WriteHeader(http.StatusNoContent)
	})
	handler := readServicesDjangoPathConverterMiddleware(policy, nil)(
		readServicesCompatibilityMiddleware(next),
	)
	malformed := "/api/workspaces/acme/projects/00000000-0000-4000-8000-00000000000Z/modules/"

	for _, method := range []string{http.MethodGet, http.MethodOptions} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(method, malformed, nil)
		request.Header.Set("Origin", "https://app.example.test")
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound || response.Body.String() != string(djangoNotFoundBody) {
			t.Errorf("%s malformed UUID = %d %q", method, response.Code, response.Body.String())
		}
		if response.Header().Get("Allow") != "" {
			t.Errorf("%s malformed UUID leaked endpoint Allow %q", method, response.Header().Get("Allow"))
		}
		if response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" {
			t.Errorf("%s malformed UUID CORS headers = %#v", method, response.Header())
		}
		if response.Header().Get("Vary") != "origin" {
			t.Errorf("%s malformed UUID Vary = %q", method, response.Header().Get("Vary"))
		}
	}

	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, malformed, nil))
	if head.Code != http.StatusNotFound || head.Body.Len() != 0 ||
		head.Header().Get("Content-Length") != fmt.Sprint(len(djangoNotFoundBody)) {
		t.Errorf("HEAD malformed UUID = %d %q headers=%#v", head.Code, head.Body.String(), head.Header())
	}

	preflightRequest := httptest.NewRequest(http.MethodOptions, malformed, nil)
	preflightRequest.Header.Set("Origin", "https://app.example.test")
	preflightRequest.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight := httptest.NewRecorder()
	handler.ServeHTTP(preflight, preflightRequest)
	if preflight.Code != http.StatusNoContent {
		t.Errorf("malformed UUID preflight = %d %q", preflight.Code, preflight.Body.String())
	}

	valid := httptest.NewRecorder()
	handler.ServeHTTP(valid, httptest.NewRequest(
		http.MethodGet,
		"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/modules/",
		nil,
	))
	if valid.Code != http.StatusNoContent || nextCalls != 2 {
		t.Errorf("valid UUID response = %d, downstream calls = %d; want malformed preflight plus valid GET", valid.Code, nextCalls)
	}
}

func TestReadServicesGzipMiddlewareMatchesDjangoThreshold(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		size       int
		acceptGzip bool
		wantVary   string
		wantGzip   bool
	}{
		{name: "under threshold", size: 199, acceptGzip: true, wantVary: "Cookie, origin"},
		{name: "threshold without acceptance", size: 200, wantVary: "Accept-Encoding, Cookie, origin"},
		{name: "compressible representation", size: 300, acceptGzip: true, wantVary: "Accept-Encoding, Cookie, origin", wantGzip: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			body := bytes.Repeat([]byte{'a'}, testCase.size)
			handler := readServicesGzipMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.Header().Set("Content-Type", "application/json")
				response.Header().Set("Vary", "Cookie, origin")
				_, _ = response.Write(body)
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if testCase.acceptGzip {
				request.Header.Set("Accept-Encoding", "gzip")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if got := response.Header().Get("Vary"); got != testCase.wantVary {
				t.Errorf("Vary = %q, want %q", got, testCase.wantVary)
			}
			if got := response.Header().Get("Content-Encoding") == "gzip"; got != testCase.wantGzip {
				t.Fatalf("gzip response = %v, want %v", got, testCase.wantGzip)
			}
			if !testCase.wantGzip {
				if !bytes.Equal(response.Body.Bytes(), body) {
					t.Errorf("plain body changed")
				}
				return
			}
			reader, err := gzip.NewReader(bytes.NewReader(response.Body.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			uncompressed, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if err := reader.Close(); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(uncompressed, body) {
				t.Errorf("uncompressed body differs")
			}
		})
	}
}

func TestReadServicesPageDescriptionMatchesDjangoStreamingGzip(t *testing.T) {
	body := []byte("page description")
	handler := readServicesGzipMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = response.Write(body)
	}))
	path := "/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/pages/00000000-0000-4000-8000-000000000002/description/"

	plainRequest := httptest.NewRequest(http.MethodGet, path, nil)
	plainResponse := httptest.NewRecorder()
	handler.ServeHTTP(plainResponse, plainRequest)
	if plainResponse.Code != http.StatusOK || !bytes.Equal(plainResponse.Body.Bytes(), body) {
		t.Fatalf("plain GET = %d %q", plainResponse.Code, plainResponse.Body.String())
	}
	if plainResponse.Header().Get("Vary") != "Accept-Encoding" {
		t.Errorf("plain GET Vary = %q", plainResponse.Header().Get("Vary"))
	}
	if plainResponse.Header().Get("Content-Length") != strconv.Itoa(len(body)) {
		t.Errorf("plain GET Content-Length = %q", plainResponse.Header().Get("Content-Length"))
	}

	gzipRequest := httptest.NewRequest(http.MethodGet, path, nil)
	gzipRequest.Header.Set("Accept-Encoding", "gzip")
	gzipResponse := httptest.NewRecorder()
	handler.ServeHTTP(gzipResponse, gzipRequest)
	if gzipResponse.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("compressed GET Content-Encoding = %q", gzipResponse.Header().Get("Content-Encoding"))
	}
	if gzipResponse.Header().Get("Content-Length") != "" {
		t.Errorf("compressed GET Content-Length = %q, want absent", gzipResponse.Header().Get("Content-Length"))
	}
	reader, err := gzip.NewReader(bytes.NewReader(gzipResponse.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	uncompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(uncompressed, body) {
		t.Errorf("compressed GET body = %q", uncompressed)
	}

	headRequest := httptest.NewRequest(http.MethodHead, path, nil)
	headRequest.Header.Set("Accept-Encoding", "gzip")
	headResponse := httptest.NewRecorder()
	handler.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK || headResponse.Body.Len() != 0 {
		t.Fatalf("HEAD = %d %q", headResponse.Code, headResponse.Body.String())
	}
	if headResponse.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("HEAD Content-Encoding = %q", headResponse.Header().Get("Content-Encoding"))
	}
	if headResponse.Header().Get("Content-Length") != "" {
		t.Errorf("HEAD Content-Length = %q, want absent", headResponse.Header().Get("Content-Length"))
	}
}

func TestReadServicesDerivedHEADUsesCompressedGETRepresentation(t *testing.T) {
	body := bytes.Repeat([]byte{'a'}, 300)
	get := readServicesGzipMiddleware(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(body)
	}))
	mux := loomhttp.NewMuxer()
	mountProtocolHandler(mux, http.MethodGet, "/api/users/me/workspaces/", get)
	loomhttp.MountDerivedHead(mux, "/api/users/me/workspaces/", get)
	request := httptest.NewRequest(http.MethodHead, "/api/users/me/workspaces/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.Len() != 0 {
		t.Fatalf("HEAD = %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Content-Encoding = %q", response.Header().Get("Content-Encoding"))
	}
	if response.Header().Get("Content-Length") == "" {
		t.Error("compressed HEAD is missing representation Content-Length")
	}
	if response.Header().Get("Vary") != "Accept-Encoding" {
		t.Errorf("Vary = %q", response.Header().Get("Vary"))
	}
}
func TestReadServicesSessionLifecycleMatchesDjango(t *testing.T) {
	config := djangoSessionCookieConfig{Name: "session-id", Domain: ".example.test", Secure: true, Configured: true}
	next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte(`{"ok":true}`))
	})

	t.Run("last configured session cookie wins", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{}
		request := httptest.NewRequest(http.MethodGet, "/api/workspaces/", nil)
		request.Header.Add("Cookie", "session-id=first; session-id=second")
		response := httptest.NewRecorder()
		readServicesSessionMiddleware(sessions, config)(next).ServeHTTP(response, request)
		if response.Code != http.StatusOK || sessions.inspectedCookie != "second" || sessions.calls != 1 {
			t.Fatalf("status=%d cookie=%q calls=%d", response.Code, sessions.inspectedCookie, sessions.calls)
		}
	})

	t.Run("browser preflight bypasses session storage", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{err: fmt.Errorf("must not inspect")}
		request := httptest.NewRequest(http.MethodOptions, "/api/workspaces/", nil)
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)
		response := httptest.NewRecorder()
		readServicesSessionMiddleware(sessions, config)(next).ServeHTTP(response, request)
		if response.Code != http.StatusOK || sessions.calls != 0 {
			t.Fatalf("status=%d session calls=%d", response.Code, sessions.calls)
		}
	})

	t.Run("session storage failure is Django HTML 500", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{err: fmt.Errorf("session store unavailable: %w", djangosession.ErrSessionStore)}
		response := httptest.NewRecorder()
		response.Header().Set("Allow", "GET, HEAD, OPTIONS")
		request := httptest.NewRequest(http.MethodGet, "/api/workspaces/", nil)
		readServicesSessionMiddleware(sessions, config)(next).ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError || response.Body.String() != string(djangoServerErrorBody) {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
		if response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("Allow") != "" {
			t.Errorf("headers=%#v", response.Header())
		}
	})
	t.Run("unknown timezone is modeled JSON 400 for TimezoneMixin reads", func(t *testing.T) {
		for _, path := range []string{
			"/api/workspaces/",
			"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/cycles/00000000-0000-4000-8000-000000000002/cycle-issues/",
			"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/modules/00000000-0000-4000-8000-000000000002/issues/",
			"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/pages/",
			"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/intake-issues/",
			"/api/workspaces/acme/estimates/",
			"/api/workspaces/acme/projects/00000000-0000-4000-8000-000000000001/project-deploy-boards/",
		} {
			sessions := &publicProjectMetadataThrottleStub{
				err:     fmt.Errorf("activate user timezone: %w", djangosession.ErrTimezoneNotFound),
				rotated: &djangosession.SessionRefresh{SessionKey: "prepared"},
				result:  djangosession.SessionRefresh{SessionKey: "refreshed"},
			}
			response := httptest.NewRecorder()
			response.Header().Set("Allow", "GET, HEAD, OPTIONS")
			request := httptest.NewRequest(http.MethodGet, path, nil)
			readServicesSessionMiddleware(sessions, config)(next).ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest ||
				response.Body.String() != `{"error":"The required key does not exist."}` {
				t.Errorf("%s response=%d %q", path, response.Code, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" ||
				response.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
				t.Errorf("%s headers=%#v", path, response.Header())
			}
			if sessions.refreshCalls != 1 {
				t.Errorf("%s session refresh calls=%d, want 1 before JSON 400", path, sessions.refreshCalls)
			}
		}
	})

	t.Run("malformed timezone is modeled JSON 500", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{
			err:     fmt.Errorf("activate user timezone: %w", djangosession.ErrInvalidTimezone),
			rotated: &djangosession.SessionRefresh{SessionKey: "prepared"},
			result:  djangosession.SessionRefresh{SessionKey: "must-not-save"},
		}
		response := httptest.NewRecorder()
		response.Header().Set("Allow", "GET, HEAD, OPTIONS")
		request := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/draft-issues/", nil)
		readServicesSessionMiddleware(sessions, config)(next).ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError ||
			response.Body.String() != `{"error":"Something went wrong please try again later"}` {
			t.Fatalf("response=%d %q", response.Code, response.Body.String())
		}
		if response.Header().Get("Content-Type") != "application/json" ||
			response.Header().Get("Allow") != "GET, HEAD, OPTIONS" {
			t.Errorf("headers=%#v", response.Header())
		}
		if sessions.refreshCalls != 0 {
			t.Errorf("session refresh calls=%d, want none for JSON 500", sessions.refreshCalls)
		}
	})

	t.Run("prepared session is saved before response commit", func(t *testing.T) {
		sessions := &publicProjectMetadataThrottleStub{
			rotated: &djangosession.SessionRefresh{SessionKey: "prepared"},
			result:  djangosession.SessionRefresh{SessionKey: "refreshed", BrowserClose: true},
		}
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/workspaces/", nil)
		readServicesSessionMiddleware(sessions, config)(next).ServeHTTP(response, request)
		if response.Code != http.StatusOK || sessions.refreshCalls != 1 {
			t.Fatalf("status=%d refresh calls=%d", response.Code, sessions.refreshCalls)
		}
		cookies := response.Header().Values("Set-Cookie")
		if len(cookies) != 1 || !strings.HasPrefix(cookies[0], "session-id=refreshed;") {
			t.Fatalf("Set-Cookie=%q", cookies)
		}
	})
}

func TestDjangoPageInternalErrorsUseHTML500(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			ctx := systemapi.WithReadServiceDjangoHTML500(context.Background())
			endpoint := markDjangoPageInternalError(func(context.Context, any) (any, error) {
				return nil, pagereadsstore.ErrInvalidOrder
			})
			if _, err := endpoint(ctx, nil); !errors.Is(err, pagereadsstore.ErrInvalidOrder) || !systemapi.ReadServiceDjangoHTML500(ctx) {
				t.Fatalf("endpoint err=%v marker=%v", err, systemapi.ReadServiceDjangoHTML500(ctx))
			}

			response := httptest.NewRecorder()
			request := httptest.NewRequest(method, "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/pages/", nil)
			request = request.WithContext(ctx)
			writer := &djangoReadServiceHTML500ResponseWriter{
				ResponseWriter: response,
				request:        request,
			}
			writer.Header().Set("Content-Type", "application/json")
			writer.Header().Set("loom-error", "internal")
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte(`{"error":"internal"}`))

			if response.Code != http.StatusInternalServerError ||
				response.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
				response.Header().Get("loom-error") != "" {
				t.Fatalf("response=%d headers=%#v", response.Code, response.Header())
			}
			wantBody := string(djangoServerErrorBody)
			if method == http.MethodHead {
				wantBody = ""
			}
			if response.Body.String() != wantBody {
				t.Fatalf("body=%q, want %q", response.Body.String(), wantBody)
			}
		})
	}
}

func TestReadServiceProtocolRegistryOwnsDeclaredRoutes(t *testing.T) {
	// Membership lifecycle, the 36 legacy assets paths plus draft association, the 16 invitation/membership and role catalogue
	// paths, custom view creation, API-v1 Sticky format-suffix companions,
	// administration paths, four workspace user-insights routes, and three custom-property
	// history routes and the instance telemetry route contribute here.
	if len(assetsProtocolMetadata) != 36 {
		t.Fatalf("assets protocol paths = %d, want 36", len(assetsProtocolMetadata))
	}
	if len(readServiceProtocols) != 279 {
		t.Fatalf("protocol routes = %d, want 279", len(readServiceProtocols))
	}
	telemetry := mustReadServiceProtocol("/api/instances/telemetry/")
	if !telemetry.authenticateMethod || !telemetry.derivedHead || telemetry.allow != "GET, HEAD, OPTIONS" {
		t.Fatal("instance telemetry must enforce authenticated reads with HEAD and OPTIONS")
	}
	for _, path := range []string{"/api/v1/workspaces/{slug}/roles/", "/api/v1/workspaces/{slug}/roles/{role_id}/"} {
		protocol := mustReadServiceProtocol(path)
		if protocol.allow != "GET" || !protocol.generatedMethods {
			t.Fatalf("role catalogue route %s must use generated GET only", path)
		}
	}

	draftAssets := mustReadServiceProtocol("/api/assets/v2/workspaces/{slug}/{entity_id}/bulk/")
	if !draftAssets.authenticateMethod || !draftAssets.csrfForAuthenticatedSession || draftAssets.allow != "POST, OPTIONS" {
		t.Fatal("draft association must enforce session authentication and CSRF")
	}
	for _, protocol := range readServiceProtocols {
		if protocol.pattern == "/api/v1/workspaces/{slug}/projects/{project_id}/cycles/date-check/" {
			t.Fatal("API-v1 cycle date-check has no source route and must not be registered")
		}
	}
	for path, wantAllow := range map[string]string{
		"/api/v1/workspaces/{slug}/stickies/":           "GET, POST, HEAD, OPTIONS",
		"/api/v1/workspaces/{slug}/stickies/{pk}/":      "GET, PUT, PATCH, DELETE, HEAD, OPTIONS",
		"/api/v1/workspaces/{slug}/stickies.json/":      "GET, POST, HEAD, OPTIONS",
		"/api/v1/workspaces/{slug}/stickies/{pk}.json/": "GET, PUT, PATCH, DELETE, HEAD, OPTIONS",
	} {
		protocol := mustReadServiceProtocol(path)
		if protocol.allow != wantAllow || !protocol.generatedMethods {
			t.Errorf("Sticky API protocol %s = %#v, want generated methods and Allow %q", path, protocol, wantAllow)
		}
	}
	wantOptions := map[string]bool{
		"/api/users/me/workspaces/":                                                                                true,
		"/api/workspaces/{slug}/project-stats/":                                                                    true,
		"/api/workspaces/{slug}/projects/details/":                                                                 true,
		"/api/workspaces/{slug}/cycles/":                                                                           true,
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/":                                true,
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/":                                 true,
		"/api/workspaces/{slug}/modules/":                                                                          true,
		"/api/workspaces/{slug}/estimates/":                                                                        true,
		"/api/workspaces/{slug}/projects/{project_id}/pages-summary/":                                              true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/":                                   true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/":                              true,
		"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/":      true,
		"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/": true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/":                                                      true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/":                                            true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/":                                true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/access/":                                     true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/":                                    true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/duplicate/":                                  true,
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/":                                       true,
	}

	seen := make(map[string]bool, len(readServiceProtocols))
	derivedHeads := 0
	for _, protocol := range readServiceProtocols {
		if seen[protocol.pattern] {
			t.Errorf("duplicate protocol owner for %s", protocol.pattern)
		}
		seen[protocol.pattern] = true
		resolved, ok := resolveReadServiceProtocol(protocol.pattern)
		if !ok || resolved.pattern != protocol.pattern || resolved.allow != protocol.allow {
			t.Errorf("resolve %s = %#v, %v", protocol.pattern, resolved, ok)
		}
		if protocol.derivedHead {
			derivedHeads++
		}
	}
	for path := range assetsProtocolMetadata {
		if !seen[path] {
			t.Errorf("assets source protocol path %s is not registered", path)
		}
	}
	// Invitations add 10 HEAD owners; the existing member-detail read mounts its own HEAD.
	// The two API rosters remain GET-only. Administration adds ten HEAD owners,
	// workspace user insights adds three, and work item history adds nine derived routes.
	// Member preference writes also preserve the existing member-detail GET/HEAD.
	if derivedHeads != 142 {
		t.Errorf("derived HEAD owners = %d, want 142", derivedHeads)
	}
	for _, path := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/issue-labels/",
		"/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/",
	} {
		protocol := mustReadServiceProtocol(path)
		if !protocol.authoredOptions || !protocol.authenticateMethod || protocol.derivedHead {
			t.Errorf("label protocol must use authored OPTIONS, authenticated dispatch and existing read-owned HEAD: %#v", protocol)
		}
	}
	description := mustReadServiceProtocol("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/")
	if description.derivedHead {
		t.Error("page FileResponse HEAD must be owned by the generated explicit route")
	}
	mux := loomhttp.NewMuxer()
	type generatedOption struct {
		method  string
		pattern string
	}
	var generatedOptions []generatedOption
	var generatedGets []string
	collect := func(method, verb, pattern string) {
		switch verb {
		case http.MethodGet:
			generatedGets = append(generatedGets, pattern)
		case http.MethodOptions:
			generatedOptions = append(generatedOptions, generatedOption{method: method, pattern: pattern})
		}
	}
	workspaceServer := workspaceprojectreadssvr.New(&workspaceprojectreads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	for _, mount := range workspaceServer.Mounts {
		collect(mount.Method, mount.Verb, mount.Pattern)
	}
	cycleServer := cyclereadssvr.New(&cyclereads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	for _, mount := range cycleServer.Mounts {
		collect(mount.Method, mount.Verb, mount.Pattern)
	}
	moduleServer := modulereadssvr.New(&modulereads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	for _, mount := range moduleServer.Mounts {
		collect(mount.Method, mount.Verb, mount.Pattern)
	}
	estimateServer := estimatereadssvr.New(&estimatereads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	for _, mount := range estimateServer.Mounts {
		collect(mount.Method, mount.Verb, mount.Pattern)
	}
	pageServer := pagereadssvr.New(&pagereads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	for _, mount := range pageServer.Mounts {
		collect(mount.Method, mount.Verb, mount.Pattern)
	}
	intakeServer := intakeinboxreadssvr.New(&intakeinboxreads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	for _, mount := range intakeServer.Mounts {
		collect(mount.Method, mount.Verb, mount.Pattern)
	}
	deployServer := deployboardreadssvr.New(&deployboardreads.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	for _, mount := range deployServer.Mounts {
		collect(mount.Method, mount.Verb, mount.Pattern)
	}
	historyServer := workitemhistorysvr.New(&workitemhistoryapi.Endpoints{}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	for _, mount := range historyServer.Mounts {
		collect(mount.Method, mount.Verb, mount.Pattern)
	}
	if len(generatedGets) != 59 || len(generatedOptions) != 63 {
		t.Errorf("generated read mounts have %d GET and %d OPTIONS routes, want 59 GET and 63 total OPTIONS", len(generatedGets), len(generatedOptions))
	}
	getPaths := make(map[string]bool, len(generatedGets))
	for _, path := range generatedGets {
		if getPaths[path] {
			t.Errorf("duplicate generated GET %s", path)
		}
		getPaths[path] = true
	}
	seenOptions := make(map[string]bool, len(generatedOptions))
	authoredOptions := 0
	for _, option := range generatedOptions {
		if !getPaths[option.pattern] && !wantOptions[option.pattern] {
			t.Errorf("generated OPTIONS %s has no selected GET route", option.pattern)
		}
		if seenOptions[option.pattern] {
			t.Errorf("duplicate generated OPTIONS %s", option.pattern)
		}
		seenOptions[option.pattern] = true
		if wantOptions[option.pattern] {
			authoredOptions++
			if option.method == "CORS preflight" {
				t.Errorf("authored OPTIONS %s was replaced by generated CORS preflight", option.pattern)
			}
			delete(wantOptions, option.pattern)
		} else if option.method != "CORS preflight" {
			t.Errorf("unexpected authored OPTIONS %s from %s", option.pattern, option.method)
		}
	}
	for path := range getPaths {
		if !seenOptions[path] {
			t.Errorf("generated OPTIONS is missing selected GET path %s", path)
		}
	}
	if authoredOptions != 20 {
		t.Errorf("authored OPTIONS routes = %d, want 20", authoredOptions)
	}
	for path := range wantOptions {
		t.Errorf("generated authored OPTIONS is missing %s", path)
	}
}

func TestStickyProtocolKeepsLookupOpaqueAndRejectsExtraFormatDots(t *testing.T) {
	canonical := mustReadServiceProtocol("/api/v1/workspaces/{slug}/stickies/{pk}/")
	suffixed := mustReadServiceProtocol("/api/v1/workspaces/{slug}/stickies/{pk}.json/")
	malformedPK := "not-a-uuid"
	if !readServiceProtocolMatchesSegments(canonical, []string{"api", "v1", "workspaces", "demo", "stickies", malformedPK}) {
		t.Fatal("canonical Sticky lookup rejected an opaque nonempty pk before API-key/scope checks")
	}
	if readServiceProtocolHasMalformedUUID("/api/v1/workspaces/demo/stickies/"+malformedPK+"/", canonical) {
		t.Fatal("Sticky malformed pk was classified as a Django UUID-converter miss")
	}
	if !readServiceProtocolMatchesSegments(suffixed, []string{"api", "v1", "workspaces", "demo", "stickies", malformedPK + ".json"}) {
		t.Fatal("JSON format alias rejected opaque Sticky pk")
	}
	for _, pk := range []string{"", "not.a.uuid", "not-a-uuid.xml.json"} {
		if readServiceProtocolMatchesSegments(suffixed, []string{"api", "v1", "workspaces", "demo", "stickies", pk + ".json"}) {
			t.Errorf("JSON format alias accepted invalid source token %q", pk)
		}
	}
}

func assertReadServiceProtocolHeaders(t *testing.T, response *httptest.ResponseRecorder, allow string) {
	t.Helper()
	headers := response.Header()
	if headers.Get("Allow") != allow ||
		headers.Get("Cross-Origin-Opener-Policy") != "same-origin" ||
		headers.Get("Referrer-Policy") != "same-origin" ||
		headers.Get("X-Content-Type-Options") != "nosniff" ||
		headers.Get("X-Frame-Options") != "DENY" ||
		!strings.Contains(headers.Get("Vary"), "origin") {
		t.Errorf("protocol headers = %#v, want Allow %q and Django security headers", headers, allow)
	}
}
