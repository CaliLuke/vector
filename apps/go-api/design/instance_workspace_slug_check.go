package design

import . "github.com/CaliLuke/loom/dsl"

var planeAdminSessionCookie = APIKeySecurity("plane_admin_session_cookie", func() {
	Description("Plane Django instance-admin session cookie.")
})

func instanceAdminSessionPayload() {
	APIKey("plane_admin_session_cookie", "session_id", String)
}

var instanceWorkspaceSlugAvailability = Type("InstanceWorkspaceSlugAvailability", func() {
	Attribute("status", Boolean)
	Required("status")
})

var instanceWorkspaceSlugError = Type("InstanceWorkspaceSlugError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var instanceWorkspaceSlugUnauthenticated = Type("InstanceWorkspaceSlugUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var instanceWorkspaceSlugForbidden = Type("InstanceWorkspaceSlugForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var instanceWorkspaceSlugInternalError = Type("InstanceWorkspaceSlugInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("instance_workspace_slug_check", func() {
	Description("Checks whether an instance administrator can use a workspace slug.")

	Method("check", func() {
		Security(planeAdminSessionCookie)
		Payload(func() {
			instanceAdminSessionPayload()
			Attribute("slug", String)
		})
		Result(instanceWorkspaceSlugAvailability)
		Error("invalid_slug", instanceWorkspaceSlugError)
		Error("unauthenticated", instanceWorkspaceSlugUnauthenticated)
		Error("forbidden", instanceWorkspaceSlugForbidden)
		Error("internal_error", instanceWorkspaceSlugInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/instances/workspace-slug-check/")
			Cookie("session_id:admin-session-id")
			Param("slug")
			Response(StatusOK)
			Response("invalid_slug", StatusBadRequest)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
