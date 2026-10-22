package djangoapi

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSerializeChoosesForeignKeyOnlyWhenSourceRowContainsIt(t *testing.T) {
	fields := []string{"description"}
	for _, test := range []struct {
		name string
		row  map[string]any
		want any
	}{
		{name: "scalar description", row: map[string]any{"description": "plain text"}, want: "plain text"},
		{name: "nonnull foreign key", row: map[string]any{"description": "ignored", "description_id": "asset-1"}, want: "asset-1"},
		{name: "nullable foreign key", row: map[string]any{"description": "ignored", "description_id": nil}, want: nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Serialize(context.Background(), nil, test.row, fields, "", "", false)
			if err != nil {
				t.Fatal(err)
			}
			if got["description"] != test.want {
				t.Fatalf("description=%#v, want %#v", got["description"], test.want)
			}
		})
	}
}

func TestSerializeExpansionUsesSourceForeignKeyEvenForScalarField(t *testing.T) {
	for _, test := range []struct {
		name string
		row  map[string]any
		want any
	}{
		{name: "scalar has no implicit id", row: map[string]any{"description": "plain text"}, want: nil},
		{name: "foreign key expansion", row: map[string]any{"description_id": "asset-1"}, want: "asset-1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Serialize(context.Background(), nil, test.row, []string{"description"}, "description", "description", false)
			if err != nil {
				t.Fatal(err)
			}
			if got["description"] != test.want {
				t.Fatalf("expanded description=%#v, want %#v", got["description"], test.want)
			}
		})
	}
}

func TestSerializeKeepsJsonNumberForCallerSpecificFloatNormalization(t *testing.T) {
	got, err := Serialize(context.Background(), nil, map[string]any{"sequence": json.Number("10000")}, []string{"sequence"}, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if got["sequence"] != json.Number("10000") {
		t.Fatalf("sequence=%#v", got["sequence"])
	}
}
