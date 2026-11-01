package requestinput

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/google/uuid"
)

func TestNativeJSONRejectsMalformedUnicodeAndAmbiguousDocuments(t *testing.T) {
	cases := map[string][]byte{
		"invalid UTF-8":             append([]byte(`{"name":"`), 0xff, '"', '}'),
		"incomplete UTF-8 suffix":   append([]byte(`{"name":"ok"}`), 0xc3),
		"unpaired surrogate":        []byte(`{"name":"\ud800"}`),
		"nested unpaired surrogate": []byte(`{"data":{"name":"\udfff"}}`),
		"trailing document":         []byte(`{"name":"ok"} {}`),
		"nonfinite number":          []byte(`{"value":NaN}`),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(bytes.NewReader(body), "application/json")
			var response *apicontract.HTTPError
			if !errors.As(err, &response) || response.Status != 400 {
				t.Fatalf("invalid body accepted or wrong error: %v", err)
			}
		})
	}
}

func TestNativeJSONPreservesUnicodeAndExactNumbers(t *testing.T) {
	input, err := Parse(bytes.NewBufferString(`{"name":"Zoë","emoji":"\ud83d\ude80","number":9007199254740993}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if input.Fields["name"] != "Zoë" || input.Fields["emoji"] != "🚀" || input.Fields["number"] != json.Number("9007199254740993") {
		t.Fatalf("decoded input lost data: %#v", input.Fields)
	}
}

func TestNativeUUIDRequiresTextualIdentity(t *testing.T) {
	for _, value := range []any{json.Number("1"), true, nil, []any{}, "x123e4567-e89b-12d3-a456-426614174000y"} {
		if id, err := UUID(value); err == nil {
			t.Fatalf("nontext UUID accepted as %s", id)
		}
	}
	expected := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	for _, text := range []string{expected.String(), "urn:uuid:" + expected.String()} {
		if id, err := UUID(text); err != nil || id != expected {
			t.Fatalf("text UUID=%s, error=%v", id, err)
		}
	}
}

func TestNativeIntegerRejectsCompatibilityCoercions(t *testing.T) {
	for _, value := range []any{"1.0", "1_0", "١", json.Number("1.1"), true, nil} {
		if number, err := IntegerField(value); err == nil {
			t.Fatalf("coerced %#v to %s", value, number)
		}
	}
	for _, value := range []any{"10", json.Number("10"), json.Number("10.00"), json.Number("1e1"), " +10 "} {
		number, err := IntegerField(value)
		if err != nil || number.String() != "10" {
			t.Fatalf("integer %#v=%v, error=%v", value, number, err)
		}
	}
}

func TestNativeIntegerPreservesPrecisionAndBoundsExponentWork(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"9007199254740993.0", "9007199254740993"},
		{"9223372036854775807.000", "9223372036854775807"},
		{"1e20", "100000000000000000000"},
	} {
		number, err := IntegerField(json.Number(test.input))
		if err != nil || number.String() != test.want {
			t.Fatalf("integer %s=%v error=%v", test.input, number, err)
		}
	}
	for _, input := range []string{"1.00000000000000000001", "1e100000000", "1e-100000000", "1_0", "not-a-number"} {
		if number, err := IntegerField(json.Number(input)); err == nil {
			t.Fatalf("invalid integer %s accepted as %s", input, number)
		}
	}
}
