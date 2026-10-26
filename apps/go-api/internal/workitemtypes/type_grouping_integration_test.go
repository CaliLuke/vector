//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/google/uuid"
)

func discoveryGroupedRead(f projectTypeFixture, repo *workitems.Repository, project, actor string, params map[string]string) (workitems.Result, error) {
	operation := "api_workspaces_by_slug_projects_by_project_id_issues"
	if project == "" {
		operation = "api_workspaces_by_slug_issues"
	}
	present := map[string]bool{}
	for key := range params {
		present[key] = true
	}
	return repo.Execute(f.ctx, workitems.Request{Operation: operation, Method: "GET", ActorID: actor, Slug: "types", ProjectID: project, Timezone: "UTC", Params: params, Present: present}, nil, "")
}

type discoveryBucket struct {
	Total   int `json:"total_results"`
	Results []struct {
		ID string `json:"id"`
	} `json:"results"`
}

func discoveryBuckets(t *testing.T, result workitems.Result) map[string]discoveryBucket {
	t.Helper()
	raw, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	var page struct {
		Results map[string]discoveryBucket `json:"results"`
	}
	if err = json.Unmarshal(raw, &page); err != nil {
		t.Fatalf("invalid grouped response: %s: %v", raw, err)
	}
	return page.Results
}

func TestDiscoveryTypeGroupingPagesAndRetainedTypes(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	f.mutate(t, "pause", f.bug)
	f.mutate(t, "remove", f.bug)
	for _, project := range []string{f.project, ""} {
		first, err := discoveryGroupedRead(f, repo, project, f.owner, map[string]string{"group_by": "type_id", "per_page": "1"})
		if err != nil {
			t.Fatal(err)
		}
		groups := discoveryBuckets(t, first)
		if groups[f.task].Total != 2 || len(groups[f.task].Results) != 1 || groups[f.bug].Total != 1 || groups[f.bug].Results[0].ID != ids[2] {
			t.Fatalf("wrong groups/counts: %+v", groups)
		}
		second, err := discoveryGroupedRead(f, repo, project, f.owner, map[string]string{"group_by": "type_id", "per_page": "1", "cursor": "1:1:0"})
		if err != nil {
			t.Fatal(err)
		}
		next := discoveryBuckets(t, second)
		if next[f.task].Total != 2 || len(next[f.task].Results) != 1 || next[f.task].Results[0].ID == groups[f.task].Results[0].ID || len(next[f.bug].Results) != 0 {
			t.Fatalf("wrong next groups: %+v", next)
		}
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	f.exec(`UPDATE issues SET type_id=NULL WHERE id=$1`, ids[2])
	f.exec(`UPDATE work_item_type_settings SET enabled=true WHERE workspace_id=$1`, f.workspace)
	none, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": "type_id"})
	if err != nil {
		t.Fatal(err)
	}
	if got := discoveryBuckets(t, none)["None"]; got.Total != 1 || got.Results[0].ID != ids[2] {
		t.Fatalf("untyped bucket: %+v", got)
	}
}

func TestDiscoveryTypeSubgroupsAndSelectedBucketIntersection(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	result, err := discoveryGroupedRead(f, repo, f.project, f.owner, map[string]string{"group_by": "priority", "sub_group_by": "type_id", "per_page": "1"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	var nested struct {
		Results map[string]struct {
			Results map[string]discoveryBucket `json:"results"`
		} `json:"results"`
	}
	if err = json.Unmarshal(raw, &nested); err != nil {
		t.Fatal(err)
	}
	if got := nested.Results["none"].Results[f.task]; got.Total != 2 || len(got.Results) != 1 {
		t.Fatalf("subgroup count/page: %s", raw)
	}
	workspace, err := discoveryGroupedRead(f, repo, "", f.owner, map[string]string{"group_by": "created_by", "sub_group_by": "type_id", "per_page": "1"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(workspace.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &nested); err != nil {
		t.Fatal(err)
	}
	if got := nested.Results[f.owner].Results[f.task]; got.Total != 2 || len(got.Results) != 1 {
		t.Fatalf("workspace creator subgroup lost: %s", raw)
	}

	params := map[string]string{"group_filter_by": "type_id", "group_filter_value": f.task, "filters": `{"or":[{"type_id__exact":"` + f.task + `"},{"type_id__exact":"` + f.bug + `"}]}`, "per_page": "1", "cursor": "1:1:0"}
	result, err = discoveryGroupedRead(f, repo, f.project, f.owner, params)
	if err != nil {
		t.Fatal(err)
	}
	assertDiscoveryPage(t, result, 2)
	params["filters"] = `{"not":{"type_id__exact":"` + f.task + `"}}`
	result, err = discoveryGroupedRead(f, repo, f.project, f.owner, params)
	if err != nil {
		t.Fatal(err)
	}
	body := result.Body.(map[string]any)
	if body["total_results"] != 0 {
		t.Fatalf("selector overwrote expression: %+v", body)
	}
}

func TestDiscoveryTypeGroupingRejectsInvalidAndUnauthorizedRequests(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _ := seedDiscoveryItems(t, f)
	expect := func(actor string, params map[string]string, status int) {
		t.Helper()
		_, err := discoveryGroupedRead(f, repo, f.project, actor, params)
		var failure *djangoapi.HTTPError
		if !errors.As(err, &failure) || failure.Status != status {
			t.Fatalf("want %d, got %v", status, err)
		}
	}
	expect(f.outsider, map[string]string{"group_by": "type_id"}, 403)
	expect(f.owner, map[string]string{"group_by": "type_id", "sub_group_by": "type_id"}, 400)
	expect(f.owner, map[string]string{"group_filter_by": "type_id", "group_filter_value": uuid.NewString()}, 400)
	expect(f.owner, map[string]string{"group_filter_value": f.task}, 400)
	expect(f.owner, map[string]string{"group_filter_by": "type_id"}, 400)
	f.exec(`UPDATE projects SET guest_view_all_features=false WHERE id=$1`, f.project)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	guest, err := discoveryGroupedRead(f, repo, f.project, f.member, map[string]string{"group_by": "type_id"})
	if err != nil {
		t.Fatal(err)
	}
	if len(discoveryBuckets(t, guest)) != 0 {
		t.Fatal("guest saw another creator's groups")
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	expect(f.owner, map[string]string{"group_by": "type_id"}, 400)
	expect(f.owner, map[string]string{"group_by": "type_id", "include_properties": "true"}, 400)
	expect(f.owner, map[string]string{"group_filter_by": "type_id", "group_filter_value": f.task}, 400)
}
