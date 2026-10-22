package design

import . "github.com/CaliLuke/loom/dsl"

var entityFavorite = Type("EntityFavorite", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("entity_type", String)
	Attribute("entity_identifier", String, func() { Format(FormatUUID); Nullable() })
	Attribute("entity_data", Any, func() { Nullable() })
	Attribute("name", String, func() { Nullable() })
	Attribute("is_folder", Boolean)
	Attribute("sequence", Float64)
	Attribute("parent", String, func() { Format(FormatUUID); Nullable() })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "entity_type", "entity_identifier", "entity_data", "name", "is_folder", "sequence", "parent", "workspace_id", "project_id")
})

var entityFavoriteUnauthenticated = Type("EntityFavoriteUnauthenticated", func() { Attribute("detail", String); Required("detail") })
var entityFavoriteNotFound = Type("EntityFavoriteNotFound", func() { Attribute("error", String, func() { Meta("struct:field:name", "Message") }); Required("error") })
var entityFavoriteInternal = Type("EntityFavoriteInternal", func() { Attribute("error", String, func() { Meta("struct:field:name", "Message") }); Required("error") })

func entityFavoriteErrors() {
	Error("unauthenticated", entityFavoriteUnauthenticated)
	Error("not_found", entityFavoriteNotFound)
	Error("internal_error", entityFavoriteInternal, func() { Fault() })
}

func entityFavoriteSession() { APIKey("plane_session_cookie", "session_id", String) }
func entityFavoritePayload(project bool, entity string) {
	entityFavoriteSession()
	Attribute("slug", String)
	if project {
		Attribute("project_id", String, func() { Format(FormatUUID) })
	}
	if entity != "" {
		Attribute(entity, String, func() { Format(FormatUUID) })
	}
	Required("slug")
}

var _ = Service("entity_favorites", func() {
	Description("Entity-specific user favorites.")
	for _, spec := range []struct{ name, typ, path string }{
		{"project", "project", "/api/workspaces/{slug}/user-favorite-projects/"},
		{"cycle", "cycle", "/api/workspaces/{slug}/projects/{project_id}/user-favorite-cycles/"},
		{"module", "module", "/api/workspaces/{slug}/projects/{project_id}/user-favorite-modules/"},
		{"view", "view", "/api/workspaces/{slug}/projects/{project_id}/user-favorite-views/"},
	} {
		method := spec.name + "_list"
		Method(method, func() {
			Security(planeSessionCookie)
			Payload(func() { entityFavoritePayload(spec.name != "project", "") })
			Result(ArrayOf(entityFavorite))
			entityFavoriteErrors()
			HTTP(func() {
				GET(spec.path)
				Cookie("session_id:session-id")
				Param("slug")
				if spec.name != "project" {
					Param("project_id")
				}
				Response(StatusOK)
			})
		})
		method = spec.name + "_create"
		Method(method, func() {
			Security(planeSessionCookie)
			Payload(func() { entityFavoritePayload(spec.name != "project", spec.name) })
			entityFavoriteErrors()
			HTTP(func() {
				POST(spec.path)
				Cookie("session_id:session-id")
				Param("slug")
				if spec.name != "project" {
					Param("project_id")
				}
				Response(StatusNoContent)
			})
		})
		method = spec.name + "_delete"
		Method(method, func() {
			Security(planeSessionCookie)
			Payload(func() { entityFavoritePayload(spec.name != "project", spec.name+"_id") })
			entityFavoriteErrors()
			HTTP(func() {
				DELETE(spec.path + "{" + spec.name + "_id}/")
				Cookie("session_id:session-id")
				Param("slug")
				Param(spec.name + "_id")
				if spec.name != "project" {
					Param("project_id")
				}
				Response(StatusNoContent)
			})
		})
	}
	Method("page_create", func() {
		Security(planeSessionCookie)
		Payload(func() { entityFavoritePayload(true, "page_id") })
		entityFavoriteErrors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/favorite-pages/{page_id}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			Response(StatusNoContent)
		})
	})
	Method("page_delete", func() {
		Security(planeSessionCookie)
		Payload(func() { entityFavoritePayload(true, "page_id") })
		entityFavoriteErrors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/projects/{project_id}/favorite-pages/{page_id}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			Response(StatusNoContent)
		})
	})
})
