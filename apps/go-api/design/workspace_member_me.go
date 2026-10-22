package design

import . "github.com/CaliLuke/loom/dsl"

var workspaceMemberMe = Type("WorkspaceMemberMe", func() {
	Attribute("company_role", String, func() { Nullable() })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("default_props", Any)
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("draft_issue_count", Int)
	Attribute("explored_features", Any)
	Attribute("getting_started_checklist", Any)
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("is_active", Boolean)
	Attribute("issue_props", Any)
	Attribute("member", String, func() { Format(FormatUUID); Nullable() })
	// Django's serializer emits role:null for the null-instance compatibility
	// response, so this field must preserve an explicit JSON null.
	Attribute("role", Int, func() { Enum(5, 15, 20); Nullable() })
	Attribute("tips", Any)
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("view_props", Any)
	Attribute("workspace", String, func() { Format(FormatUUID); Nullable() })
	Required(
		"company_role",
		"created_by",
		"default_props",
		"deleted_at",
		"explored_features",
		"getting_started_checklist",
		"is_active",
		"issue_props",
		"member",
		"role",
		"tips",
		"updated_by",
		"view_props",
		"workspace",
	)
})
var workspaceMemberMeUnauthenticated = Type("WorkspaceMemberMeUnauthenticated", func() { Attribute("detail", String); Required("detail") })
var _ = Service("workspace_member_me", func() {
	Method("get", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload(); Attribute("slug", String); Required("slug") })
		Result(workspaceMemberMe)
		Error("unauthenticated", workspaceMemberMeUnauthenticated)
		HTTP(func() {
			GET("/api/workspaces/{slug}/workspace-members/me/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
		})
	})
})
