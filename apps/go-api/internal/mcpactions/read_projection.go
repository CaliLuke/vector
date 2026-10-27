package mcpactions

import (
	"encoding/json"
	"errors"
	"math/big"
)

// projectReadResult decodes pagination separately so each entry can apply its
// pinned SDK normalization before constructing the generated result union.
func projectReadResult[Record, Page, Result any](value any, detail bool, record func(map[string]any) (*Record, error), detailResult func(*Record) Result, pageResult func(*Page, []*Record) Result) (*Result, error) {
	document, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("invalid resource document")
	}
	if detail {
		entry, err := record(document)
		if err != nil {
			return nil, err
		}
		result := detailResult(entry)
		return &result, nil
	}
	rows, ok := document["results"].([]any)
	if !ok {
		return nil, errors.New("invalid resource page")
	}
	envelope := make(map[string]any, len(document))
	for key, value := range document {
		if key != "results" {
			envelope[key] = value
		}
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	page := new(Page)
	if err = json.Unmarshal(body, page); err != nil {
		return nil, err
	}
	entries := make([]*Record, 0, len(rows))
	for _, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok {
			return nil, errors.New("invalid resource row")
		}
		entry, err := record(row)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	result := pageResult(page, entries)
	return &result, nil
}

// The HTTP serializer emits integral estimates as e.g. 3.0; the pinned SDK
// coerces them to integers. Preserve precision rather than converting via float.
func sdkInteger(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	number, ok := new(big.Rat).SetString(string(raw))
	if !ok || !number.IsInt() || !number.Num().IsInt64() {
		return nil, errors.New("invalid SDK integer")
	}
	return number.Num().Int64(), nil
}
