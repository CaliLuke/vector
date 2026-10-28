package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

var dependencyKinds = []string{"blocking", "blocked_by", "start_before", "start_after", "finish_before", "finish_after"}

func relationAction(action string, item bool) {
	Attribute("action", String, func() { Enum(action) })
	Required("action")
	projectWorkspaceSelector()
	if item {
		for _, key := range []string{"project_id", "workitem_id"} {
			Attribute(key, String, func() { Format(FormatUUID) })
			Required(key)
		}
	}
	Meta("openapi:additionalProperties", "false")
}

var relationList = Type("RelationList", func() { relationAction("list", true); Attribute("include_revision", Boolean) })
var relationCreate = Type("RelationCreate", func() {
	relationAction("create", true)
	Attribute("workitem_ids", ArrayOf(String, func() { Format(FormatUUID) }), func() { MinLength(1); MaxLength(1000) })
	Required("workitem_ids")
	Attribute("relation_type", String, func() { Enum("blocking", "blocked_by", "start_before", "start_after", "finish_before", "finish_after") })
	Attribute("relation_definition_id", String, func() { Format(FormatUUID) })
	Attribute("relation_definition_label", String)
	mutationExtensions(true)
})
var relationDelete = Type("RelationDelete", func() {
	relationAction("delete", true)
	Attribute("related_workitem_id", String, func() { Format(FormatUUID) })
	Required("related_workitem_id")
	Attribute("is_dependency", Boolean)
	mutationExtensions(true)
})
var relationDefinitions = Type("RelationDefinitions", func() {
	relationAction("list_definitions", false)
	Attribute("is_default", Boolean)
	Attribute("is_active", Boolean)
	Attribute("include_revision", Boolean)
})

func definitionChanges() {
	for _, key := range []string{"name", "outward", "inward", "color"} {
		Attribute(key, String, func() { MaxLength(255) })
	}
	Attribute("is_active", Boolean)
}

var relationDefinitionCreate = Type("RelationDefinitionCreate", func() {
	relationAction("create_definition", false)
	definitionChanges()
	Required("name")
	mutationExtensions(true)
})
var relationDefinitionUpdate = Type("RelationDefinitionUpdate", func() {
	relationAction("update_definition", false)
	Attribute("definition_id", String, func() { Format(FormatUUID) })
	Required("definition_id")
	definitionChanges()
	mutationExtensions(true)
})
var relationDefinitionDelete = Type("RelationDefinitionDelete", func() {
	relationAction("delete_definition", false)
	Attribute("definition_id", String, func() { Format(FormatUUID) })
	Required("definition_id")
	mutationExtensions(true)
})
var relationItem = Type("RelationItem", func() {
	for _, key := range []string{"id", "name", "project_id", "state_id", "priority", "type_id", "created_at", "updated_at", "created_by", "updated_by", "relation_type"} {
		Attribute(key, String, func() { Nullable() })
		Required(key)
	}
	Attribute("sequence_id", Int64, func() { Nullable() })
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("is_epic", Boolean, func() { Nullable() })
	Attribute("label_ids", ArrayOf(String))
	Attribute("assignee_ids", ArrayOf(String))
	Required("sequence_id", "sort_order", "is_epic", "label_ids", "assignee_ids")
	Meta("openapi:additionalProperties", "false")
})
var relationItems = Type("RelationItems", ArrayOf(relationItem))
var relationDefinitionRecord = Type("RelationDefinitionRecord", func() {
	for _, key := range []string{"id", "name", "outward", "inward", "color", "created_at", "updated_at", "workspace"} {
		Attribute(key, String, func() { Nullable() })
		Required(key)
	}
	for _, key := range []string{"is_default", "is_active"} {
		Attribute(key, Boolean, func() { Nullable() })
		Required(key)
	}
	Attribute("sort_order", Float64, func() { Nullable() })
	Required("sort_order")
	Meta("openapi:additionalProperties", "false")
})
var relationDependencies = Type("RelationDependencies", func() {
	for _, key := range dependencyKinds {
		Attribute(key, relationItems)
		Required(key)
	}
	Meta("openapi:additionalProperties", "false")
})
var relationCollection = Type("RelationCollection", func() {
	Attribute("dependencies", relationDependencies)
	// Restore MapOf(String, relationItems) after loom#1041 is released.
	Attribute("custom", Any, "Custom definition label to related item array.")
	Attribute("revision", String)
	Required("dependencies", "custom")
	Meta("openapi:additionalProperties", "false")
})
var relationDefinitionCollection = Type("RelationDefinitionCollection", func() {
	Attribute("built_in_dependencies", ArrayOf(String))
	Attribute("custom_definitions", ArrayOf(relationDefinitionRecord))
	Attribute("revision", String)
	Required("built_in_dependencies", "custom_definitions")
	Meta("openapi:additionalProperties", "false")
})
var relationReceipt = Type("RelationMutationReceipt", func() {
	Attribute("resource_id", String)
	Attribute("revision", String)
	// Restore the typed nested union after loom#1036 is released.
	Attribute("result", Any, func() { Nullable() })
	Attribute("request_id", String)
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("resource_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var relationRequest = Type("WorkitemRelationRequest", OneOf(relationList, relationCreate, relationDelete, relationDefinitions, relationDefinitionCreate, relationDefinitionUpdate, relationDefinitionDelete), func() { Untagged() })
var relationResult = Type("WorkitemRelationResult", OneOf(relationCollection, relationItems, relationDefinitionCollection, relationDefinitionRecord, relationReceipt), func() { Untagged() })

func workitemRelation() {
	Method("workitem_relation", func() {
		Description("Read and manage six built-in dependency directions and workspace-defined custom relations. Call list_definitions to select an exact outward/inward label. create accepts relation_type, or relation_definition_id plus relation_definition_label. delete selects dependencies with is_dependency=true; the default removes custom relations. All endpoints must be writable. Workspace administrators manage definitions. include_revision lists the workspace relation revision for guarded writes.")
		Payload(relationRequest)
		Result(relationResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("workitem_relation", "List, create and delete relations; list, create, update and delete their definitions.", mcp.ToolTitle("Work item relations"))
	})
}
