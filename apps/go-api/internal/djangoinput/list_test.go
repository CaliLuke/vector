package djangoinput

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestFormListSourceVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/form_lists.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Name        string
			ContentType string `json:"content_type"`
			Body        string
			Partial     bool
			Fields      map[string]any
			Present     bool
			Value       any
		}
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) == 0 {
		t.Fatal("missing Django input vectors")
	}
	for _, test := range fixture.Cases {
		t.Run(test.Name, func(t *testing.T) {
			input, err := Parse(strings.NewReader(test.Body), test.ContentType)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(input.Fields, test.Fields) {
				t.Fatalf("scalar fields=%v; source=%v", input.Fields, test.Fields)
			}
			value, present := input.ListValue("labels", test.Partial)
			if present != test.Present || !reflect.DeepEqual(value, test.Value) {
				t.Fatalf("list=(%v,%t); source=(%v,%t)", value, present, test.Value, test.Present)
			}
		})
	}
}

func TestJSONListValuePreservesScalarAndMissingDistinction(t *testing.T) {
	for _, test := range []struct {
		body    string
		present bool
		value   any
	}{
		{`{}`, false, nil},
		{`{"labels":null}`, true, nil},
		{`{"labels":"value"}`, true, "value"},
		{`{"labels":[]}`, true, []any{}},
		{`{"labels":["one","two"]}`, true, []any{"one", "two"}},
	} {
		t.Run(test.body, func(t *testing.T) {
			input, err := Parse(strings.NewReader(test.body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			for _, partial := range []bool{false, true} {
				value, present := input.ListValue("labels", partial)
				if present != test.present || !reflect.DeepEqual(value, test.value) {
					t.Fatalf("partial=%t: (%v,%t); want (%v,%t)", partial, value, present, test.value, test.present)
				}
			}
		})
	}
}
