package projectstatereads

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestStateWriteInputAggregatesRequiredBlankAndNullErrors(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       map[string][]string
	}{
		{"required", `{}`, map[string][]string{"name": {"This field is required."}, "color": {"This field is required."}}},
		{"blank-null", `{"name":" ","color":null,"default":"invalid"}`, map[string][]string{"name": {"This field may not be blank."}, "color": {"This field may not be null."}, "default": {"Must be a valid boolean."}}},
		{"triage", `{"name":"Triage","color":"red","group":"triage"}`, map[string][]string{"non_field_errors": {"Cannot create triage state"}}},
		{"list", `[]`, map[string][]string{"non_field_errors": {"Invalid data. Expected a dictionary, but got list."}}},
		{"null", `null`, map[string][]string{"non_field_errors": {"No data provided"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input, err := ParseStateWriteBody(strings.NewReader(test.body), "application/json")
			if err != nil {
				t.Fatal(err)
			}
			err = validateStateFields(input.Fields, true, true, &input)
			var failure *WriteFailure
			if !errors.As(err, &failure) || failure.Status != 400 || !reflect.DeepEqual(failure.Body, test.want) {
				t.Fatalf("error=%+v, want=%+v", err, test.want)
			}
		})
	}
}

func TestStateWriteInputPersistsNormalizedValues(t *testing.T) {
	input, err := ParseStateWriteBody(strings.NewReader(`{"name":42,"color":"  red  ","default":"ON","sequence":"12.5","order":3,"description":"  test  ","external_id":"ignored by app"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if err = validateStateFields(input.Fields, true, true, &input); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "42", "color": "red", "default": true, "sequence": 12.5, "order": float64(3), "description": "test"}
	if !reflect.DeepEqual(input.Fields, want) {
		t.Fatalf("fields=%#v, want=%#v", input.Fields, want)
	}
}

func TestStateWriteInputPreservesFormAndUnicodeValidation(t *testing.T) {
	for _, partial := range []bool{false, true} {
		input, err := ParseStateWriteBody(strings.NewReader("name=Ready&color=red"), "application/x-www-form-urlencoded")
		if err != nil {
			t.Fatal(err)
		}
		if err = validateStateFields(input.Fields, !partial, false, &input); err != nil {
			t.Fatal(err)
		}
		_, defaultPresent := input.Fields["default"]
		_, triagePresent := input.Fields["is_triage"]
		if defaultPresent == partial || triagePresent == partial {
			t.Fatalf("partial=%v fields=%#v", partial, input.Fields)
		}
	}
	name := strings.Repeat("界", 255)
	encoded, _ := json.Marshal(map[string]any{"name": name, "color": "red"})
	input, err := ParseStateWriteBody(strings.NewReader(string(encoded)), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if err = validateStateFields(input.Fields, true, true, &input); err != nil {
		t.Fatal(err)
	}
	if input.Fields["name"] != name {
		t.Fatal("Unicode name changed")
	}
	input, err = ParseStateWriteBody(strings.NewReader(`{"name":"\ud800","color":"red"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if err = validateStateFields(input.Fields, true, true, &input); err == nil {
		t.Fatal("unpaired surrogate accepted")
	}
}
