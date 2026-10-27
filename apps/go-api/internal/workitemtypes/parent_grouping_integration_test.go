//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestParentTypeGroupsIncludeChildrenAndKeepPaginationConjunctive(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	parent := addHierarchyItem(t, f, f.bug, "")
	empty := addHierarchyItem(t, f, f.bug, "")
	f.exec(`UPDATE issues SET parent_id=$2 WHERE id=ANY($1::uuid[])`, ids[:2], parent)
	axis := "parent_type_" + f.bug
	first, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": axis, "sub_issue": "false", "per_page": "1"})
	if err != nil {
		t.Fatal(err)
	}
	groups := discoveryBuckets(t, first)
	if groups[parent].Total != 2 || len(groups[parent].Results) != 1 {
		t.Fatalf("children not grouped: %+v", groups)
	}
	metadata := first.Body.(map[string]any)["parent_groups"].([]map[string]any)
	found := false
	for _, row := range metadata {
		if row["id"] == empty {
			found = true
		}
	}
	if !found {
		t.Fatal("empty parent metadata absent")
	}
	page, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_filter_by": axis, "group_filter_value": parent, "sub_issue": "false", "per_page": "1", "cursor": "1:1:0", "filters": `{"or":[{"type_id__exact":"` + f.task + `"},{"priority__exact":"urgent"}]}`})
	if err != nil {
		t.Fatal(err)
	}
	if next := assertDiscoveryPage(t, page, 2); next == groups[parent].Results[0].ID {
		t.Fatal("repeated first child")
	}
	none, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_filter_by": axis, "group_filter_value": parent, "filters": `{"not":{"type_id__exact":"` + f.task + `"}}`})
	if err != nil {
		t.Fatal(err)
	}
	if none.Body.(map[string]any)["total_results"] != 0 {
		t.Fatal("selector broadened NOT")
	}
	_, err = discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": "priority", "sub_group_by": axis})
	if err == nil {
		t.Fatal("parent subgroup accepted")
	}
	_, err = discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": "parent_type_" + uuid.NewString()})
	if err == nil {
		t.Fatal("unknown parent type accepted")
	}
}

func TestParentTypeGroupsHideInaccessibleCrossProjectParents(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	parent := addHierarchyItem(t, f, f.task, "")
	project := uuid.NewString()
	f.addProject(project)
	f.exec(`UPDATE issues SET project_id=$2,name='Private parent' WHERE id=$1`, parent, project)
	f.exec(`UPDATE issues SET parent_id=$2 WHERE id=$1`, ids[0], parent)
	axis := "parent_type_" + f.task
	read := func() []byte {
		t.Helper()
		result, err := discoveryGroupedRead(f, repo, f.project, f.member, map[string]string{"group_by": axis, "sub_issue": "false"})
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(result.Body)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	raw := read()
	var response struct {
		Parents []struct {
			ID string `json:"id"`
		} `json:"parent_groups"`
		Groups map[string]discoveryBucket `json:"results"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	if _, exists := response.Groups[parent]; exists {
		t.Fatal("hidden parent bucket disclosed")
	}
	for _, item := range response.Parents {
		if item.ID == parent {
			t.Fatal("hidden parent metadata disclosed")
		}
	}
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, project, f.member)
	if err := json.Unmarshal(read(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Groups[parent].Total != 1 || response.Groups[parent].Results[0].ID != ids[0] {
		t.Fatalf("cross-project parent not grouped: %+v", response)
	}
}
