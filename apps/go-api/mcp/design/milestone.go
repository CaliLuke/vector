package design

import (
	milestonetypes "github.com/CaliLuke/go-plane/milestones/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func milestoneTarget(action string, detail bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	if detail {
		Attribute("milestone_id", String, func() { Format(FormatUUID) })
		Required("milestone_id")
	}
	Meta("openapi:additionalProperties", "false")
}
func milestonePaging() {
	Attribute("cursor", String)
	Attribute("per_page", Int, "Zero or omitted uses 1000.", func() { Minimum(0); Maximum(1000) })
}
func milestoneFields() {
	// Do not constrain target_date with FormatDate here: the pinned exact empty
	// string means omission. The native operation validates nonempty values.
	for _, name := range []string{"title", "target_date", "external_source", "external_id"} {
		Attribute(name, String)
	}
	milestonetypes.MutationExtensions()
}

var milestoneList = Type("MilestoneList", func() { milestoneTarget("list", false); milestonePaging() })
var milestoneRetrieve = Type("MilestoneRetrieve", func() {
	milestoneTarget("retrieve", true)
	Attribute("include_revision", Boolean)
})
var milestoneCreate = Type("MilestoneCreate", func() {
	milestoneTarget("create", false)
	milestoneFields()
	Required("title")
})
var milestoneUpdate = Type("MilestoneUpdate", func() { milestoneTarget("update", true); milestoneFields() })
var milestoneDelete = Type("MilestoneDelete", func() { milestoneTarget("delete", true); milestonetypes.MutationExtensions() })
var milestoneListWorkitems = Type("MilestoneListWorkitems", func() { milestoneTarget("list_workitems", true); milestonePaging() })
var milestoneManageWorkitems = Type("MilestoneManageWorkitems", func() {
	milestoneTarget("manage_workitems", true)
	Attribute("add_ids", String, "Work item UUIDs as a comma-separated string or a JSON array encoded as a string.")
	Attribute("remove_ids", String, "Work item UUIDs as a comma-separated string or a JSON array encoded as a string. Removal wins on overlap.")
	milestonetypes.MutationExtensions()
})

// A typed carrier avoids nested untagged unions (https://github.com/CaliLuke/loom/issues/1036).
// Projection emits exactly a seven-field milestone or three-field association,
// never a mixture.
// Remove this carrier when the minimum Loom release supports nested unions.
var milestonePageEntry = Type("MilestonePageEntry", func() {
	Attribute("id", String, func() { Nullable(); Format(FormatUUID) })
	Required("id")
	for _, name := range []string{"title", "created_at", "updated_at", "issue", "milestone"} {
		Attribute(name, String)
	}
	for _, name := range []string{"target_date", "external_source", "external_id"} {
		Attribute(name, String, func() { Nullable() })
	}
	Meta("openapi:additionalProperties", "false")
})
var milestonePage = Type("MilestonePage", func() {
	Attribute("results", ArrayOf(milestonePageEntry))
	Attribute("count", Int)
	Attribute("total_count", Int)
	Attribute("next_cursor", String)
	Attribute("prev_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_page_results", Boolean)
	Required("results", "count", "total_count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results")
	Meta("openapi:additionalProperties", "false")
})
var milestoneRequest = Type("MilestoneRequest", OneOf(milestoneList, milestoneRetrieve, milestoneCreate, milestoneUpdate, milestoneDelete, milestoneListWorkitems, milestoneManageWorkitems), func() { Untagged() })
var milestoneResult = Type("MilestoneResult", OneOf(milestonePage, milestonetypes.Record, milestonetypes.MutationReceipt), func() { Untagged() })

func milestone() {
	Method("milestone", func() {
		Description("Milestones within a project. Use an advertised action.")
		Payload(milestoneRequest)
		Result(milestoneResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Error("conflict", func() { Remedy(func() { SafeMessage("The operation conflicts with an existing resource.") }) })
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("milestone", "Milestones within a project. Use an advertised action.", mcp.ToolTitle("Milestones"))
	})
}
