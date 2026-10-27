package milestones

import (
	"strings"
	"time"
	"unicode/utf8"
)

// StringField distinguishes omission from explicit null in native HTTP input.
// The MCP adapter omits its pinned exact-empty optional strings before calling.
type StringField struct {
	Present bool
	Value   *string
}

type Fields struct{ Title, TargetDate, ExternalSource, ExternalID StringField }

func normalizedFields(fields Fields, create bool) (map[string]any, error) {
	if create && !fields.Title.Present {
		return nil, ErrInvalidInput
	}
	values := make(map[string]any)
	for _, field := range []struct {
		name  string
		input StringField
	}{
		{"title", fields.Title}, {"target_date", fields.TargetDate}, {"external_source", fields.ExternalSource}, {"external_id", fields.ExternalID},
	} {
		if !field.input.Present {
			continue
		}
		if field.input.Value == nil {
			if field.name == "title" {
				return nil, ErrInvalidInput
			}
			values[field.name] = nil
			continue
		}
		value := *field.input.Value
		if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
			return nil, ErrInvalidInput
		}
		if field.name == "title" {
			value = strings.Trim(value, " \t\n\r\v\f")
			if value == "" {
				return nil, ErrInvalidInput
			}
		}
		if field.name == "target_date" {
			date, err := time.Parse("2006-01-02", value)
			if err != nil || len(value) != 10 || date.Year() < 1 {
				return nil, ErrInvalidInput
			}
		} else if utf8.RuneCountInString(value) > 255 {
			return nil, ErrInvalidInput
		}
		values[field.name] = value
	}
	return values, nil
}

func recordFields(record Record) map[string]any {
	return map[string]any{"title": record.Title, "target_date": stringValue(record.TargetDate), "external_source": stringValue(record.ExternalSource), "external_id": stringValue(record.ExternalID)}
}

func stringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
