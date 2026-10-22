package djangoinput

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// FieldValue applies DRF Field.get_value to the existing decoded input.
// Parse already retains the last repeated form value in Fields.
func (input Input) FieldValue(field string, required, allowBlank, allowNull bool) (any, bool) {
	value, present := input.Fields[field]
	if !input.Form || !present || value != "" {
		return value, present
	}
	if allowNull && !allowBlank {
		return nil, true
	}
	if !required && !allowBlank && !allowNull {
		return nil, false
	}
	return value, true
}

// BooleanFieldValue retains DRF's full-form checkbox default while a partial
// form update leaves omitted booleans untouched.
func (input Input) BooleanFieldValue(field string, partial bool) (any, bool) {
	if _, present := input.Fields[field]; input.Form && !present && !partial {
		return false, true
	}
	return input.FieldValue(field, false, false, false)
}

// JSONFieldValue uses the shared parsed form boundary and standard JSON decoder,
// matching DRF JSONField's HTML JSONString handling without changing task input.
func (input Input) JSONFieldValue(field string) (any, bool, error) {
	value, present := input.Fields[field]
	if !present || !input.Form {
		return value, present, nil
	}
	text, ok := value.(string)
	if !ok {
		return nil, true, fmt.Errorf("JSON form field is not text")
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewBufferString(text))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, true, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, true, fmt.Errorf("JSON form field has trailing data")
	}
	return decoded, true, nil
}

// JSONFieldSource preserves dictionary insertion order for source serializers
// that convert a JSON value to Python text before sanitizing it.
func (input Input) JSONFieldSource(field string) []byte {
	if input.Form {
		text, _ := input.Fields[field].(string)
		return []byte(text)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(input.Raw, &fields) != nil {
		return nil
	}
	return fields[field]
}
