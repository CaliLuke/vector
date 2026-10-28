package cyclereads

import (
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
)

func ValidateCreateInput(fields map[string]any, surrogates map[string][]string, apiV1 bool) (CycleCreate, map[string]any) {
	var result CycleCreate
	errorsByField := map[string]any{}
	stringField := func(key string, required, allowBlank, allowNull bool, max int) (*string, bool) {
		value, present := fields[key]
		if !present {
			if required {
				errorsByField[key] = []string{"This field is required."}
			}
			return nil, false
		}
		if value == nil {
			if allowNull {
				return nil, true
			}
			errorsByField[key] = []string{"This field may not be null."}
			return nil, false
		}
		text, messages := djangoinput.TextField(value, allowBlank, max, surrogates[key])
		if len(messages) > 0 {
			errorsByField[key] = messages
			return nil, false
		}
		return &text, true
	}
	if name, ok := stringField("name", true, false, false, 255); ok && name != nil {
		result.Name = *name
	}
	if description, ok := stringField("description", false, true, false, 0); ok && description != nil {
		result.Description = *description
	}
	if zone, ok := stringField("timezone", false, false, false, 0); ok && zone != nil {
		result.Timezone = *zone
	}
	dateField := func(key string) *string {
		value, present := fields[key]
		if !present || value == nil {
			return nil
		}
		text, ok := value.(string)
		if !ok {
			errorsByField[key] = []string{"Datetime has wrong format."}
			return nil
		}
		return &text
	}
	result.StartDate = dateField("start_date")
	result.EndDate = dateField("end_date")
	optionalString := func(key string, max int) *string {
		text, ok := stringField(key, false, true, true, max)
		if !ok || text == nil {
			return nil
		}
		return text
	}
	result.ExternalSource = optionalString("external_source", 255)
	result.ExternalID = optionalString("external_id", 255)
	if apiV1 {
		if value, present := fields["owned_by"]; present && value != nil {
			parsed, err := djangoinput.UUID(value)
			if err != nil {
				if _, ok := value.(string); ok {
					errorsByField["owned_by"] = []string{"“" + djangoinput.ScalarString(value) + "” is not a valid UUID."}
				} else {
					errorsByField["owned_by"] = []string{"Incorrect type. Expected pk value, received " + djangoinput.PythonType(value) + "."}
				}
			} else {
				owner := parsed.String()
				result.OwnedBy = &owner
			}
		}
	} else {
		if value, present := fields["sort_order"]; present {
			sortOrder, message := djangoinput.FloatField(value)
			if message != "" {
				errorsByField["sort_order"] = []string{message}
			} else {
				result.SortOrder = &sortOrder
			}
		}
		for key, destination := range map[string]*any{
			"view_props": &result.ViewProps, "progress_snapshot": &result.ProgressSnapshot, "logo_props": &result.LogoProps,
		} {
			if value, present := fields[key]; present {
				if value == nil {
					errorsByField[key] = []string{"This field may not be null."}
				} else {
					*destination = value
				}
			}
		}
		if value, present := fields["version"]; present {
			integer, err := pythoncompat.DecimalInteger(djangoinput.ScalarString(value))
			if err != nil || !integer.IsInt64() {
				errorsByField["version"] = []string{"A valid integer is required."}
			} else if number := integer.Int64(); number < -2147483648 || number > 2147483647 {
				errorsByField["version"] = []string{"Ensure this value is less than or equal to 2147483647."}
			} else {
				version := int32(number)
				result.Version = &version
			}
		}
	}
	if len(errorsByField) == 0 {
		return result, nil
	}
	return result, errorsByField
}

func ValidatePatchInput(fields map[string]any, surrogates map[string][]string, apiV1 bool) (map[string]any, map[string]any) {
	validated := make(map[string]any, len(fields))
	errorsByField := map[string]any{}
	readonly := map[string]bool{"id": true, "created_at": true, "updated_at": true, "deleted_at": true, "created_by": true, "updated_by": true, "archived_at": true, "project": true, "workspace": true}
	if !apiV1 {
		readonly["owned_by"] = true
	}
	for key, value := range fields {
		if readonly[key] {
			continue
		}
		allowed := key == "name" || key == "description" || key == "start_date" || key == "end_date" || key == "sort_order" || key == "external_source" || key == "external_id" || key == "timezone"
		if apiV1 {
			allowed = allowed || key == "owned_by"
		}
		if !apiV1 {
			allowed = allowed || key == "view_props" || key == "progress_snapshot" || key == "logo_props" || key == "version"
		}
		if !allowed {
			continue
		}
		switch key {
		case "name", "description", "external_source", "external_id", "timezone":
			allowBlank, allowNull, maximum := true, false, 0
			if key == "name" || key == "timezone" {
				allowBlank = false
			}
			if key == "external_source" || key == "external_id" {
				allowNull, maximum = true, 255
			}
			if key == "name" {
				maximum = 255
			}
			if value == nil {
				if allowNull {
					validated[key] = nil
				} else {
					errorsByField[key] = []string{"This field may not be null."}
				}
				continue
			}
			text, messages := djangoinput.TextField(value, allowBlank, maximum, surrogates[key])
			if len(messages) > 0 {
				errorsByField[key] = messages
				continue
			}
			validated[key] = text
		case "start_date", "end_date":
			if value != nil {
				if _, ok := value.(string); !ok {
					errorsByField[key] = []string{"Datetime has wrong format."}
					continue
				}
			}
			validated[key] = value
		case "sort_order":
			number, message := djangoinput.FloatField(value)
			if message != "" {
				errorsByField[key] = []string{message}
				continue
			}
			validated[key] = number
		case "view_props", "progress_snapshot", "logo_props":
			if value == nil {
				errorsByField[key] = []string{"This field may not be null."}
				continue
			}
			validated[key] = value
		case "owned_by":
			if value == nil {
				validated[key] = nil
				continue
			}
			owner, err := djangoinput.UUID(value)
			if err != nil {
				errorsByField[key] = []string{"“" + djangoinput.ScalarString(value) + "” is not a valid UUID."}
				continue
			}
			validated[key] = owner.String()
		case "version":
			integer, err := pythoncompat.DecimalInteger(djangoinput.ScalarString(value))
			if err != nil || !integer.IsInt64() {
				errorsByField[key] = []string{"A valid integer is required."}
				continue
			}
			number := integer.Int64()
			if number < -2147483648 {
				errorsByField[key] = []string{"Ensure this value is greater than or equal to -2147483648."}
				continue
			}
			if number > 2147483647 {
				errorsByField[key] = []string{"Ensure this value is less than or equal to 2147483647."}
				continue
			}
			validated[key] = number
		}
	}
	if len(errorsByField) > 0 {
		return nil, errorsByField
	}
	return validated, nil
}
