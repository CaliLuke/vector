package design

import . "github.com/CaliLuke/loom/dsl"

var planeSessionCookie = APIKeySecurity("plane_session_cookie", func() {
	Description("Plane Django session cookie.")
})

var planeDjangoSession = SessionAuth("plane_django_session", func() {
	CookieTransport(planeSessionCookie, "", func() {
		CookieName("session-id")
	})
})

var stickyResponse = Type("StickyResponse", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("owner", String, func() { Format(FormatUUID) })
	Attribute("created_by", String, func() { Format(FormatUUID); Meta("openapi:extension:x-nullable", "true") })
	Attribute("updated_by", String, func() { Format(FormatUUID); Meta("openapi:extension:x-nullable", "true") })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Meta("openapi:extension:x-nullable", "true") })
	Attribute("name", String, func() { Meta("openapi:extension:x-nullable", "true") })
	Attribute("color", String, func() { MaxLength(255); Meta("openapi:extension:x-nullable", "true") })
	Attribute("background_color", String, func() { MaxLength(255); Meta("openapi:extension:x-nullable", "true") })
	Attribute("description", Any)
	Attribute("description_html", String)
	Attribute("description_stripped", String, func() { Meta("openapi:extension:x-nullable", "true") })
	Attribute("description_binary", Bytes, func() {
		Meta("openapi:format", "binary")
		ReadOnly()
		Nullable()
	})
	Attribute("logo_props", Any)
	Attribute("sort_order", Float64)
	Required(
		"id", "workspace", "owner", "created_at", "updated_at", "description", "description_html",
		"logo_props", "sort_order",
	)
})

var stickyPage = Type("StickyPage", func() {
	Attribute("grouped_by", String, func() { Meta("openapi:extension:x-nullable", "true") })
	Attribute("sub_grouped_by", String, func() { Meta("openapi:extension:x-nullable", "true") })
	Attribute("total_count", Int, func() { Example(1) })
	Attribute("next_cursor", String)
	Attribute("prev_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_page_results", Boolean)
	Attribute("count", Int, func() { Example(1) })
	Attribute("total_pages", Int, func() { Example(1) })
	Attribute("total_results", Int, func() { Example(1) })
	Attribute("extra_stats", String, func() { Meta("openapi:extension:x-nullable", "true") })
	Attribute("results", ArrayOf(stickyResponse))
	Required(
		"total_count", "next_cursor", "prev_cursor",
		"next_page_results", "prev_page_results", "count", "total_pages", "total_results",
		"results",
	)
})

var invalidPaginationError = Type("StickyInvalidPaginationError", func() {
	Attribute("detail", String)
	Required("detail")
})

var unauthenticatedError = Type("StickyUnauthenticatedError", func() {
	Attribute("detail", String)
	Required("detail")
})

var notFoundError = Type("StickyNotFoundError", func() {
	Attribute("detail", String)
	Required("detail")
})

var permissionError = Type("StickyPermissionError", func() {
	Attribute("message", String, func() {
		Meta("struct:tag:json", "error")
		Meta("struct:tag:form", "error")
		Meta("struct:tag:xml", "error")
	})
	Required("message")
})

var validationErrors = Type("StickyValidationErrors", MapOf(String, ArrayOf(String)))

func stickyWritePayload() {
	Attribute("background_color", String, func() { MaxLength(255); Nullable() })
	Attribute("color", String, func() { MaxLength(255); Nullable() })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("description", Any)
	Attribute("description_html", String)
	Attribute("description_stripped", String, func() { Nullable() })
	Attribute("logo_props", Any)
	Attribute("name", String, func() { Nullable() })
	Attribute("sort_order", Float64)
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
}

func stickySessionPayload() {
	APIKey("plane_session_cookie", "session_id", String)
}

var _ = Service("stickies", func() {
	Description("Serves the reviewed frontend Sticky API.")
	HTTP(func() {
		RuntimeCORS()
	})
	Method("list", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("cursor", String)
			Attribute("per_page", Int, func() { Maximum(1000); Example(20) })
			Attribute("query", String)
			Required("slug")
		})
		Result(stickyPage, func() {
			Example(map[string]any{
				"total_count": 1, "next_cursor": "20:0:0", "prev_cursor": "20:0:0",
				"next_page_results": false, "prev_page_results": false, "count": 1,
				"total_pages": 1, "total_results": 1, "results": []any{},
			})
		})
		Error("invalid_pagination", invalidPaginationError)
		Error("unauthenticated", unauthenticatedError)
		Error("forbidden", permissionError)
		HTTP(func() {
			GET("/api/workspaces/{slug}/stickies/")
			Cookie("session_id:session-id")
			Param("cursor")
			Param("per_page")
			Param("query")
			Response(StatusOK)
			Response("invalid_pagination", StatusBadRequest)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
		})
	})

	Method("create", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			stickyWritePayload()
			Required("slug")
		})
		Result(stickyResponse)
		Error("validation_failed", validationErrors)
		Error("unauthenticated", unauthenticatedError)
		Error("forbidden", permissionError)
		Error("payload_too_large", String)
		HTTP(func() {
			POST("/api/workspaces/{slug}/stickies/")
			Cookie("session_id:session-id")
			Response(StatusCreated)
			Response("validation_failed", StatusBadRequest)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("payload_too_large", StatusRequestEntityTooLarge)
		})
	})

	Method("retrieve", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "pk")
		})
		Result(stickyResponse)
		Error("unauthenticated", unauthenticatedError)
		Error("not_found", notFoundError)
		HTTP(func() {
			GET("/api/workspaces/{slug}/stickies/{pk}/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
		})
	})

	Method("update", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			stickyWritePayload()
			Required("slug", "pk")
		})
		Result(stickyResponse)
		Error("validation_failed", validationErrors)
		Error("unauthenticated", unauthenticatedError)
		Error("forbidden", permissionError)
		Error("not_found", notFoundError)
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/stickies/{pk}/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("validation_failed", StatusBadRequest)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("not_found", StatusNotFound)
		})
	})

	Method("delete", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "pk")
		})
		Error("unauthenticated", unauthenticatedError)
		Error("forbidden", permissionError)
		Error("not_found", notFoundError)
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/stickies/{pk}/")
			Cookie("session_id:session-id")
			Response(StatusNoContent)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("not_found", StatusNotFound)
		})
	})
})
