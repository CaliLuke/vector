package design

import . "github.com/CaliLuke/loom/dsl"

var workspaceSlugAvailability = Type("WorkspaceSlugAvailability", func() {
	Attribute("status", Boolean)
	Required("status")
})

var workspaceSlugError = Type("WorkspaceSlugError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceSlugUnauthenticated = Type("WorkspaceSlugUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceSlugInternalError = Type("WorkspaceSlugInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("workspace_slug_check", func() {
	Description("Checks whether an authenticated user can use a workspace slug.")

	Method("check", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
		})
		Result(workspaceSlugAvailability)
		Error("invalid_slug", workspaceSlugError)
		Error("unauthenticated", workspaceSlugUnauthenticated)
		Error("internal_error", workspaceSlugInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspace-slug-check/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			Response("invalid_slug", StatusBadRequest)
			Response("unauthenticated", StatusUnauthorized)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
