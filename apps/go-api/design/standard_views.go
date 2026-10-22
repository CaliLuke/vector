package design

import . "github.com/CaliLuke/loom/dsl"

// standardView mirrors IssueViewSerializer's complete representation.  The
// project and workspace fields are identifiers because DRF serializes those
// relations by primary key when no nested representation is requested.
var standardView = Type("StandardView", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("name", String)
	Attribute("description", String)
	Attribute("query", Any)
	Attribute("filters", Any)
	Attribute("display_filters", Any)
	Attribute("display_properties", Any)
	Attribute("rich_filters", Any)
	Attribute("access", Int)
	Attribute("sort_order", Float64)
	Attribute("logo_props", Any)
	Attribute("owned_by", String, func() { Format(FormatUUID) })
	Attribute("is_locked", Boolean)
	Attribute("archived_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID); Nullable() })
	// Only project-scoped list/retrieve responses expose is_favorite.  Keep the
	// field optional so workspace responses and mutation responses match DRF.
	Attribute("is_favorite", Boolean, func() { Nullable() })
	Required("id", "created_at", "updated_at", "deleted_at", "name", "description", "query", "filters", "display_filters", "display_properties", "rich_filters", "access", "sort_order", "logo_props", "owned_by", "is_locked", "archived_at", "created_by", "updated_by", "workspace", "project")
})

var standardViewUnauthenticated = Type("StandardViewUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var standardViewForbidden = Type("StandardViewForbidden", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var standardViewNotFound = Type("StandardViewNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var standardViewInvalid = Type("StandardViewInvalid", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var standardViewInternal = Type("StandardViewInternal", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

func standardViewErrors() {
	Error("unauthenticated", standardViewUnauthenticated)
	Error("forbidden", standardViewForbidden)
	Error("not_found", standardViewNotFound)
	Error("invalid", standardViewInvalid)
	Error("internal_error", standardViewInternal, func() { Fault() })
}

func standardViewResponses() {
	Response("unauthenticated", StatusUnauthorized)
	Response("forbidden", StatusForbidden)
	Response("not_found", StatusNotFound)
	Response("invalid", StatusBadRequest)
	Response("internal_error", StatusInternalServerError)
}

func standardViewWritableFields() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("filters", Any, func() { Nullable() })
	Attribute("display_filters", Any, func() { Nullable() })
	Attribute("display_properties", Any, func() { Nullable() })
	Attribute("rich_filters", Any, func() { Nullable() })
	Attribute("logo_props", Any, func() { Nullable() })
	Attribute("archived_at", String, func() { Format(FormatDateTime); Nullable() })
}

var _ = Service("standard_views", func() {
	Method("list_project", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id")
		})
		Result(ArrayOf(standardView))
		standardViewErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/views/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Response(StatusOK)
			standardViewResponses()
		})
	})
	Method("create_project", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			standardViewWritableFields()
			Required("slug", "project_id", "name")
		})
		Result(standardView)
		standardViewErrors()
		Error("invalid_json", jsonFieldValidation)
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/views/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Response("invalid_json", StatusBadRequest)
			Response(StatusCreated)
			standardViewResponses()
		})
	})
	Method("retrieve_project", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "pk")
		})
		Result(standardView)
		standardViewErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/views/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(StatusOK)
			standardViewResponses()
		})
	})
	Method("update_project", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			standardViewWritableFields()
			Required("slug", "project_id", "pk")
		})
		Result(standardView)
		standardViewErrors()
		Error("invalid_json", jsonFieldValidation)
		HTTP(func() {
			PUT("/api/workspaces/{slug}/projects/{project_id}/views/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response("invalid_json", StatusBadRequest)
			Response(StatusOK)
			standardViewResponses()
		})
	})
	Method("patch_project", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			standardViewWritableFields()
			Required("slug", "project_id", "pk")
		})
		Result(standardView)
		standardViewErrors()
		Error("invalid_json", jsonFieldValidation)
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/projects/{project_id}/views/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response("invalid_json", StatusBadRequest)
			Response(StatusOK)
			standardViewResponses()
		})
	})
	Method("delete_project", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "pk")
		})
		standardViewErrors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/projects/{project_id}/views/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(StatusNoContent)
			standardViewResponses()
		})
	})
	Method("list_workspace", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload(); Attribute("slug", String); Required("slug") })
		Result(ArrayOf(standardView))
		standardViewErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/views/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			standardViewResponses()
		})
	})
	Method("create_workspace", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			standardViewWritableFields()
			Required("slug", "name")
		})
		Result(standardView)
		standardViewErrors()
		Error("invalid_json", jsonFieldValidation)
		HTTP(func() {
			POST("/api/workspaces/{slug}/views/")
			Cookie("session_id:session-id")
			Param("slug")
			Response("invalid_json", StatusBadRequest)
			Response(StatusCreated)
			standardViewResponses()
		})
	})
	Method("retrieve_workspace", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "pk")
		})
		Result(standardView)
		standardViewErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/views/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("pk")
			Response(StatusOK)
			standardViewResponses()
		})
	})
	Method("update_workspace", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			standardViewWritableFields()
			Required("slug", "pk")
		})
		Result(standardView)
		standardViewErrors()
		Error("invalid_json", jsonFieldValidation)
		HTTP(func() {
			PUT("/api/workspaces/{slug}/views/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("pk")
			Response("invalid_json", StatusBadRequest)
			Response(StatusOK)
			standardViewResponses()
		})
	})
	Method("patch_workspace", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			standardViewWritableFields()
			Required("slug", "pk")
		})
		Result(standardView)
		standardViewErrors()
		Error("invalid_json", jsonFieldValidation)
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/views/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("pk")
			Response("invalid_json", StatusBadRequest)
			Response(StatusOK)
			standardViewResponses()
		})
	})
	Method("delete_workspace", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "pk")
		})
		standardViewErrors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/views/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("pk")
			Response(StatusNoContent)
			standardViewResponses()
		})
	})
})
