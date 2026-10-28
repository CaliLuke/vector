package assets

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
)

var legacyReadOnlyFields = map[string]bool{
	"id": true, "created_by": true, "updated_by": true, "created_at": true, "updated_at": true,
}

func legacyWritableFields(raw map[string]string) (map[string]any, error) {
	fields := make(map[string]any, len(raw))
	errorsByField := make(map[string]any)
	names := make([]string, 0, len(raw))
	for name := range raw {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		value := raw[name]
		if legacyReadOnlyFields[name] {
			continue
		}
		switch name {
		case "attributes", "storage_metadata":
			var parsed any
			if err := json.Unmarshal([]byte(value), &parsed); err != nil {
				errorsByField[name] = []string{"Value must be valid JSON."}
				continue
			}
			fields[name] = parsed
		case "user", "workspace", "draft_issue", "project", "issue", "comment", "page":
			if value == "" {
				fields[name] = nil
				continue
			}
			id, err := uuid.Parse(value)
			if err != nil {
				errorsByField[name] = []string{"“" + value + "” is not a valid UUID."}
				continue
			}
			fields[name] = id
		case "size":
			number, message := djangoinput.FloatField(value)
			if message != "" {
				errorsByField[name] = []string{message}
				continue
			}
			fields[name] = number
		case "is_uploaded", "is_deleted", "is_archived":
			if value == "" {
				continue
			}
			parsed, err := parseDRFBool(value)
			if err != nil {
				errorsByField[name] = []string{"Must be a valid boolean."}
				continue
			}
			fields[name] = parsed
		case "entity_type", "entity_identifier", "external_id", "external_source":
			text := strings.TrimSpace(value)
			if len([]rune(text)) > 255 {
				errorsByField[name] = []string{"Ensure this field has no more than 255 characters."}
				continue
			}
			fields[name] = text
		case "deleted_at":
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				fields[name] = nil
				continue
			}
			parsed, err := time.Parse("2006-01-02", trimmed)
			if err != nil {
				parsed, err = pythoncompat.DjangoDateTime(trimmed, time.UTC)
			}
			if err != nil {
				errorsByField[name] = []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}
				continue
			}
			fields[name] = parsed
		default:
			// DRF serializers ignore fields that are not declared by the model.
		}
	}
	if len(errorsByField) > 0 {
		return nil, &djangoapi.HTTPError{Status: 400, Body: errorsByField}
	}
	return fields, nil
}

func legacyRelationError(name, value string) error {
	return &djangoapi.HTTPError{Status: 400, Body: map[string]any{name: []string{fmt.Sprintf("Invalid pk \"%s\" - object does not exist.", value)}}}
}

func parseDRFBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "t", "yes", "y", "on", "1":
		return true, nil
	case "false", "f", "no", "n", "off", "0":
		return false, nil
	default:
		return false, strconv.ErrSyntax
	}
}

func legacyJSONField(fields map[string]any, name string) []byte {
	if value, ok := fields[name]; ok {
		if encoded, err := json.Marshal(value); err == nil {
			return encoded
		}
	}
	return []byte(`{}`)
}

func legacyUUIDField(fields map[string]any, name string) any {
	if value, ok := fields[name].(uuid.UUID); ok {
		return value
	}
	return nil
}

func legacyNullableStringField(fields map[string]any, name string) any {
	if value, ok := fields[name].(string); ok {
		return value
	}
	return nil
}

func legacyFloatField(fields map[string]any, name string) float64 {
	if value, ok := fields[name].(float64); ok {
		return value
	}
	return 0
}

func legacyBoolField(fields map[string]any, name string) bool {
	if value, ok := fields[name].(bool); ok {
		return value
	}
	return false
}

func legacyTimeField(fields map[string]any, name string) any {
	if value, ok := fields[name].(time.Time); ok {
		return value
	}
	return nil
}
