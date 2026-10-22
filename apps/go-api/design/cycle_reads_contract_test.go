package design

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestCycleReadsExposeExactlyTenSessionProtectedOperations(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	paths := []string{
		"/api/workspaces/{slug}/cycles/",
		"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/",
		"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/",
	}
	for _, path := range paths {
		operation, ok := document.Paths[path]["get"]
		if !ok {
			t.Errorf("generated cycle surface missing GET %s", path)
			continue
		}
		if len(operation.Security) != 1 || len(operation.Security[0]["plane_session_cookie"]) != 0 {
			t.Errorf("GET %s security=%#v, want Django session cookie", path, operation.Security)
		}
		for _, status := range []string{"200", "401", "403", "500"} {
			if _, ok := operation.Responses[status]; !ok {
				t.Errorf("GET %s missing response %s", path, status)
			}
		}
	}
}

func TestCycleReadsOwnPermissionAwareOptionsOperations(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	for path, statuses := range map[string][]string{
		"/api/workspaces/{slug}/cycles/":                                            {"200", "401", "403", "500"},
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/": {"200", "401"},
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/":  {"200", "401"},
	} {
		operation, ok := document.Paths[path]["options"]
		if !ok {
			t.Errorf("generated cycle surface missing OPTIONS %s", path)
			continue
		}
		if len(operation.Security) != 1 || len(operation.Security[0]["plane_session_cookie"]) != 0 {
			t.Errorf("OPTIONS %s security=%#v, want Django session cookie", path, operation.Security)
		}
		for _, status := range statuses {
			if _, ok := operation.Responses[status]; !ok {
				t.Errorf("OPTIONS %s missing response %s", path, status)
			}
		}
		if schema := responseSchemaName(t, operation.Responses["200"]); schema != "CycleReadsMetadata" {
			t.Errorf("OPTIONS %s response schema=%q", path, schema)
		}
		if path != "/api/workspaces/{slug}/cycles/" {
			uuidParameters := map[string]bool{"project_id": false, "cycle_id": false}
			for _, parameter := range resolvedPageParameters(t, document, operation.Parameters) {
				if _, tracked := uuidParameters[parameter.Name]; !tracked {
					continue
				}
				uuidParameters[parameter.Name] = parameter.Schema.Resolved().Type.Value == "string" &&
					parameter.Schema.Resolved().Format == "uuid"
			}
			for name, valid := range uuidParameters {
				if !valid {
					t.Errorf("OPTIONS %s parameter %s is not a UUID string", path, name)
				}
			}
		}
	}
}

func TestCycleReadsContractIncludesRealFiltersAndRepairedIdentifiers(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	list := document.Paths["/api/workspaces/{slug}/projects/{project_id}/cycles/"]["get"]
	if names := parameterNames(document, list.Parameters); !slices.Contains(names, "cycle_view") {
		t.Fatalf("cycle list parameters=%v, want cycle_view", names)
	}
	analytics := document.Paths["/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/"]["get"]
	if names := parameterNames(document, analytics.Parameters); !slices.Contains(names, "type") {
		t.Fatalf("analytics parameters=%v, want type", names)
	}
	archive := document.Paths["/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/"]["get"]
	if names := parameterNames(document, archive.Parameters); !slices.Contains(names, "cycle_id") || slices.Contains(names, "pk") {
		t.Fatalf("archive parameters=%v, want cycle_id and no pk", names)
	}
	issueDetail := document.Paths["/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/"]["get"]
	if names := parameterNames(document, issueDetail.Parameters); !slices.Contains(names, "issue_id") || slices.Contains(names, "pk") {
		t.Fatalf("cycle-issue detail parameters=%v, want issue_id and no pk", names)
	}
	issues := document.Paths["/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/"]["get"]
	names := parameterNames(document, issues.Parameters)
	for _, parameter := range []string{
		"order_by", "group_by", "sub_group_by", "cursor", "per_page",
		"state", "state_group", "estimate_point", "priority", "assignees", "labels", "created_by", "logged_by", "parent",
		"mentions", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "project", "cycle",
		"module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date", "type", "filters",
	} {
		if !slices.Contains(names, parameter) {
			t.Errorf("cycle-issue list parameters missing %q: %v", parameter, names)
		}
	}
	for _, parameter := range issues.Parameters {
		if parameter.Ref != "" {
			parameter = document.Components.Parameters[strings.TrimPrefix(parameter.Ref, "#/components/parameters/")]
		}
		if parameter.Name == "per_page" && parameter.Schema.Type.Value != "string" {
			t.Errorf("per_page schema type=%q, want string so the service can preserve Django parse errors", parameter.Schema.Type.Value)
		}
	}
	for _, parameter := range []string{
		"assignee_id", "assignee_id__in", "cycle_id__in", "module_id", "module_id__in", "mention_id",
		"mention_id__in", "label_id", "label_id__in", "created_by_id", "created_by_id__in", "is_archived",
		"state_group__in", "state_id", "state_id__in", "project_id__in", "subscriber_id", "subscriber_id__in",
		"created_at__exact", "created_at__range", "updated_at__exact", "updated_at__range", "start_date__exact",
		"start_date__range", "target_date__exact", "target_date__range", "is_draft", "priority__in",
	} {
		if slices.Contains(names, parameter) {
			t.Errorf("cycle-issue list unexpectedly exposes rich-only flat parameter %q: %v", parameter, names)
		}
	}
}

func TestCycleIssueListUsesPaginatorEnvelopeNotBareCycleIssueArray(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	operation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/"]["get"]
	schemaName := responseSchemaName(t, operation.Responses["200"])
	if schemaName != "CycleIssuePage" {
		t.Fatalf("cycle-issue list response schema=%q, want CycleIssuePage", schemaName)
	}
	page := document.Components.Schemas[schemaName]
	for _, field := range []string{
		"grouped_by", "sub_grouped_by", "total_count", "next_cursor", "prev_cursor",
		"next_page_results", "prev_page_results", "count", "total_pages", "total_results", "extra_stats", "results",
	} {
		if _, ok := page.Properties[field]; !ok {
			t.Errorf("CycleIssuePage missing %q", field)
		}
		if !slices.Contains(page.Required, field) {
			t.Errorf("CycleIssuePage does not require %q", field)
		}
	}
	if page.Type.Value == "array" {
		t.Fatal("cycle-issue list still advertises a bare array")
	}
}

func TestCycleIssueListBadRequestSchemaCoversEveryRuntimeBody(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	operation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/"]["get"]
	names := responseAlternativeSchemaNames(t, operation.Responses["400"])
	if len(names) != 5 {
		t.Fatalf("cycle issue 400 schema alternatives=%v, want all five runtime error types", names)
	}
	for _, name := range names {
		schema := document.Components.Schemas[name]
		for _, field := range []string{"error", "detail", "message", "code", "errors"} {
			if _, ok := schema.Properties[field]; !ok {
				t.Errorf("cycle issue 400 schema %q missing %q", name, field)
			}
		}
		if len(schema.Required) != 0 {
			t.Errorf("cycle issue 400 schema %q required=%v, want optional superset", name, schema.Required)
		}
	}
}

func TestCycleSchemasContainCorrectedResponseFields(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	cycle := document.Components.Schemas["CycleRead"]
	for _, field := range []string{
		"progress_snapshot", "logo_props", "version", "created_by", "assignee_ids", "sub_issues",
		"archived_at", "estimate_distribution", "distribution",
	} {
		if _, ok := cycle.Properties[field]; !ok {
			t.Errorf("CycleRead missing corrected field %q", field)
		}
	}
	if progress := cycle.Properties["progress_snapshot"].Resolved(); progress.Type.Value == "number" || progress.Type.Value == "integer" {
		t.Fatalf("progress_snapshot=%#v, must preserve JSON rather than generated float", progress)
	}
	issue := document.Components.Schemas["CycleIssueRead"]
	for _, field := range []string{"issue", "cycle", "issue_detail", "sub_issues_count"} {
		if _, ok := issue.Properties[field]; !ok {
			t.Errorf("CycleIssueRead missing %q", field)
		}
	}
	progress := document.Components.Schemas["CycleProgress"]
	for _, field := range []string{
		"backlog_estimate_points", "unstarted_estimate_points", "started_estimate_points",
		"cancelled_estimate_points", "completed_estimate_points", "total_estimate_points",
		"backlog_issues", "unstarted_issues", "started_issues", "cancelled_issues", "completed_issues", "total_issues",
	} {
		if _, ok := progress.Properties[field]; !ok {
			t.Errorf("CycleProgress missing %q", field)
		}
	}
}

func TestWorkspaceCycleForbiddenUsesDRFDetailEnvelope(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	operation := document.Paths["/api/workspaces/{slug}/cycles/"]["get"]
	schemaName := responseSchemaName(t, operation.Responses["403"])
	if schemaName != "CycleReadsWorkspaceForbidden" {
		t.Fatalf("workspace cycle 403 schema=%q", schemaName)
	}
	schema := document.Components.Schemas[schemaName]
	if _, ok := schema.Properties["detail"]; !ok {
		t.Error("workspace cycle 403 missing detail")
	}
	if _, ok := schema.Properties["error"]; ok {
		t.Error("workspace cycle 403 incorrectly uses project permission envelope")
	}
}

func TestCycleAnalyticsResponseUsesUnconstrainedItems(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	operation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/"]["get"]
	if schemaName := responseSchemaName(t, operation.Responses["200"]); schemaName != "CycleAnalytics" {
		t.Fatalf("analytics response schema=%q, want CycleAnalytics", schemaName)
	}

	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var rawDocument struct {
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &rawDocument); err != nil {
		t.Fatal(err)
	}
	var analytics struct {
		Properties map[string]struct {
			Type  string                     `json:"type"`
			Items map[string]json.RawMessage `json:"items"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(rawDocument.Components.Schemas["CycleAnalytics"], &analytics); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"assignees", "labels"} {
		property := analytics.Properties[field]
		if property.Type != "array" {
			t.Errorf("CycleAnalytics.%s type=%q, want array", field, property.Type)
		}
		for _, constraint := range []string{"$ref", "type", "format", "properties", "allOf", "anyOf", "oneOf"} {
			if _, exists := property.Items[constraint]; exists {
				t.Errorf("CycleAnalytics.%s items unexpectedly constrain %s", field, constraint)
			}
		}
	}
}

func TestCycleDetailAndAggregateFailuresAreDocumented(t *testing.T) {
	document := readOpenAPI(t, "../gen/http/openapi.json")
	for _, path := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/",
	} {
		if _, ok := document.Paths[path]["get"].Responses["404"]; !ok {
			t.Errorf("GET %s missing repaired not-found response", path)
		}
	}
	for _, path := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/",
		"/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/",
	} {
		if _, ok := document.Paths[path]["get"].Responses["400"]; !ok {
			t.Errorf("GET %s missing bad-request response", path)
		}
	}
}

func responseSchemaName(t *testing.T, response json.RawMessage) string {
	t.Helper()
	var decoded struct {
		Content map[string]struct {
			Schema struct {
				Ref string `json:"$ref"`
			} `json:"schema"`
		} `json:"content"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	ref := decoded.Content["application/json"].Schema.Ref
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	if name == "" || name == ref {
		t.Fatalf("response schema ref=%q", ref)
	}
	return name
}
