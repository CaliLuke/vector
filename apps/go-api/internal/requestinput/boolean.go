package requestinput

import (
	"encoding/json"
	"strconv"
	"strings"
)

// BooleanField accepts JSON booleans, zero/one and conventional form checkbox values.
func BooleanField(value any) (bool, string) {
	if value == nil {
		return false, "This field may not be null."
	}
	switch value := value.(type) {
	case bool:
		return value, ""
	case json.Number:
		if number, err := IntegerField(value); err == nil && number.IsInt64() && (number.Int64() == 0 || number.Int64() == 1) {
			return number.Int64() == 1, ""
		}
	case string:
		text := strings.ToLower(value)
		if result, err := strconv.ParseBool(text); err == nil {
			return result, ""
		}
		switch text {
		case "yes", "y", "on":
			return true, ""
		case "no", "n", "off":
			return false, ""
		}
	}
	return false, "Must be a valid boolean."
}
