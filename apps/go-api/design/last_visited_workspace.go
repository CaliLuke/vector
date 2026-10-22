package design

import . "github.com/CaliLuke/loom/dsl"

var lastVisitedWorkspaceResult = Type("LastVisitedWorkspaceResult", func() {
	Attribute("project_details", Any)
	Attribute("workspace_details", Any)
	Required("project_details", "workspace_details")
})
var lastVisitedWorkspaceUnauthenticated = Type("LastVisitedWorkspaceUnauthenticated", func() { Attribute("detail", String); Required("detail") })
var lastVisitedWorkspaceInternal = Type("LastVisitedWorkspaceInternal", func() { Attribute("error", String, func() { Meta("struct:field:name", "Message") }); Required("error") })

var _ = Service("last_visited_workspace", func() {
	Method("get", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload() })
		Result(lastVisitedWorkspaceResult)
		Error("unauthenticated", lastVisitedWorkspaceUnauthenticated)
		Error("internal_error", lastVisitedWorkspaceInternal, func() { Fault() })
		HTTP(func() {
			GET("/api/users/last-visited-workspace/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
