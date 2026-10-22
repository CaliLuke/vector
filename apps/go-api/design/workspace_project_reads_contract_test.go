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

func TestWorkspaceProjectReadsDesignDeclaresSevenDistinctGETOperations(t *testing.T) {
	data, err := os.ReadFile("workspace_project_reads.go")
	if err != nil {
		t.Fatal(err)
	}
	design := string(data)
	methods := []string{
		`Method("user_workspaces"`,
		`Method("workspaces"`,
		`Method("workspace"`,
		`Method("project_stats"`,
		`Method("projects"`,
		`Method("project_details"`,
		`Method("project"`,
	}
	for _, method := range methods {
		if strings.Count(design, method) != 1 {
			t.Errorf("design occurrence count for %s = %d, want 1", method, strings.Count(design, method))
		}
	}
	for _, parameter := range []string{`Param("owner")`, `Param("fields")`, `Param("project_ids")`, `Param("per_page")`, `Param("cursor")`, `Param("order_by")`} {
		if !strings.Contains(design, parameter) {
			t.Errorf("design missing operative query parameter %s", parameter)
		}
	}
	if strings.Count(design, "Result(rawJSONResponseMetadata)") != 7 || strings.Count(design, "rawJSONReadResponse(Any)") != 1 {
		t.Fatalf("raw GET result declarations do not preserve the seven wire contracts")
	}
	for _, metadataContract := range []string{`Type("RawJSONResponseMetadata"`, `Required("content_type")`, `Header("content_type:Content-Type")`} {
		if !strings.Contains(design, metadataContract) {
			t.Errorf("raw response metadata missing %s", metadataContract)
		}
	}
	for _, method := range []string{`Method("user_workspaces_options"`, `Method("project_stats_options"`, `Method("project_details_options"`} {
		if strings.Count(design, method) != 1 {
			t.Errorf("design occurrence count for %s = %d, want 1", method, strings.Count(design, method))
		}
	}
	if !strings.Contains(design, `ContentType("application/json")`) {
		t.Error("raw JSON responses must declare application/json")
	}

	if count := strings.Count(design, `Attribute("owner", String)`); count != 3 {
		t.Errorf("plain post-auth owner payload declarations = %d, want 3", count)
	}
	if strings.Contains(design, `Attribute("pk", String, func() { Format(FormatUUID) })`) {
		t.Error("project pk must reach service-level Django 404 mapping")
	}
	for _, modeledError := range []string{"WorkspaceProjectOwnerBadRequest", "WorkspaceProjectValidationError"} {
		if !strings.Contains(design, modeledError) {
			t.Errorf("design missing %s", modeledError)
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
