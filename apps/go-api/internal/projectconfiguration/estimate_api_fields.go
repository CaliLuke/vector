package projectconfiguration

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
)

// estimateAPIFields validates the SDK-shaped API independently of the app's
// estimate-plus-points editor envelope. Zero is a real point key.
func estimateAPIFields(action string, input map[string]any) (map[string]any, error) {
	allowed := map[string]bool{}
	switch action {
	case "create", "update":
		for _, key := range []string{"name", "description", "external_source", "external_id"} {
			allowed[key] = true
		}
		if action == "create" {
			allowed["type"], allowed["last_used"] = true, true
		}
	case "create_points":
		allowed["points"] = true
	case "update_point":
		for _, key := range []string{"value", "key", "description", "external_source", "external_id"} {
			allowed[key] = true
		}
	case "delete", "delete_point":
	default:
		return nil, projectmutations.ErrInvalid
	}
	fields := map[string]any{}
	for key, value := range input {
		if !allowed[key] {
			return nil, projectmutations.ErrInvalid
		}
		switch key {
		case "last_used":
			flag, ok := value.(bool)
			if !ok {
				return nil, projectmutations.ErrInvalid
			}
			fields[key] = flag
		case "key":
			n, ok := integerValue(value)
			if !ok || n < 0 || n > 2147483647 {
				return nil, projectmutations.ErrInvalid
			}
			fields[key] = n
		case "points":
			raw, ok := value.(string)
			if !ok {
				return nil, projectmutations.ErrInvalid
			}
			points, err := parseEstimatePoints(raw)
			if err != nil {
				return nil, err
			}
			fields[key] = points
		default:
			text, ok := value.(string)
			if !ok {
				return nil, projectmutations.ErrInvalid
			}
			if key != "description" {
				text = strings.TrimSpace(text)
			}
			if (key == "name" || key == "value") && text == "" {
				return nil, projectmutations.ErrInvalid
			}
			max := 255
			if key == "value" {
				max = 20
			}
			if key != "description" && utf8.RuneCountInString(text) > max || strings.ContainsRune(text, 0) {
				return nil, projectmutations.ErrInvalid
			}
			fields[key] = text
		}
	}
	if action == "create" {
		if fields["name"] == nil {
			return nil, projectmutations.ErrInvalid
		}
		if fields["type"] == nil {
			fields["type"] = "categories"
		}
		if fields["last_used"] == nil {
			fields["last_used"] = true
		}
		switch fields["type"] {
		case "categories", "points", "time":
		default:
			return nil, projectmutations.ErrInvalid
		}
		estimateFieldDefaults(fields)
	}
	if action == "create_points" && fields["points"] == nil {
		return nil, projectmutations.ErrInvalid
	}
	return fields, nil
}
func estimateFieldDefaults(fields map[string]any) {
	if fields["description"] == nil {
		fields["description"] = ""
	}
	for _, key := range []string{"external_source", "external_id"} {
		if _, ok := fields[key]; !ok {
			fields[key] = nil
		}
	}
}
func parseEstimatePoints(raw string) ([]map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var points []map[string]any
	if decoder.Decode(&points) != nil || len(points) == 0 {
		return nil, projectmutations.ErrInvalid
	}
	var tail any
	if decoder.Decode(&tail) != io.EOF {
		return nil, projectmutations.ErrInvalid
	}
	result := make([]map[string]any, len(points))
	for i, point := range points {
		// The pinned SDK ignores extra fields in point request models.
		fields := map[string]any{}
		for _, key := range []string{"value", "key", "description", "external_source", "external_id"} {
			if v, ok := point[key]; ok && v != nil {
				fields[key] = v
			}
		}
		if fields["value"] == nil {
			return nil, projectmutations.ErrInvalid
		}
		validated, err := estimateAPIFields("update_point", fields)
		if err != nil {
			return nil, err
		}
		if validated["key"] == nil {
			validated["key"] = int64(0)
		}
		estimateFieldDefaults(validated)
		result[i] = validated
	}
	return result, nil
}
