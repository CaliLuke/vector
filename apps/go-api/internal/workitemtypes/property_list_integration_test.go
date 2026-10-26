//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/workitems"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

type propertyListPage struct {
	Total   int `json:"total_results"`
	Results []struct {
		ID          string                          `json:"id"`
		Values      map[string]types.ProjectedValue `json:"property_values"`
		EvaluatedAt time.Time                       `json:"properties_evaluated_at"`
	} `json:"results"`
}

func readPropertyPage(t *testing.T, f projectTypeFixture, repo *workitems.Repository, operation, actor string, params map[string]string, api bool) propertyListPage {
	t.Helper()
	project := f.project
	if !strings.Contains(operation, "_projects_") {
		project = ""
	}
	present := map[string]bool{}
	for key := range params {
		present[key] = true
	}
	result, err := repo.Execute(f.ctx, workitems.Request{Operation: operation, Method: "GET", ActorID: actor, Slug: "types", ProjectID: project, API: api, Timezone: "UTC", Params: params, Present: present}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	var page propertyListPage
	if err = json.Unmarshal(raw, &page); err != nil {
		t.Fatalf("response %s: %v", raw, err)
	}
	return page
}

func TestPropertyListProjectsEffectiveValuesAcrossAuthenticatedRoutes(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	fixed, clock, length := uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: fixed, Name: "instructions", Title: "Instructions", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Configured guidance"}},
		property.Definition{ID: clock, Title: "Clock", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "NOW()", Result: "Date"}},
		property.Definition{ID: length, Title: "Length", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "LEN({{name}})", Result: "Number"}},
	)
	for _, operation := range []string{"api_workspaces_by_slug_projects_by_project_id_issues", "api_workspaces_by_slug_projects_by_project_id_v2_issues", "api_workspaces_by_slug_projects_by_project_id_issues_detail", "api_workspaces_by_slug_user_issues_by_user_id", "api_workspaces_by_slug_issues"} {
		t.Run(operation, func(t *testing.T) {
			page := readPropertyPage(t, f, repo, operation, f.owner, map[string]string{"include_properties": "true", "per_page": "2", "cursor": "2:0:0", "user_id": f.owner, "filters": `{"type_id__exact":"` + f.task + `"}`}, false)
			if page.Total != 2 || len(page.Results) != 2 {
				t.Fatalf("wrong page: %+v", page)
			}
			for _, row := range page.Results {
				if row.Values[fixed].Value != "Configured guidance" || row.Values[length].Value != float64(11) || row.EvaluatedAt.IsZero() {
					t.Fatalf("incorrect projection: %+v", row)
				}
				if row.Values[clock].Value != page.Results[0].Values[clock].Value || row.EvaluatedAt != page.Results[0].EvaluatedAt {
					t.Fatal("response used different formula instants")
				}
			}
		})
	}
}

func TestPropertyListPreservesGuestVisibilityAndUntypedDomain(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	fixed := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: fixed, Title: "Instructions", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Scoped"}})
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, ids[2], f.member)
	page := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.member, map[string]string{"include_properties": "true", "per_page": "10"}, false)
	if page.Total != 1 || len(page.Results) != 1 || page.Results[0].ID != ids[2] || len(page.Results[0].Values) != 0 || page.Results[0].EvaluatedAt.IsZero() {
		t.Fatalf("guest/unattached domain: %+v", page)
	}
}

func TestPropertyListOptInDoesNotChangeLegacyOrAPIResponses(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	for _, test := range []struct {
		operation string
		api       bool
		include   string
	}{
		{"api_workspaces_by_slug_projects_by_project_id_issues", false, "false"},
		{"api_v1_workspaces_by_slug_projects_by_project_id_issues", true, "true"},
	} {
		page := readPropertyPage(t, f, repo, test.operation, f.owner, map[string]string{"include_properties": test.include, "per_page": "10"}, test.api)
		if page.Total != 3 || len(page.Results) != 3 {
			t.Fatalf("wrong legacy page: %+v", page)
		}
		for _, row := range page.Results {
			if row.Values != nil || !row.EvaluatedAt.IsZero() {
				t.Fatal("opt-in leaked into legacy/public response")
			}
		}
	}
}

func TestPropertyListMatchesDetailFormulaAndNumericEstimate(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	effort, calculation, estimate, point := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: calculation, Title: "Calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} + {{estimate_point}}", Result: "Number"}},
	)
	f.exec(`INSERT INTO estimates(id,workspace_id,project_id,name,description,type,last_used,created_at,updated_at) VALUES($1,$2,$3,'Points','','points',true,now(),now())`, estimate, f.workspace, f.project)
	f.exec(`INSERT INTO estimate_points(id,workspace_id,project_id,estimate_id,key,value,description,created_at,updated_at) VALUES($1,$2,$3,$4,1,'2.5','',now(),now())`, point, f.workspace, f.project, estimate)
	f.exec(`UPDATE issues SET priority='urgent',estimate_point_id=$2,custom_properties=jsonb_build_object($3::text,4) WHERE id=$1`, ids[0], point, effort)
	schema, err := f.repo.ItemSchema(f.ctx, f.owner, "types", types.ItemSchemaInput{Mode: "edit", ProjectID: f.project, IssueID: ids[0]})
	if err != nil {
		t.Fatal(err)
	}
	page := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.owner, map[string]string{"include_properties": "true", "per_page": "1", "filters": `{"priority__exact":"urgent"}`}, false)
	if page.Total != 1 || len(page.Results) != 1 {
		t.Fatalf("wrong filtered page: %+v", page)
	}
	values := page.Results[0].Values
	if values[effort].Value != float64(4) || values[calculation].Value != float64(6.5) || values[calculation].Value != schema.Computed[calculation].Value {
		t.Fatalf("list/detail mismatch: %+v / %+v", values, schema.Computed)
	}
}

func TestPropertyListRetainsProjectionInGroupedAndIDResponses(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	fixed := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: fixed, Title: "Instructions", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Visible guidance"}})
	catalog, err := f.repo.DiscoveryProperties(f.ctx, f.owner, "types", f.project)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, operation string
		params          map[string]string
	}{
		{"grouped", "api_workspaces_by_slug_projects_by_project_id_issues", map[string]string{"group_by": "priority", "per_page": "10"}},
		{"by IDs", "api_workspaces_by_slug_projects_by_project_id_issues_list", map[string]string{"issues": ids[0]}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.params["include_properties"] = "true"
			present := map[string]bool{}
			for key := range test.params {
				present[key] = true
			}
			result, err := repo.Execute(f.ctx, workitems.Request{Operation: test.operation, Method: "GET", ActorID: f.owner, Slug: "types", ProjectID: f.project, Timezone: "UTC", Params: test.params, Present: present}, nil, "")
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(result.Body)
			if err != nil {
				t.Fatal(err)
			}
			var rows []struct {
				ID       string                          `json:"id"`
				Revision *int64                          `json:"properties_revision"`
				Values   map[string]types.ProjectedValue `json:"property_values"`
			}
			if test.name == "grouped" {
				var page struct {
					Total   int                        `json:"total_results"`
					Results map[string]json.RawMessage `json:"results"`
				}
				if err = json.Unmarshal(raw, &page); err != nil || page.Total != 3 {
					t.Fatalf("grouped page: %s %v", raw, err)
				}
				var group struct {
					Results json.RawMessage `json:"results"`
				}
				if err = json.Unmarshal(page.Results["none"], &group); err != nil {
					t.Fatal(err)
				}
				raw = group.Results
			}
			if err = json.Unmarshal(raw, &rows); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, row := range rows {
				if row.ID == ids[0] {
					found = true
					if row.Values[fixed].Value != "Visible guidance" || row.Revision == nil || *row.Revision != catalog.Revision {
						t.Fatalf("lost grouped/ID projection: %+v", row)
					}
				}
			}
			if !found {
				t.Fatal("requested item missing from response")
			}
		})
	}
}
