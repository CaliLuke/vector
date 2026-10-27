package workitemtypes

import (
	"context"
	"strconv"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

// GroupingProperties resolves only authorized scalar categorical axes. Formula
// expressions and private context definitions never enter grouping responses.
func (snapshot *PropertyDiscoverySnapshot) GroupingProperties(ctx context.Context, keys []string) (map[string]*DiscoveryProperty, error) {
	fields, err := snapshot.discoveryFields()
	if err != nil {
		return nil, err
	}
	selected := map[string]*DiscoveryProperty{}
	for _, key := range keys {
		field := fields[key]
		if field == nil || !field.Groupable {
			return nil, reject(400, "This custom property cannot be grouped. Choose another group in Display.")
		}
		selected[key] = field
	}
	if err = snapshot.discoveryChoices(ctx, snapshot.tx, selected); err != nil {
		return nil, err
	}
	return selected, nil
}

// PropertyGroupValue keeps empty, inapplicable and invalid values distinct.
// Reference identities are retained even when their labels are unavailable.
func PropertyGroupValue(field DiscoveryProperty, values map[string]ProjectedValue) string {
	value, exists := values[field.PropertyID]
	if !exists || value.Status == "not_applicable" || value.Kind != field.Kind {
		return "NotApplicable"
	}
	current := value.Value
	if field.Kind == property.Cascading && value.CatalogID != field.CatalogID {
		return "NotApplicable"
	}
	if value.Status == "error" {
		return "Error"
	}
	if field.Kind == property.Cascading {
		current, exists = value.Levels[field.LevelID]
		if !exists {
			return "NotApplicable"
		}
	}
	if value.Status == "empty" || current == nil {
		return "None"
	}
	switch item := current.(type) {
	case bool:
		return strconv.FormatBool(item)
	case string:
		if item != "" {
			return item
		}
	}
	return "Error"
}
