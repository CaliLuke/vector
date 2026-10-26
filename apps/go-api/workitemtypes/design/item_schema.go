package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var resolvedProperty = Type("ResolvedProperty", func() {
	for _, name := range []string{"id", "name", "title", "description", "icon", "kind", "context_id"} {
		Attribute(name, String)
	}
	Attribute("configuration", propertyConfiguration)
	Required("id", "name", "title", "description", "icon", "kind", "context_id", "configuration")
})
var itemReferenceChoice = Type("ItemReferenceChoice", func() {
	Attribute("id", String)
	Attribute("name", String)
	Attribute("eligible", Boolean)
	Attribute("reason", String)
	Required("id", "name", "eligible", "reason")
})
var formulaResult = Type("FormulaResult", func() {
	Attribute("kind", String)
	Attribute("status", String, func() { Enum("value", "empty", "error") })
	Attribute("value", Any)
	Attribute("error", String)
	Required("kind", "status", "error")
})
var itemSchema = Type("ItemSchema", func() {
	Attribute("stored_values", MapOf(String, Any))
	Attribute("item_updated_at", String, func() { Format(FormatDateTime) })
	Attribute("computed", MapOf(String, formulaResult))
	Attribute("evaluated_at", String, func() { Format(FormatDateTime) })
	Attribute("enabled", Boolean)
	Attribute("revision", Int64)
	Attribute("type_id", String)
	Attribute("eligible_type_ids", ArrayOf(String))
	Attribute("selection_reason", String)
	Attribute("properties", ArrayOf(resolvedProperty))
	Attribute("members", ArrayOf(itemReferenceChoice))
	Attribute("releases", ArrayOf(itemReferenceChoice))
	Required("stored_values", "enabled", "revision", "type_id", "properties", "members", "releases", "computed", "evaluated_at", "eligible_type_ids", "selection_reason")
})

var itemFormulaPreview = Type("ItemFormulaPreview", func() {
	Attribute("name", String, func() { MaxLength(255) })
	Attribute("priority", String, func() { Enum("none", "low", "medium", "high", "urgent") })
	for _, name := range []string{"start_date", "target_date"} {
		Attribute(name, String, func() { Pattern(`^(|[0-9]{4}-[0-9]{2}-[0-9]{2})$`) })
	}
	Attribute("estimate_point", String, func() { Pattern(`^(|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`) })
	Attribute("custom_properties", MapOf(String, Any), func() { MaxLength(1000) })
})

func itemSchemaMethod() {
	for _, name := range []string{"ItemSchema", "PreviewItemSchema"} {
		Method(name, func() {
			SessionSecurity(session)
			Payload(func() {
				schemaScopePayload()
				if name == "PreviewItemSchema" {
					Attribute("preview", itemFormulaPreview)
					Required("preview")
				}
			})
			Result(itemSchema)
			for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
				Error("status_"+strconv.Itoa(code), failure)
			}
			HTTP(func() {
				if name == "ItemSchema" {
					GET("/api/workspaces/{slug}/work-item-schema/")
				} else {
					POST("/api/workspaces/{slug}/work-item-schema/preview/")
					Body("preview")
				}
				for _, param := range []string{"slug", "mode", "project_id", "issue_id", "draft_id", "type_id", "parent_id"} {
					Param(param)
				}
				Response(StatusOK)
				for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
					Response("status_"+strconv.Itoa(code), code)
				}
			})
		})
	}
}
func schemaScopePayload() {
	Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
	Attribute("mode", String, func() { Enum("create", "edit", "draft", "publish") })
	for _, name := range []string{"project_id", "issue_id", "draft_id"} {
		Attribute(name, String, func() { Format(FormatUUID) })
	}
	for _, name := range []string{"type_id", "parent_id"} {
		Attribute(name, String, func() { Pattern(`^(|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`) })
	}
	Required("slug", "mode")
}
