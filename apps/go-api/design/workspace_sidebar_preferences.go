package design

import . "github.com/CaliLuke/loom/dsl"

var workspaceSidebarPreference = Type("WorkspaceSidebarPreference", func() {
	Attribute("is_pinned", Boolean)
	Attribute("sort_order", Float64)
	Required("is_pinned", "sort_order")
})

var workspaceSidebarPreferencePatch = Type("WorkspaceSidebarPreferencePatch", func() {
	Attribute("key", String, func() { MaxLength(255) })
	Attribute("is_pinned", Boolean, func() { Nullable() })
	Attribute("sort_order", Float64, func() { Nullable() })
})

var workspaceSidebarPreferenceMessage = Type("WorkspaceSidebarPreferenceMessage", func() {
	Attribute("message", String)
	Required("message")
})

var workspaceSidebarPreferenceUnauthenticated = Type("WorkspaceSidebarPreferenceUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceSidebarPreferenceForbidden = Type("WorkspaceSidebarPreferenceForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceSidebarPreferenceInternal = Type("WorkspaceSidebarPreferenceInternal", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("workspace_sidebar_preferences", func() {
	Description("Manages the authenticated user's workspace sidebar preferences.")

	Method("get", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Required("slug")
		})
		Result(MapOf(String, workspaceSidebarPreference))
		Error("unauthenticated", workspaceSidebarPreferenceUnauthenticated)
		Error("forbidden", workspaceSidebarPreferenceForbidden)
		Error("internal_error", workspaceSidebarPreferenceInternal, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/sidebar-preferences/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("patch", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("body", ArrayOf(workspaceSidebarPreferencePatch))
			Required("slug", "body")
		})
		Result(workspaceSidebarPreferenceMessage)
		Error("unauthenticated", workspaceSidebarPreferenceUnauthenticated)
		Error("forbidden", workspaceSidebarPreferenceForbidden)
		Error("internal_error", workspaceSidebarPreferenceInternal, func() { Fault() })
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/sidebar-preferences/")
			Cookie("session_id:session-id")
			Param("slug")
			Body("body")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
