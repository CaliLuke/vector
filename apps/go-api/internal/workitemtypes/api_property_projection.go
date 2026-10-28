package workitemtypes

import (
	"fmt"
	"slices"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

var apiPropertyKinds = map[string]property.Kind{"TEXT": property.Text, "DATETIME": property.Date, "DECIMAL": property.Number, "BOOLEAN": property.Boolean, "OPTION": property.Dropdown, "URL": property.URL, "FORMULA": property.Formula, "CASCADING": property.Cascading, "EMAIL": property.Email, "FILE": property.File}
var apiRelationKinds = map[string]property.Kind{"ISSUE": property.Issue, "USER": property.Member, "RELEASE": property.Release, "RICH_TEXT": property.RichText}
var apiTextFormats = map[string]string{"single-line": "Single line", "multi-line": "Paragraph", "readonly": "Read-only"}
var apiDateFormats = map[string]string{"MMM dd, yyyy": "Jan 15, 2025", "dd/MM/yyyy": "15/01/2025", "MM/dd/yyyy": "01/15/2025", "yyyy/MM/dd": "2025/01/15"}

func apiKind(kind property.Kind) (string, any) {
	for label, native := range apiPropertyKinds {
		if native == kind {
			return label, nil
		}
	}
	for label, native := range apiRelationKinds {
		if native == kind {
			return "RELATION", label
		}
	}
	return string(kind), nil
}
func apiSettings(kind property.Kind, config property.Configuration) any {
	if kind == property.Formula {
		return map[string]any{"expression": config.Expression, "result": config.Result}
	}
	if kind == property.Cascading {
		return map[string]any{"levels": config.Levels, "options": config.Options}
	}
	formats := apiTextFormats
	if kind == property.Date {
		formats = apiDateFormats
	} else if kind != property.Text {
		return nil
	}
	for external, native := range formats {
		if native == config.Format {
			if kind == property.Text && config.Format == "Read-only" {
				return map[string]any{"display_format": external, "content": config.Content}
			}
			return map[string]any{"display_format": external}
		}
	}
	return nil
}
func apiDefaults(value any) any {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []any:
		return v
	case []string:
		return v
	default:
		return []string{fmt.Sprint(v)}
	}
}
func projectAPIProperty(row apiPropertyRow, config property.Configuration, q PropertyAPIQuery) map[string]any {
	def := row.Definition
	kind, relation := apiKind(def.Kind)
	options := []map[string]any{}
	for index, option := range config.Options {
		options = append(options, projectAPIOption(row, config, option, index, q))
	}
	var project, typeID any
	if q.ProjectID != uuid.Nil {
		project = q.ProjectID.String()
	}
	if q.TypeID != uuid.Nil {
		typeID = q.TypeID.String()
	}
	return map[string]any{"id": def.ID, "deleted_at": nil, "relation_type": relation, "created_at": row.CreatedAt, "updated_at": row.UpdatedAt, "name": def.Name, "display_name": def.Title, "description": def.Description, "logo_props": map[string]any{"name": def.Icon}, "sort_order": 65535, "property_type": kind, "is_required": config.Required, "default_value": apiDefaults(config.Default), "settings": apiSettings(def.Kind, config), "is_active": config.Active, "is_multi": config.Multiple, "validation_rules": nil, "external_source": row.Metadata["external_source"], "external_id": row.Metadata["external_id"], "created_by": row.CreatedBy, "updated_by": row.UpdatedBy, "workspace": q.WorkspaceID.String(), "project": project, "issue_type": typeID, "options": options}
}
func optionMetadata(row apiPropertyRow, id string) map[string]any {
	all, _ := row.Metadata["options"].(map[string]any)
	meta, _ := all[id].(map[string]any)
	if meta == nil {
		return map[string]any{}
	}
	return meta
}
func projectAPIOption(row apiPropertyRow, config property.Configuration, option property.Option, index int, q PropertyAPIQuery) map[string]any {
	metadata := optionMetadata(row, option.ID)
	defaults := memberValueIDs(config.Default)
	var project, parent any
	if q.ProjectID != uuid.Nil {
		project = q.ProjectID.String()
	}
	if option.Parent != "" {
		parent = option.Parent
	}
	created, updated, creator, updater := row.CreatedAt, row.UpdatedAt, row.CreatedBy, row.UpdatedBy
	if metadata["created_at"] != nil {
		created = metadata["created_at"]
	}
	if metadata["updated_at"] != nil {
		updated = metadata["updated_at"]
	}
	if metadata["created_by"] != nil {
		creator = metadata["created_by"]
	}
	if metadata["updated_by"] != nil {
		updater = metadata["updated_by"]
	}
	return map[string]any{"id": option.ID, "deleted_at": nil, "created_at": created, "updated_at": updated, "name": option.Name, "sort_order": index, "description": metadata["description"], "logo_props": map[string]any{"name": option.Icon, "color": metadata["color"]}, "is_active": config.Active, "is_default": slices.Contains(defaults, option.ID), "external_source": metadata["external_source"], "external_id": metadata["external_id"], "created_by": creator, "updated_by": updater, "workspace": q.WorkspaceID.String(), "project": project, "property": row.Definition.ID, "parent": parent}
}
