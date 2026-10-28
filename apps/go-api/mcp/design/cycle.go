package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

var cycleList = Type("CycleList", func() {
	Attribute("action", String, func() { Enum("list") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("archived", Boolean)
	Attribute("status", String, "Active cycle status: current, upcoming, completed, draft, or incomplete. Ignored for archived cycles.")
	Attribute("cursor", String)
	Attribute("order_by", String)
	Attribute("per_page", Int, "Zero or omitted uses 1000.", func() { Minimum(0); Maximum(1000) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	Meta("openapi:additionalProperties", "false")
})

var cycleRetrieve = Type("CycleRetrieve", func() {
	Attribute("action", String, func() { Enum("retrieve") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("cycle_id", String, func() { Format(FormatUUID) })
	Attribute("include_revision", Boolean)
	projectWorkspaceSelector()
	Required("action", "project_id", "cycle_id")
	Meta("openapi:additionalProperties", "false")
})

// One typed carrier preserves lite/detail field omission without a nested
// untagged union (Loom #1036). Remove it when nested unions are supported.
var cycleRecord = Type("CycleRecord", func() {
	Attribute("revision", String)
	Attribute("name", String)
	Required("name")
	for _, field := range []string{"id", "created_at", "updated_at", "deleted_at", "description", "start_date", "end_date", "external_source", "external_id", "archived_at", "timezone", "created_by", "updated_by", "project", "workspace", "owned_by"} {
		Attribute(field, String, func() { Nullable() })
		Required(field)
	}
	for _, field := range []string{"view_props", "progress_snapshot", "logo_props"} {
		Attribute(field, Any, func() { Nullable() })
		Required(field)
	}
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("version", Int, func() { Nullable() })
	Required("sort_order", "version")
	for _, field := range []string{"total_issues", "cancelled_issues", "completed_issues", "started_issues", "unstarted_issues", "backlog_issues", "total_estimates", "completed_estimates", "started_estimates"} {
		Attribute(field, Int64, func() { Nullable() })
	}
	Meta("openapi:additionalProperties", "false")
})
var cyclePage = Type("CyclePage", func() {
	membertypes.MemberPageFields()
	Attribute("results", ArrayOf(cycleRecord))
	Meta("openapi:additionalProperties", "false")
})
var cycleRequest = Type("CycleRequest", OneOf(cycleListWorkitems, cycleList, cycleRetrieve, cycleCreate, cycleUpdate, cycleArchive, cycleManageWorkitems, cycleTransferWorkitems), func() { Untagged() })
var cycleResult = Type("CycleResult", OneOf(workitemPage, workitemPQLFailure, cyclePage, cycleRecord, cycleReceipt), func() { Untagged() })

func cycle() {
	Method("cycle", func() {
		Description("Cycles in a project. Use an advertised action.")
		Payload(cycleRequest)
		Result(cycleResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("cycle", "Cycles in a project. Use an advertised action.", mcp.ToolTitle("Cycles"))
	})
}

var cycleArchive = Type("CycleArchive", func() {
	Attribute("action", String, func() { Enum("archive", "unarchive", "complete", "delete") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("cycle_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	mutationExtensions(true)
	Required("action", "project_id", "cycle_id")
	Meta("openapi:additionalProperties", "false")
})
var cycleReceipt = Type("CycleMutationReceipt", func() {
	Attribute("cycle_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("result", cycleRecord, func() { Nullable() })
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("cycle_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})

func cycleWriteFields() {
	for _, field := range []string{"name", "description", "start_date", "external_source", "external_id", "end_date", "owned_by", "timezone"} {
		Attribute(field, String)
	}
}

var cycleCreate = Type("CycleCreate", func() {
	Attribute("action", String, func() { Enum("create") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	cycleWriteFields()
	mutationExtensions(false)
	Required("action", "project_id")
	Required("name", "owned_by")
	Meta("openapi:additionalProperties", "false")
})
var cycleUpdate = Type("CycleUpdate", func() {
	Attribute("action", String, func() { Enum("update") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	cycleWriteFields()
	mutationExtensions(true)
	Required("action", "project_id")
	Attribute("cycle_id", String, func() { Format(FormatUUID) })
	Required("cycle_id")
	Meta("openapi:additionalProperties", "false")
})

var cycleManageWorkitems = Type("CycleManageWorkitems", func() {
	Attribute("action", String, func() { Enum("manage_workitems") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("cycle_id", String, func() { Format(FormatUUID) })
	Attribute("add_ids", String, "Work item UUIDs as a comma-separated string or a JSON array encoded as a string.")
	Attribute("remove_ids", String, "Work item UUIDs as a comma-separated string or a JSON array encoded as a string. Removal wins on overlap.")
	projectWorkspaceSelector()
	mutationExtensions(true)
	Required("action", "project_id", "cycle_id")
	Meta("openapi:additionalProperties", "false")
})

var cycleTransferWorkitems = Type("CycleTransferWorkitems", func() {
	Attribute("action", String, func() { Enum("transfer_workitems") })
	for _, field := range []string{"project_id", "cycle_id", "new_cycle_id"} {
		Attribute(field, String, func() { Format(FormatUUID) })
	}
	projectWorkspaceSelector()
	mutationExtensions(true)
	Required("action", "project_id", "cycle_id", "new_cycle_id")
	Meta("openapi:additionalProperties", "false")
})
