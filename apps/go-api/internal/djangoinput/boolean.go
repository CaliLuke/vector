package djangoinput

import (
	"encoding/json"
	"strconv"
	"strings"
)

// BooleanField accepts DRF's BooleanField vocabulary. strconv handles the
// ordinary spellings; the additional words are DRF's compatibility contract.
func BooleanField(value any) (bool, string) {
	if value == nil {
		return false, "This field may not be null."
	}
	switch value := value.(type) {
	case bool:
		return value, ""
	case json.Number:
		if number, err := value.Float64(); err == nil && (number == 0 || number == 1) {
			return number == 1, ""
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
