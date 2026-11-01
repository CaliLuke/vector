package requestinput

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Text keeps strings unchanged and renders other JSON values as JSON text.
func Text(value any) (string, error) {
	if text, ok := value.(string); ok {
		if !utf8.ValidString(text) {
			return "", errors.New("text must be UTF-8")
		}
		return text, nil
	}
	switch value.(type) {
	case nil, bool, json.Number, float64, int, int64, []any, map[string]any:
		encoded, err := json.Marshal(value)
		return string(encoded), err
	default:
		return "", fmt.Errorf("unsupported text value %T", value)
	}
}

// Nonempty identifies optional values that still need field validation.
// It does not validate their type or authorize their use.
func Nonempty(value any) bool {
	switch value := value.(type) {
	case nil:
		return false
	case bool:
		return value
	case string:
		return value != ""
	case json.Number:
		coefficient, _, _ := strings.Cut(strings.ToLower(string(value)), "e")
		return strings.Trim(coefficient, "+-0.") != ""
	case float64:
		return value != 0
	case int:
		return value != 0
	case int64:
		return value != 0
	case []any:
		return len(value) > 0
	case map[string]any:
		return len(value) > 0
	default:
		return true
	}
}
