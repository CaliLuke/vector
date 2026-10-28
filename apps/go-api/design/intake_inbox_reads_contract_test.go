package design

import (
	"encoding/json"
	"encoding/json/jsontext"
	"os"
	"reflect"
	"slices"
	"testing"
)

type intakeContractDocument struct {
	Paths map[string]map[string]struct {
		Parameters []openAPIParameter    `json:"parameters"`
		Security   []map[string][]string `json:"security"`
		Responses  map[string]struct {
			Content map[string]struct {
				Schema struct {
					Ref  string `json:"$ref"`
					Type string `json:"type"`
				} `json:"schema"`
			} `json:"content"`
		} `json:"responses"`
	} `json:"paths"`
	Components struct {
		Schemas map[string]struct {
			Type       string                    `json:"type"`
			Properties map[string]jsontext.Value `json:"properties"`
			Required   []string                  `json:"required"`
		} `json:"schemas"`
	} `json:"components"`
}

func TestIntakeInboxReadsExposeAllTwelveGETOperations(t *testing.T) {
	document := readIntakeContract(t)
	paths := []string{
		"/api/workspaces/{slug}/draft-issues/",
		"/api/workspaces/{slug}/draft-issues/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/",
		"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/",
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/intake-issues/",
		"/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/",
		"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/",
	}
	for _, path := range paths {
		operation, ok := document.Paths[path]["get"]
		if !ok {
			t.Errorf("generated contract missing GET %s", path)
			continue
		}
		if len(operation.Security) != 1 || operation.Security[0]["plane_session_cookie"] == nil {
			t.Errorf("GET %s security=%v", path, operation.Security)
		}
		for _, status := range []string{"200", "400", "401", "403", "404", "500"} {
			if _, ok := operation.Responses[status]; !ok {
				t.Errorf("GET %s missing response %s", path, status)
			}
		}
	}
	if got := intakeResponseSchemaRef(document.Paths[paths[0]]["get"].Responses["200"]); got != "#/components/schemas/IntakeInboxDraftPage" {
		t.Errorf("draft list response schema=%q, want IntakeInboxDraftPage", got)
	}
}

func TestIntakeCollectionContractsAllowDjangoEmptyObjectWhileDetailsStayModeled(t *testing.T) {
	document := readIntakeContract(t)
	for _, path := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/",
	} {
		if ref := intakeResponseSchemaRef(document.Paths[path]["get"].Responses["200"]); ref != "" {
			t.Errorf("collection %s response schema=%q, want unconstrained empty-object-compatible schema", path, ref)
		}
	}
	for _, path := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/",
		"/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/",
	} {
		if ref := intakeResponseSchemaRef(document.Paths[path]["get"].Responses["200"]); ref != "#/components/schemas/IntakeInbox" {
			t.Errorf("detail %s response schema=%q, want IntakeInbox", path, ref)
		}
	}
}

func TestDescriptionVersionPathsExposeAuthenticatedOptions(t *testing.T) {
	document := readIntakeContract(t)
	for _, path := range []string{
		"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/",
		"/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/",
	} {
		operation, ok := document.Paths[path]["options"]
		if !ok {
			t.Errorf("generated contract missing OPTIONS %s", path)
			continue
		}
		if len(operation.Security) != 1 || operation.Security[0]["plane_session_cookie"] == nil {
			t.Errorf("OPTIONS %s security=%v", path, operation.Security)
		}
		for _, status := range []string{"200", "401"} {
			if _, ok := operation.Responses[status]; !ok {
				t.Errorf("OPTIONS %s missing response %s", path, status)
			}
		}
	}
}

func TestIntakeAndInboxAliasesHaveEquivalentContracts(t *testing.T) {
	document := readIntakeContract(t)
	generated := readOpenAPI(t)
	pairs := [][2]string{
		{"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/", "/api/workspaces/{slug}/projects/{project_id}/intake-issues/"},
		{"/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/", "/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/"},
		{"/api/workspaces/{slug}/projects/{project_id}/inboxes/", "/api/workspaces/{slug}/projects/{project_id}/intakes/"},
		{"/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/", "/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/"},
	}
	for _, pair := range pairs {
		left := document.Paths[pair[0]]["get"]
		right := document.Paths[pair[1]]["get"]
		if !reflect.DeepEqual(intakeResponseSchemaRef(left.Responses["200"]), intakeResponseSchemaRef(right.Responses["200"])) {
			t.Errorf("alias response schemas differ: %s and %s", pair[0], pair[1])
		}
		leftParameters := parameterNames(generated, left.Parameters)
		rightParameters := parameterNames(generated, right.Parameters)
		if !slices.Equal(leftParameters, rightParameters) {
			t.Errorf("alias parameters differ: %s=%v %s=%v", pair[0], leftParameters, pair[1], rightParameters)
		}
	}
}

func TestIntakeInboxReadSchemasCorrectArchivedContractGaps(t *testing.T) {
	document := readIntakeContract(t)
	for _, name := range []string{"IntakeInboxDraftPage", "IntakeInboxIssuePage", "IntakeInboxDescriptionVersionPage"} {
		if schema := document.Components.Schemas[name]; schema.Type != "object" {
			t.Errorf("%s type=%q, want object envelope", name, schema.Type)
		}
	}
	if schema := document.Components.Schemas["IntakeInbox"]; schema.Type != "object" {
		t.Errorf("intake collection result type=%q, want one object", schema.Type)
	}
	detail := document.Components.Schemas["IntakeInboxIssueDetail"]
	for _, field := range []string{"duplicate_issue_detail", "issue", "snoozed_till"} {
		if _, ok := detail.Properties[field]; !ok {
			t.Errorf("intake issue detail missing %q", field)
		}
	}
	if _, ok := detail.Properties["created_by"]; ok {
		t.Error("intake issue detail exposes list-only created_by")
	}
	summary := document.Components.Schemas["IntakeInboxDescriptionVersionSummary"]
	detailVersion := document.Components.Schemas["IntakeInboxDescriptionVersionDetail"]
	for _, field := range []string{"description_binary", "description_html", "description_json", "description_stripped"} {
		if _, ok := summary.Properties[field]; ok {
			t.Errorf("description-version list row exposes detail-only %q", field)
		}
		if _, ok := detailVersion.Properties[field]; !ok {
			t.Errorf("description-version detail missing %q", field)
		}
	}
	listOperation := document.Paths["/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/"]["get"]
	generated := readOpenAPI(t)
	if !slices.Contains(parameterNames(generated, listOperation.Parameters), "cursor") {
		t.Error("description-version list omits cursor")
	}
}

func readIntakeContract(t *testing.T) intakeContractDocument {
	t.Helper()
	data, err := os.ReadFile("../gen/http/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var document intakeContractDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func intakeResponseSchemaRef(response struct {
	Content map[string]struct {
		Schema struct {
			Ref  string `json:"$ref"`
			Type string `json:"type"`
		} `json:"schema"`
	} `json:"content"`
}) string {
	return response.Content["application/json"].Schema.Ref
}
