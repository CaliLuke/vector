package design

import . "github.com/CaliLuke/loom/dsl"

var projectLabelRead = Type("ProjectLabelRead", func() {
	Attribute("parent", String, func() { Format(FormatUUID); Nullable() })
	Attribute("name", String)
	Attribute("color", String)
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Attribute("sort_order", Float64)
	Required("parent", "name", "color", "id", "project_id", "workspace_id", "sort_order")
})

var projectLabelReadsUnauthenticated = Type("ProjectLabelReadsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var projectLabelReadsForbidden = Type("ProjectLabelReadsForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var projectLabelReadsNotFound = Type("ProjectLabelReadsNotFound", func() {
	Attribute("detail", String)
	Required("detail")
})

var projectLabelReadsInternalError = Type("ProjectLabelReadsInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("project_label_reads", func() {
	Description("Returns authenticated project issue-label read models.")

	Method("list", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id")
		})
		Result(ArrayOf(projectLabelRead))
		Error("unauthenticated", projectLabelReadsUnauthenticated)
		Error("forbidden", projectLabelReadsForbidden)
		Error("internal_error", projectLabelReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/issue-labels/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("detail", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "pk")
		})
		Result(projectLabelRead)
		Error("unauthenticated", projectLabelReadsUnauthenticated)
		Error("forbidden", projectLabelReadsForbidden)
		Error("not_found", projectLabelReadsNotFound)
		Error("internal_error", projectLabelReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
