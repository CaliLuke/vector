//go:build integration

package workitemtypes_test

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

func TestCustomPropertyFiltersRunBeforePaginationAndPreserveUnknown(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	effort := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,CASE WHEN id=$3 THEN 2 ELSE 9 END),created_at=CASE WHEN id=$3 THEN '2026-01-01'::timestamptz ELSE '2026-02-01'::timestamptz END WHERE type_id=$1`, f.task, effort, ids[0])
	key := "customproperty_" + effort + "__gte"
	filter := fmt.Sprintf(`{"%s":8}`, key)
	for _, op := range []string{"api_workspaces_by_slug_projects_by_project_id_issues", "api_workspaces_by_slug_projects_by_project_id_v2_issues", "api_workspaces_by_slug_issues"} {
		page := readPropertyPage(t, f, repo, op, f.owner, map[string]string{"filters": filter, "per_page": "1", "order_by": "created_at"}, false)
		if page.Total != 1 || len(page.Results) != 1 || page.Results[0].ID != ids[1] {
			t.Fatalf("filtered page: %+v", page)
		}
		if page.Results[0].Values != nil {
			t.Fatal("filtering enabled unrequested value projection")
		}
	}
	negative := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.owner, map[string]string{"filters": `{"not":` + filter + `}`, "per_page": "10"}, false)
	if negative.Total != 1 || negative.Results[0].ID != ids[0] {
		t.Fatalf("negation included unattached bug property: %+v", negative)
	}
}

func TestCustomPropertyFormulaFilterAndMixedLegacyDisjunction(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	effort, formulaID := uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: formulaID, Title: "Double", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} * 2", Result: "Number"}},
	)
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,4) WHERE id=$1`, ids[0], effort)
	f.exec(`UPDATE issues SET priority='urgent' WHERE id=$1`, ids[2])
	filter := fmt.Sprintf(`{"or":[{"customproperty_%s__exact":8},{"priority":"urgent"}]}`, formulaID)
	page := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.owner, map[string]string{"filters": filter, "per_page": "10", "include_properties": "true"}, false)
	if page.Total != 2 || len(page.Results) != 2 {
		t.Fatalf("mixed expression lost valid branch: %+v", page)
	}
	found := false
	for _, row := range page.Results {
		if row.ID == ids[0] {
			found = true
			if row.Values[formulaID].Value != float64(8) {
				t.Fatalf("formula/filter projection disagreement: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("formula match missing")
	}
}
