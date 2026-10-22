package standardviews

import (
	"encoding/json"
	"testing"
)

func TestNormalizeFiltersUsesIssueFilterNames(t *testing.T) {
	got, err := normalizeFilters(json.RawMessage(`{"state":"state-id","state_group":["started"],"priority":["high"],"labels":["None","label-id"],"assignees":["user-id"],"sub_issue":"false","type":"active","unknown":"ignored"}`), "GET")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"state__in":                          "state-id",
		"state__group__in":                   []any{"unstarted", "started"},
		"priority__in":                       []any{"high"},
		"labels__isnull":                     true,
		"labels__in":                         []any{"label-id"},
		"label_issue__deleted_at__isnull":    true,
		"issue_assignee__deleted_at__isnull": true,
		"parent__isnull":                     true,
	} {
		if !jsonEquivalent(result[key], want) {
			t.Errorf("%s = %#v, want %#v", key, result[key], want)
		}
	}
	if _, ok := result["unknown"]; ok {
		t.Error("unknown filter was retained")
	}
}

func TestNormalizeFiltersDateAndRelativeForms(t *testing.T) {
	got, err := normalizeFilters(json.RawMessage(`{"created_at":["2024-01-01;after"],"start_target_date":"true"}`), "GET")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatal(err)
	}
	if result["created_at__date__gte"] != "2024-01-01" {
		t.Fatalf("created_at filter = %#v", result["created_at__date__gte"])
	}
	if result["start_date__isnull"] != false || result["target_date__isnull"] != false {
		t.Fatalf("start_target_date = %#v", result)
	}
}

func TestNormalizeFiltersPreservesPostValues(t *testing.T) {
	got, err := normalizeFilters(json.RawMessage(`{"parent":["None"],"labels":["None"],"start_date":["2024-01-01;after"],"sub_issue":false}`), "POST")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatal(err)
	}
	if !jsonEquivalent(result["parent__in"], []any{"None"}) || !jsonEquivalent(result["labels__in"], []any{"None"}) {
		t.Fatalf("POST relation filters = %#v", result)
	}
	if !jsonEquivalent(result["start_date"], []any{"2024-01-01;after"}) {
		t.Fatalf("POST date filter = %#v", result["start_date"])
	}
	if _, ok := result["parent__isnull"]; ok {
		t.Error("POST parent unexpectedly applied GET None handling")
	}
}

func TestNormalizeFiltersEmptyIsEmptyObject(t *testing.T) {
	for _, input := range []json.RawMessage{nil, []byte(`null`), []byte(`{}`)} {
		got, err := normalizeFilters(input, "POST")
		if err != nil || string(got) != `{}` {
			t.Fatalf("normalizeFilters(%s) = %s, %v", input, got, err)
		}
	}
}

func jsonEquivalent(got, want any) bool {
	a, _ := json.Marshal(got)
	b, _ := json.Marshal(want)
	return string(a) == string(b)
}
