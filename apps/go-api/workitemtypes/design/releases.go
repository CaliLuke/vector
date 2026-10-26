package design

import (
	. "github.com/CaliLuke/loom/dsl"
)

var releaseValues = Type("ReleaseValues", func() {
	Attribute("name", String, func() { MaxLength(255) })
	Attribute("description", String, func() { MaxLength(10000) })
	Attribute("target_date", String, func() { MaxLength(10) })
	Required("name", "description", "target_date")
})
var releaseItem = Type("ProjectRelease", func() {
	Extend(releaseValues)
	Attribute("id", String)
	Attribute("created_at", String)
	Required("id", "created_at")
})
var releaseCatalog = Type("ReleaseCatalog", func() {
	Attribute("revision", Int64)
	Attribute("project_name", String)
	Attribute("can_manage", Boolean)
	Attribute("releases", ArrayOf(releaseItem))
	Required("revision", "project_name", "can_manage", "releases")
})
var releaseInput = Type("ReleaseInput", func() {
	Attribute("revision", Int64, func() { Minimum(0) })
	Attribute("action", String, func() { Enum("create", "update", "delete") })
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("values", releaseValues)
	Required("revision", "action", "id", "values")
})

func releaseMethods() {
	projectCatalogMethods("ReleaseCatalog", "UpdateReleaseCatalog", "work-item-releases", releaseCatalog, releaseInput)
}
