package workspaceprojectreads

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/CaliLuke/go-plane/internal/stickies/content"
	"github.com/go-json-experiment/json/jsontext"
)

// projectFieldValidation collects normalized values and field errors for one request.
// Relationship scope checks run only after field validation succeeds.
type projectFieldValidation struct {
	input    djangoinput.Input
	values   map[string]any
	failures map[string]any
}

func (v *projectFieldValidation) validateScalars(app, replace, create bool) error {
	if app {
		for _, key := range []string{"created_by", "updated_by", "archived_at"} {
			if _, supplied := v.input.Fields[key]; supplied {
				v.failures[key] = []string{"This field is read-only."}
			}
		}
	}
	v.text("name", create || replace, false, false, 255)
	v.text("identifier", create || replace, false, false, 12)
	v.text("description", false, true, false, 0)
	v.text("cover_image", false, true, true, 0)
	v.text("emoji", false, true, true, 255)
	v.text("external_source", false, true, true, 255)
	v.text("external_id", false, true, true, 255)
	if err := v.timezone(); err != nil {
		return err
	}
	v.booleans(!create && !replace)
	v.durations()
	v.jsonFields(app)
	if app && !replace {
		return v.network()
	}
	return nil
}

func (v *projectFieldValidation) text(key string, required, blank, nullable bool, max int) {
	raw, present := v.input.FieldValue(key, required, blank, nullable)
	if !present {
		if required {
			v.failures[key] = []string{"This field is required."}
		}
		return
	}
	if raw == nil {
		if nullable {
			v.values[key] = nil
		} else {
			v.failures[key] = []string{"This field may not be null."}
		}
		return
	}
	value, messages := djangoinput.TextField(raw, blank, max, v.input.Surrogates[key])
	if len(messages) > 0 {
		v.failures[key] = messages
		return
	}
	v.values[key] = value
}

func (v *projectFieldValidation) timezone() error {
	if raw, present := v.input.FieldValue("timezone", false, false, false); present {
		zone, err := projectFieldErrorString(v.input, "timezone", raw)
		if err != nil {
			return err
		}
		if raw == nil {
			v.failures["timezone"] = []string{"This field may not be null."}
		} else if !validProjectTimezone(zone) {
			v.failures["timezone"] = []string{fmt.Sprintf("\"%s\" is not a valid choice.", zone)}
		} else {
			v.values["timezone"] = zone
		}
	}

	return nil
}

func (v *projectFieldValidation) booleans(partial bool) {
	for _, key := range projectBooleanFields {
		raw, present := v.input.BooleanFieldValue(key, partial)
		if !present {
			continue
		}
		if raw == nil {
			v.failures[key] = []string{"This field may not be null."}
			continue
		}
		if value, ok := projectBoolean(raw); ok {
			v.values[key] = value
		} else {
			v.failures[key] = []string{"Must be a valid boolean."}
		}
	}
}

func (v *projectFieldValidation) durations() {
	for _, key := range []string{"archive_in", "close_in"} {
		raw, present := v.input.FieldValue(key, false, false, false)
		if !present {
			continue
		}
		if raw == nil {
			v.failures[key] = []string{"This field may not be null."}
			continue
		}
		number, err := djangoinput.IntegerField(raw)
		if err != nil || !number.IsInt64() {
			v.failures[key] = []string{"A valid integer is required."}
			continue
		}
		value := number.Int64()
		if value < 0 {
			v.failures[key] = []string{"Ensure this value is greater than or equal to 0."}
		} else if value > 12 {
			v.failures[key] = []string{"Ensure this value is less than or equal to 12."}
		} else {
			v.values[key] = value
		}
	}
}

func (v *projectFieldValidation) jsonFields(app bool) {
	for _, key := range []string{"icon_prop", "description_html", "description_text", "logo_props"} {
		if !app && key != "icon_prop" {
			continue
		}
		if _, present := v.input.Fields[key]; present && !projectJSONFieldUnicodeValid(v.input, key) {
			v.failures[key] = []string{"Value must be valid JSON."}
			continue
		}
		raw, present, err := v.input.JSONFieldValue(key)
		if err != nil {
			v.failures[key] = []string{"Value must be valid JSON."}
			continue
		}
		if present {
			if raw == nil && key == "logo_props" {
				v.failures[key] = []string{"This field may not be null."}
			} else {
				normalized, err := normalizeProjectJSONField(raw)
				if err != nil {
					v.failures[key] = []string{"Value must be valid JSON."}
				} else {
					v.values[key] = normalized
				}
			}
		}
	}
}

func (v *projectFieldValidation) network() error {
	if raw, present := v.input.FieldValue("network", false, false, false); present {
		choice, err := projectFieldErrorString(v.input, "network", raw)
		if err != nil {
			return err
		}
		if raw == nil {
			v.failures["network"] = []string{"This field may not be null."}
		} else if choice != "0" && choice != "2" {
			v.failures["network"] = []string{fmt.Sprintf("\"%s\" is not a valid choice.", choice)}
		} else {
			if choice == "0" {
				v.values["network"] = int64(0)
			} else {
				v.values["network"] = int64(2)
			}
		}
	}

	return nil
}

func (v *projectFieldValidation) descriptionHTML() error {
	if raw, present := v.values["description_html"]; present && pythoncompat.Truthy(raw) {
		text, err := pythoncompat.String(raw, v.input.JSONFieldSource("description_html"))
		if err != nil {
			return err
		}
		clean, hasClean, err := content.ValidateHTML(text)
		if err != nil {
			return projectFailure(400, map[string]any{"error": []string{"html content is not valid"}})
		}
		if hasClean {
			v.values["description_html"] = clean
		}
	}
	return nil
}

func projectFieldErrorString(input djangoinput.Input, field string, value any) (string, error) {
	switch value.(type) {
	case []any, map[string]any:
		return pythoncompat.String(value, input.JSONFieldSource(field))
	default:
		return djangoinput.ScalarString(value), nil
	}
}

func projectBoolean(raw any) (bool, bool) {
	switch v := raw.(type) {
	case bool:
		return v, true
	case json.Number:
		label := djangoinput.ScalarString(v)
		if label == "1" || label == "1.0" {
			return true, true
		}
		if label == "0" || label == "0.0" {
			return false, true
		}
		if strings.ContainsAny(string(v), ".eE") {
			number, err := v.Float64()
			if err == nil {
				if number == 1 {
					return true, true
				}
				if number == 0 {
					return false, true
				}
			}
		}
	case string:
		switch strings.ToLower(v) {
		case "t", "y", "yes", "true", "on", "1":
			return true, true
		case "f", "n", "no", "false", "off", "0":
			return false, true
		}
	}
	return false, false
}

func normalizeProjectJSONField(value any) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	normalized, err := pythoncompat.NormalizeJSONNumbers(raw)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(normalized)))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// projectJSONFieldUnicodeValid checks the original JSONField text before the
// request decoder can replace lone-surrogate escapes with U+FFFD. Duplicate
// names remain valid; an invalid escape in an overwritten duplicate is still
// rejected as part of validating the submitted field source.
func projectJSONFieldUnicodeValid(input djangoinput.Input, field string) bool {
	source := input.JSONFieldSource(field)
	if len(source) == 0 {
		value, present := input.Fields[field]
		if !present {
			return true
		}
		var err error
		source, err = json.Marshal(value)
		if err != nil {
			return false
		}
	}
	return jsontext.Value(source).IsValid(jsontext.AllowDuplicateNames(true))
}
