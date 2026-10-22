package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	client "github.com/CaliLuke/go-plane/projectlifecycle/gen/http/project_lifecycle/client"
	api "github.com/CaliLuke/go-plane/projectlifecycle/gen/project_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
)

type projectLifecycleClientDoer struct{ response *http.Response }

func (d *projectLifecycleClientDoer) Do(*http.Request) (*http.Response, error) {
	return d.response, nil
}

func projectLifecycleOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("projectlifecycle/gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func TestProjectLifecycleGeneratedClientPreservesSourceBodyAndMediaError(t *testing.T) {
	raw := `{"cover_image_url":null,"anchor":null,"member_role":null,"sort_order":65535.0,"members":["00000000-0000-4000-8000-000000000001"],"next_work_item_sequence":1}`
	response := func(status int, body string) *http.Response {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
	}
	doer := &projectLifecycleClientDoer{response: response(201, raw)}
	transport := client.NewClient("http", "example.test", doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	endpoint := transport.PostAPIWorkspacesBySlugProjects()
	request := &api.PostAPIWorkspacesBySlugProjectsRequestData{Payload: &api.PostAPIWorkspacesBySlugProjectsPayload{Slug: "space"}, Body: io.NopCloser(strings.NewReader(`{"name":"Project","identifier":"P"}`))}
	result, err := endpoint(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	body := result.(*api.PostAPIWorkspacesBySlugProjectsResponseData).Body
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil || string(got) != raw {
		t.Fatalf("raw response changed: %s,%v", got, err)
	}
	doer.response = response(415, `{"detail":"Unsupported media type in request."}`)
	_, err = endpoint(context.Background(), request)
	var failure *api.ProjectLifecycleErrorIdentity
	if !errors.As(err, &failure) || failure.Kind != "status_415" {
		t.Fatalf("media error=%v", err)
	}
}
func projectLifecycleSchema(t *testing.T, document map[string]any, value any) map[string]any {
	t.Helper()
	schema, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("missing schema: %#v", value)
	}
	if ref, ok := schema["$ref"].(string); ok {
		kind, name, found := strings.Cut(strings.TrimPrefix(ref, "#/components/"), "/")
		if !found {
			t.Fatalf("unsupported component reference %s", ref)
		}
		return projectLifecycleSchema(t, document, document["components"].(map[string]any)[kind].(map[string]any)[name])
	}
	return schema
}
func TestProjectLifecycleAPIProjectReadsArePublished(t *testing.T) {
	doc := projectLifecycleOpenAPI(t)
	paths := doc["paths"].(map[string]any)
	for _, path := range []string{"/api/v1/workspaces/{slug}/projects/", "/api/v1/workspaces/{slug}/projects/{pk}/"} {
		operation, ok := paths[path].(map[string]any)["get"].(map[string]any)
		if !ok {
			t.Fatalf("source-recorded GET %s is absent from generated OpenAPI", path)
		}
		if operation["operationId"] == nil || operation["responses"].(map[string]any)["200"] == nil {
			t.Errorf("GET %s is missing its operation id or success response", path)
		}
	}
}
func TestProjectLifecycleArchiveContractPublishesObservedBodyAndErrors(t *testing.T) {
	doc := projectLifecycleOpenAPI(t)
	operation := doc["paths"].(map[string]any)["/api/workspaces/{slug}/projects/{project_id}/archive/"].(map[string]any)["post"].(map[string]any)
	responses := operation["responses"].(map[string]any)
	success, ok := responses["200"].(map[string]any)
	if !ok {
		t.Fatal("session archive lacks observed 200 response")
	}
	media, ok := success["content"].(map[string]any)
	if !ok {
		t.Fatal("archive raw response lost documented body")
	}
	schema := projectLifecycleSchema(t, doc, media["application/json"].(map[string]any)["schema"])
	properties := schema["properties"].(map[string]any)
	if properties["archived_at"].(map[string]any)["type"] != "string" {
		t.Fatalf("archive schema=%v", schema)
	}
	for _, status := range []string{"401", "403", "404", "500"} {
		if responses[status] == nil {
			t.Errorf("archive runtime status %s is not modeled", status)
		}
	}
}
func TestProjectLifecycleSummaryContractUsesOptionalSupportedIntegerCounts(t *testing.T) {
	doc := projectLifecycleOpenAPI(t)
	operation := doc["paths"].(map[string]any)["/api/v1/workspaces/{slug}/projects/{project_id}/summary/"].(map[string]any)["get"].(map[string]any)
	response := operation["responses"].(map[string]any)["200"].(map[string]any)
	schema := projectLifecycleSchema(t, doc, response["content"].(map[string]any)["application/json"].(map[string]any)["schema"])
	counts := projectLifecycleSchema(t, doc, schema["properties"].(map[string]any)["counts"])
	if required, ok := counts["required"].([]any); ok && len(required) > 0 {
		t.Fatalf("selected counts cannot require omitted fields: %v", required)
	}
	props := counts["properties"].(map[string]any)
	for _, field := range []string{"members", "states", "labels", "cycles", "modules", "issues", "intakes", "pages"} {
		if props[field].(map[string]any)["type"] != "integer" {
			t.Errorf("count %s is not an integer", field)
		}
	}
}

func TestProjectLifecycleSessionWriteSchemaUsesActualMutableFields(t *testing.T) {
	doc := projectLifecycleOpenAPI(t)
	paths := doc["paths"].(map[string]any)
	for _, target := range []struct{ path, method string }{
		{"/api/workspaces/{slug}/projects/", "post"},
		{"/api/workspaces/{slug}/projects/{pk}/", "patch"},
		{"/api/workspaces/{slug}/projects/{pk}/", "put"},
	} {
		op := paths[target.path].(map[string]any)[target.method].(map[string]any)
		requestBody := projectLifecycleSchema(t, doc, op["requestBody"])
		body := requestBody["content"].(map[string]any)["application/json"].(map[string]any)
		schema := projectLifecycleSchema(t, doc, body["schema"])
		fields := schema["properties"].(map[string]any)
		for _, key := range []string{"created_by", "updated_by", "archived_at", "deleted_at"} {
			if _, offered := fields[key]; offered {
				t.Errorf("%s offers server-owned %s", target.method, key)
			}
		}
		if fields["cover_image_asset"] == nil {
			t.Errorf("%s lost supported cover clearing", target.method)
		}
		if target.method != "post" && fields["inbox_view"] == nil {
			t.Errorf("%s lacks session intake alias", target.method)
		}
		if required, ok := schema["required"].([]any); ok {
			for _, key := range required {
				if key == "workspace" {
					t.Errorf("%s requires read-only workspace", target.method)
				}
			}
		}
		if target.method == "put" {
			for _, key := range []string{"network", "logo_props", "description_text"} {
				if fields[key] != nil {
					t.Errorf("PUT offers unexpected %s", key)
				}
			}
		}
		if op["responses"].(map[string]any)["415"] == nil {
			t.Errorf("%s lacks parser media error", target.method)
		}
	}
}

func TestProjectLifecycleSessionResponsePublishesMemberArrayAndSequenceInteger(t *testing.T) {
	doc := projectLifecycleOpenAPI(t)
	op := doc["paths"].(map[string]any)["/api/workspaces/{slug}/projects/"].(map[string]any)["post"].(map[string]any)
	body := op["responses"].(map[string]any)["201"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)
	schema := projectLifecycleSchema(t, doc, body["schema"])
	fields := schema["properties"].(map[string]any)
	members := projectLifecycleSchema(t, doc, fields["members"])
	if members["type"] != "array" {
		t.Fatalf("members=%v", members)
	}
	if items := projectLifecycleSchema(t, doc, members["items"]); items["type"] != "string" || items["format"] != "uuid" {
		t.Errorf("member identity=%v", items)
	}
	if sequence := projectLifecycleSchema(t, doc, fields["next_work_item_sequence"]); sequence["type"] != "integer" {
		t.Errorf("next sequence=%v", sequence)
	}
	for _, key := range []string{"cover_image_url", "anchor", "member_role", "sort_order"} {
		field := projectLifecycleSchema(t, doc, fields[key])
		nullable := false
		for _, branch := range field["anyOf"].([]any) {
			if branch.(map[string]any)["type"] == "null" {
				nullable = true
			}
		}
		if !nullable {
			t.Errorf("%s does not admit observed null", key)
		}
	}
}
