package design

import (
	. "github.com/CaliLuke/loom/dsl"
)

func propertyTarget(action string, propertyID, optionID, types bool) {
	Attribute("action", String, func() { Enum(action) })
	Required("action")
	projectWorkspaceSelector()
	Attribute("project_id", String, "Optional project scope; definitions are workspace owned.")
	if types {
		Attribute("workitem_type_id", String)
	}
	if propertyID {
		key := "property_id"
		if action == "retrieve" || action == "update" || action == "delete" {
			key = "workitem_property_id"
		}
		Attribute(key, String, func() { Format(FormatUUID) })
		Required(key)
	}
	if optionID {
		Attribute("option_id", String, func() { Format(FormatUUID) })
		Required("option_id")
	}
	Meta("openapi:additionalProperties", "false")
}
func propertyPaging() {
	Attribute("cursor", String)
	Attribute("per_page", Int, func() { Minimum(0); Maximum(1000) })
}

func propertyDefinitionFields(create bool) {
	for _, key := range []string{"display_name", "description", "default_value", "display_format", "external_source", "external_id"} {
		Attribute(key, String)
	}
	Attribute("property_type", String, func() {
		Enum("TEXT", "DATETIME", "DECIMAL", "BOOLEAN", "OPTION", "RELATION", "URL", "EMAIL", "FILE", "FORMULA", "CASCADING")
	})
	Attribute("relation_type", String, func() { Enum("ISSUE", "USER", "RELEASE", "RICH_TEXT") })
	configuration := propertyUpdateConfiguration
	if create {
		configuration = propertyCreateConfiguration
	}
	Attribute("configuration", configuration, "Vector extension: FORMULA creation requires expression and result; CASCADING creation requires two to four levels and its option tree; readonly TEXT can supply content.")
	for _, key := range []string{"is_required", "is_multi", "is_active"} {
		Attribute(key, Boolean)
	}
	if create {
		Attribute("options", String, "JSON array of option objects.")
		Required("display_name", "property_type")
	}
	mutationExtensions(true)
}
func propertyOptionFields(create bool) {
	for _, key := range []string{"name", "description", "color", "external_source", "external_id"} {
		Attribute(key, String)
	}
	Attribute("is_default", Boolean)
	if create {
		Required("name")
	}
	mutationExtensions(true)
}

var propertyList = Type("WorkitemPropertyList", func() {
	propertyTarget("list", false, false, true)
	propertyPaging()
	Attribute("include_revision", Boolean)
})
var propertyRetrieve = Type("WorkitemPropertyRetrieve", func() { propertyTarget("retrieve", true, false, true); Attribute("include_revision", Boolean) })
var propertyCreate = Type("WorkitemPropertyCreate", func() { propertyTarget("create", false, false, true); propertyDefinitionFields(true) })
var propertyUpdate = Type("WorkitemPropertyUpdate", func() { propertyTarget("update", true, false, true); propertyDefinitionFields(false) })
var propertyDelete = Type("WorkitemPropertyDelete", func() { propertyTarget("delete", true, false, true); mutationExtensions(true) })
var propertyManage = Type("WorkitemPropertyManage", func() {
	propertyTarget("manage_type_properties", false, false, true)
	Required("workitem_type_id")
	Attribute("attach_ids", String)
	Attribute("detach_ids", String)
	mutationExtensions(true)
})
var propertyOptions = Type("WorkitemPropertyOptions", func() {
	propertyTarget("list_options", true, false, false)
	propertyPaging()
	Attribute("include_revision", Boolean)
})
var propertyOptionRetrieve = Type("WorkitemPropertyOptionRetrieve", func() { propertyTarget("retrieve_option", true, true, false); Attribute("include_revision", Boolean) })
var propertyOptionCreate = Type("WorkitemPropertyOptionCreate", func() { propertyTarget("create_option", true, false, false); propertyOptionFields(true) })
var propertyOptionUpdate = Type("WorkitemPropertyOptionUpdate", func() { propertyTarget("update_option", true, true, false); propertyOptionFields(false) })
var propertyOptionDelete = Type("WorkitemPropertyOptionDelete", func() { propertyTarget("delete_option", true, true, false); mutationExtensions(true) })

func nullablePropertyFields(keys []string) {
	for _, key := range keys {
		Attribute(key, String, func() { Nullable() })
		Required(key)
	}
}

var propertyOptionRecord = Type("WorkitemPropertyOptionRecord", func() {
	nullablePropertyFields([]string{"id", "deleted_at", "created_at", "updated_at", "description", "external_source", "external_id", "created_by", "updated_by", "workspace", "project", "property", "parent"})
	Attribute("name", String)
	Required("name")
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("logo_props", Any, func() { Nullable() })
	Attribute("is_active", Boolean, func() { Nullable() })
	Attribute("is_default", Boolean, func() { Nullable() })
	Required("sort_order", "logo_props", "is_active", "is_default")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var propertyRecord = Type("WorkitemPropertyRecord", func() {
	nullablePropertyFields([]string{"id", "deleted_at", "relation_type", "created_at", "updated_at", "name", "description", "external_source", "external_id", "created_by", "updated_by", "workspace", "project", "issue_type"})
	Attribute("display_name", String)
	Attribute("property_type", String)
	Required("display_name", "property_type")
	Attribute("sort_order", Float64, func() { Nullable() })
	Required("sort_order")
	for _, key := range []string{"logo_props", "settings", "validation_rules"} {
		Attribute(key, Any, func() { Nullable() })
		Required(key)
	}
	for _, key := range []string{"is_required", "is_active", "is_multi"} {
		Attribute(key, Boolean, func() { Nullable() })
		Required(key)
	}
	Attribute("default_value", ArrayOf(String), func() { Nullable() })
	Required("default_value")
	Attribute("options", ArrayOf(propertyOptionRecord), func() { Nullable() })
	Required("options")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})

// One array carrier preserves empty [] for property, option and attached-ID lists.
// Replace JSON items with the nested record/string union after loom#1036 ships.
var propertyArray = Type("WorkitemPropertyArray", ArrayOf(Any))
var propertyReceipt = Type("WorkitemPropertyMutationReceipt", func() {
	Attribute("resource_id", String)
	Attribute("revision", String)
	Attribute("result", Any, func() { Nullable() })
	Attribute("request_id", String)
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("resource_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var propertyRequest = Type("WorkitemPropertyRequest", OneOf(propertyList, propertyRetrieve, propertyCreate, propertyUpdate, propertyDelete, propertyManage, propertyOptions, propertyOptionRetrieve, propertyOptionCreate, propertyOptionUpdate, propertyOptionDelete, propertyGetValue, propertySetValue, propertyDeleteValue), func() { Untagged() })
var propertyResult = Type("WorkitemPropertyResult", OneOf(propertyRecord, propertyOptionRecord, propertyArray, propertyReceipt, propertyValueRecord), func() { Untagged() })

func workitemProperty() {
	Method("workitem_property", func() {
		Description("Custom property definitions, options and type associations. Omit project_id for workspace-owned changes. Workspace administrators manage definitions and associations. Native validation and preservation rules apply; used values are never silently discarded. Type-scoped empty lists fall back to project-flat then workspace type properties. include_revision returns the current catalogue revision for guarded writes.")
		Payload(propertyRequest)
		Result(propertyResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Error("workspace_owned", func() {
			Remedy(func() {
				SafeMessage("Properties are managed by the workspace. Omit project_id for definition and option changes.")
			})
		})
		Error("conflict", func() {
			Remedy(func() {
				SafeMessage("The operation conflicts with the current property configuration or requires preservation review.")
			})
		})
		mutationTool("workitem_property", "Work item properties", "Manage property definitions, options and type associations.")
	})
}
