package design

import . "github.com/CaliLuke/loom/dsl"

// These are the exact fields emitted by LabelSerializer and StateSerializer
// for the workspace collection endpoints.  Keep UUIDs as strings because the
// Django JSON representation is the canonical wire format.
var workspaceLabel = Type("WorkspaceLabel", func() {
	Attribute("parent", String, func() { Format(FormatUUID); Nullable() })
	Attribute("name", String)
	Attribute("color", String)
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Attribute("sort_order", Float64)
	Required("parent", "name", "color", "id", "project_id", "workspace_id", "sort_order")
})

var workspaceState = Type("WorkspaceState", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("color", String)
	Attribute("group", String)
	Attribute("default", Boolean)
	Attribute("description", String, func() { Nullable() })
	Attribute("sequence", Float64)
	Attribute("order", Float64, func() { Nullable() })
	Required("id", "project_id", "workspace_id", "name", "color", "group", "default", "description", "sequence", "order")
})

var workspaceMetadataUnauthenticated = Type("WorkspaceMetadataUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceMetadataForbidden = Type("WorkspaceMetadataForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceMetadataInternal = Type("WorkspaceMetadataInternal", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

func workspaceMetadataPayload() {
	stickySessionPayload()
	Attribute("slug", String)
	Required("slug")
}

func workspaceMetadataHTTP(path string) {
	GET(path)
	Cookie("session_id:session-id")
	Param("slug")
	Response(StatusOK)
	Response("unauthenticated", StatusUnauthorized)
	Response("forbidden", StatusForbidden)
	Response("internal_error", StatusInternalServerError)
}

var _ = Service("workspace_metadata", func() {
	Description("Lists labels and states visible in the authenticated user's workspace projects.")

	Method("labels", func() {
		Security(planeSessionCookie)
		Payload(workspaceMetadataPayload)
		Result(ArrayOf(workspaceLabel))
		Error("unauthenticated", workspaceMetadataUnauthenticated)
		Error("forbidden", workspaceMetadataForbidden)
		Error("internal_error", workspaceMetadataInternal, func() { Fault() })
		HTTP(func() { workspaceMetadataHTTP("/api/workspaces/{slug}/labels/") })
	})

	Method("states", func() {
		Security(planeSessionCookie)
		Payload(workspaceMetadataPayload)
		Result(ArrayOf(workspaceState))
		Error("unauthenticated", workspaceMetadataUnauthenticated)
		Error("forbidden", workspaceMetadataForbidden)
		Error("internal_error", workspaceMetadataInternal, func() { Fault() })
		HTTP(func() { workspaceMetadataHTTP("/api/workspaces/{slug}/states/") })
	})
})
