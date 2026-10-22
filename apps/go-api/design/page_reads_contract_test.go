package design

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPageReadsExposeSixAuthenticatedGetOperations(t *testing.T) {
	const descriptionPath = "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/"
	document := readOpenAPI(t, "../gen/http/openapi.json")
	expected := map[string][]string{
		"/api/workspaces/{slug}/projects/{project_id}/pages-summary/":   {"project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/pages/":           {"order_by", "project_id", "search", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/": {"order_by", "page_id", "project_id", "search", "slug", "track_visit"},
		descriptionPath: {"page_id", "project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/":      {"page_id", "project_id", "search", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/": {"page_id", "pk", "project_id", "slug"},
	}
	for path, parameters := range expected {
		operation, exists := document.Paths[path]["get"]
		if !exists {
			t.Fatalf("generated surface missing GET %s", path)
		}
		if len(operation.Security) != 1 || len(operation.Security[0]["plane_session_cookie"]) != 0 {
			t.Errorf("GET %s security = %#v, want required session cookie", path, operation.Security)
		}
		got := pageParameterNames(t, document, operation.Parameters)
		if path == descriptionPath {
			got = pageParameterNames(t, document, operation.Parameters, "path", "query")
		}
		slices.Sort(parameters)
		if !slices.Equal(got, parameters) {
			t.Errorf("GET %s parameters = %v, want %v", path, got, parameters)
		}
	}
	description := document.Paths[descriptionPath]["get"]
	wantHeaders := []string{"If-Match", "If-Modified-Since", "If-None-Match", "If-Range", "If-Unmodified-Since", "Range"}
	gotHeaders := pageParameterNames(t, document, description.Parameters, "header")
	if !slices.Equal(gotHeaders, wantHeaders) {
		t.Errorf("GET %s standard range/conditional headers = %v, want %v", descriptionPath, gotHeaders, wantHeaders)
	}
	if got, want := len(description.Parameters), len(expected[descriptionPath])+len(wantHeaders); got != want {
		t.Errorf("GET %s has %d parameters, want %d path/query plus standard header parameters", descriptionPath, got, want)
	}
	for _, parameter := range resolvedPageParameters(t, document, description.Parameters) {
		if parameter.In == "header" && parameter.Schema.Resolved().Type.Value != "string" {
			t.Errorf("GET %s header %s schema = %#v, want string", descriptionPath, parameter.Name, parameter.Schema)
		}
	}
	head, exists := document.Paths[descriptionPath]["head"]
	if !exists {
		t.Fatalf("generated surface missing explicit HEAD %s", descriptionPath)
	}
	if description.OperationID == "" || head.OperationID == "" || description.OperationID == head.OperationID {
		t.Errorf("GET/HEAD description operation IDs = %q / %q, want distinct non-empty IDs", description.OperationID, head.OperationID)
	}
	if len(head.Security) != 1 || len(head.Security[0]["plane_session_cookie"]) != 0 {
		t.Errorf("HEAD %s security = %#v, want required session cookie", descriptionPath, head.Security)
	}
	headHeaders := pageParameterNames(t, document, head.Parameters, "header")
	if !slices.Equal(headHeaders, wantHeaders) {
		t.Errorf("HEAD %s standard range/conditional headers = %v, want %v", descriptionPath, headHeaders, wantHeaders)
	}
	detail := document.Paths["/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/"]["get"]
	foundTrackVisit := false
	for _, parameter := range resolvedPageParameters(t, document, detail.Parameters) {
		if parameter.Name != "track_visit" {
			continue
		}
		foundTrackVisit = true
		if got := parameter.Schema.Resolved().Type.Value; got != "string" {
			t.Errorf("track_visit type = %q, want string to preserve Django query parsing", got)
		}
	}
	if !foundTrackVisit {
		t.Error("detail track_visit parameter is missing")
	}
}
func TestPageReadsExposeAuthenticatedOrdinaryOptionsOperations(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	expected := map[string][]string{
		"/api/workspaces/{slug}/projects/{project_id}/pages-summary/": {
			"project_id", "slug",
		},
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/": {
			"page_id", "project_id", "slug",
		},
		"/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/": {
			"page_id", "pk", "project_id", "slug",
		},
	}
	for path, parameters := range expected {
		operation, exists := document.Paths[path]["options"]
		if !exists {
			t.Fatalf("generated surface missing OPTIONS %s", path)
		}
		if len(operation.Security) != 1 || len(operation.Security[0]["plane_session_cookie"]) != 0 {
			t.Errorf("OPTIONS %s security = %#v, want required session cookie", path, operation.Security)
		}
		got := pageParameterNames(t, document, operation.Parameters)
		slices.Sort(parameters)
		if !slices.Equal(got, parameters) {
			t.Errorf("OPTIONS %s parameters = %v, want %v", path, got, parameters)
		}
	}
}

func TestPageSummaryAndDetailSchemasCorrectGeneratedContractGaps(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")

	summary, exists := document.Components.Schemas["PageSummaryRead"]
	if !exists {
		t.Fatal("PageSummaryRead schema is missing")
	}
	for _, field := range []string{"public_pages", "private_pages", "archived_pages"} {
		property, found := summary.Properties[field]
		if !found || property.Resolved().Type.Value != "integer" || !slices.Contains(summary.Required, field) {
			t.Errorf("PageSummaryRead.%s = %#v required=%v", field, property, summary.Required)
		}
	}

	detail, exists := document.Components.Schemas["PageDetailRead"]
	if !exists {
		t.Fatal("PageDetailRead schema is missing")
	}
	for _, field := range []string{"description_html", "issue_ids"} {
		if _, found := detail.Properties[field]; !found {
			t.Errorf("PageDetailRead.%s is missing", field)
		}
		if !slices.Contains(detail.Required, field) {
			t.Errorf("PageDetailRead.%s is not required", field)
		}
	}
}

func TestPageDetailIssueIDsAllowNullArrayItems(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "gen", "http", "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Items struct {
						AnyOf []struct {
							Type   openAPIType `json:"type"`
							Format string      `json:"format"`
						} `json:"anyOf"`
					} `json:"items"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	items := document.Components.Schemas["PageDetailRead"].Properties["issue_ids"].Items
	hasUUID, hasNull := false, false
	for _, item := range items.AnyOf {
		hasUUID = hasUUID || item.Type.Value == "string" && item.Format == "uuid"
		hasNull = hasNull || item.Type.Value == "null"
	}
	if !hasUUID || !hasNull {
		t.Fatalf("PageDetailRead.issue_ids items = %#v, want string/uuid and null alternatives", items)
	}
}

func TestPageReadSchemaPreservesStoredAndNullableFields(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	page := document.Components.Schemas["PageRead"]
	for _, field := range []string{
		"id", "name", "owned_by", "access", "color", "parent", "is_favorite", "is_locked",
		"archived_at", "workspace", "created_at", "updated_at", "created_by", "updated_by",
		"view_props", "logo_props", "label_ids", "project_ids",
	} {
		if _, exists := page.Properties[field]; !exists {
			t.Errorf("PageRead.%s is missing", field)
		}
		if !slices.Contains(page.Required, field) {
			t.Errorf("PageRead.%s is not required", field)
		}
	}
	for _, field := range []string{"parent", "archived_at", "created_by", "updated_by"} {
		if !page.Properties[field].Resolved().Type.Nullable {
			t.Errorf("PageRead.%s must preserve explicit null", field)
		}
	}
}

func TestPageVersionDetailPreservesHTMLJSONAndBinaryNullability(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	version := document.Components.Schemas["PageVersionDetailRead"]
	for _, field := range []string{"description_binary", "description_html", "description_json"} {
		if _, exists := version.Properties[field]; !exists {
			t.Errorf("PageVersionDetailRead.%s is missing", field)
		}
		if !slices.Contains(version.Required, field) {
			t.Errorf("PageVersionDetailRead.%s is not required", field)
		}
	}
	binaryProperty := version.Properties["description_binary"]
	if len(binaryProperty.AnyOf) != 2 {
		t.Fatalf("description_binary anyOf = %#v, want base64 byte string and null branches", binaryProperty.AnyOf)
	}
	hasBinary := false
	hasNull := false
	for _, branch := range binaryProperty.AnyOf {
		switch {
		case branch.Type.Value == "string" && branch.Format == "byte":
			hasBinary = true
		case branch.Type.Value == "null" && branch.Format == "":
			hasNull = true
		}
	}
	binary := binaryProperty.Resolved()
	if !hasBinary || !hasNull ||
		binaryProperty.Type.Value != "" || binaryProperty.Format != "" ||
		binary.Type.Value != "string" || binary.Format != "byte" || !binary.Type.Nullable {
		t.Errorf("description_binary = %#v, want required nullable format=byte string represented by anyOf", binaryProperty)
	}
}

func TestPageDescriptionGETAndHEADAreRawAttachments(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	path := "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/"
	for _, method := range []string{"get", "head"} {
		operation, exists := document.Paths[path][method]
		if !exists {
			t.Fatalf("description %s operation is missing", method)
		}
		responseData := operation.Responses["200"]
		var response struct {
			Content map[string]json.RawMessage `json:"content"`
			Headers map[string]json.RawMessage `json:"headers"`
		}
		if err := json.Unmarshal(responseData, &response); err != nil {
			t.Fatal(err)
		}
		if method == "get" {
			if _, exists := response.Content["application/octet-stream"]; !exists {
				t.Fatalf("GET description content types = %v", sortedKeys(response.Content))
			}
		} else if len(response.Content) != 0 {
			t.Fatalf("HEAD description unexpectedly declares body content %v", sortedKeys(response.Content))
		}
		if _, exists := response.Headers["Content-Disposition"]; !exists {
			t.Fatalf("%s description headers = %v", method, sortedKeys(response.Headers))
		}
	}
}

func resolvedPageParameters(t *testing.T, document openAPIDocument, parameters []openAPIParameter) []openAPIParameter {
	t.Helper()
	resolved := make([]openAPIParameter, 0, len(parameters))
	for _, parameter := range parameters {
		if parameter.Ref != "" {
			componentName := filepath.Base(parameter.Ref)
			component, exists := document.Components.Parameters[componentName]
			if !exists {
				t.Fatalf("OpenAPI parameter reference %q is missing", parameter.Ref)
			}
			parameter = component
		}
		resolved = append(resolved, parameter)
	}
	return resolved
}

func pageParameterNames(t *testing.T, document openAPIDocument, parameters []openAPIParameter, locations ...string) []string {
	t.Helper()
	names := make([]string, 0, len(parameters))
	for _, parameter := range resolvedPageParameters(t, document, parameters) {
		if len(locations) == 0 || slices.Contains(locations, parameter.In) {
			names = append(names, parameter.Name)
		}
	}
	slices.Sort(names)
	return names
}
