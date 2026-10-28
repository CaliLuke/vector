package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func workitemTarget(action string, detail bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	if detail {
		Attribute("workitem_id", String, func() { Format(FormatUUID) })
		Required("workitem_id")
	}
	Meta("openapi:additionalProperties", "false")
}
func workitemWriteFields() {
	for _, key := range []string{"name", "type_id", "description_html", "description_stripped", "priority", "parent", "state", "estimate_point", "external_source", "external_id"} {
		Attribute(key, String)
	}
	for _, key := range []string{"assignees", "labels"} {
		Attribute(key, ArrayOf(String), func() { Nullable() })
	}
	for _, key := range []string{"start_date", "target_date"} {
		Attribute(key, String, "Omitted or empty leaves unchanged; null clears the date.", func() { Nullable() })
	}
	Attribute("point", Int)
	Attribute("sort_order", Float64)
	Attribute("is_draft", Boolean, func() { Nullable() })
}

var workitemCreate = Type("WorkitemCreate", func() {
	workitemTarget("create", false)
	workitemWriteFields()
	Required("name")
	mutationExtensions(false)
})
var workitemUpdate = Type("WorkitemUpdate", func() { workitemTarget("update", true); workitemWriteFields(); mutationExtensions(true) })
var workitemDelete = Type("WorkitemDelete", func() { workitemTarget("delete", true); mutationExtensions(true) })
var workitemArchive = Type("WorkitemArchive", func() {
	workitemTarget("archive", true)
	Attribute("archive", Boolean, func() { Nullable() })
	mutationExtensions(true)
})
var workitemManageAssignee = Type("WorkitemManageAssignee", func() {
	workitemTarget("manage_assignee", true)
	Attribute("add_user_id", String)
	Attribute("remove_user_id", String)
	mutationExtensions(true)
})
var workitemManageLabel = Type("WorkitemManageLabel", func() {
	workitemTarget("manage_label", true)
	Attribute("add_label_id", String)
	Attribute("remove_label_id", String)
	mutationExtensions(true)
})

// Mutations use the SDK's unexpanded work-item response. Read projections with
// sparse or expanded fields are authored separately when their actions are added.
var workitemRecord = Type("WorkitemRecord", func() {
	for _, key := range []string{"id", "type_id", "created_at", "updated_at", "deleted_at", "name", "description_html", "description_stripped", "description_binary", "priority", "start_date", "target_date", "completed_at", "archived_at", "external_source", "external_id", "created_by", "updated_by", "project", "workspace", "parent", "state", "estimate_point", "type"} {
		Attribute(key, String, func() { Nullable() })
		Required(key)
	}
	for _, key := range []string{"point", "sequence_id"} {
		Attribute(key, Int64, func() { Nullable() })
		Required(key)
	}
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("is_draft", Boolean, func() { Nullable() })
	Required("sort_order", "is_draft")
	for _, key := range []string{"assignees", "labels"} {
		Attribute(key, ArrayOf(String))
		Required(key)
	}
	// https://github.com/CaliLuke/loom/issues/1041 tracks maps inside untagged result branches.
	// Restore MapOf(String, Any) when a released generator accepts that shape.
	Attribute("custom_properties", Any)
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var workitemArchiveStatus = Type("WorkitemArchiveStatus", func() {
	Attribute("workitem_id", String, func() { Format(FormatUUID) })
	Attribute("archived", Boolean)
	Required("workitem_id", "archived")
	Meta("openapi:additionalProperties", "false")
})
var workitemReceipt = Type("WorkitemMutationReceipt", func() {
	Attribute("workitem_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("result", workitemRecord, func() { Nullable() })
	Attribute("archived", Boolean)
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("workitem_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var workitemRequest = Type("WorkitemRequest", OneOf(workitemCount, workitemList, workitemListArchived, workitemRetrieve, workitemIdentifier, workitemSearch, workitemCreate, workitemUpdate, workitemDelete, workitemArchive, workitemManageAssignee, workitemManageLabel), func() { Untagged() })
var workitemResult = Type("WorkitemResult", OneOf(workitemCountResult, workitemReadRecord, workitemPage, workitemSearchResult, workitemPQLFailure, workitemArchiveStatus, workitemReceipt), func() { Untagged() })

func workitem() {
	Method("workitem", func() {
		Description("Read, search, create and change work items. Lists support PQL, sparse fields, expanded relations and cursor pagination. Assignee and label management removes first, then adds, atomically. Dates accept null to clear. Archive only accepts completed or cancelled items; archive=false restores them.")
		Payload(workitemRequest)
		Result(workitemResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("workitem", "Work items: issues, tasks and epics. Use an advertised action.", mcp.ToolTitle("Work items"))
	})
}
