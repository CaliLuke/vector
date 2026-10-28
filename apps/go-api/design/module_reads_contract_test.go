package design

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
)

func TestModuleReadsExposeTenAuthenticatedGetOperations(t *testing.T) {
	document := readOpenAPI(t)
	expected := map[string][]string{
		"/api/workspaces/{slug}/modules/":                                                     {"slug"},
		"/api/workspaces/{slug}/projects/{project_id}/archived-modules/":                      {"project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/":                 {"pk", "project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/modules/":                               {"fields", "project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/":           {"module_id", "project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/":            {"assignees", "completed_at", "created_at", "created_by", "cursor", "cycle", "estimate_point", "filters", "group_by", "inbox_status", "intake_status", "labels", "logged_by", "mentions", "module", "module_id", "name", "order_by", "parent", "per_page", "priority", "project", "project_id", "slug", "start_date", "start_target_date", "state", "state_group", "sub_group_by", "sub_issue", "subscriber", "target_date", "type", "updated_at"},
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/": {"issue_id", "module_id", "project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/":      {"module_id", "project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/": {"module_id", "pk", "project_id", "slug"},
		"/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/":                          {"pk", "project_id", "slug"},
	}
	if len(expected) != 10 {
		t.Fatalf("test setup has %d operations, want 10", len(expected))
	}
	for path, parameters := range expected {
		operation, exists := document.Paths[path]["get"]
		if !exists {
			t.Fatalf("generated surface missing GET %s", path)
		}
		if len(operation.Security) != 1 || len(operation.Security[0]["plane_session_cookie"]) != 0 {
			t.Errorf("GET %s security = %#v, want required session cookie", path, operation.Security)
		}
		got := parameterNames(document, operation.Parameters)
		slices.Sort(parameters)
		if !slices.Equal(got, parameters) {
			t.Errorf("GET %s parameters = %v, want %v", path, got, parameters)
		}
	}
}

func TestModuleDetailContractsIncludeRuntimeFields(t *testing.T) {
	document := readOpenAPI(t)
	detail := document.Components.Schemas["ModuleDetailRead"]
	for _, field := range []string{
		"link_module", "sub_issues", "backlog_estimate_points", "unstarted_estimate_points",
		"started_estimate_points", "cancelled_estimate_points", "distribution", "estimate_distribution",
	} {
		if _, exists := detail.Properties[field]; !exists {
			t.Errorf("ModuleDetailRead.%s is missing", field)
		}
		if !slices.Contains(detail.Required, field) {
			t.Errorf("ModuleDetailRead.%s is not required", field)
		}
	}
	paths := []string{
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/",
	}
	for _, path := range paths {
		operation := document.Paths[path]["get"]
		if _, exists := operation.Responses["404"]; !exists {
			t.Errorf("GET %s omits not-found behavior", path)
		}
		if got := responseSchemaRef(operation.Responses["200"]); got != "#/components/schemas/ModuleDetailRead" {
			t.Errorf("GET %s response schema = %q", path, got)
		}
	}
	archivedDetail := document.Paths["/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/"]["get"]
	if _, exists := archivedDetail.Responses["404"]; exists {
		t.Error("archived-module detail must preserve Django's sparse 200 for a missing row")
	}
	if got := responseSchemaRef(archivedDetail.Responses["200"]); got != "" {
		t.Errorf("archived-module detail must allow full and sparse success objects, got schema %q", got)
	}
}

func TestModuleIssueListUsesPaginatorEnvelope(t *testing.T) {
	document := readOpenAPI(t)
	page := document.Components.Schemas["ModuleIssuePage"]
	for _, field := range []string{
		"grouped_by", "sub_grouped_by", "total_count", "next_cursor", "prev_cursor",
		"next_page_results", "prev_page_results", "count", "total_pages", "total_results",
		"extra_stats", "results",
	} {
		if _, exists := page.Properties[field]; !exists {
			t.Errorf("ModuleIssuePage.%s is missing", field)
		}
		if !slices.Contains(page.Required, field) {
			t.Errorf("ModuleIssuePage.%s is not required", field)
		}
	}
	operation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/"]["get"]
	if got := responseSchemaRef(operation.Responses["200"]); got != "#/components/schemas/ModuleIssuePage" {
		t.Errorf("issue-list response schema = %q", got)
	}
	if _, exists := operation.Responses["400"]; !exists {
		t.Error("issue-list contract omits invalid cursor/grouping response")
	}
}

func TestModuleIssueListBadRequestRepresentsEverySourceEnvelope(t *testing.T) {
	document := readOpenAPI(t)
	path := "/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/"
	operation := document.Paths[path]["get"]
	if got := responseSchemaRef(operation.Responses["400"]); got != "#/components/schemas/ModuleIssueListBadRequest" {
		t.Fatalf("issue-list 400 schema = %q", got)
	}
	badRequest := document.Components.Schemas["ModuleIssueListBadRequest"]
	for _, field := range []string{"error", "detail", "message", "code", "errors"} {
		if _, exists := badRequest.Properties[field]; !exists {
			t.Errorf("module issue bad request omits %s", field)
		}
		if slices.Contains(badRequest.Required, field) {
			t.Errorf("module issue bad request unexpectedly requires %s", field)
		}
	}
}

func TestModuleIssueDetailReturnsRelationshipAndUsesIssueID(t *testing.T) {
	document := readOpenAPI(t)
	path := "/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/issues/{issue_id}/"
	operation := document.Paths[path]["get"]
	if !slices.Contains(parameterNames(document, operation.Parameters), "issue_id") {
		t.Fatal("module-issue detail does not bind issue_id")
	}
	componentName, relationship := responseComponentSchema(t, document, operation.Responses["200"])
	if componentName != "ModuleIssueRead" {
		t.Fatalf("module-issue detail response schema = %q, want ModuleIssueRead", componentName)
	}
	for _, field := range []string{"id", "module", "issue", "module_detail", "issue_detail", "sub_issues_count"} {
		if _, exists := relationship.Properties[field]; !exists {
			t.Errorf("ModuleIssueRead.%s is missing", field)
		}
		if !slices.Contains(relationship.Required, field) {
			t.Errorf("ModuleIssueRead.%s is not required", field)
		}
	}
	if _, exists := operation.Responses["404"]; !exists {
		t.Error("module-issue detail omits not-found response")
	}
}

func TestArchivedListAndLinkSchemasPreserveDistinctContainers(t *testing.T) {
	document := readOpenAPI(t)
	archived := document.Components.Schemas["ArchivedModuleRead"]
	for _, field := range []string{"archived_at", "member_ids", "total_issues", "is_favorite"} {
		if _, exists := archived.Properties[field]; !exists {
			t.Errorf("ArchivedModuleRead.%s is missing", field)
		}
	}
	for _, omitted := range []string{"logo_props", "total_estimate_points", "completed_estimate_points"} {
		if _, exists := archived.Properties[omitted]; exists {
			t.Errorf("ArchivedModuleRead exposes source-omitted %s", omitted)
		}
	}
	workspace := document.Components.Schemas["WorkspaceModuleRead"]
	for _, omitted := range []string{"is_favorite", "total_estimate_points", "completed_estimate_points"} {
		if _, exists := workspace.Properties[omitted]; exists {
			t.Errorf("WorkspaceModuleRead exposes source-omitted %s", omitted)
		}
	}
	link := document.Components.Schemas["ModuleLinkRead"]
	for _, field := range []string{"id", "workspace", "project", "module", "title", "url", "metadata"} {
		if _, exists := link.Properties[field]; !exists {
			t.Errorf("ModuleLinkRead.%s is missing", field)
		}
	}
	listOperation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/"]["get"]
	detailOperation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/"]["get"]
	if responseSchemaRef(listOperation.Responses["200"]) == responseSchemaRef(detailOperation.Responses["200"]) {
		t.Error("module-link list and detail use the same response container")
	}
}

func responseSchemaRef(response json.RawMessage) string {
	direct, _ := responseSchemaRefs(response)
	return direct
}

func responseSchemaRefs(response json.RawMessage) (direct string, item string) {
	var envelope struct {
		Content map[string]struct {
			Schema struct {
				Ref   string `json:"$ref"`
				Items *struct {
					Ref string `json:"$ref"`
				} `json:"items"`
			} `json:"schema"`
		} `json:"content"`
	}
	if json.Unmarshal(response, &envelope) != nil {
		return "", ""
	}
	schema := envelope.Content["application/json"].Schema
	if schema.Items == nil {
		return schema.Ref, ""
	}
	return schema.Ref, schema.Items.Ref
}

func responseComponentSchema(
	t *testing.T,
	document openAPIDocument,
	response json.RawMessage,
) (string, openAPISchema) {
	t.Helper()
	directRef, itemRef := responseSchemaRefs(response)
	ref := directRef
	if ref == "" {
		ref = itemRef
	}
	if ref == "" {
		t.Fatal("response has neither a schema reference nor an array item schema reference")
	}
	componentName := filepath.Base(ref)
	schema, exists := document.Components.Schemas[componentName]
	if !exists {
		t.Fatalf("response schema reference %q is missing", ref)
	}
	return componentName, schema
}

func TestModuleAndLinkDetailPreserveDistinctNotFoundEnvelopes(t *testing.T) {
	document := readOpenAPI(t)
	moduleOperation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/"]["get"]
	if got := responseSchemaRef(moduleOperation.Responses["404"]); got != "#/components/schemas/ModuleReadsModuleNotFound" {
		t.Errorf("module detail 404 schema = %q", got)
	}
	moduleError := document.Components.Schemas["ModuleReadsModuleNotFound"]
	if _, exists := moduleError.Properties["error"]; !exists {
		t.Error("module detail 404 omits error")
	}
	linkOperation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/"]["get"]
	if got := responseSchemaRef(linkOperation.Responses["404"]); got != "#/components/schemas/ModuleReadsLinkNotFound" {
		t.Errorf("module link detail 404 schema = %q", got)
	}
	linkError := document.Components.Schemas["ModuleReadsLinkNotFound"]
	if _, exists := linkError.Properties["detail"]; !exists {
		t.Error("module link detail 404 omits detail")
	}
}

func TestModulePermissionClassEndpointsUseDRFPermissionEnvelope(t *testing.T) {
	document := readOpenAPI(t)
	permissionPaths := []string{
		"/api/workspaces/{slug}/modules/",
		"/api/workspaces/{slug}/projects/{project_id}/archived-modules/",
		"/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/",
		"/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/",
	}
	for _, path := range permissionPaths {
		operation := document.Paths[path]["get"]
		if got := responseSchemaRef(operation.Responses["403"]); got != "#/components/schemas/ModuleReadsPermissionDenied" {
			t.Errorf("GET %s 403 schema = %q", path, got)
		}
	}
	permissionError := document.Components.Schemas["ModuleReadsPermissionDenied"]
	if _, exists := permissionError.Properties["detail"]; !exists {
		t.Error("permission-class 403 omits detail")
	}

	decoratorOperation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/modules/"]["get"]
	if got := responseSchemaRef(decoratorOperation.Responses["403"]); got != "#/components/schemas/ModuleReadsForbidden" {
		t.Errorf("decorator-backed list 403 schema = %q", got)
	}
}
