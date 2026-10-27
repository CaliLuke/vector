package workitemtypes

import (
	"cmp"
	"context"
	"slices"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

// PropertyExportColumn preserves stable field identity even when titles collide.
type PropertyExportColumn struct{ Key, Title string }

// ExportRows projects published rows admitted by the export query in this
// transaction. The selected project set bounds both values and column metadata.
func (snapshot *PropertyDiscoverySnapshot) ExportRows(ctx context.Context, rows []map[string]any, projects []string) ([]PropertyExportColumn, error) {
	selected := map[string]bool{}
	for _, id := range projects {
		if !snapshot.projects[id] {
			return nil, reject(403, "Export project access changed.")
		}
		selected[id] = true
	}
	snapshot.projects = selected
	if err := snapshot.ProjectStoredRows(ctx, rows, DiscoveryItemVisibility); err != nil {
		return nil, err
	}
	fields, err := snapshot.discoveryFields()
	if err != nil {
		return nil, err
	}
	if err = snapshot.discoveryChoices(ctx, snapshot.tx, fields); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if order := cmp.Compare(fields[a].Name, fields[b].Name); order != 0 {
			return order
		}
		return cmp.Compare(a, b)
	})
	columns := []PropertyExportColumn{{Key: "work_item_type", Title: "Work item type"}, {Key: "type_id", Title: "Work item type ID"}}
	for _, key := range keys {
		columns = append(columns, PropertyExportColumn{Key: key, Title: fields[key].Name + " [" + key + "]"})
	}
	names := map[string]string{}
	for _, item := range snapshot.Types.Types {
		names[item.ID] = item.Name
	}
	for _, row := range rows {
		typeID, _ := row["type_id"].(string)
		row["work_item_type"] = names[typeID]
		values := row["property_values"].(map[string]ProjectedValue)
		for _, key := range keys {
			row[key] = propertyExportValue(*fields[key], values)
		}
	}
	return columns, nil
}

func propertyExportValue(field DiscoveryProperty, values map[string]ProjectedValue) any {
	value, exists := values[field.PropertyID]
	if !exists || value.Status == "not_applicable" || value.Kind != field.Kind {
		return "#NOT_APPLICABLE"
	}
	if field.Kind == property.Cascading && value.CatalogID != field.CatalogID {
		return "#NOT_APPLICABLE"
	}
	if value.Status == "error" {
		return "#ERROR: " + value.Error
	}
	current := value.Value
	if field.Kind == property.Cascading {
		current = value.Levels[field.LevelID]
	}
	if current == nil || value.Status == "empty" {
		return nil
	}
	if field.Kind != property.Dropdown && field.Kind != property.Member && field.Kind != property.Release && field.Kind != property.Cascading {
		return current
	}
	names := map[string]string{}
	for _, option := range field.Options {
		names[option.ID] = option.Name
	}
	return exportReferenceNames(current, names)
}

func exportReferenceNames(value any, names map[string]string) any {
	switch item := value.(type) {
	case string:
		if name, exists := names[item]; exists {
			return name
		}
		return item
	case []any:
		out := make([]any, len(item))
		for i, entry := range item {
			out[i] = exportReferenceNames(entry, names)
		}
		return out
	case []string:
		out := make([]any, len(item))
		for i, entry := range item {
			out[i] = exportReferenceNames(entry, names)
		}
		return out
	default:
		return value
	}
}
