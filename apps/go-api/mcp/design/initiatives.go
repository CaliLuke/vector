package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func initiativeTarget(action string) {
	if action == "list" || action == "create" {
		pageTarget(action, false)
	} else {
		pageTarget(action, false, "initiative_id")
	}
}
func initiativeFields() {
	for _, key := range []string{"name", "description_html", "start_date", "end_date", "lead"} {
		Attribute(key, String)
	}
	Attribute("state", String, func() { Enum("", "DRAFT", "PLANNED", "ACTIVE", "COMPLETED", "CLOSED") })
}

var initiativeList = Type("InitiativeList", func() { initiativeTarget("list"); Attribute("include_revision", Boolean) })
var initiativeRetrieve = Type("InitiativeRetrieve", func() { initiativeTarget("retrieve"); Attribute("include_revision", Boolean) })
var initiativeCreate = Type("InitiativeCreate", func() { initiativeTarget("create"); initiativeFields(); Required("name"); mutationExtensions(false) })
var initiativeUpdate = Type("InitiativeUpdate", func() { initiativeTarget("update"); initiativeFields(); mutationExtensions(true) })
var initiativeDelete = Type("InitiativeDelete", func() { initiativeTarget("delete"); mutationExtensions(true) })
var initiativeListProjects = Type("InitiativeListProjects", func() { initiativeTarget("list_projects"); pagePagination(); Attribute("include_revision", Boolean) })
var initiativeListWorkitems = Type("InitiativeListWorkitems", func() { initiativeTarget("list_workitems"); pagePagination(); Attribute("include_revision", Boolean) })
var initiativeAddProjects = Type("InitiativeAddProjects", func() {
	initiativeTarget("add_projects")
	Attribute("project_ids", String)
	Required("project_ids")
	mutationExtensions(true)
})
var initiativeRemoveProjects = Type("InitiativeRemoveProjects", func() {
	initiativeTarget("remove_projects")
	Attribute("project_ids", String)
	Required("project_ids")
	mutationExtensions(true)
})
var initiativeManageWorkitems = Type("InitiativeManageWorkitems", func() {
	initiativeTarget("manage_workitems")
	Attribute("add_ids", String)
	Attribute("remove_ids", String)
	mutationExtensions(true)
})
var initiativeRecord = Type("InitiativeRecord", func() {
	Attribute("id", String)
	Attribute("name", String)
	Attribute("workspace", String)
	Attribute("logo_props", Any)
	Required("id", "name", "workspace", "logo_props")
	nullableStrings("description", "description_html", "description_stripped", "description_binary", "start_date", "end_date", "state", "lead", "created_at", "updated_at", "deleted_at")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var initiativeRecords = Type("InitiativeRecords", ArrayOf(initiativeRecord))

// One page carrier avoids ambiguous empty result arrays and nested unions until
// https://github.com/CaliLuke/loom/issues/1036 is fixed. Service projections validate
// each project or work-item record against its generated native SDK contract.
var initiativePage = Type("InitiativePage", func() {
	toolkitPageFields()
	Attribute("results", ArrayOf(Any))
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var initiativeRequest = Type("InitiativeRequest", OneOf(initiativeList, initiativeRetrieve, initiativeCreate, initiativeUpdate, initiativeDelete, initiativeListProjects, initiativeListWorkitems, initiativeAddProjects, initiativeRemoveProjects, initiativeManageWorkitems), func() { Untagged() })
var initiativeResult = Type("InitiativeResult", OneOf(initiativeRecord, initiativeRecords, initiativePage, pageMutationReceipt), func() { Untagged() })

func initiatives() {
	Method("initiative", func() {
		Description("Workspace initiatives, gated by workspace initiatives feature settings. List returns all initiatives. Dates use YYYY-MM-DD and state is DRAFT, PLANNED, ACTIVE, COMPLETED or CLOSED. Workspace administrators edit the catalogue; changing associations also requires writable project membership. Project/item lists filter current visibility. Project links and work-item links are independent. manage_workitems applies removals before additions; association writes return null. Delete keeps projects/items and removes their links. Optional receipts and revisions support retries.")
		Payload(initiativeRequest)
		Result(initiativeResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Error("feature_disabled", func() {
			Remedy(func() {
				SafeMessage("Initiatives are disabled. A workspace administrator can enable initiatives using workspace update_features. For work-item equivalents, resolve the Initiative work-item type in the intended project; project/initiative links require native initiatives.")
			})
		})
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("initiative", "Manage initiatives and their project/work-item associations.", mcp.ToolTitle("Initiatives"))
	})
}
