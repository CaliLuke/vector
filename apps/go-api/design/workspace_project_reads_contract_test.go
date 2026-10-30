package design

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

type workspaceOpenAPIParameter struct {
	Ref  string `json:"$ref"`
	Name string `json:"name"`
	In   string `json:"in"`
}

type workspaceOpenAPIOperation struct {
	Parameters []workspaceOpenAPIParameter `json:"parameters"`
}

func TestWorkspaceProjectReadsGeneratedOperationsPreserveWireContracts(t *testing.T) {
	document := readOpenAPI(t)
	paths := map[string]string{
		"/api/users/me/workspaces/":                "get_api_users_me_workspaces",
		"/api/workspaces/":                         "get_api_workspaces",
		"/api/workspaces/{slug}/":                  "get_api_workspaces_by_slug",
		"/api/workspaces/{slug}/project-stats/":    "get_api_workspaces_by_slug_project_stats",
		"/api/workspaces/{slug}/projects/":         "get_api_workspaces_by_slug_projects",
		"/api/workspaces/{slug}/projects/details/": "get_api_workspaces_by_slug_projects_details",
		"/api/workspaces/{slug}/projects/{pk}/":    "get_api_workspaces_by_slug_projects_by_pk",
	}
	operationIDs := map[string]bool{}
	owners := 0
	for path, operationID := range paths {
		t.Run(path, func(t *testing.T) {
			operation, exists := document.Paths[path]["get"]
			if !exists {
				t.Fatal("GET operation missing")
			}
			if operation.OperationID != operationID || operationIDs[operation.OperationID] {
				t.Fatalf("missing or duplicate operation ID %q", operation.OperationID)
			}
			operationIDs[operation.OperationID] = true
			var response struct {
				Headers map[string]json.RawMessage `json:"headers"`
				Content map[string]json.RawMessage `json:"content"`
			}
			if err := json.Unmarshal(operation.Responses["200"], &response); err != nil {
				t.Fatal(err)
			}
			if _, exists := response.Content["application/json"]; !exists {
				t.Error("200 response must publish application/json")
			}
			if _, exists := response.Headers["Content-Type"]; !exists {
				t.Error("200 response must publish Content-Type metadata")
			}
			for _, parameter := range resolvedPageParameters(t, document, operation.Parameters) {
				if parameter.Name != "owner" && parameter.Name != "pk" {
					continue
				}
				if parameter.Schema.Type.Value != "string" || parameter.Schema.Format != "" {
					t.Errorf("%s must reach service validation as an unformatted string: %#v", parameter.Name, parameter.Schema)
				}
				if parameter.Name == "owner" {
					owners++
				}
			}
		})
	}
	if owners != 3 {
		t.Errorf("owner parameters=%d want=3", owners)
	}
	for _, name := range []string{"WorkspaceProjectOwnerBadRequest", "WorkspaceProjectValidationError"} {
		if _, exists := document.Components.Schemas[name]; !exists {
			t.Errorf("missing validation schema %s", name)
		}
	}
}

func TestWorkspaceProjectReadsOpenAPIRepairsGeneratedContractGaps(t *testing.T) {
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths      map[string]map[string]workspaceOpenAPIOperation `json:"paths"`
		Components struct {
			Parameters map[string]workspaceOpenAPIParameter `json:"parameters"`
			Schemas    map[string]struct {
				Type       any `json:"type"`
				Properties map[string]struct {
					Type  any `json:"type"`
					AnyOf []struct {
						Type any `json:"type"`
					} `json:"anyOf"`
					Items *struct {
						Type   any    `json:"type"`
						Format string `json:"format"`
					} `json:"items"`
				} `json:"properties"`
				Required []string `json:"required"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}

	project, ok := document.Components.Schemas["WorkspaceProjectProject"]
	if !ok {
		t.Fatal("WorkspaceProjectProject schema is missing")
	}
	members := project.Properties["members"]
	if !openAPITypeIncludes(members.Type, "array") || members.Items == nil || !openAPITypeIncludes(members.Items.Type, "string") || members.Items.Format != "uuid" {
		t.Fatalf("members schema = %#v, want UUID array", members)
	}
	if !openAPITypeIncludes(project.Properties["next_work_item_sequence"].Type, "integer") {
		t.Fatalf("next_work_item_sequence schema = %#v, want integer", project.Properties["next_work_item_sequence"])
	}
	for _, required := range []string{"members", "next_work_item_sequence"} {
		if !slices.Contains(project.Required, required) {
			t.Errorf("detailed project schema does not require %q", required)
		}
	}
	for _, nullable := range []string{"member_role", "sort_order", "anchor", "cover_image_url"} {
		property := project.Properties[nullable]
		allowsNull := openAPITypeIncludes(property.Type, "null")
		for _, branch := range property.AnyOf {
			allowsNull = allowsNull || openAPITypeIncludes(branch.Type, "null")
		}
		if !allowsNull {
			t.Errorf("project %s must permit the null value emitted by Django", nullable)
		}
	}

	compact := document.Components.Schemas["WorkspaceProjectCompactProject"]
	for _, field := range []string{"intake_count", "inbox_view", "member_role", "sort_order"} {
		if _, exists := compact.Properties[field]; !exists {
			t.Errorf("compact project schema missing %q", field)
		}
	}
	if _, exists := compact.Properties["members"]; exists {
		t.Error("compact project schema incorrectly exposes detailed-project members")
	}

	workspace := document.Components.Schemas["WorkspaceProjectWorkspace"]
	logoURL := workspace.Properties["logo_url"]
	logoAllowsNull := openAPITypeIncludes(logoURL.Type, "null")
	for _, branch := range logoURL.AnyOf {
		logoAllowsNull = logoAllowsNull || openAPITypeIncludes(branch.Type, "null")
	}
	if !logoAllowsNull {
		t.Error("workspace logo_url must permit the null emitted when no logo exists")
	}
	if _, exists := workspace.Properties["role"]; exists {
		t.Error("workspace collection/detail schema exposes the unannotated role field")
	}
	userWorkspace := document.Components.Schemas["WorkspaceProjectUserWorkspace"]
	if _, exists := userWorkspace.Properties["role"]; !exists || !slices.Contains(userWorkspace.Required, "role") {
		t.Error("users/me workspace schema must require its annotated role field")
	}

	stats := document.Components.Schemas["WorkspaceProjectStats"]
	for _, field := range []string{"id", "total_issues", "completed_issues", "total_members", "total_cycles", "total_modules"} {
		if _, exists := stats.Properties[field]; !exists {
			t.Errorf("project stats schema missing %q", field)
		}
	}
	if len(stats.Properties) != 6 {
		t.Errorf("project stats exposes %d fields, want the six real projection fields", len(stats.Properties))
	}
	if len(stats.Required) != 1 || stats.Required[0] != "id" {
		t.Errorf("project stats required fields = %v, want only id because counters are selected by query", stats.Required)
	}

	assertOpenAPIQueryParameters(t, document.Paths, document.Components.Parameters, "/api/users/me/workspaces/", []string{"owner", "search", "fields"})
	assertOpenAPIQueryParameters(t, document.Paths, document.Components.Parameters, "/api/workspaces/{slug}/project-stats/", []string{"fields", "project_ids", "search"})
	assertOpenAPIQueryParameters(t, document.Paths, document.Components.Parameters, "/api/workspaces/{slug}/projects/details/", []string{"fields", "per_page", "cursor", "order_by", "search"})
	for _, path := range []string{"/api/users/me/workspaces/", "/api/workspaces/{slug}/project-stats/", "/api/workspaces/{slug}/projects/details/"} {
		if _, exists := document.Paths[path]["options"]; !exists {
			t.Errorf("OPTIONS %s is missing from the generated service", path)
		}
	}
}

func assertOpenAPIQueryParameters(
	t *testing.T,
	paths map[string]map[string]workspaceOpenAPIOperation,
	components map[string]workspaceOpenAPIParameter,
	path string,
	wanted []string,
) {
	t.Helper()
	operation, ok := paths[path]["get"]
	if !ok {
		t.Errorf("GET %s is missing", path)
		return
	}
	names := make([]string, 0, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		if parameter.Ref != "" {
			const prefix = "#/components/parameters/"
			componentName := strings.TrimPrefix(parameter.Ref, prefix)
			if componentName == parameter.Ref {
				t.Errorf("GET %s parameter has unsupported reference %q", path, parameter.Ref)
				continue
			}
			var exists bool
			parameter, exists = components[componentName]
			if !exists {
				t.Errorf("GET %s parameter reference %q is missing", path, parameter.Ref)
				continue
			}
		}
		if parameter.In == "query" {
			names = append(names, parameter.Name)
		}
	}
	slices.Sort(names)
	wanted = slices.Clone(wanted)
	slices.Sort(wanted)
	if !slices.Equal(names, wanted) {
		t.Errorf("GET %s query parameters = %v, want %v", path, names, wanted)
	}
}

func openAPITypeIncludes(value any, wanted string) bool {
	switch value := value.(type) {
	case string:
		return value == wanted
	case []any:
		return slices.ContainsFunc(value, func(item any) bool { return item == wanted })
	default:
		return false
	}
}
