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
	projectWorkspaceSelector()
	Required("action", "project_id")
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

var projectRequest = Type("ProjectRequest", OneOf(projectList, projectRetrieve), func() { Untagged() })
var projectResult = Type("ProjectResult", OneOf(projectPage, projectDetail), func() { Untagged() })

func project() {
	Method("project", func() {
		Description("List authorized projects or retrieve one full project.")
		Payload(projectRequest)
		Result(projectResult)
		readErrors()
		Meta("mcp:annotation:readOnlyHint", "true")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("project", "List projects or retrieve full detail using the selected action.", mcp.ToolTitle("Projects"))
	})
}
