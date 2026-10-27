package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

var moduleList = Type("ModuleList", func() {
	Attribute("action", String, func() { Enum("list") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("archived", Boolean)
	Attribute("cursor", String)
	Attribute("order_by", String)
	Attribute("per_page", Int, "Zero or omitted uses 1000.", func() { Minimum(0); Maximum(1000) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	Meta("openapi:additionalProperties", "false")
})

var moduleRetrieve = Type("ModuleRetrieve", func() {
	Attribute("action", String, func() { Enum("retrieve") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("module_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id", "module_id")
	Meta("openapi:additionalProperties", "false")
})

// One typed carrier preserves lite/detail field omission without a nested
// untagged union (Loom #1036). Remove it when nested unions are supported.
var moduleRecord = Type("ModuleRecord", func() {
	Attribute("name", String)
	Required("name")
	for _, field := range []string{"id", "created_at", "updated_at", "deleted_at", "description", "start_date", "target_date", "external_source", "external_id", "archived_at", "created_by", "updated_by", "project", "workspace", "lead"} {
		Attribute(field, String, func() { Nullable() })
		Required(field)
	}
	Attribute("status", String, func() { Nullable(); Enum("backlog", "planned", "in-progress", "paused", "completed", "cancelled") })
	Required("status")
	for _, field := range []string{"description_text", "description_html", "view_props", "logo_props"} {
		Attribute(field, Any, func() { Nullable() })
		Required(field)
	}
	Attribute("sort_order", Float64, func() { Nullable() })
	Required("sort_order")
	Attribute("members", ArrayOf(String), func() { Nullable() })
	Required("members")
	for _, field := range []string{"total_issues", "cancelled_issues", "completed_issues", "started_issues", "unstarted_issues", "backlog_issues"} {
		Attribute(field, Int64, func() { Nullable() })
	}

	Meta("openapi:additionalProperties", "false")
})
var modulePage = Type("ModulePage", func() {
	membertypes.MemberPageFields()
	Attribute("results", ArrayOf(moduleRecord))
	Meta("openapi:additionalProperties", "false")
})
var moduleRequest = Type("ModuleRequest", OneOf(moduleList, moduleRetrieve), func() { Untagged() })
var moduleResult = Type("ModuleResult", OneOf(modulePage, moduleRecord), func() { Untagged() })

func module() {
	Method("module", func() {
		Description("Modules in a project. List active or archived modules, or retrieve an active module.")
		Payload(moduleRequest)
		Result(moduleResult)
		readErrors()
		Meta("mcp:annotation:readOnlyHint", "true")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("module", "Modules in a project. Use an advertised action.", mcp.ToolTitle("Modules"))
	})
}
