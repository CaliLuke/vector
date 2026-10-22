package design

import . "github.com/CaliLuke/loom/dsl"

var pageRead = Type("PageRead", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("owned_by", String, func() { Format(FormatUUID) })
	Attribute("access", Int, func() { Enum(0, 1) })
	Attribute("color", String)
	Attribute("parent", String, func() { Format(FormatUUID); Nullable() })
	Attribute("is_favorite", Boolean)
	Attribute("is_locked", Boolean)
	Attribute("archived_at", String, func() { Format(FormatDate); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("view_props", Any)
	Attribute("logo_props", Any)
	Attribute("label_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("project_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Required(
		"id", "name", "owned_by", "access", "color", "parent", "is_favorite", "is_locked",
		"archived_at", "workspace", "created_at", "updated_at", "created_by", "updated_by",
		"view_props", "logo_props", "label_ids", "project_ids",
	)
})

var pageDetailRead = Type("PageDetailRead", func() {
	Extend(pageRead)
	Attribute("description_html", String)
	Attribute("issue_ids", ArrayOf(String, func() { Format(FormatUUID); Nullable() }))
	Required("description_html", "issue_ids")
})

var pageSummaryRead = Type("PageSummaryRead", func() {
	Attribute("public_pages", Int64)
	Attribute("private_pages", Int64)
	Attribute("archived_pages", Int64)
	Required("public_pages", "private_pages", "archived_pages")
})

var pageVersionRead = Type("PageVersionRead", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("page", String, func() { Format(FormatUUID) })
	Attribute("last_saved_at", String, func() { Format(FormatDateTime) })
	Attribute("owned_by", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Required(
		"id", "workspace", "page", "last_saved_at", "owned_by", "created_at", "updated_at",
		"created_by", "updated_by",
	)
})

var pageVersionDetailRead = Type("PageVersionDetailRead", func() {
	Extend(pageVersionRead)
	Attribute("description_binary", Bytes, func() { Meta("openapi:format", "byte"); Nullable() })
	Attribute("description_html", String)
	Attribute("description_json", Any)
	Required("description_binary", "description_html", "description_json")
})

var pageDescriptionMetadata = Type("PageDescriptionMetadata", func() {
	Attribute("disposition", String)
	Required("disposition")
})
var pageOptionsMetadata = Type("PageOptionsMetadata", func() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("renders", ArrayOf(String))
	Attribute("parses", ArrayOf(String))
	Required("name", "description", "renders", "parses")
})

var pageReadsUnauthenticated = Type("PageReadsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var pageReadsForbidden = Type("PageReadsForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var pageReadsBadRequest = Type("PageReadsBadRequest", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var pageReadsNotFound = Type("PageReadsNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var pageReadsInternalError = Type("PageReadsInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("page_reads", func() {
	Description("Returns authenticated project page and page-version read models.")
	HTTP(func() {
		RuntimeCORS()
	})

	Method("summary", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_pages_summary")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id")
		})
		Result(rawJSONResponseMetadata)
		pageReadErrors(false, false)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/pages-summary/")
			rawJSONReadResponse(pageSummaryRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			pageReadResponses(false, false)
		})
	})
	Method("summary_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_projects_by_project_id_pages_summary")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id")
		})
		Result(rawJSONResponseMetadata)
		pageReadErrors(false, false)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/pages-summary/")
			rawJSONReadResponse(pageOptionsMetadata)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			pageReadResponses(false, false)
		})
	})

	Method("list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_pages")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("search", String)
			Attribute("order_by", String)
			Required("slug", "project_id")
		})
		Result(rawJSONResponseMetadata)
		pageReadErrors(false, false)
		Error("invalid_search", ArrayOf(String))
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/pages/")
			rawJSONReadResponse(ArrayOf(pageRead))
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("search")
			Param("order_by")
			pageReadResponses(false, false)
			Response("invalid_search", StatusBadRequest)
		})
	})

	Method("detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("page_id", String, func() { Format(FormatUUID) })
			Attribute("search", String)
			Attribute("order_by", String)
			Attribute("track_visit", String)
			Required("slug", "project_id", "page_id")
		})
		Result(rawJSONResponseMetadata)
		pageReadErrors(true, true)
		Error("invalid_search", ArrayOf(String))
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/")
			rawJSONReadResponse(pageDetailRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			Param("search")
			Param("order_by")
			Param("track_visit")
			pageReadResponses(true, true)
			Response("invalid_search", StatusBadRequest)
		})
	})

	Method("description", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_description(#{routeIndex})")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("page_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "page_id")
		})
		Result(pageDescriptionMetadata)
		pageReadErrors(false, true)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/")
			HEAD("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			FileResponse()
			Response(StatusOK, func() {
				ContentType("application/octet-stream")
				Header("disposition:Content-Disposition")
			})
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("version_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_versions")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("page_id", String, func() { Format(FormatUUID) })
			Attribute("search", String)
			Required("slug", "project_id", "page_id")
		})
		Result(rawJSONResponseMetadata)
		pageReadErrors(false, true)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/")
			rawJSONReadResponse(ArrayOf(pageVersionRead))
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			Param("search")
			pageReadResponses(false, true)
		})
	})
	Method("version_list_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_versions")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("page_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "page_id")
		})
		Result(rawJSONResponseMetadata)
		pageReadErrors(false, false)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/")
			rawJSONReadResponse(pageOptionsMetadata)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			pageReadResponses(false, false)
		})
	})

	Method("version_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_versions_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("page_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "page_id", "pk")
		})
		Result(rawJSONResponseMetadata)
		pageReadErrors(false, true)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/")
			rawJSONReadResponse(pageVersionDetailRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			Param("pk")
			pageReadResponses(false, true)
		})
	})
	Method("version_detail_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_versions_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("page_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String)
			Required("slug", "project_id", "page_id", "pk")
		})
		Result(rawJSONResponseMetadata)
		pageReadErrors(false, false)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/versions/{pk}/")
			rawJSONReadResponse(pageOptionsMetadata)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			Param("pk")
			pageReadResponses(false, false)
		})
	})

	pageLifecycleMethods()
})

func pageReadErrors(badRequest, notFound bool) {
	Error("unauthenticated", pageReadsUnauthenticated)
	Error("forbidden", pageReadsForbidden)
	if badRequest {
		Error("bad_request", pageReadsBadRequest)
	}
	if notFound {
		Error("not_found", pageReadsNotFound)
	}
	Error("internal_error", pageReadsInternalError, func() { Fault() })
}

func pageReadResponses(badRequest, notFound bool) {
	Response("unauthenticated", StatusUnauthorized)
	Response("forbidden", StatusForbidden)
	if badRequest {
		Response("bad_request", StatusBadRequest)
	}
	if notFound {
		Response("not_found", StatusNotFound)
	}
	Response("internal_error", StatusInternalServerError)
}
