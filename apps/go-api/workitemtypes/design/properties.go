package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var propertyOption = Type("PropertyOption", func() {
	for _, name := range []string{"id", "name", "icon", "parent", "next_label"} {
		Attribute(name, String)
	}
	Attribute("level", Int)
	Required("id", "name", "icon", "parent", "next_label", "level")
})
var propertyLevel = Type("PropertyLevel", func() {
	Attribute("id", String)
	Attribute("name", String)
	Attribute("required", Boolean)
	Required("id", "name", "required")
})
var propertyConfiguration = Type("PropertyConfiguration", func() {
	for _, name := range []string{"active", "required", "multiple"} {
		Attribute(name, Boolean)
	}
	for _, name := range []string{"format", "content", "expression", "result"} {
		Attribute(name, String)
	}
	Attribute("default_value", Any)
	Attribute("options", ArrayOf(propertyOption))
	Attribute("levels", ArrayOf(propertyLevel))
	Required("active", "required", "multiple", "format", "content", "expression", "result", "options", "levels")
})
var propertyContext = Type("PropertyContext", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String, func() { MinLength(1); MaxLength(255) })
	for _, name := range []string{"project_ids", "type_ids", "override_keys"} {
		Attribute(name, ArrayOf(String), func() { MaxLength(1000) })
	}
	Attribute("configuration", propertyConfiguration)
	Required("id", "name", "project_ids", "type_ids", "override_keys", "configuration")
})
var propertyDefinition = Type("PropertyDefinition", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	for _, name := range []string{"name", "title", "description", "icon", "kind"} {
		Attribute(name, String)
	}
	Attribute("configuration", propertyConfiguration)
	Attribute("contexts", ArrayOf(propertyContext), func() { MaxLength(100) })
	Required("id", "name", "title", "description", "icon", "kind", "configuration", "contexts")
})
var propertyRecord = Type("PropertyRecord", func() {
	Attribute("definition", propertyDefinition)
	Attribute("used", Boolean)
	Required("definition", "used")
})
var propertyAttachment = Type("PropertyAttachment", func() {
	Attribute("type_id", String)
	Attribute("property_ids", ArrayOf(String))
	Required("type_id", "property_ids")
})
var propertyChoice = Type("PropertyChoice", func() { Attribute("id", String); Attribute("name", String); Required("id", "name") })
var propertyCatalog = Type("PropertyCatalog", func() {
	Attribute("enabled", Boolean)
	Attribute("project_count", Int)
	Attribute("types", ArrayOf(item))
	Attribute("revision", Int64)
	Attribute("can_manage", Boolean)
	Attribute("properties", ArrayOf(propertyRecord))
	Attribute("attachments", ArrayOf(propertyAttachment))
	Attribute("projects", ArrayOf(propertyChoice))
	Attribute("members", ArrayOf(propertyChoice))
	Required("enabled", "project_count", "types", "revision", "can_manage", "properties", "attachments", "projects", "members")
})
var propertyInput = Type("PropertyInput", func() {
	Attribute("revision", Int64, func() { Minimum(0) })
	Attribute("action", String, func() { Enum("create", "update", "attach", "delete", "detach") })
	Attribute("definition", propertyDefinition)
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("type_id", String, func() { Format(FormatUUID) })
	Attribute("property_ids", ArrayOf(String), func() { MaxLength(100) })
	Attribute("review_id", String, func() { Format(FormatUUID) })
	Attribute("permanent", Boolean)
	Required("revision", "action")
})

var propertyReview = Type("PropertyReview", func() {
	Attribute("id", String)
	Attribute("items", Int)
	Attribute("types", Int)
	Attribute("projects", Int)
	Required("id", "items", "types", "projects")
})

func propertyMethods() {
	for _, action := range []string{"PropertyCatalog", "MutateProperty", "PreviewPropertyRemoval"} {
		Method(action, func() {
			SessionSecurity(session)
			Payload(func() {
				Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
				Required("slug")
				if action != "PropertyCatalog" {
					Attribute("data", propertyInput)
					Required("data")
				}
			})
			if action == "PreviewPropertyRemoval" {
				Result(propertyReview)
			} else {
				Result(propertyCatalog)
			}
			for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
				Error("status_"+strconv.Itoa(code), failure)
			}
			HTTP(func() {
				path := "/api/workspaces/{slug}/work-item-properties/"
				if action == "PreviewPropertyRemoval" {
					path += "removal-preview/"
				}
				if action == "PropertyCatalog" {
					GET(path)
				} else {
					POST(path)
					Body("data")
				}
				Param("slug")
				Response(StatusOK)
				for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
					Response("status_"+strconv.Itoa(code), code)
				}
			})
		})
	}
}
