package issuequery

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"
)

type PQLPropertyValue struct {
	ItemID string
	Value  any
}

var errPQLProjection = errors.New("invalid PQL property materialization")

func (plan *PreparedPQL) propertyBindings(values map[string][]PQLPropertyValue) (map[string]string, error) {
	if len(values) != len(plan.properties) {
		return nil, errPQLProjection
	}
	bindings := make(map[string]string, len(values))
	for id, definition := range plan.properties {
		rows, exists := values[id]
		if !exists {
			return nil, errPQLProjection
		}
		items := make(map[string]any, len(rows))
		for _, row := range rows {
			item, err := uuid.Parse(row.ItemID)
			if err != nil {
				return nil, errPQLProjection
			}
			if _, duplicate := items[item.String()]; duplicate {
				return nil, errPQLProjection
			}
			value, err := pqlProjectedValue(definition.Kind, row.Value)
			if err != nil {
				return nil, err
			}
			items[item.String()] = value
		}
		encoded, err := json.Marshal(items)
		if err != nil {
			return nil, errPQLProjection
		}
		bindings[id] = string(encoded)
	}
	return bindings, nil
}

func pqlProjectedValue(kind PQLPropertyKind, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch kind {
	case PQLPropertyText:
		text, ok := value.(string)
		if ok && utf8.ValidString(text) && !strings.ContainsRune(text, 0) && len(text) <= 65536 {
			if strings.TrimSpace(text) == "" {
				return nil, nil
			}
			return text, nil
		}
	case PQLPropertyNumber:
		number, ok := value.(float64)
		if ok && !math.IsNaN(number) && !math.IsInf(number, 0) {
			return number, nil
		}
	case PQLPropertyBoolean:
		if boolean, ok := value.(bool); ok {
			return boolean, nil
		}
	case PQLPropertyDate:
		if text, ok := value.(string); ok {
			date, err := time.Parse(time.DateOnly, text)
			if err == nil && date.Year() >= 1 && date.Year() <= 9999 && date.Format(time.DateOnly) == text {
				return text, nil
			}
		}
	case PQLPropertySelection:
		return pqlProjectedSelections(value)
	}
	return nil, errPQLProjection
}

func pqlProjectedSelections(value any) (any, error) {
	var selected []string
	switch current := value.(type) {
	case string:
		selected = []string{current}
	case []string:
		selected = current
	default:
		return nil, errPQLProjection
	}
	if len(selected) == 0 {
		return nil, nil
	}
	normalized := make([]string, len(selected))
	for index, raw := range selected {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, errPQLProjection
		}
		normalized[index] = id.String()
	}
	return normalized, nil
}
