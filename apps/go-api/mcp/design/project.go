package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func projectWorkspaceSelector() {
	Attribute("workspace_slug", String, func() { MaxLength(255) })
}

var projectList = Type("ProjectList", func() {
	Attribute("action", String, func() { Enum("list") })
	Attribute("cursor", String)
	Attribute("order_by", String)
	Attribute("per_page", Int, "Zero or omitted uses 100.", func() { Minimum(0); Maximum(1000) })
	projectWorkspaceSelector()
	Required("action")
	Meta("openapi:additionalProperties", "false")
})

var projectRetrieve = Type("ProjectRetrieve", func() {
	Attribute("action", String, func() { Enum("retrieve") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("include_revision", Boolean, "Vector extension: include an opaque revision token for guarded writes.")
	projectWorkspaceSelector()
	Required("action", "project_id")
	Meta("openapi:additionalProperties", "false")
})

var projectArchive = Type("ProjectArchive", func() {
	Attribute("action", String, func() { Enum("archive", "unarchive") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("request_id", String, "Optional Vector retry identity; retained for at least 24 hours.", func() { Format(FormatUUID) })
	Attribute("expected_revision", String, "Optional opaque Vector revision from retrieve with include_revision. Conflicts fail atomically.", func() { MinLength(1); MaxLength(128) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	Meta("openapi:additionalProperties", "false")
})

var projectMutationReceipt = Type("ProjectMutationReceipt", func() {
	Description("Vector extension result, returned only when request_id or expected_revision is supplied. Record mutations include result; archive and restore retain their metadata-only receipt.")
	Attribute("result", projectDetail, func() { Nullable() })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String, func() { Format(FormatDateTime) })
	Attribute("replayed", Boolean)
	Required("project_id", "revision")
	Meta("openapi:additionalProperties", "false")
})

var projectLite = Type("ProjectLite", func() {
	Attribute("name", String)
	Attribute("identifier", String)
	Attribute("id", String, func() { Nullable(); Format(FormatUUID) })
	for _, field := range []string{"cover_image", "emoji", "description", "cover_image_url", "archived_at"} {
		Attribute(field, String, func() { Nullable() })
		Required(field)
	}
	Attribute("icon_prop", Any, func() { Nullable() })
	Required("id", "name", "identifier", "icon_prop")
	Meta("openapi:additionalProperties", "false")
})

var projectPage = Type("ProjectPage", func() {
	membertypes.MemberPageFields()
	Attribute("results", ArrayOf(projectLite))
	Meta("openapi:additionalProperties", "false")
})

var projectDetail = Type("ProjectDetail", func() {
	Extend(projectLite)
	Attribute("revision", String, "Vector extension, present only when include_revision was requested.")
	for _, field := range []string{"total_members", "total_cycles", "total_modules", "member_role", "archive_in", "close_in"} {
		Attribute(field, Int, func() { Nullable() })
		Required(field)
	}
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("network", Int, func() { Nullable(); Enum(0, 2) })
	Required("sort_order", "network")
	for _, field := range []string{"is_member", "is_deployed", "module_view", "cycle_view", "issue_views_view", "page_view", "intake_view", "is_time_tracking_enabled", "is_issue_type_enabled", "guest_view_all_features"} {
		Attribute(field, Boolean, func() { Nullable() })
		Required(field)
	}
	for _, field := range []string{"created_at", "updated_at", "deleted_at", "timezone", "external_source", "external_id", "created_by", "updated_by", "workspace", "default_assignee", "project_lead", "cover_image_asset", "estimate", "default_state"} {
		Attribute(field, String, func() { Nullable() })
		Required(field)
	}
	for _, field := range []string{"description_text", "description_html", "logo_props"} {
		Attribute(field, Any, func() { Nullable() })
		Required(field)
	}
	Meta("openapi:additionalProperties", "false")
})

var projectRequest = Type("ProjectRequest", OneOf(projectList, projectRetrieve, projectArchive, projectCreate, projectUpdate, projectDelete, projectWorklogSummary, projectGetFeatures, projectUpdateFeatures), func() { Untagged() })
var projectResult = Type("ProjectResult", OneOf(projectPage, projectDetail, projectMutationReceipt, projectWorklogTotals, projectFeatureRecord, pageMutationReceipt), func() { Untagged() })

func project() {
	Method("project", func() {
		Description("Projects in a workspace. Use an advertised action.")
		Payload(projectRequest)
		Result(projectResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("project", "Projects in a workspace. Use an advertised action.", mcp.ToolTitle("Projects"))
	})
}

func writeErrors() {
	Error("request_conflict", func() { Remedy(func() { SafeMessage("This request_id was already used with different input.") }) })
	Error("revision_conflict", func() {
		Remedy(func() { SafeMessage("The resource changed. Read its current revision before trying a new update.") })
	})
	Error("indeterminate", func() {
		Remedy(func() {
			SafeMessage("The commit outcome is unknown. Retry with the same request_id, or reconcile with a read before another unkeyed write.")
		})
	})
}

func projectWriteFields() {
	for _, field := range []string{"name", "identifier", "description", "description_html", "project_lead", "default_assignee", "emoji", "cover_image", "timezone", "external_source", "external_id"} {
		Attribute(field, String)
	}
	for _, field := range []string{"modules", "cycles", "views", "pages", "intakes", "workitem_types"} {
		Attribute(field, Boolean, func() { Nullable() })
	}
	Attribute("archive_in", Int, "Zero or omitted leaves the setting unchanged.")
	Attribute("close_in", Int, "Zero or omitted leaves the setting unchanged.")
	projectWorkspaceSelector()
	Meta("openapi:additionalProperties", "false")
}

var projectCreate = Type("ProjectCreate", func() {
	Attribute("action", String, func() { Enum("create") })
	projectWriteFields()
	mutationExtensions(false)
	Required("action", "name", "identifier")
})
var projectUpdate = Type("ProjectUpdate", func() {
	Attribute("action", String, func() { Enum("update") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWriteFields()
	Attribute("network", Int, func() { Nullable(); Enum(0, 2) })
	Attribute("is_time_tracking_enabled", Boolean, func() { Nullable() })
	Attribute("default_state", String)
	Attribute("estimate", String)
	mutationExtensions(true)
	Required("action", "project_id")
})
var projectDelete = Type("ProjectDelete", func() {
	Attribute("action", String, func() { Enum("delete") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	mutationExtensions(true)
	Required("action", "project_id")
	Meta("openapi:additionalProperties", "false")
})
