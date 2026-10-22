package design

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
)

func TestIntakeDescriptionBinaryPreservesNullableBinarySchema(t *testing.T) {
	document := readIntakeContract(t)
	version := document.Components.Schemas["IntakeInboxDescriptionVersionDetail"]
	if !slices.Contains(version.Required, "description_binary") {
		t.Fatal("description_binary must remain required even when its value is null")
	}
	var property map[string]any
	if err := json.Unmarshal(version.Properties["description_binary"], &property); err != nil {
		t.Fatal(err)
	}
	// Generated examples are not constraints. The established response schema
	// permits a binary string or null, without imposing base64, contentEncoding,
	// contentMediaType, or extra string validation on the binary branch.
	delete(property, "example")
	want := map[string]any{
		"anyOf": []any{
			map[string]any{"type": "string", "format": "binary"},
			map[string]any{"type": "null"},
		},
	}
	if !reflect.DeepEqual(property, want) {
		t.Fatalf("description_binary schema = %#v, want nullable format=binary string without encoding constraints: %#v", property, want)
	}
}
