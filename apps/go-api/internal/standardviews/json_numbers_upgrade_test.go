package standardviews

import (
	"encoding/json"
	"testing"
)

func TestDerivedFiltersPreserveNormalizedJSONNumbers(t *testing.T) {
	got, err := normalizeFilters(json.RawMessage(`{"state":[9007199254740993],"project":[9007199254740992.0]}`), "POST")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(got, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["state__in"]) != "[9007199254740993]" || string(fields["project__in"]) != "[9007199254740992.0]" {
		t.Fatalf("derived filters changed numeric values: %s", got)
	}
}
