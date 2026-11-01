package pagereads

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/requestinput"
)

func TestPageLifecycleMetadataFieldValidationMatchesSerializerBoundaries(t *testing.T) {
	t.Run("metadata description rejects blank text", func(t *testing.T) {
		_, failures, _ := validatePageLifecycleFields(requestinput.Input{Fields: map[string]any{"description_html": ""}})
		if !reflect.DeepEqual(failures["description_html"], []string{"This field may not be blank."}) {
			t.Fatalf("description_html failures=%#v", failures["description_html"])
		}
	})

	t.Run("metadata description has no arbitrary short text cap", func(t *testing.T) {
		value := strings.Repeat("<p>long page text</p>", 200)
		fields, failures, _ := validatePageLifecycleFields(requestinput.Input{Fields: map[string]any{"description_html": value}})
		if len(failures) != 0 || fields["description_html"] != value {
			t.Fatalf("validated=%v failures=%v", fields["description_html"] == value, failures)
		}
	})

	t.Run("explicit null labels are invalid", func(t *testing.T) {
		_, failures, _ := validatePageLifecycleFields(requestinput.Input{Fields: map[string]any{"labels": nil}})
		if !reflect.DeepEqual(failures["labels"], []string{"This field may not be null."}) {
			t.Fatalf("labels failures=%#v", failures["labels"])
		}
	})
}

func TestPageLifecycleLabelListKeepsIndexedValidationErrors(t *testing.T) {
	t.Run("invalid list shape", func(t *testing.T) {
		labels := lifecycleParseLabelList("not-a-list", true)
		if len(labels.references) != 0 || !reflect.DeepEqual(labels.errors, []string{`Expected a list of items but got type "str".`}) {
			t.Fatalf("labels=%#v", labels)
		}
	})

	t.Run("typed errors, aggregate children, and preserve raw primary keys", func(t *testing.T) {
		noncanonical := "71000000000040008000000000000099"
		labels := lifecycleParseLabelList([]any{nil, true, map[string]any{}, []any{}, "", json.Number("1.5"), json.Number("99"), noncanonical, "not-a-uuid"}, true)
		want := map[string]any{
			"0": []string{"This field may not be null."},
			"1": []string{"Incorrect type. Expected pk value, received bool."},
			"2": []string{"“{}” is not a valid UUID."},
			"3": []string{"“[]” is not a valid UUID."},
			"4": []string{"This field may not be null."},
			"5": []string{"“1.5” is not a valid UUID."},
			"6": []string{"“99” is not a valid UUID."},
			"8": []string{"“not-a-uuid” is not a valid UUID."},
		}
		if !reflect.DeepEqual(labels.errors, want) {
			t.Fatalf("child errors=%#v want %#v", labels.errors, want)
		}
		if len(labels.references) != 1 || labels.references[0].index != 7 || labels.references[0].raw != noncanonical || labels.references[0].id.String() != "71000000-0000-4000-8000-000000000099" {
			t.Fatalf("textual UUID reference=%#v", labels.references)
		}
	})

	t.Run("omitted labels remain optional", func(t *testing.T) {
		labels := lifecycleParseLabelList(nil, false)
		if labels.present || len(labels.references) != 0 || labels.errors != nil {
			t.Fatalf("labels=%#v", labels)
		}
	})

	t.Run("composite child error preserves JSON object order and nesting", func(t *testing.T) {
		source := []byte(`[{"z":1,"a":{"y":2,"b":3}}]`)
		var values []any
		if err := json.Unmarshal(source, &values); err != nil {
			t.Fatal(err)
		}
		labels := lifecycleParseLabelList(values, true)
		want := map[string]any{"0": []string{`“{"a":{"b":3,"y":2},"z":1}” is not a valid UUID.`}}
		if !reflect.DeepEqual(labels.errors, want) {
			t.Fatalf("child errors=%#v want %#v", labels.errors, want)
		}
	})
}

func TestPageLifecycleRawAccessOwnershipCheckPrecedesChoiceCoercion(t *testing.T) {
	owner := uuid.New()
	member := uuid.New()
	state := pageLifecycleState{ownerID: owner, access: 0}
	for _, test := range []struct {
		name string
		user uuid.UUID
		raw  any
		want bool
	}{
		{name: "same numeric scalar", user: member, raw: json.Number("0"), want: false},
		{name: "same numeric string differs before choice coercion", user: member, raw: "0", want: true},
		{name: "same numeric float compares equal before choice coercion", user: member, raw: json.Number("0.0"), want: false},
		{name: "invalid choice is an ownership change first", user: member, raw: "9", want: true},
		{name: "owner bypasses raw ownership check", user: owner, raw: "9", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := lifecycleRawAccessChanged(test.raw, state, test.user); got != test.want {
				t.Fatalf("lifecycleRawAccessChanged(%#v)=%v want %v", test.raw, got, test.want)
			}
		})
	}
}

func TestPageLifecycleParentLookupAndSerializerRejectBooleanIdentity(t *testing.T) {
	if parent, err := lifecycleParentUUID(true); err == nil {
		t.Fatalf("boolean parent accepted as %s", parent)
	}
	if _, err := lifecycleParentUUID(json.Number("1")); err == nil {
		t.Fatal("numeric parent identity was accepted")
	}

	if requestinput.Nonempty(false) || requestinput.Nonempty(json.Number("0")) || requestinput.Nonempty([]any{}) || requestinput.Nonempty(map[string]any{}) {
		t.Fatal("falsey parent inputs would incorrectly enter the raw view lookup")
	}
	if !requestinput.Nonempty(true) || !requestinput.Nonempty(json.Number("1")) {
		t.Fatal("truthy parent inputs would incorrectly skip the raw view lookup")
	}
}
