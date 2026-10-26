package design

import (
	. "github.com/CaliLuke/loom/dsl"
)

var projectItem = Type("ProjectWorkItemType", func() {
	Extend(item)
	Attribute("imported", Boolean)
	Attribute("project_active", Boolean)
	Attribute("available", Boolean)
	Required("imported", "project_active", "available")
})
var projectCatalog = Type("ProjectCatalog", func() {
	Attribute("enabled", Boolean)
	Attribute("can_manage", Boolean)
	Attribute("revision", Int64)
	Attribute("project_name", String)
	Attribute("default_type_id", String)
	Attribute("types", ArrayOf(projectItem))
	Required("enabled", "can_manage", "revision", "project_name", "default_type_id", "types")
})
var projectInput = Type("ProjectTypeInput", func() {
	Attribute("revision", Int64, func() { Minimum(0) })
	Attribute("action", String, func() { Enum("import", "pause", "resume", "remove", "set_default") })
	Attribute("type_ids", ArrayOf(String, func() { Format(FormatUUID) }), func() { MaxLength(100) })
	Required("revision", "action", "type_ids")
})

func projectMethods() {
	projectCatalogMethods("ProjectCatalog", "UpdateProject", "work-item-types", projectCatalog, projectInput)
}
