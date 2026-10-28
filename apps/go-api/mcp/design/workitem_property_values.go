package design

import . "github.com/CaliLuke/loom/dsl"

func propertyValueTarget(action string) {
	propertyTarget(action, true, false, false)
	Attribute("workitem_id", String, func() { Format(FormatUUID) })
	Required("project_id", "workitem_id")
}

var propertyGetValue = Type("WorkitemPropertyGetValue", func() { propertyValueTarget("get_value"); Attribute("include_revision", Boolean) })
var propertySetValue = Type("WorkitemPropertySetValue", func() {
	propertyValueTarget("set_value")
	// JSON carrier until loom#1036 supports the nested scalar/string-array union.
	Attribute("value", Any, "A string, boolean, number, or array of selection IDs. Zero and false are values.")
	Attribute("description_html", String, "Use instead of value for a rich text property.")
	Attribute("external_source", String, func() { MaxLength(255) })
	Attribute("external_id", String, func() { MaxLength(255) })
	mutationExtensions(true)
})
var propertyDeleteValue = Type("WorkitemPropertyDeleteValue", func() { propertyValueTarget("delete_value"); mutationExtensions(true) })
var propertyRichValue = Type("WorkitemPropertyRichValue", func() {
	Attribute("id", String, func() { Nullable() })
	Attribute("description_html", String)
	Attribute("description_stripped", String)
	Required("id", "description_html", "description_stripped")
	Meta("openapi:additionalProperties", "false")
})
var propertyValueRecord = Type("WorkitemPropertyValueRecord", func() {
	for _, key := range []string{"id", "property_id", "issue_id"} {
		Attribute(key, String)
		Required(key)
	}
	Attribute("value", Any, func() { Nullable() })
	Required("value")
	nullablePropertyFields([]string{"value_type", "external_id", "external_source", "created_at", "updated_at"})
	Attribute("value_detail", propertyRichValue, func() { Nullable() })
	Required("value_detail")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
