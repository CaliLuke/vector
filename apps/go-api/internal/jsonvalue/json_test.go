package jsonvalue

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderedPayloadRetainsActivityOrderAndExactNumbers(t *testing.T) {
	raw := []byte(`{"z":1,"nested":{"b":2,"a":"hé😀"},"z":9007199254740993,"a":-0,"float":1e-7}`)
	value, err := DecodeObject(raw)
	require.NoError(t, err)
	got, err := Ordered(value, raw)
	require.NoError(t, err)
	require.Equal(t, `{"z": 9007199254740993, "nested": {"b": 2, "a": "hé😀"}, "a": -0, "float": 1e-7}`, got)
	delete(value, "z")
	value["state_id"] = "new"
	got, err = Ordered(value, raw)
	require.NoError(t, err)
	require.Equal(t, `{"nested": {"b": 2, "a": "hé😀"}, "a": -0, "float": 1e-7, "state_id": "new"}`, got)
}
func TestJSONValidationPreservesExactDecimalText(t *testing.T) {
	raw := []byte(`{"large":9007199254740993.0,"small":0.100000000000000005,"negative":-0,"nested":[1e-999,"1e999"]}`)
	got, err := Validate(raw)
	require.NoError(t, err)
	require.Equal(t, raw, got)
	for _, raw := range []string{`{"a":NaN}`, `{"a":1e999}`, `{"a":`, `{"a":Infinity}`} {
		_, validationErr := Validate([]byte(raw))
		require.Error(t, validationErr)
	}
	_, err = Ordered(json.Number("not-a-number"), nil)
	require.Error(t, err)
}

func TestActivityPayloadEncodingKeepsNestedValuesAndTypedArguments(t *testing.T) {
	type payload struct {
		ID     uint64   `json:"id"`
		Active bool     `json:"active"`
		Tags   []string `json:"tags"`
	}
	got, err := Ordered([]any{nil, true, 2, int64(-3), 1.25, payload{ID: 9007199254740993, Active: true, Tags: []string{"<tag>", "a\\b"}}}, []byte(`[null,true,2,-3,1.25,{"id":0,"active":false,"tags":[]}]`))
	require.NoError(t, err)
	require.JSONEq(t, `[null,true,2,-3,1.25,{"id":9007199254740993,"active":true,"tags":["<tag>","a\\b"]}]`, got)
	require.Contains(t, got, "9007199254740993")
	_, err = Ordered(map[string]any{"nested": []any{make(chan int)}}, nil)
	require.Error(t, err)
	empty, err := DecodeObject(nil)
	require.NoError(t, err)
	require.Empty(t, empty)
	_, err = DecodeObject([]byte(`[]`))
	require.Error(t, err)
}
