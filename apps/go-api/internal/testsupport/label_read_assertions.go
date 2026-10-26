//go:build integration

package testsupport

import (
	"encoding/json"
	"slices"
	"sort"
	"testing"
)

func LabelReadObject(t testing.TB, raw []byte) map[string]any {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	if object == nil {
		t.Fatal("expected a current issue object")
	}
	return object
}
func RequireLabelIDs(t testing.TB, value any, want ...string) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "null" {
		t.Fatal("label IDs must be an explicit array")
	}
	var actual []string
	if err = json.Unmarshal(raw, &actual); err != nil {
		t.Fatal(err)
	}
	expected := append([]string(nil), want...)
	sort.Strings(actual)
	sort.Strings(expected)
	if !slices.Equal(actual, expected) {
		t.Fatalf("label IDs=%v want=%v", actual, expected)
	}
}
