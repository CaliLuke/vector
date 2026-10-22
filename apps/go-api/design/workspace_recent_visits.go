package design

import . "github.com/CaliLuke/loom/dsl"

var workspaceRecentVisit = Type("WorkspaceRecentVisit", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("entity_name", String, func() { MaxLength(30) })
	Attribute("entity_identifier", String, func() { Format(FormatUUID); Nullable() })
	Attribute("entity_data", Any, func() { Nullable() })
	Attribute("visited_at", String, func() { Format(FormatDateTime) })
	Required("id", "entity_name", "entity_identifier", "entity_data", "visited_at")
})

var workspaceRecentVisitUnauthenticated = Type("WorkspaceRecentVisitUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceRecentVisitForbidden = Type("WorkspaceRecentVisitForbidden", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceRecentVisitInternal = Type("WorkspaceRecentVisitInternal", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("workspace_recent_visits", func() {
	Method("list", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("entity_name", String)
			Required("slug")
		})
		Result(ArrayOf(workspaceRecentVisit))
		Error("unauthenticated", workspaceRecentVisitUnauthenticated)
		Error("forbidden", workspaceRecentVisitForbidden)
		Error("internal_error", workspaceRecentVisitInternal, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/recent-visits/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("entity_name")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
