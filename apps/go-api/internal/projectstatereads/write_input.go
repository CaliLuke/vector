package projectstatereads

import (
	"errors"
	"io"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

// ParseStateWriteBody preserves the shared parser's form and Unicode metadata
// until the route-specific serializer runs.
func ParseStateWriteBody(body io.Reader, contentType string) (djangoinput.Input, error) {
	input, err := djangoinput.Parse(body, contentType)
	var response *djangoapi.HTTPError
	if errors.As(err, &response) {
		return input, &WriteFailure{Status: response.Status, Body: response.Body}
	}
	return input, err
}

func validateStateFields(fields map[string]any, create, app bool, parsed *djangoinput.Input) error {
	input := djangoinput.Input{Fields: fields}
	if parsed != nil {
		input = *parsed
		input.Fields = fields
	}
	if input.Null {
		return &WriteFailure{Status: 400, Body: map[string][]string{"non_field_errors": {"No data provided"}}}
	}
	if input.Kind != "" {
		return &WriteFailure{Status: 400, Body: map[string][]string{"non_field_errors": {"Invalid data. Expected a dictionary, but got " + input.Kind + "."}}}
	}
	failures := map[string][]string{}
	allowed := map[string]bool{"name": true, "color": true, "group": true, "default": true, "description": true, "sequence": true}
	textFields := []struct {
		name                      string
		required, blank, nullable bool
		maximum                   int
	}{{"name", true, false, false, 255}, {"color", true, false, false, 255}, {"description", false, true, false, 0}}
	if !app {
		for _, name := range []string{"external_source", "external_id"} {
			allowed[name] = true
			textFields = append(textFields, struct {
				name                      string
				required, blank, nullable bool
				maximum                   int
			}{name, false, true, true, 255})
		}
		allowed["is_triage"] = true
	} else {
		allowed["order"] = true
	}
	for _, field := range textFields {
		value, present := input.FieldValue(field.name, field.required, field.blank, field.nullable)
		if !present {
			delete(fields, field.name)
			if create && field.required {
				failures[field.name] = []string{"This field is required."}
			}
			continue
		}
		if value == nil && field.nullable {
			fields[field.name] = nil
			continue
		}
		text, messages := djangoinput.TextField(value, field.blank, field.maximum, input.Surrogates[field.name])
		if len(messages) != 0 {
			failures[field.name] = messages
		} else {
			fields[field.name] = text
		}
	}
	for _, field := range []string{"default", "is_triage"} {
		if !allowed[field] {
			continue
		}
		value, present := input.BooleanFieldValue(field, !create)
		if !present {
			delete(fields, field)
			continue
		}
		boolean, message := djangoinput.BooleanField(value)
		if message != "" {
			failures[field] = []string{message}
		} else {
			fields[field] = boolean
		}
	}
	for _, field := range []string{"sequence", "order"} {
		if !allowed[field] {
			continue
		}
		value, present := input.FieldValue(field, false, false, false)
		if !present {
			delete(fields, field)
			continue
		}
		if value == nil {
			failures[field] = []string{"This field may not be null."}
			continue
		}
		number, message := djangoinput.FloatField(value)
		if message != "" {
			failures[field] = []string{message}
		} else {
			fields[field] = number
		}
	}
	if value, present := input.FieldValue("group", false, false, false); present {
		if value == nil {
			failures["group"] = []string{"This field may not be null."}
		} else if text, ok := value.(string); !ok || !oneOf(text, "backlog", "unstarted", "started", "completed", "cancelled", "triage") {
			text, err := pythoncompat.String(value, input.JSONFieldSource("group"))
			if err != nil {
				return err
			}
			failures["group"] = []string{"\"" + text + "\" is not a valid choice."}
		}
	} else {
		delete(fields, "group")
	}
	if len(failures) != 0 {
		return &WriteFailure{Status: 400, Body: failures}
	}
	if fields["group"] == "triage" {
		return &WriteFailure{Status: 400, Body: map[string][]string{"non_field_errors": {"Cannot create triage state"}}}
	}
	for field := range fields {
		if !allowed[field] {
			delete(fields, field)
		}
	}
	return nil
}
