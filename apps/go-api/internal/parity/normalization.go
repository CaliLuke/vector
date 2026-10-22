package parity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

var defaultVolatileReplacement = json.RawMessage(`"<volatile>"`)

// Normalizations contains only explicitly approved top-level column rules.
type Normalizations struct {
	VolatileFields []VolatileFieldRule `json:"volatile_fields"`
	VolatileValues []VolatileValueRule `json:"volatile_values"`
}

// VolatileFieldRule replaces one complete top-level column value.
type VolatileFieldRule struct {
	Table       string          `json:"table"`
	Column      string          `json:"column"`
	Replacement json.RawMessage `json:"replacement,omitempty"`
}

// VolatileValueRule replaces only listed JSON scalar or object values in one column.
type VolatileValueRule struct {
	Table       string            `json:"table"`
	Column      string            `json:"column"`
	Values      []json.RawMessage `json:"values"`
	Replacement json.RawMessage   `json:"replacement"`
}

// LoadNormalizations reads a strict JSON normalization file.
func LoadNormalizations(path string) (Normalizations, error) {
	file, err := os.Open(path)
	if err != nil {
		return Normalizations{}, fmt.Errorf("open normalization file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var rules Normalizations
	if err := decoder.Decode(&rules); err != nil {
		return Normalizations{}, fmt.Errorf("decode normalization file: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Normalizations{}, err
	}
	if err := rules.Validate(); err != nil {
		return Normalizations{}, err
	}
	return rules, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode normalization file: multiple JSON values")
		}
		return fmt.Errorf("decode normalization file: %w", err)
	}
	return nil
}

// Validate rejects incomplete and invalid rules before a database is queried.
func (n Normalizations) Validate() error {
	seen := make(map[string]struct{}, len(n.VolatileFields)+len(n.VolatileValues))
	for index := range n.VolatileFields {
		rule := &n.VolatileFields[index]
		if rule.Table == "" || rule.Column == "" {
			return fmt.Errorf("volatile_fields[%d] requires table and column", index)
		}
		if len(rule.Replacement) == 0 {
			rule.Replacement = bytes.Clone(defaultVolatileReplacement)
		}
		if !json.Valid(rule.Replacement) {
			return fmt.Errorf("volatile_fields[%d] replacement is not valid JSON", index)
		}
		key := "field\x00" + rule.Table + "\x00" + rule.Column
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate volatile field rule for %s.%s", rule.Table, rule.Column)
		}
		seen[key] = struct{}{}
	}
	for index, rule := range n.VolatileValues {
		if rule.Table == "" || rule.Column == "" || len(rule.Values) == 0 || len(rule.Replacement) == 0 {
			return fmt.Errorf("volatile_values[%d] requires table, column, values, and replacement", index)
		}
		if !json.Valid(rule.Replacement) {
			return fmt.Errorf("volatile_values[%d] replacement is not valid JSON", index)
		}
		for valueIndex, value := range rule.Values {
			if !json.Valid(value) {
				return fmt.Errorf("volatile_values[%d].values[%d] is not valid JSON", index, valueIndex)
			}
		}
		key := "value\x00" + rule.Table + "\x00" + rule.Column
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate volatile value rule for %s.%s", rule.Table, rule.Column)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (n Normalizations) validateColumns(tables []TableDump) error {
	columns := make(map[string]map[string]struct{}, len(tables))
	for _, table := range tables {
		columns[table.Name] = make(map[string]struct{}, len(table.Columns))
		for _, column := range table.Columns {
			columns[table.Name][column.Name] = struct{}{}
		}
	}
	for _, rule := range n.VolatileFields {
		if _, exists := columns[rule.Table][rule.Column]; !exists {
			return fmt.Errorf("volatile field rule references unknown column %s.%s", rule.Table, rule.Column)
		}
	}
	for _, rule := range n.VolatileValues {
		if _, exists := columns[rule.Table][rule.Column]; !exists {
			return fmt.Errorf("volatile value rule references unknown column %s.%s", rule.Table, rule.Column)
		}
	}
	return nil
}

func (n Normalizations) canonicalize(table string, row json.RawMessage) (json.RawMessage, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(row, &values); err != nil {
		return nil, fmt.Errorf("decode row object: %w", err)
	}
	for _, rule := range n.VolatileFields {
		if rule.Table != table {
			continue
		}
		if _, exists := values[rule.Column]; exists {
			values[rule.Column] = bytes.Clone(rule.Replacement)
		}
	}
	for _, rule := range n.VolatileValues {
		if rule.Table != table {
			continue
		}
		value, exists := values[rule.Column]
		if !exists {
			continue
		}
		for _, volatileValue := range rule.Values {
			if bytes.Equal(value, volatileValue) {
				values[rule.Column] = bytes.Clone(rule.Replacement)
				break
			}
		}
	}
	canonical, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("encode canonical row object: %w", err)
	}
	return canonical, nil
}
