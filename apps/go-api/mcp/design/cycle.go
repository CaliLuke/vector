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
	projectWorkspaceSelector()
	Required("action", "project_id", "cycle_id")
	Meta("openapi:additionalProperties", "false")
})

// One typed carrier preserves lite/detail field omission without a nested
// untagged union (Loom #1036). Remove it when nested unions are supported.
var cycleRecord = Type("CycleRecord", func() {
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
var cycleRequest = Type("CycleRequest", OneOf(cycleList, cycleRetrieve), func() { Untagged() })
var cycleResult = Type("CycleResult", OneOf(cyclePage, cycleRecord), func() { Untagged() })

func cycle() {
	Method("cycle", func() {
		Description("Cycles in a project. List active or archived cycles, or retrieve an active cycle.")
		Payload(cycleRequest)
		Result(cycleResult)
		readErrors()
		Meta("mcp:annotation:readOnlyHint", "true")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("cycle", "Cycles in a project. Use an advertised action.", mcp.ToolTitle("Cycles"))
	})
}
