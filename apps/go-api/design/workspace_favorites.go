package design

import . "github.com/CaliLuke/loom/dsl"

var workspaceFavorite = Type("WorkspaceFavorite", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("entity_type", String, func() { MaxLength(100) })
	Attribute("entity_identifier", String, func() { Format(FormatUUID); Nullable() })
	Attribute("entity_data", Any, func() { Nullable() })
	Attribute("name", String, func() { MaxLength(255); Nullable() })
	Attribute("is_folder", Boolean)
	Attribute("sequence", Float64)
	Attribute("parent", String, func() { Format(FormatUUID); Nullable() })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "entity_type", "entity_identifier", "entity_data", "name", "is_folder", "sequence", "parent", "workspace_id", "project_id")
})

var workspaceFavoriteUnauthenticated = Type("WorkspaceFavoriteUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceFavoriteForbidden = Type("WorkspaceFavoriteForbidden", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceFavoriteNotFound = Type("WorkspaceFavoriteNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceFavoriteInvalid = Type("WorkspaceFavoriteInvalid", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Attribute("entity_type", ArrayOf(String))
	Attribute("entity_identifier", ArrayOf(String))
	Attribute("name", ArrayOf(String))
	Attribute("is_folder", ArrayOf(String))
	Attribute("sequence", ArrayOf(String))
	Attribute("parent", ArrayOf(String))
	Attribute("project_id", ArrayOf(String))
})

var workspaceFavoriteInternal = Type("WorkspaceFavoriteInternal", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

func workspaceFavoriteFields() {
	Attribute("entity_type", String, func() { MaxLength(100); Nullable() })
	Attribute("entity_identifier", String, func() { Format(FormatUUID); Nullable() })
	Attribute("name", String, func() { MaxLength(255); Nullable() })
	Attribute("is_folder", Boolean, func() { Nullable() })
	Attribute("sequence", Float64, func() { Nullable() })
	Attribute("parent", String, func() { Format(FormatUUID); Nullable() })
	Attribute("project_id", String, func() { Format(FormatUUID); Nullable() })
}

func workspaceFavoriteErrors() {
	Error("unauthenticated", workspaceFavoriteUnauthenticated)
	Error("forbidden", workspaceFavoriteForbidden)
	Error("not_found", workspaceFavoriteNotFound)
	Error("invalid", workspaceFavoriteInvalid)
	Error("internal_error", workspaceFavoriteInternal, func() { Fault() })
}

func workspaceFavoriteResponses() {
	Response("unauthenticated", StatusUnauthorized)
	Response("forbidden", StatusForbidden)
	Response("not_found", StatusNotFound)
	Response("invalid", StatusBadRequest)
	Response("internal_error", StatusInternalServerError)
}

var _ = Service("workspace_favorites", func() {
	Method("list", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("all", Boolean)
			Required("slug")
		})
		Result(ArrayOf(workspaceFavorite))
		workspaceFavoriteErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/user-favorites/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("all")
			Response(StatusOK)
			workspaceFavoriteResponses()
		})
	})
	Method("create", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			workspaceFavoriteFields()
			Required("slug")
		})
		Result(workspaceFavorite)
		workspaceFavoriteErrors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/user-favorites/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			workspaceFavoriteResponses()
		})
	})
	Method("patch_collection", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload(); Attribute("slug", String); workspaceFavoriteFields(); Required("slug") })
		Result(workspaceFavorite)
		workspaceFavoriteErrors()
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/user-favorites/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			workspaceFavoriteResponses()
		})
	})
	Method("delete_collection", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload(); Attribute("slug", String); Required("slug") })
		workspaceFavoriteErrors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/user-favorites/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusNoContent)
			workspaceFavoriteResponses()
		})
	})
	Method("retrieve", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("favorite_id", String, func() { Format(FormatUUID) })
			Required("slug", "favorite_id")
		})
		Result(workspaceFavorite)
		workspaceFavoriteErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/user-favorites/{favorite_id}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("favorite_id")
			Response(StatusOK)
			workspaceFavoriteResponses()
		})
	})
	Method("create_detail", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("favorite_id", String, func() { Format(FormatUUID) })
			workspaceFavoriteFields()
			Required("slug", "favorite_id")
		})
		Result(workspaceFavorite)
		workspaceFavoriteErrors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/user-favorites/{favorite_id}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("favorite_id")
			Response(StatusOK)
			workspaceFavoriteResponses()
		})
	})
	Method("patch", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("favorite_id", String, func() { Format(FormatUUID) })
			workspaceFavoriteFields()
			Required("slug", "favorite_id")
		})
		Result(workspaceFavorite)
		workspaceFavoriteErrors()
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/user-favorites/{favorite_id}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("favorite_id")
			Response(StatusOK)
			workspaceFavoriteResponses()
		})
	})
	Method("delete", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("favorite_id", String, func() { Format(FormatUUID) })
			Required("slug", "favorite_id")
		})
		workspaceFavoriteErrors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/user-favorites/{favorite_id}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("favorite_id")
			Response(StatusNoContent)
			workspaceFavoriteResponses()
		})
	})
	Method("group", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("favorite_id", String, func() { Format(FormatUUID) })
			Required("slug", "favorite_id")
		})
		Result(ArrayOf(workspaceFavorite))
		workspaceFavoriteErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/user-favorites/{favorite_id}/group/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("favorite_id")
			Response(StatusOK)
			workspaceFavoriteResponses()
		})
	})
})
