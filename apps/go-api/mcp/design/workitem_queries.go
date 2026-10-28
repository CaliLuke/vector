package design

import . "github.com/CaliLuke/loom/dsl"

func workitemQueryFields(page bool, external bool) {
	for _, key := range []string{"order_by", "expand", "fields"} {
		Attribute(key, String)
	}
	if external {
		Attribute("external_id", String)
		Attribute("external_source", String)
	}
	if page {
		Attribute("pql", String, "PQL predicate; call get_pql_reference for syntax.")
		Attribute("cursor", String)
		Attribute("per_page", Int, "Zero or omitted uses the native page size.", func() { Minimum(0); Maximum(1000) })
	}
}

var workitemList = Type("WorkitemList", func() {
	Attribute("action", String, func() { Enum("list") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	workitemQueryFields(true, true)
	Required("action")
	Meta("openapi:additionalProperties", "false")
})
var workitemListArchived = Type("WorkitemListArchived", func() {
	workitemTarget("list_archived", false)
	workitemQueryFields(true, true)
})
var workitemRetrieve = Type("WorkitemRetrieve", func() {
	workitemTarget("retrieve", true)
	workitemQueryFields(false, true)
	Attribute("include_revision", Boolean)
})
var workitemIdentifier = Type("WorkitemIdentifier", func() {
	Attribute("action", String, func() { Enum("retrieve_by_identifier") })
	Attribute("workitem_identifier", String)
	projectWorkspaceSelector()
	workitemQueryFields(false, true)
	Required("action", "workitem_identifier")
	Meta("openapi:additionalProperties", "false")
})
var workitemSearch = Type("WorkitemSearch", func() {
	Attribute("action", String, func() { Enum("search") })
	Attribute("query", String, func() { MinLength(1) })
	projectWorkspaceSelector()
	workitemQueryFields(false, true)
	Required("action", "query")
	Meta("openapi:additionalProperties", "false")
})

// Optional fields preserve sparse pages. Polymorphic relation fields use JSON
// carriers until https://github.com/CaliLuke/loom/issues/1036 supports nested
// untagged unions; restore their ID/object unions after that generator release.
var workitemReadRecord = Type("WorkitemReadRecord", func() {
	for _, key := range []string{"id", "type_id", "created_at", "updated_at", "deleted_at", "name", "description_html", "description_stripped", "description_binary", "priority", "start_date", "target_date", "completed_at", "archived_at", "external_source", "external_id"} {
		Attribute(key, String, func() { Nullable() })
	}
	for _, key := range []string{"created_by", "updated_by", "workspace", "parent", "estimate_point", "project", "state", "type"} {
		Attribute(key, Any, func() { Nullable() })
	}
	Attribute("point", Int64, func() { Nullable() })
	Attribute("sequence_id", Int64, func() { Nullable() })
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("is_draft", Boolean, func() { Nullable() })
	Attribute("assignees", ArrayOf(Any))
	Attribute("labels", ArrayOf(Any))
	// Map fields under untagged result branches: https://github.com/CaliLuke/loom/issues/1041.
	Attribute("custom_properties", Any)
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var workitemPage = Type("WorkitemPage", func() {
	Attribute("results", ArrayOf(workitemReadRecord))
	for _, key := range []string{"count", "total_count"} {
		Attribute(key, Int)
	}
	for _, key := range []string{"next_cursor", "prev_cursor"} {
		Attribute(key, String, func() { Nullable() })
	}
	Attribute("next_page_results", Boolean)
	Attribute("prev_page_results", Boolean)
	Required("results", "count", "total_count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results")
	Meta("openapi:additionalProperties", "false")
})
var workitemSearchItem = Type("WorkitemSearchItem", func() {
	for _, key := range []string{"id", "name", "project__identifier", "project_id", "workspace__slug"} {
		Attribute(key, String)
		Required(key)
	}
	Attribute("sequence_id", Int64)
	Required("sequence_id")
	Meta("openapi:additionalProperties", "false")
})
var workitemSearchResult = Type("WorkitemSearchResult", func() {
	Attribute("issues", ArrayOf(workitemSearchItem))
	Required("issues")
	Meta("openapi:additionalProperties", "false")
})
var workitemPQLFailure = Type("WorkitemPQLFailure", func() {
	Attribute("error", Any)
	Attribute("failed_pql", String)
	Attribute("pql_reference", String)
	Attribute("hint", String)
	Required("error", "failed_pql", "pql_reference", "hint")
	Meta("openapi:additionalProperties", "false")
})
var cycleListWorkitems = Type("CycleListWorkitems", func() {
	Attribute("action", String, func() { Enum("list_workitems") })
	Attribute("cycle_id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	workitemQueryFields(true, false)
	Required("action", "project_id", "cycle_id")
	Meta("openapi:additionalProperties", "false")
})
var moduleListWorkitems = Type("ModuleListWorkitems", func() {
	Attribute("action", String, func() { Enum("list_workitems") })
	Attribute("module_id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	workitemQueryFields(true, false)
	Required("action", "project_id", "module_id")
	Meta("openapi:additionalProperties", "false")
})

var workitemCount = Type("WorkitemCount", func() {
	Attribute("action", String, func() { Enum("count") })
	Required("action")
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Attribute("pql", String)
	for _, key := range []string{"group_by", "sub_group_by"} {
		Attribute(key, String, func() {
			Enum("", "state_id", "state__group", "priority", "project_id", "type_id", "labels__id", "assignees__id", "issue_module__module_id", "release_work_items__release_id", "cycle_id", "milestone_id", "created_by", "target_date", "start_date")
		})
	}
	Meta("openapi:additionalProperties", "false")
})
var workitemCountResult = Type("WorkitemCountResult", func() {
	nullableStrings("grouped_by", "sub_grouped_by")
	Attribute("total_count", Int64)
	// Raw grouping keys need a map under an untagged union; restore MapOf after
	// https://github.com/CaliLuke/loom/issues/1041 is released and wire checks pass.
	Attribute("grouped_counts", Any)
	Required("total_count", "grouped_counts")
	Meta("openapi:additionalProperties", "false")
})
