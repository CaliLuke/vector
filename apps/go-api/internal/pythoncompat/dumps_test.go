package pythoncompat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDumpsRetainsWorkerIterationOrderAndLastDuplicateValue(t *testing.T) {
	raw := `{"z":1,"nested":{"b":2,"a":"hé😀"},"z":9007199254740993,"a":-0,"float":1e-7}`
	var value map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	got, err := Dumps(value, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	// CPython json.dumps(json.loads(raw)): dictionaries preserve insertion
	// order; repeated keys overwrite without moving; ASCII output uses pairs.
	want := `{"z": 9007199254740993, "nested": {"b": 2, "a": "h\u00e9\ud83d\ude00"}, "a": 0, "float": 1e-07}`
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	delete(value, "z")
	value["state_id"] = "new"
	got, err = Dumps(value, []byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, `{"nested":`) || !strings.HasSuffix(got, `, "state_id": "new"}`) {
		t.Fatalf("mutation changed surviving key order: %s", got)
	}
}
