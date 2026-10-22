package design_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	client "github.com/CaliLuke/go-plane/work_item_core/gen/http/work_item_core/client"
	api "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

// The complete identifier is one native string path parameter. The domain owns
// Django's split and numeric lookup semantics; transport preserves the spelling.
func TestWorkItemGeneratedIdentifierSchemaPreservesSourceSpelling(t *testing.T) {
	for _, fixtureName := range []string{
		"issue-core-entry-v1-issues-identifier-unicode-decimal",
		"issue-core-entry-v1-issues-identifier-whitespace",
		"issue-core-entry-v1-issues-identifier-out-of-range",
	} {
		t.Run(fixtureName, func(t *testing.T) {
			data, err := os.ReadFile("../../../../apps/go-api/internal/parity/testdata/fixtures/work_item_core/" + fixtureName + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var fixture struct{ Request struct{ Path string } }
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatal(err)
			}
			path, err := url.PathUnescape(strings.TrimSuffix(fixture.Request.Path, "/"))
			if err != nil {
				t.Fatal(err)
			}
			spelling := path[strings.LastIndex(path, "/")+1:]
			if !strings.HasPrefix(spelling, "IC-") {
				t.Fatalf("source identifier missing from %q", path)
			}
			body, err := json.Marshal(map[string]any{"identifier": spelling, "slug": "entry"})
			if err != nil {
				t.Fatal(err)
			}
			for _, payload := range []any{
				&api.GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifierPayload{},
				&api.GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload{},
				&api.GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload{},
			} {
				if err := json.Unmarshal(body, payload); err != nil {
					t.Fatalf("generated %T cannot express source spelling %q: %v", payload, spelling, err)
				}
				value := reflect.ValueOf(payload).Elem().FieldByName("Identifier")
				if value.Kind() != reflect.String || value.String() != spelling {
					t.Fatalf("%T changed source spelling %q to %v", payload, spelling, value)
				}
			}
		})
	}
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	type parameter struct {
		Ref      string `json:"$ref"`
		Name     string
		In       string
		Required bool
		Schema   struct{ Type string }
	}
	var document struct {
		Components struct {
			Parameters map[string]parameter
		}
		Paths map[string]map[string]struct {
			Parameters []parameter
		}
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for path, methods := range document.Paths {
		if !strings.Contains(path, "{identifier}") {
			continue
		}
		found := false
		for _, parameter := range methods["get"].Parameters {
			if parameter.Ref != "" {
				key, ok := strings.CutPrefix(parameter.Ref, "#/components/parameters/")
				resolved, exists := document.Components.Parameters[key]
				if !ok || !exists {
					t.Fatalf("cannot resolve parameter %q", parameter.Ref)
				}
				parameter = resolved
			}
			if parameter.Name == "identifier" {
				found = true
				if parameter.Schema.Type != "string" || parameter.In != "path" || !parameter.Required {
					t.Errorf("%s identifier=%+v; want required string path parameter", path, parameter)
				}
			}
		}
		if !found {
			t.Errorf("%s omits identifier parameter", path)
		}
		checked++
	}
	if checked != 3 {
		t.Fatalf("checked %d identifier routes; want 3", checked)
	}
}

type identifierTransportService struct {
	transportService
	slug, identifier string
}

func (s *identifierTransportService) captureIdentifier(slug, identifier string) (*api.WorkItemResponse, io.ReadCloser, error) {
	s.slug, s.identifier = slug, identifier
	return &api.WorkItemResponse{}, io.NopCloser(strings.NewReader(`{}`)), nil
}

func (s *identifierTransportService) GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifier(_ context.Context, p *api.GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifierPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.captureIdentifier(p.Slug, p.Identifier)
}

func (s *identifierTransportService) GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier(_ context.Context, p *api.GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.captureIdentifier(p.Slug, p.Identifier)
}

func (s *identifierTransportService) GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier(_ context.Context, p *api.GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload) (*api.WorkItemResponse, io.ReadCloser, error) {
	return s.captureIdentifier(p.Slug, p.Identifier)
}

// This is generated-client/server transport proof. Source-specific routing,
// numeric conversion, authorization and lookup still belong to HTTP parity.
func TestWorkItemGeneratedIdentifierClientServerPreservesSpelling(t *testing.T) {
	key := "spike-key"
	for _, route := range []struct {
		name     string
		endpoint func(*client.Client) loom.Endpoint
		payload  func(string) any
		scheme   string
	}{
		{"v1 issues", (*client.Client).GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifier, func(value string) any {
			return &api.GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifierPayload{Slug: "demo", Identifier: value, APIKeyAuthCredential: &key}
		}, "ApiKeyAuth"},
		{"v1 work-items", (*client.Client).GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier, func(value string) any {
			return &api.GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload{Slug: "demo", Identifier: value, APIKeyAuthCredential: &key}
		}, "ApiKeyAuth"},
		{"app work-items", (*client.Client).GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifier, func(value string) any {
			return &api.GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierPayload{Slug: "demo", Identifier: value}
		}, "PlaneSessionCookie"},
	} {
		t.Run(route.name, func(t *testing.T) {
			service := &identifierTransportService{}
			remote := httptest.NewServer(transportWithService(t, service))
			defer remote.Close()
			transport := client.NewClient("http", strings.TrimPrefix(remote.URL, "http://"), workItemCookieDoer{remote.Client()}, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			for _, identifier := range []string{"PROJ-17", "FOO-BAR-1", "IC--1", "IC-١", "IC- 1 ", "IC-999999999999999999999999", "A%B#C-1"} {
				t.Run(identifier, func(t *testing.T) {
					result, err := route.endpoint(transport)(context.Background(), route.payload(identifier))
					if err != nil {
						t.Fatal(err)
					}
					body := identifierResponseBody(t, result)
					defer body.Close()
					data, err := io.ReadAll(body)
					if err != nil || string(data) != `{}` {
						t.Fatalf("response=%s error=%v", data, err)
					}
					if service.slug != "demo" || service.identifier != identifier || service.scheme != route.scheme {
						t.Fatalf("delivered slug=%q identifier=%q scheme=%q", service.slug, service.identifier, service.scheme)
					}
				})
			}
		})
	}
}

func identifierResponseBody(t *testing.T, result any) io.ReadCloser {
	t.Helper()
	switch response := result.(type) {
	case *api.GetAPIV1WorkspacesBySlugIssuesProjectIdentifierIssueIdentifierResponseData:
		return response.Body
	case *api.GetAPIV1WorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierResponseData:
		return response.Body
	case *api.GetAPIWorkspacesBySlugWorkItemsProjectIdentifierIssueIdentifierResponseData:
		return response.Body
	default:
		t.Fatalf("unexpected generated response %T", result)
		return nil
	}
}
