package workitems

import (
	"context"
	"reflect"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

func TestCustomPropertyProjectionRemainsPrivateAndHonorsFieldSelection(t *testing.T) {
	row := map[string]any{"id": "item", "custom_properties": map[string]any{"private": "secret"}}
	repo := &Repository{}
	for _, fields := range []string{"", "id,custom_properties", "custom_properties"} {
		result, err := repo.serializeAuthorizedIssue(context.Background(), row, fields, "")
		if err != nil || !reflect.DeepEqual(result["custom_properties"], row["custom_properties"]) {
			t.Fatalf("authorized projection %q: %#v %v", fields, result, err)
		}
	}
	result, err := repo.serializeAuthorizedIssue(context.Background(), row, "id", "")
	if err != nil || len(result) != 1 || result["id"] != "item" {
		t.Fatalf("field selection ignored: %#v %v", result, err)
	}
	for _, fields := range []string{"", "custom_properties"} {
		public, publicErr := djangoapi.SerializeIssue(context.Background(), nil, row, fields, "")
		if _, exposed := public["custom_properties"]; publicErr != nil || exposed {
			t.Fatalf("public custom property exposed: %#v %v", public, publicErr)
		}
	}
}

func TestSubmittedCustomValuesDistinguishesOmissionClearAndMalformedObject(t *testing.T) {
	for _, test := range []struct {
		name            string
		input           djangoinput.Input
		invalid, absent bool
	}{
		{name: "omitted", input: djangoinput.Input{Fields: map[string]any{}}, absent: true},
		{name: "empty", input: djangoinput.Input{Fields: map[string]any{"custom_properties": map[string]any{}}}},
		{name: "explicit field clear", input: djangoinput.Input{Fields: map[string]any{"custom_properties": map[string]any{"field": nil}}}},
		{name: "null object", input: djangoinput.Input{Fields: map[string]any{"custom_properties": nil}}, invalid: true},
		{name: "form object", input: djangoinput.Input{Form: true, Fields: map[string]any{"custom_properties": `{"field":false}`}}},
		{name: "form trailing json", input: djangoinput.Input{Form: true, Fields: map[string]any{"custom_properties": `{} {}`}}, invalid: true},
		{name: "form array", input: djangoinput.Input{Form: true, Fields: map[string]any{"custom_properties": `[]`}}, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, err := submittedCustomValues(test.input)
			if (err != nil) != test.invalid || !test.invalid && (value == nil) != test.absent {
				t.Fatalf("shape handling: %#v %v", value, err)
			}
		})
	}
}
