package djangoinput

import (
	"cmp"
	"math/big"
	"slices"
	"strings"
)

// ListValue follows DRF ListField.get_value. A partial form update must contain
// the bare key: indexed keys alone do not opt the field into the update.
func (input Input) ListValue(field string, partial bool) (any, bool) {
	value, present := input.Fields[field]
	if !input.Form || partial && !present {
		return value, present
	}
	if values := input.FormValues[field]; len(values) > 0 {
		return values, true
	}
	indexed := map[string]any{}
	for _, name := range input.FormKeys {
		rest, found := strings.CutPrefix(name, field+"[")
		if !found {
			continue
		}
		number, key, found := strings.Cut(rest, "]")
		if !found || number == "" || strings.ContainsFunc(number, func(r rune) bool { return r < '0' || r > '9' }) {
			continue
		}
		index, _ := new(big.Int).SetString(number, 10)
		canonical := index.String()
		if key == "" {
			indexed[canonical] = input.Fields[name]
		} else {
			object, ok := indexed[canonical].(map[string]any)
			if !ok {
				object = map[string]any{}
				indexed[canonical] = object
			}
			object[key] = input.Fields[name]
		}
	}
	if len(indexed) == 0 {
		return nil, false
	}
	keys := make([]string, 0, len(indexed))
	for key := range indexed {
		keys = append(keys, key)
	}
	// Canonical nonnegative decimal strings sort numerically by length then
	// lexically, without imposing a machine-integer bound absent in Python.
	slices.SortFunc(keys, func(a, b string) int {
		if order := cmp.Compare(len(a), len(b)); order != 0 {
			return order
		}
		return strings.Compare(a, b)
	})
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		values = append(values, indexed[key])
	}
	return values, true
}
