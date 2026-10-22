package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

func TestPublicVoteDisabledGuardsPrecedeBodyAndTargetAccess(t *testing.T) {
	repository := &Repository{}
	scope := PublicFeedbackScope{VotesEnabled: false}
	_, err := repository.PublicVoteCreate(context.Background(), scope, "not-a-uuid", "actor", strings.NewReader("{"), "application/json")
	var createError *djangoapi.HTTPError
	if !errors.As(err, &createError) || createError.Status != 403 || createError.Body["error"] != publicVoteDisabled {
		t.Fatalf("create err=%v", err)
	}
	_, err = repository.PublicVoteDelete(context.Background(), scope, "not-a-uuid", "actor")
	var deleteError *djangoapi.HTTPError
	if !errors.As(err, &deleteError) || deleteError.Status != 403 || deleteError.Body["error"] != publicVoteDisabled {
		t.Fatalf("delete err=%v", err)
	}
	result, err := repository.PublicVoteList(context.Background(), scope, "not-a-uuid")
	if err != nil || result.Status != 200 || len(result.Body.([]any)) != 0 {
		t.Fatalf("list result=%+v err=%v", result, err)
	}
}

func TestPublicVoteValueUsesDjangoIntegerCoercion(t *testing.T) {
	for input, expected := range map[any]int64{
		json.Number("1"):   1,
		json.Number("1.0"): 1,
		json.Number("0"):   0,
		int64(1):           1,
		int(0):             0,
		true:               1,
		false:              0,
		"-1":               -1,
		" 1 ":              1,
		"١":                1,
	} {
		got, ok := publicVoteValue(input)
		if !ok || got != expected {
			t.Errorf("publicVoteValue(%v)=(%d,%v), want (%d,true)", input, got, ok, expected)
		}
	}
	if _, ok := publicVoteValue([]any{1}); ok {
		t.Fatal("array vote unexpectedly parsed as an integer")
	}
}

func TestPublicFeedbackJSONObjectRootErrorsMatchDRFTypes(t *testing.T) {
	cases := []struct {
		body string
		kind string
	}{
		{body: "null", kind: "No data provided"},
		{body: "[]", kind: "Invalid data. Expected a dictionary, but got list."},
		{body: `"value"`, kind: "Invalid data. Expected a dictionary, but got str."},
		{body: "1", kind: "Invalid data. Expected a dictionary, but got int."},
		{body: "1.5", kind: "Invalid data. Expected a dictionary, but got float."},
		{body: "true", kind: "Invalid data. Expected a dictionary, but got bool."},
	}
	for _, test := range cases {
		t.Run(test.body, func(t *testing.T) {
			input, err := djangoinput.Parse(strings.NewReader(test.body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			err = publicFeedbackObjectRootError(input)
			var response *djangoapi.HTTPError
			if !errors.As(err, &response) || response.Status != 400 || response.Body["non_field_errors"] == nil {
				t.Fatalf("root validation error=%v", err)
			}
			messages, ok := response.Body["non_field_errors"].([]string)
			if !ok || len(messages) != 1 || messages[0] != test.kind {
				t.Fatalf("non_field_errors=%#v, want [%q]", response.Body["non_field_errors"], test.kind)
			}
		})
	}
	object, err := djangoinput.Parse(strings.NewReader(`{"issue":{}}`), "application/json")
	if err != nil || publicFeedbackObjectRootError(object) != nil {
		t.Fatalf("object root validation error=%v", err)
	}
}

func TestPublicFeedbackNestedIssueShapeErrorIsStable(t *testing.T) {
	err := publicFeedbackNestedIssueError()
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 400 {
		t.Fatalf("nested issue error=%v", err)
	}
	messages, ok := response.Body["issue"].([]string)
	if !ok || len(messages) != 1 || messages[0] != "Expected an object." {
		t.Fatalf("issue error=%#v", response.Body["issue"])
	}
}

func TestPublicIntakeQueryOptionsIgnoreUnsupportedFilters(t *testing.T) {
	params := map[string]string{"filters": `{"state__group":"started"}`, "order_by": "name"}
	options := issuequeryOptions(params, map[string]bool{"filters": true, "order_by": true})
	if options.Filters != "" || options.OrderBy != "name" {
		t.Fatalf("intake options=%#v, want ignored filters and retained order_by", options)
	}
	if params["filters"] == "" {
		t.Fatal("query option construction mutated caller parameters")
	}
}

func TestPublicIssueGroupingUsesReturnedCreatorAndDateValues(t *testing.T) {
	repository := &Repository{}
	rows := []map[string]any{
		{"created_by_id": "creator-a", "start_date": "2026-03-04", "target_date": "2026-03-10"},
		{"created_by_id": "creator-b", "start_date": "2026-03-04", "target_date": nil},
		{"created_by_id": "creator-a", "start_date": nil, "target_date": "2026-03-10"},
	}
	for field, want := range map[string][]any{
		"created_by":  {"creator-a", "creator-b"},
		"start_date":  {"2026-03-04", nil},
		"target_date": {"2026-03-10", nil},
	} {
		got, err := repository.groupUniverse(context.Background(), Request{}, field, rows)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("groupUniverse(%q)=%#v, %v; want %#v", field, got, err, want)
		}
	}
}

func TestPublicIntakeCreateProjectionOmitsSubIssueCount(t *testing.T) {
	issue := map[string]any{"id": "issue", "sub_issues_count": 0}
	projection := publicIntakeCreateProjection(issue)
	if projection["id"] != "issue" {
		t.Fatalf("projection=%#v", projection)
	}
	if _, exists := projection["sub_issues_count"]; exists {
		t.Fatalf("create projection contains sub_issues_count: %#v", projection)
	}
}

func TestPublicIntakeSerializerProjectionsMatchEachOperation(t *testing.T) {
	readIssue := map[string]any{
		"project_id": "project", "workspace_id": "workspace", "state_id": "state", "parent_id": "parent",
		"type_id": "type", "assignee_ids": []any{"assignee"}, "label_ids": []any{"label"},
		"sub_issues_count": 2, "assignee_details": []any{}, "label_details": []any{}, "issue_intake": []any{},
	}
	list := publicIntakeReadProjection(maps.Clone(readIssue), true)
	if list["sub_issues_count"] != 2 {
		t.Fatalf("collection projection omitted sub_issues_count: %#v", list)
	}
	for _, field := range []string{"project_id", "workspace_id", "state_id", "parent_id", "type_id", "assignee_ids", "label_ids"} {
		if _, exists := list[field]; exists {
			t.Fatalf("read projection contains internal relation field %q: %#v", field, list)
		}
	}
	detail := publicIntakeReadProjection(maps.Clone(readIssue), false)
	if _, exists := detail["sub_issues_count"]; exists {
		t.Fatalf("detail projection contains sub_issues_count: %#v", detail)
	}
	create := publicIntakeCreateProjection(maps.Clone(readIssue))
	if _, exists := create["sub_issues_count"]; exists {
		t.Fatalf("create projection contains sub_issues_count: %#v", create)
	}
	patch := publicIntakePatchProjection(maps.Clone(readIssue))
	for _, field := range []string{"project_id", "workspace_id", "state_id", "parent_id", "assignee_ids", "label_ids"} {
		if _, exists := patch[field]; !exists {
			t.Fatalf("PATCH projection omitted relation field %q: %#v", field, patch)
		}
	}
	for _, field := range []string{"sub_issues_count", "assignee_details", "label_details", "issue_intake", "project_detail", "state_detail"} {
		if _, exists := patch[field]; exists {
			t.Fatalf("PATCH projection contains read-only field %q: %#v", field, patch)
		}
	}
}

func TestPublicIntakeBindingErrorMatchesDjangoNotFound(t *testing.T) {
	configured := "configured-intake"
	err := (&Repository{}).validatePublicIntake(context.Background(), PublicFeedbackScope{IntakeID: &configured}, "other-intake")
	var response *djangoapi.HTTPError
	if !errors.As(err, &response) || response.Status != 404 || response.Body["error"] != "Not found" {
		t.Fatalf("binding error=%v, want Django 404 Not found", err)
	}
}

func TestPublicIssueListSameGroupingFieldsUseSameParametersError(t *testing.T) {
	result, err := (&Repository{}).PublicIssueList(
		context.Background(),
		PublicFeedbackScope{WorkspaceSlug: "demo", ProjectID: "project"},
		"",
		map[string]string{"group_by": "state", "sub_group_by": "state"},
		map[string]bool{"group_by": true, "sub_group_by": true},
	)
	var listError *djangoapi.HTTPError
	if !errors.As(err, &listError) || listError.Status != 400 || listError.Body["error"] != "Group by and sub group by cannot have same parameters" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestPublicIssueListUnsupportedGroupingFieldUsesDetailError(t *testing.T) {
	_, err := (&Repository{}).PublicIssueList(
		context.Background(),
		PublicFeedbackScope{WorkspaceSlug: "demo", ProjectID: "project"},
		"",
		map[string]string{"group_by": "unsupported"},
		map[string]bool{"group_by": true},
	)
	var listError *djangoapi.HTTPError
	if !errors.As(err, &listError) || listError.Status != 400 || listError.Body["detail"] != "Invalid group_by field: unsupported" {
		t.Fatalf("err=%v", err)
	}
}

func TestPublicIssueListGroupedLegacyDateFilterReturnsGroupedEmptyEnvelope(t *testing.T) {
	for _, test := range []struct {
		name      string
		date      string
		subgroup  string
		wantGroup string
	}{
		{name: "week-only grouped", date: "2026-W23;after", wantGroup: "state_id"},
		{name: "subgrouped week-only date", date: "2026-W23;after", subgroup: "priority", wantGroup: "state_id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			params := map[string]string{"start_date": test.date, "group_by": test.wantGroup}
			if test.subgroup != "" {
				params["sub_group_by"] = test.subgroup
			}
			result, err := (&Repository{}).PublicIssueList(
				context.Background(),
				PublicFeedbackScope{WorkspaceSlug: "demo", ProjectID: "project"},
				"",
				params,
				map[string]bool{"start_date": true, "group_by": true, "sub_group_by": test.subgroup != ""},
			)
			if err != nil {
				t.Fatal(err)
			}
			body, ok := result.Body.(map[string]any)
			if !ok || result.Status != 200 || body["grouped_by"] != test.wantGroup || body["count"] != 0 || body["total_results"] != 0 {
				t.Fatalf("result=%#v; want 200 grouped empty envelope", result)
			}
			groups, ok := body["results"].(map[string]any)
			if !ok || len(groups) != 0 {
				t.Fatalf("grouped results=%#v, want empty mapping", body["results"])
			}
			if test.subgroup != "" && body["sub_grouped_by"] != test.subgroup {
				t.Fatalf("sub_grouped_by=%#v, want %q", body["sub_grouped_by"], test.subgroup)
			}
		})
	}
}

func TestPublicIssueListEqualGroupingFieldsPrecedePaginationErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		params map[string]string
	}{
		{name: "invalid per_page", params: map[string]string{"per_page": "bogus"}},
		{name: "invalid cursor", params: map[string]string{"per_page": "10", "cursor": "not-a-cursor"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.params["group_by"] = "state_id"
			test.params["sub_group_by"] = "state_id"
			present := map[string]bool{"group_by": true, "sub_group_by": true}
			for key := range test.params {
				present[key] = true
			}
			_, err := (&Repository{}).PublicIssueList(
				context.Background(),
				PublicFeedbackScope{WorkspaceSlug: "demo", ProjectID: "project"},
				"",
				test.params,
				present,
			)
			var listError *djangoapi.HTTPError
			if !errors.As(err, &listError) || listError.Status != 400 || listError.Body["error"] != "Group by and sub group by cannot have same parameters" {
				t.Fatalf("err=%v; want same-group 400 error", err)
			}
		})
	}
}

func TestPublicIssueActivitySnapshotKeepsPythonFloatSpelling(t *testing.T) {
	value, err := orderedDump(publicIssueActivitySnapshot(map[string]any{
		"id":         "issue",
		"name":       "Issue",
		"sort_order": json.Number("65535"),
	}), publicIssueActivityOrder)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(value, `"sort_order": 65535.0`) {
		t.Fatalf("activity snapshot=%s, want source float spelling", value)
	}
}
