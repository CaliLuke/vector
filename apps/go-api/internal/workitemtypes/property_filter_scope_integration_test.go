//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

func TestCustomPropertyCatalogIdentityAndForeignSelections(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	cascade, level, childLevel, root, child, contextID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	options := []property.Option{{ID: root, Name: "Root"}, {ID: child, Name: "Child", Parent: root, Level: 1}}
	attachValueProperties(t, f, property.Definition{ID: cascade, Title: "Location", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: level, Name: "Country"}, {ID: childLevel, Name: "Region"}}, Options: options}, Contexts: []property.Context{{ID: contextID, Name: "Bug catalog", Types: []string{f.bug}, Overrides: []string{"options"}, Configuration: property.Configuration{Options: options}}}})
	f.exec(`INSERT INTO work_item_type_properties(type_id,property_id) VALUES($1,$2)`, f.bug, cascade)
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($1::text,jsonb_build_array($2::text,$3::text))`, cascade, root, child)
	for _, test := range []struct {
		catalog string
		count   int
	}{{"base", 2}, {contextID, 1}} {
		filter := fmt.Sprintf(`{"customproperty_%s_level_%s_catalog_%s__exact":"%s"}`, cascade, level, test.catalog, root)
		page := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.owner, map[string]string{"filters": filter, "per_page": "10"}, false)
		if page.Total != test.count || len(page.Results) != test.count {
			t.Fatalf("catalog domain collapsed: %+v", page)
		}
	}
	for _, filter := range []string{
		fmt.Sprintf(`{"customproperty_%s_level_%s_catalog_%s__exact":"%s"}`, cascade, level, uuid.NewString(), root),
		fmt.Sprintf(`{"customproperty_%s_level_%s_catalog_base__exact":"%s"}`, cascade, childLevel, root),
		fmt.Sprintf(`{"customproperty_%s_level_%s__exact":"%s"}`, cascade, level, root),
	} {
		_, err := typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.project, f.owner, filter, "", false)
		var invalid *djangoapi.HTTPError
		if !errors.As(err, &invalid) || invalid.Status != 400 {
			t.Fatalf("unavailable catalog/level accepted: %v", err)
		}
	}
}

func TestCustomPropertyReferenceFiltersRespectScope(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	member, release, own, foreign, private := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: member, Title: "Reviewer", Kind: property.Member, Configuration: property.Configuration{Active: true}}, property.Definition{ID: release, Title: "Delivery", Kind: property.Release, Configuration: property.Configuration{Active: true}})
	f.addProject(private)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Visible'),($4,$2,$5,'Private')`, own, f.workspace, f.project, foreign, private)
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,$3::text,$4::text,$5::text) WHERE id=$1`, ids[0], member, f.member, release, own)
	filter := fmt.Sprintf(`{"and":[{"customproperty_%s__exact":"%s"},{"customproperty_%s__in":["%s"]}]}`, member, f.member, release, own)
	page := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.owner, map[string]string{"filters": filter, "per_page": "10"}, false)
	if page.Total != 1 || page.Results[0].ID != ids[0] {
		t.Fatalf("authorized references did not match: %+v", page)
	}
	for _, filter := range []string{fmt.Sprintf(`{"customproperty_%s__exact":"%s"}`, member, f.outsider), fmt.Sprintf(`{"customproperty_%s__exact":"%s"}`, release, foreign), fmt.Sprintf(`{"customproperty_%s__exact":4}`, uuid.NewString())} {
		_, err := typeDiscoveryList(f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.project, f.owner, filter, "", false)
		var invalid *djangoapi.HTTPError
		if !errors.As(err, &invalid) || invalid.Status != 400 {
			t.Fatalf("foreign identity accepted: %v", err)
		}
	}
}

func TestCustomPropertyFilterPreservesGuestAndInactiveDomains(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	effort := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}, Contexts: []property.Context{{ID: uuid.NewString(), Name: "Inactive for bugs", Types: []string{f.bug}, Overrides: []string{"active"}, Configuration: property.Configuration{Active: false}}}})
	f.exec(`INSERT INTO work_item_type_properties(type_id,property_id) VALUES($1,$2)`, f.bug, effort)
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($1::text,4)`, effort)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, ids[0], f.member)
	filter := fmt.Sprintf(`{"customproperty_%s__exact":4}`, effort)
	page := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.member, map[string]string{"filters": filter, "per_page": "10"}, false)
	if page.Total != 1 || page.Results[0].ID != ids[0] {
		t.Fatalf("guest scope changed: %+v", page)
	}
	negative := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.owner, map[string]string{"filters": `{"not":` + filter + `}`, "per_page": "10"}, false)
	if negative.Total != 0 {
		t.Fatalf("inactive property matched exclusion: %+v", negative)
	}
}

func TestCustomPropertyDisjunctionCannotBypassGuestOrLiveScope(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	effort := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($1::text,4),priority='urgent'`, effort)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, ids[0], f.member)
	filter := fmt.Sprintf(`{"or":[{"customproperty_%s__exact":4},{"priority":"urgent"}]}`, effort)
	params := map[string]string{"filters": filter, "per_page": "10"}
	page := readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.member, params, false)
	if page.Total != 1 || len(page.Results) != 1 || page.Results[0].ID != ids[0] {
		t.Fatalf("legacy OR branch bypassed guest scope: %+v", page)
	}
	f.exec(`UPDATE issues SET deleted_at=now() WHERE id=$1`, ids[0])
	page = readPropertyPage(t, f, repo, "api_workspaces_by_slug_projects_by_project_id_issues", f.member, params, false)
	if page.Total != 0 || len(page.Results) != 0 {
		t.Fatalf("legacy OR branch bypassed live scope: %+v", page)
	}
}

func TestCustomPropertyFiltersApplyToGroupedAndIDSelectedLists(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	effort := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($1::text,4)`, effort)
	filter := fmt.Sprintf(`{"customproperty_%s__exact":4}`, effort)
	for _, test := range []struct {
		operation string
		params    map[string]string
	}{
		{"api_workspaces_by_slug_projects_by_project_id_issues", map[string]string{"group_by": "priority", "per_page": "1"}},
		{"api_workspaces_by_slug_projects_by_project_id_issues_list", map[string]string{"issues": ids[0] + "," + ids[2]}},
	} {
		test.params["filters"] = filter
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
		if test.params["group_by"] != "" {
			var page struct {
				Total   int `json:"total_results"`
				Results map[string]struct {
					Total   int               `json:"total_results"`
					Results []json.RawMessage `json:"results"`
				} `json:"results"`
			}
			if err = json.Unmarshal(raw, &page); err != nil || page.Total != 2 || len(page.Results["none"].Results) != 1 {
				t.Fatalf("grouped filtering: %s %v", raw, err)
			}
		} else {
			var rows []struct {
				ID string `json:"id"`
			}
			if err = json.Unmarshal(raw, &rows); err != nil || len(rows) != 1 || rows[0].ID != ids[0] {
				t.Fatalf("ID filtering: %s %v", raw, err)
			}
		}
	}
}
