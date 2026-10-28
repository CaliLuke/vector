package design_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	client "github.com/CaliLuke/go-plane/work_item_core/gen/http/work_item_core/client"
	server "github.com/CaliLuke/go-plane/work_item_core/gen/http/work_item_core/server"
	api "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

// Embedding limits this transport spike to the exercised methods. The complete
// production service must implement every method before any route is activated.
type transportService struct {
	api.Service
	scheme  string
	key     string
	body    string
	search  *api.GetAPIV1WorkspacesBySlugIssuesSearchPayload
	failure error
}

func (s *transportService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	s.scheme, s.key = scheme.Name, key
	return ctx, nil
}

func (s *transportService) PostAPIWorkspacesBySlugProjectsByProjectIDIssues(_ context.Context, _ *api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesPayload, body io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}
	s.body = string(data)
	if s.failure != nil {
		return nil, nil, s.failure
	}
	return &api.WorkItemResponse{}, io.NopCloser(strings.NewReader(`{"id":"01234567-89ab-4def-8123-456789abcdef","name":"Created"}`)), nil
}

func (s *transportService) GetAPIV1WorkspacesBySlugIssuesSearch(_ context.Context, p *api.GetAPIV1WorkspacesBySlugIssuesSearchPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	s.search = p
	return &api.WorkItemResponse{}, io.NopCloser(strings.NewReader(`[]`)), nil
}

func (s *transportService) DeleteAPIWorkspacesBySlugProjectsByProjectIDBulkDeleteIssues(_ context.Context, _ *api.DeleteAPIWorkspacesBySlugProjectsByProjectIDBulkDeleteIssuesPayload, body io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, err
	}
	s.body = string(data)
	return &api.WorkItemResponse{}, io.NopCloser(strings.NewReader(`{"message":"1 issues were deleted"}`)), nil
}

func transport(t *testing.T) (http.Handler, *transportService) {
	t.Helper()
	s := &transportService{}
	return transportWithService(t, s), s
}

func transportWithService(t *testing.T, service api.Service) http.Handler {
	t.Helper()
	mux := loomhttp.NewMuxer()
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "http://localhost:3000", Methods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"}, Headers: []string{"Content-Type", "X-Api-Key"}, Credentials: true}}})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	generated := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	generated.Use(loomhttp.RequestMetadataMiddleware(metadata))
	generated.Mount(mux)
	return mux
}

func TestWorkItemGeneratedTransportRetainsRawWriteBody(t *testing.T) {
	handler, service := transport(t)
	request := httptest.NewRequest("POST", "/api/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/issues/", strings.NewReader(`{"name":1,"name":"last","unexpected":null}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", "session-id=spike")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 201 || service.body != `{"name":1,"name":"last","unexpected":null}` {
		t.Fatalf("status=%d input=%s response=%s", response.Code, service.body, response.Body.String())
	}
	if service.scheme != "PlaneSessionCookie" || service.key != "" {
		t.Fatalf("session must be transport-owned: scheme=%q key=%q", service.scheme, service.key)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatal(response.Header())
	}
}

func TestWorkItemGeneratedTransportRetainsRawDeleteBody(t *testing.T) {
	handler, service := transport(t)
	const raw = `{"issue_ids":[1],"issue_ids":["last"]}`
	request := httptest.NewRequest("DELETE", "/api/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/bulk-delete-issues/", strings.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Cookie", "session-id=spike")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || service.body != raw || response.Body.String() != `{"message":"1 issues were deleted"}` {
		t.Fatalf("status=%d input=%q response=%q", response.Code, service.body, response.Body.String())
	}
}

func TestWorkItemGeneratedCORSPreflight(t *testing.T) {
	handler, service := transport(t)
	request := httptest.NewRequest("OPTIONS", "/api/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/issues/", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "Content-Type")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 204 || response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" || response.Header().Get("Access-Control-Allow-Credentials") != "true" || !strings.Contains(response.Header().Get("Access-Control-Allow-Methods"), "POST") || service.scheme != "" {
		t.Fatalf("preflight status=%d auth=%q headers=%v", response.Code, service.scheme, response.Header())
	}
}

func TestWorkItemGeneratedQueryRetainsInvalidAndAbsentScalars(t *testing.T) {
	handler, service := transport(t)
	request := httptest.NewRequest("GET", "/api/v1/workspaces/demo/issues/search/?limit=invalid&project_id=not-a-uuid", nil)
	request.Header.Set("X-Api-Key", "spike-key")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || service.search == nil {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if service.search.Limit == nil || *service.search.Limit != "invalid" || service.search.ProjectID == nil || *service.search.ProjectID != "not-a-uuid" || service.search.Search != nil {
		t.Fatalf("source query grammar lost before the domain: %#v", service.search)
	}
	if service.scheme != "ApiKeyAuth" || service.key != "spike-key" {
		t.Fatalf("API authentication: %q %q", service.scheme, service.key)
	}
}

func TestWorkItemGeneratedErrorsPreserveFieldKeysAndJSONValues(t *testing.T) {
	handler, service := transport(t)
	expected := `{"assignee_ids":{"0":["Invalid pk."]},"description_html":["Invalid HTML"],"sequence_id":9007199254740993}`
	failure := &api.WorkItemError{Kind: "status_400", Body: loom.JSONValue(expected)}
	service.failure = failure
	request := httptest.NewRequest("POST", "/api/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/issues/", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 400 {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	var got, want any
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	decoder = json.NewDecoder(strings.NewReader(expected))
	decoder.UseNumber()
	if err := decoder.Decode(&want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("error body=%#v, want %#v", got, want)
	}
}

func TestWorkItemGeneratedErrorsPreserveRecordedClientErrorBodies(t *testing.T) {
	fixtures, err := filepath.Glob("../../../../apps/go-api/internal/parity/testdata/fixtures/work_item_core/*.json")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, path := range fixtures {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fixture struct {
			Response struct {
				Status int
				Body   json.RawMessage
			}
		}
		if err := json.Unmarshal(body, &fixture); err != nil {
			t.Fatal(err)
		}
		if fixture.Response.Status < 400 || fixture.Response.Status >= 500 {
			continue
		}
		checked++
		t.Run(filepath.Base(path), func(t *testing.T) {
			handler, service := transport(t)
			failure := &api.WorkItemError{Kind: fmt.Sprintf("status_%d", fixture.Response.Status), Body: fixture.Response.Body}
			service.failure = failure
			request := httptest.NewRequest("POST", "/api/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/issues/", strings.NewReader(`{}`))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != fixture.Response.Status {
				t.Fatalf("status=%d; source=%d", response.Code, fixture.Response.Status)
			}
			decode := func(raw []byte) any {
				t.Helper()
				var value any
				decoder := json.NewDecoder(strings.NewReader(string(raw)))
				decoder.UseNumber()
				if err := decoder.Decode(&value); err != nil {
					t.Fatal(err)
				}
				return value
			}
			if got, want := decode(response.Body.Bytes()), decode(fixture.Response.Body); !reflect.DeepEqual(got, want) {
				t.Fatalf("generated error=%s; source=%s", response.Body, fixture.Response.Body)
			}
		})
	}
	if checked == 0 {
		t.Fatal("no source client-error fixtures were checked")
	}
	t.Logf("checked %d recorded client-error bodies through generated transport", checked)
}

type workItemCookieDoer struct{ client *http.Client }

func (d workItemCookieDoer) Do(request *http.Request) (*http.Response, error) {
	request.Header.Set("Cookie", "session-id=spike")
	request.Header.Set("Content-Type", "application/json")
	return d.client.Do(request)
}

func TestWorkItemGeneratedClientDecodesTypedErrors(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"field keys and large integer", `{"assignee_ids":{"0":["Invalid pk."]},"sequence_id":9007199254740993}`},
		{"array body", `["invalid",{"field":["value"]}]`},
		{"null body", `null`},
		{"empty object body", `{}`},
		{"unknown field", `{"unmodeled_error":{"code":"new"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, service := transport(t)
			service.failure = &api.WorkItemError{Kind: "status_400", Body: loom.JSONValue(tc.body)}
			remote := httptest.NewServer(handler)
			defer remote.Close()
			transport := client.NewClient("http", strings.TrimPrefix(remote.URL, "http://"), workItemCookieDoer{remote.Client()}, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			_, err := transport.PostAPIWorkspacesBySlugProjectsByProjectIDIssues()(context.Background(), &api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesRequestData{
				Payload: &api.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesPayload{Slug: "demo", ProjectID: "01234567-89ab-4def-8123-456789abcdef"},
				Body:    io.NopCloser(strings.NewReader(`{}`)),
			})
			var decoded *api.WorkItemError
			if !errors.As(err, &decoded) {
				t.Fatalf("typed 400 error: %T %v", err, err)
			}
			if got := strings.TrimSpace(string(decoded.Body)); got != tc.body {
				t.Fatalf("decoded error body=%s, want %s", got, tc.body)
			}
		})
	}
}

func TestWorkItemGeneratedPathConstructorEscapesSegments(t *testing.T) {
	const projectID = "01234567-89ab-4def-8123-456789abcdef"
	got := client.PostAPIWorkspacesBySlugProjectsByProjectIDIssuesWorkItemCorePath("a/b", projectID)
	if got != "/api/workspaces/a%2Fb/projects/"+projectID+"/issues/" {
		t.Fatalf("generated path=%q", got)
	}
}

func TestWorkItemGeneratedSuccessStatusesMatchDjangoFixtures(t *testing.T) {
	openAPIBody, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Paths map[string]map[string]struct {
			OperationID string                     `json:"operationId"`
			Responses   map[string]json.RawMessage `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(openAPIBody, &schema); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ fixture, operationID string }{
		{"lifecycle-draft-publish-planning-and-asset", "post_api_workspaces_by_slug_draft_to_issue_by_draft_id"},
		{"lifecycle-delete-bulk", "delete_api_workspaces_by_slug_projects_by_project_id_bulk_delete_issues"},
		{"lifecycle-inbox-create", "post_api_workspaces_by_slug_projects_by_project_id_inbox_issues"},
		{"lifecycle-intake-create", "post_api_workspaces_by_slug_projects_by_project_id_intake_issues"},
		{"lifecycle-relation-remove", "post_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_remove_relation"},
		{"issue-core-app-patch-description-skip", "patch_api_workspaces_by_slug_projects_by_project_id_issues_by_pk"},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			fixtureBody, err := os.ReadFile("../../../../apps/go-api/internal/parity/testdata/fixtures/work_item_core/" + tc.fixture + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct {
				Request struct {
					Method string `json:"method"`
				} `json:"request"`
				Response struct {
					Status int `json:"status"`
				} `json:"response"`
			}
			if err := json.Unmarshal(fixtureBody, &fixture); err != nil {
				t.Fatal(err)
			}
			if fixture.Response.Status < 200 || fixture.Response.Status >= 300 {
				t.Fatalf("source fixture is not a success: %d", fixture.Response.Status)
			}
			found := false
			for _, operations := range schema.Paths {
				operation, ok := operations[strings.ToLower(fixture.Request.Method)]
				if !ok || operation.OperationID != tc.operationID {
					continue
				}
				found = true
				for status := range operation.Responses {
					code, err := strconv.Atoi(status)
					if err == nil && code >= 200 && code < 300 && code != fixture.Response.Status {
						t.Errorf("OpenAPI advertises success %d, Django fixture returned %d", code, fixture.Response.Status)
					}
				}
				if _, ok := operation.Responses[strconv.Itoa(fixture.Response.Status)]; !ok {
					t.Errorf("OpenAPI operation %s lacks source status %d", tc.operationID, fixture.Response.Status)
				}
			}
			if !found {
				t.Fatalf("OpenAPI lacks %s %s", fixture.Request.Method, tc.operationID)
			}
		})
	}
}

func TestWorkItemGeneratedContractContainsAllSelectedOperations(t *testing.T) {
	body, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, path := range schema.Paths {
		for method := range path {
			switch method {
			case "get", "post", "put", "patch", "delete":
				count++
			}
		}
	}
	if count != 72 || len(api.MethodNames) != 72 {
		t.Fatalf("OpenAPI=%d service=%d, want72", count, len(api.MethodNames))
	}
	if strings.Contains(string(body), "planeSessionCookieCredential") {
		t.Fatal("session credential leaked into public request payload")
	}
}
