package design

import . "github.com/CaliLuke/loom/dsl"

var estimatePointRead = Type("EstimatePointRead", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("estimate", String, func() { Format(FormatUUID) })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("key", Int)
	Attribute("value", String)
	Attribute("description", String)
	Required("id", "created_at", "updated_at", "created_by", "updated_by", "deleted_at", "estimate", "workspace", "project", "key", "value", "description")
})

var estimateRead = Type("EstimateRead", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("description", String)
	Attribute("type", String, func() { Enum("categories", "points") })
	Attribute("last_used", Boolean)
	Attribute("points", ArrayOf(estimatePointRead))
	Required("id", "created_at", "updated_at", "created_by", "updated_by", "deleted_at", "workspace", "project", "name", "description", "type", "last_used", "points")
})

var estimateReadsUnauthenticated = Type("EstimateReadsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var estimateReadsForbidden = Type("EstimateReadsForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var estimateReadsNotFound = Type("EstimateReadsNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var estimateReadsInternalError = Type("EstimateReadsInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("estimate_reads", func() {
	Description("Returns authenticated workspace and project estimate read models with ordered points.")
	HTTP(func() {
		RuntimeCORS()
	})

	Method("workspace_selected", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_estimates")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("search", String, func() { Description("Accepted for compatibility and intentionally ignored.") })
			Required("slug")
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", estimateReadsUnauthenticated)
		Error("forbidden", estimateReadsForbidden)
		Error("internal_error", estimateReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/estimates/")
			rawJSONReadResponse(ArrayOf(estimateRead))
			Cookie("session_id:session-id")
			Param("slug")
			Param("search")
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("workspace_selected_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_estimates")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Required("slug")
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", estimateReadsUnauthenticated)
		Error("forbidden", estimateReadsForbidden)
		Error("internal_error", estimateReadsInternalError, func() { Fault() })
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/estimates/")
			rawJSONReadResponse(Any)
			Cookie("session_id:session-id")
			Param("slug")
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("project_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_estimates")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("search", String, func() { Description("Accepted for compatibility and intentionally ignored.") })
			Required("slug", "project_id")
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", estimateReadsUnauthenticated)
		Error("forbidden", estimateReadsForbidden)
		Error("internal_error", estimateReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/estimates/")
			rawJSONReadResponse(ArrayOf(estimateRead))
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("search")
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("project_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_estimates_by_estimate_id")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("estimate_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "estimate_id")
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", estimateReadsUnauthenticated)
		Error("forbidden", estimateReadsForbidden)
		Error("not_found", estimateReadsNotFound)
		Error("internal_error", estimateReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/")
			rawJSONReadResponse(estimateRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("estimate_id")
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
