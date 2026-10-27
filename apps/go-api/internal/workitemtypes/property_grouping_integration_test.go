//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

func TestCustomGroupsUseScalarValuesCountsAndIndependentSelectors(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	flag, choice, option := uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: flag, Title: "Ready", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: choice, Title: "Category", Kind: property.Dropdown, Configuration: property.Configuration{Active: true, Options: []property.Option{{ID: option, Name: "Alpha"}}}},
	)
	saved, _ := json.Marshal(map[string]any{flag: false, choice: option})
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=ANY($1::uuid[])`, ids[:2], saved)
	axis := "customproperty_" + flag
	for _, project := range []string{f.project, ""} {
		result, err := discoveryGroupedRead(f, repo, project, f.owner, map[string]string{"group_by": axis, "per_page": "1"})
		if err != nil {
			t.Fatal(err)
		}
		groups := discoveryBuckets(t, result)
		if groups["false"].Total != 2 || len(groups["false"].Results) != 1 || groups["NotApplicable"].Total != 1 {
			t.Fatalf("false/applicability groups: %+v", groups)
		}
		params := map[string]string{"group_filter_by": axis, "group_filter_value": "false", "per_page": "1", "cursor": "1:1:0", "filters": `{"or":[{"type_id__exact":"` + f.task + `"},{"priority__exact":"urgent"}]}`}
		page, err := discoveryGroupedRead(f, repo, project, f.owner, params)
		if err != nil {
			t.Fatal(err)
		}
		second := assertDiscoveryPage(t, page, 2)
		if second == groups["false"].Results[0].ID {
			t.Fatal("group page repeated its first row")
		}
		params["filters"] = `{"not":{"type_id__exact":"` + f.task + `"}}`
		none, err := discoveryGroupedRead(f, repo, project, f.owner, params)
		if err != nil {
			t.Fatal(err)
		}
		body := none.Body.(map[string]any)
		if body["total_results"] != 0 {
			t.Fatalf("selector broadened NOT: %+v", body)
		}
	}
	conflicting, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{
		"group_filter_by": axis, "group_filter_value": "false",
		"sub_group_filter_by": axis, "sub_group_filter_value": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	if body := conflicting.Body.(map[string]any); body["total_results"] != 0 {
		t.Fatalf("conflicting selectors must intersect: %+v", body)
	}
	result, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": "customproperty_" + choice, "sub_group_by": axis, "per_page": "1"})
	if err != nil {
		t.Fatal(err)
	}
	body := result.Body.(map[string]any)
	group := body["results"].(map[string]any)[option].(map[string]any)
	if group["total_results"] != 2 || group["results"].(map[string]any)["false"].(map[string]any)["total_results"] != 2 {
		t.Fatalf("custom subgroup counts: %+v", body)
	}
}

func TestCustomGroupsCoverSingleReferencesCascadeLevelsAndEmptyValues(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	member, release, cascade, level, option, version := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: member, Title: "Owner", Kind: property.Member, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: release, Title: "Release", Kind: property.Release, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: cascade, Title: "Location", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: level, Name: "Country"}, {ID: uuid.NewString(), Name: "Region"}}, Options: []property.Option{{ID: option, Name: "Country one"}}}},
	)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Release one')`, version, f.workspace, f.project)
	saved, _ := json.Marshal(map[string]any{member: f.owner, release: version, cascade: []string{option}})
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[0], saved)
	for _, sample := range []struct{ axis, value string }{
		{"customproperty_" + member, f.owner}, {"customproperty_" + release, version}, {"customproperty_" + cascade + "_level_" + level + "_catalog_base", option},
	} {
		result, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": sample.axis})
		if err != nil {
			t.Fatal(err)
		}
		groups := discoveryBuckets(t, result)
		if groups[sample.value].Total != 1 || groups["None"].Total != 1 || groups["NotApplicable"].Total != 1 {
			t.Fatalf("%s groups: %+v", sample.axis, groups)
		}
	}
}

func TestCustomGroupsRejectUnsupportedAxesAndRespectGuestVisibility(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	flag, multiple, text := uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: flag, Title: "Ready", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: multiple, Title: "Owners", Kind: property.Member, Configuration: property.Configuration{Active: true, Multiple: true}},
		property.Definition{ID: text, Title: "Notes", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}},
	)
	for _, id := range []string{multiple, text, uuid.NewString()} {
		_, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": "customproperty_" + id})
		if err == nil {
			t.Fatalf("unsupported grouping accepted: %s", id)
		}
	}
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, ids[2], f.member)
	result, err := discoveryGroupedRead(f, repo, f.project, f.member, map[string]string{"group_by": "customproperty_" + flag})
	if err != nil {
		t.Fatal(err)
	}
	groups := discoveryBuckets(t, result)
	if len(groups) != 1 || groups["NotApplicable"].Total != 1 {
		t.Fatalf("hidden item count leaked: %+v", groups)
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	_, err = discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": "customproperty_" + flag})
	if err == nil {
		t.Fatal("disabled grouping accepted")
	}
}
