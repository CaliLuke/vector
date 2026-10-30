package design_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	client "github.com/CaliLuke/go-plane/project_label_writes/gen/http/project_label_writes/client"
	server "github.com/CaliLuke/go-plane/project_label_writes/gen/http/project_label_writes/server"
	api "github.com/CaliLuke/go-plane/project_label_writes/gen/project_label_writes"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

const (
	// Django accepts canonical UUID-shaped values without enforcing RFC variant bits.
	projectID = "01234567-89ab-0def-0123-456789abcdef"
	labelID   = "abcdef01-2345-0789-0bcd-ef0123456789"
)

// These tests exercise the generated boundary. Django permission, parsing,
// response negotiation, and effects are characterized by separate parity cases.
type labelService struct {
	t         *testing.T
	called    string
	body      string
	authCalls int
	failure   error
	parent    *string
}

func (s *labelService) APIKeyAuth(ctx context.Context, _ string, scheme *security.APIKeyScheme) (context.Context, error) {
	s.authCalls++
	metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
	if !ok || scheme.Name != "PlaneSessionCookie" {
		s.t.Error("missing generated session metadata")
		return ctx, errors.New("missing generated session metadata")
	}
	if metadata.Headers().Get("Cookie") != "session-id=spike" {
		message := "Authentication credentials were not provided."
		return ctx, &api.LabelError{Kind: "status_401", Detail: &message}
	}
	return ctx, nil
}

func (s *labelService) record(name, slug, project, pk string, body io.ReadCloser) (*api.Label, error) {
	if slug != "demo" || project != projectID || (pk != "" && pk != labelID) {
		s.t.Errorf("wrong target %q/%q/%q", slug, project, pk)
		return nil, errors.New("wrong target")
	}
	s.called = name
	if body != nil {
		data, err := io.ReadAll(body)
		if err != nil {
			s.t.Error(err)
			return nil, err
		}
		s.body = string(data)
	}
	if s.failure != nil {
		return nil, s.failure
	}
	parent := loom.NullValue[string]()
	if s.parent != nil {
		parent = loom.NullableValue(*s.parent)
	}
	return &api.Label{Color: "#123456", ID: labelID, Name: "Sprint", Parent: parent, ProjectID: loom.NullableValue(projectID), SortOrder: 2.5, WorkspaceID: projectID}, nil
}

func (s *labelService) Create(_ context.Context, p *api.CreatePayload, body io.ReadCloser) (*api.Label, error) {
	return s.record("Create", p.Slug, p.ProjectID, "", body)
}
func (s *labelService) Update(_ context.Context, p *api.UpdatePayload, body io.ReadCloser) (*api.Label, error) {
	return s.record("Update", p.Slug, p.ProjectID, p.Pk, body)
}
func (s *labelService) Patch(_ context.Context, p *api.PatchPayload, body io.ReadCloser) (*api.Label, error) {
	return s.record("Patch", p.Slug, p.ProjectID, p.Pk, body)
}
func (s *labelService) Delete(_ context.Context, p *api.DeletePayload) error {
	_, err := s.record("Delete", p.Slug, p.ProjectID, p.Pk, nil)
	return err
}
func (s *labelService) CollectionOptions(_ context.Context, p *api.CollectionOptionsPayload) (*api.LabelMetadata, error) {
	s.record("CollectionOptions", p.Slug, p.ProjectID, "", nil)
	return labelMetadata(), s.failure
}
func (s *labelService) DetailOptions(_ context.Context, p *api.DetailOptionsPayload) (*api.LabelMetadata, error) {
	s.record("DetailOptions", p.Slug, p.ProjectID, p.Pk, nil)
	return labelMetadata(), s.failure
}
func labelMetadata() *api.LabelMetadata {
	return &api.LabelMetadata{Name: "Issue label", Description: "", Renders: []string{"application/json"}, Parses: []string{"application/json"}, Actions: map[string]map[string]*api.LabelMetadataField{"POST": {"name": {Type: "string", Required: true, ReadOnly: false, Label: "Name"}}}}
}

func labelTransport(t *testing.T) (http.Handler, *labelService) {
	t.Helper()
	s := &labelService{t: t}
	mux := loomhttp.NewMuxer()
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "https://app.example.test", Methods: []string{"POST", "PUT", "PATCH", "DELETE"}, Headers: []string{"Content-Type"}, Credentials: true}}})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	generated := server.New(api.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	generated.Use(loomhttp.RequestMetadataMiddleware(metadata))
	generated.Mount(mux)
	return mux, s
}

func labelRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Cookie", "session-id=spike")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestGeneratedLabelWritesPassRawBodiesAndReturnCompleteLabel(t *testing.T) {
	for _, tc := range []struct {
		name, call, method, path, body, media string
		status                                int
	}{
		{"Create JSON object", "Create", "POST", server.CreateProjectLabelWritesPath("demo", projectID), `{"name":"Sprint"}`, "application/json", 201},
		{"Create form", "Create", "POST", server.CreateProjectLabelWritesPath("demo", projectID), "name=Sprint&parent=", "application/x-www-form-urlencoded", 201},
		{"Update nonobject JSON", "Update", "PUT", server.UpdateProjectLabelWritesPath("demo", projectID, labelID), `[1,2]`, "application/json", 200},
		{"Update multipart", "Update", "PUT", server.UpdateProjectLabelWritesPath("demo", projectID, labelID), "--boundary\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nSprint\r\n--boundary--\r\n", "multipart/form-data; boundary=boundary", 200},
		{"Patch malformed JSON", "Patch", "PATCH", server.PatchProjectLabelWritesPath("demo", projectID, labelID), `{malformed`, "application/json", 200},
		{"Delete", "Delete", "DELETE", server.DeleteProjectLabelWritesPath("demo", projectID, labelID), "", "application/json", 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, service := labelTransport(t)
			response := httptest.NewRecorder()
			request := labelRequest(tc.method, tc.path+"?ignored=%GG", tc.body)
			request.Header.Set("Content-Type", tc.media)
			handler.ServeHTTP(response, request)
			if response.Code != tc.status || service.called != tc.call || service.body != tc.body {
				t.Fatalf("status=%d call=%q body=%q response=%s", response.Code, service.called, service.body, response.Body.String())
			}
			if tc.method == "DELETE" {
				if response.Body.Len() != 0 {
					t.Fatalf("DELETE body=%s", response.Body.String())
				}
				return
			}
			var got map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"color": "#123456", "id": labelID, "name": "Sprint", "parent": nil, "project_id": projectID, "sort_order": 2.5, "workspace_id": projectID}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("label=%v want=%v", got, want)
			}
		})
	}
}

type readSpy struct{ reads int }

func (s *readSpy) Read([]byte) (int, error) { s.reads++; return 0, io.ErrUnexpectedEOF }
func (*readSpy) Close() error               { return nil }

func TestGeneratedLabelWritesAuthenticateBeforeReadingRawBody(t *testing.T) {
	handler, service := labelTransport(t)
	body := &readSpy{}
	req := httptest.NewRequest("POST", server.CreateProjectLabelWritesPath("demo", projectID), body)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != 401 || service.called != "" || body.reads != 0 || service.authCalls != 1 {
		t.Fatalf("status=%d call=%q reads=%d auth=%d", response.Code, service.called, body.reads, service.authCalls)
	}
}

func TestGeneratedLabelWritesRejectMalformedPathUUID(t *testing.T) {
	handler, service := labelTransport(t)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, labelRequest("PATCH", server.PatchProjectLabelWritesPath("demo", "not-a-uuid", labelID), `{}`))
	if response.Code != 400 || service.called != "" {
		t.Fatalf("status=%d call=%q", response.Code, service.called)
	}
}

func TestGeneratedLabelWritesPreserveValidationArraysAndJSONServerError(t *testing.T) {
	// Field validation and unexpected failures share the JSON error contract.
	serverMessage := "Something went wrong please try again later"
	for _, tc := range []struct {
		name        string
		failure     error
		status      int
		contentType string
		body        map[string]any
	}{
		{
			"field errors",
			&api.LabelError{Kind: "status_400", NameErrors: []string{"This field is required."}, Parent: []string{"Invalid pk."}},
			400,
			"application/json",
			map[string]any{"name": []any{"This field is required."}, "parent": []any{"Invalid pk."}},
		},
		{
			"server error",
			&api.LabelError{Kind: "status_500", ErrorValue: &serverMessage},
			500,
			"application/json",
			map[string]any{"error": serverMessage},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, service := labelTransport(t)
			service.failure = tc.failure
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, labelRequest("POST", server.CreateProjectLabelWritesPath("demo", projectID), `{}`))
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON body %q: %v", response.Body.String(), err)
			}
			if response.Code != tc.status || !strings.HasPrefix(response.Header().Get("Content-Type"), tc.contentType) || !reflect.DeepEqual(body, tc.body) {
				t.Fatalf("status=%d content-type=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
			}
		})
	}
}

func TestGeneratedLabelWritesOptionsDispatchAndCORS(t *testing.T) {
	for _, tc := range []struct{ name, path, call string }{
		{"collection", server.CollectionOptionsProjectLabelWritesPath("demo", projectID), "CollectionOptions"},
		{"detail", server.DetailOptionsProjectLabelWritesPath("demo", projectID, labelID), "DetailOptions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, service := labelTransport(t)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, labelRequest("OPTIONS", tc.path, ""))
			if response.Code != 200 || service.called != tc.call || service.authCalls != 1 || !strings.Contains(response.Body.String(), `"actions"`) {
				t.Fatalf("ordinary OPTIONS status=%d call=%q auth=%d body=%s", response.Code, service.called, service.authCalls, response.Body.String())
			}
			handler, service = labelTransport(t)
			unauthenticated := httptest.NewRecorder()
			handler.ServeHTTP(unauthenticated, httptest.NewRequest("OPTIONS", tc.path, nil))
			if unauthenticated.Code != 401 || service.authCalls != 1 || service.called != "" {
				t.Fatalf("unauthenticated OPTIONS status=%d auth=%d call=%q", unauthenticated.Code, service.authCalls, service.called)
			}
			handler, service = labelTransport(t)
			preflight := httptest.NewRequest("OPTIONS", tc.path, nil)
			preflight.Header.Set("Origin", "https://app.example.test")
			preflight.Header.Set("Access-Control-Request-Method", "POST")
			preflight.Header.Set("Access-Control-Request-Headers", "Content-Type")
			response = httptest.NewRecorder()
			handler.ServeHTTP(response, preflight)
			if response.Code != 204 || service.authCalls != 0 || service.called != "" || response.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" || response.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Fatalf("preflight status=%d call=%q auth=%d headers=%v", response.Code, service.called, service.authCalls, response.Header())
			}
		})
	}
}

type cookieDoer struct{ client *http.Client }

func TestGeneratedLabelWritesClientAcceptsDjangoUUIDParent(t *testing.T) {
	handler, service := labelTransport(t)
	parent := labelID
	service.parent = &parent
	remote := httptest.NewServer(handler)
	defer remote.Close()
	transport := client.NewClient("http", strings.TrimPrefix(remote.URL, "http://"), cookieDoer{remote.Client()}, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	result, err := transport.Create()(context.Background(), &api.CreateRequestData{
		Payload: &api.CreatePayload{Slug: "demo", ProjectID: projectID}, Body: io.NopCloser(strings.NewReader(`{}`)),
	})
	if err != nil {
		t.Fatal(err)
	}
	label := result.(*api.Label)
	if value, present := label.Parent.Value(); !present || value != parent {
		t.Fatalf("parent=%v, want %s", label.Parent, parent)
	}
}

func (d cookieDoer) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Cookie", "session-id=spike")
	req.Header.Set("Content-Type", "application/json")
	return d.client.Do(req)
}

func TestGeneratedLabelWritesClientRoundTripsAndPaths(t *testing.T) {
	handler, service := labelTransport(t)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	clientTransport := client.NewClient("http", strings.TrimPrefix(httpServer.URL, "http://"), cookieDoer{httpServer.Client()}, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, tc := range []struct {
		name     string
		endpoint loom.Endpoint
		request  any
		body     string
	}{
		{"Create", clientTransport.Create(), &api.CreateRequestData{Payload: &api.CreatePayload{Slug: "demo", ProjectID: projectID}, Body: io.NopCloser(strings.NewReader(`null`))}, `null`},
		{"Update", clientTransport.Update(), &api.UpdateRequestData{Payload: &api.UpdatePayload{Slug: "demo", ProjectID: projectID, Pk: labelID}, Body: io.NopCloser(strings.NewReader(`[1,2]`))}, `[1,2]`},
		{"Patch", clientTransport.Patch(), &api.PatchRequestData{Payload: &api.PatchPayload{Slug: "demo", ProjectID: projectID, Pk: labelID}, Body: io.NopCloser(strings.NewReader(`{broken`))}, `{broken`},
		{"Delete", clientTransport.Delete(), &api.DeletePayload{Slug: "demo", ProjectID: projectID, Pk: labelID}, ""},
		{"CollectionOptions", clientTransport.CollectionOptions(), &api.CollectionOptionsPayload{Slug: "demo", ProjectID: projectID}, ""},
		{"DetailOptions", clientTransport.DetailOptions(), &api.DetailOptionsPayload{Slug: "demo", ProjectID: projectID, Pk: labelID}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.endpoint(context.Background(), tc.request)
			if err != nil {
				t.Fatal(err)
			}
			if service.called != tc.name || service.body != tc.body {
				t.Fatalf("call=%q body=%q", service.called, service.body)
			}
			if tc.name == "Create" || tc.name == "Update" || tc.name == "Patch" {
				label, ok := result.(*api.Label)
				if !ok || !label.Parent.IsNull() || label.ProjectID.IsNull() {
					t.Fatalf("decoded label=%#v", result)
				}
			}
			service.body = ""
		})
	}
	if got := client.CreateProjectLabelWritesPath("a/b", projectID); !strings.Contains(got, "a%2Fb") {
		t.Fatalf("escaped client path=%q", got)
	}
	if got := server.CreateProjectLabelWritesPath("a/b", projectID); !strings.Contains(got, "a%2Fb") {
		t.Fatalf("escaped server path=%q", got)
	}
}

func TestGeneratedLabelWritesOpenAPIContract(t *testing.T) {
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths      map[string]map[string]json.RawMessage
		Components struct {
			SecuritySchemes map[string]struct{ Type, In, Name string }
			RequestBodies   map[string]struct {
				Required bool
				Content  map[string]json.RawMessage
			}
			Responses map[string]json.RawMessage
			Schemas   map[string]struct {
				Properties map[string]json.RawMessage
				Required   []string
			}
		}
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for path, operations := range doc.Paths {
		for _, method := range []string{"get", "head", "post", "patch", "put", "delete", "options"} {
			if _, ok := operations[method]; ok {
				got[path] = append(got[path], method)
			}
		}
	}
	collection := "/api/workspaces/{slug}/projects/{project_id}/issue-labels/"
	detail := collection + "{pk}/"
	want := map[string][]string{collection: {"post", "options"}, detail: {"patch", "put", "delete", "options"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("operations=%v want=%v", got, want)
	}
	scheme := doc.Components.SecuritySchemes["PlaneSessionCookie"]
	if scheme.Type != "apiKey" || scheme.In != "cookie" || scheme.Name != "session-id" {
		t.Fatalf("session scheme=%+v", scheme)
	}
	labelSchema := doc.Components.Schemas["Label"]
	if len(labelSchema.Properties) != 7 || len(labelSchema.Required) != 7 {
		t.Fatalf("incomplete Label schema: %+v", labelSchema)
	}
	for _, field := range []string{"color", "id", "name", "parent", "project_id", "sort_order", "workspace_id"} {
		if _, ok := labelSchema.Properties[field]; !ok {
			t.Fatalf("Label schema missing %q", field)
		}
	}
	for _, field := range []string{"parent", "project_id"} {
		if !strings.Contains(string(labelSchema.Properties[field]), `"null"`) {
			t.Fatalf("Label %s is not nullable: %s", field, labelSchema.Properties[field])
		}
	}
	for path, methods := range doc.Paths {
		for name, raw := range methods {
			if name != "post" && name != "put" && name != "patch" && name != "delete" && name != "options" {
				continue
			}
			var op struct{ Security []map[string][]string }
			if err := json.Unmarshal(raw, &op); err != nil {
				t.Fatal(err)
			}
			if len(op.Security) != 1 {
				t.Fatalf("%s %s missing session security", name, path)
			}
			if _, ok := op.Security[0]["PlaneSessionCookie"]; !ok {
				t.Fatalf("%s %s wrong security=%v", name, path, op.Security)
			}
		}
	}
	for _, target := range []struct {
		path, method string
		required     bool
	}{{collection, "post", true}, {detail, "put", true}, {detail, "patch", false}} {
		var op struct {
			RequestBody struct {
				Ref      string `json:"$ref"`
				Required bool
				Content  map[string]json.RawMessage
			}
			Security []map[string][]string
		}
		if err := json.Unmarshal(doc.Paths[target.path][target.method], &op); err != nil {
			t.Fatal(err)
		}
		body := op.RequestBody
		if body.Ref != "" {
			referenced := doc.Components.RequestBodies[strings.TrimPrefix(body.Ref, "#/components/requestBodies/")]
			body.Required = referenced.Required
			body.Content = referenced.Content
		}
		if body.Required != target.required || len(body.Content) != 3 || len(op.Security) != 1 {
			t.Fatalf("%s request/security=%+v", target.method, op)
		}
		for _, media := range []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"} {
			if _, ok := body.Content[media]; !ok {
				t.Fatalf("%s missing %s", target.method, media)
			}
		}
		// Unexpected failures expose the JSON contract without a legacy HTML exception.
		var responseRefs struct {
			Responses map[string]struct {
				Ref string `json:"$ref"`
			}
		}
		if err := json.Unmarshal(doc.Paths[target.path][target.method], &responseRefs); err != nil {
			t.Fatal(err)
		}
		ref := strings.TrimPrefix(responseRefs.Responses["500"].Ref, "#/components/responses/")
		var failure struct {
			Content  map[string]json.RawMessage
			Renderer json.RawMessage `json:"x-plane-renderer-failure"`
		}
		if err := json.Unmarshal(doc.Components.Responses[ref], &failure); err != nil {
			t.Fatal(err)
		}
		if _, ok := failure.Content["application/json"]; !ok || len(failure.Content) != 1 || len(failure.Renderer) != 0 {
			t.Fatalf("%s unexpected error response contract: %+v", target.method, failure)
		}
	}
}
