package design

import . "github.com/CaliLuke/loom/dsl"

var projectStateRead = Type("ProjectStateRead", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("color", String)
	Attribute("group", String, func() {
		Enum("backlog", "unstarted", "started", "completed", "cancelled", "triage")
	})
	Attribute("default", Boolean)
	Attribute("description", String)
	Attribute("sequence", Float64)
	Attribute("order", Float64)
	Required("id", "project_id", "workspace_id", "name", "color", "group", "default", "description", "sequence")
})

var projectStateReadsUnauthenticated = Type("ProjectStateReadsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var projectStateReadsPermissionDenied = Type("ProjectStateReadsPermissionDenied", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var projectStateReadsDetailNotFound = Type("ProjectStateReadsDetailNotFound", func() {
	Attribute("detail", String)
	Required("detail")
})

var projectStateReadsNotFound = Type("ProjectStateReadsNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var projectStateReadsInternalError = Type("ProjectStateReadsInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var projectStateWritesAppRawRequest = MapOf(String, Any)

var projectStateWritesAppResponseMetadata = Type("ProjectStateWritesAppResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var projectStateWritesAppError = Type("ProjectStateWritesAppError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("kind", "body")
})

func projectStateWritesAppErrors() {
	for _, item := range []struct {
		name   string
		status int
	}{
		{"bad_request", 400},
		{"unauthenticated", 401},
		{"permission_denied", 403},
		{"not_found", 404},
		{"unsupported_media_type", 415},
		{"internal_error", 500},
	} {
		Error(item.name, projectStateWritesAppError, func() {
			if item.status == 500 {
				Fault()
			}
		})
		HTTP(func() {
			Response(item.name, item.status, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body("body")
			})
		})
	}
}

func projectStateWritesAppPayload(stateID, hasBody bool) {
	Security(planeSessionCookie)
	Payload(func() {
		stickySessionPayload()
		Attribute("slug", String)
		if hasBody {
			Attribute("content_type", String)
		}
		Attribute("project_id", String, func() { Format(FormatUUID) })
		if stateID {
			Attribute("pk", String, func() { Format(FormatUUID) })
		}
		required := []string{"slug", "project_id"}
		if stateID {
			required = append(required, "pk")
		}
		Required(required...)
	})
}

func projectStateWritesAppRequestBody() {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(projectStateWritesAppRawRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
}

func projectStateWritesAppJSONResponse(status int, schema any) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Header("content_type:Content-Type")
		OpenAPIBody(schema)
	})
}

func projectStateWritesAppMethod(name, operationID, route, method string, stateID bool, response bool) {
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		projectStateWritesAppPayload(stateID, method == "POST" || method == "PATCH")
		if response {
			Result(projectStateWritesAppResponseMetadata)
		} else {
			Result(Empty)
		}
		projectStateWritesAppErrors()
		HTTP(func() {
			switch method {
			case "POST":
				POST(route)
			case "PATCH":
				PATCH(route)
			case "DELETE":
				DELETE(route)
			}
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			if stateID {
				Param("pk")
			}
			if method == "POST" || method == "PATCH" {
				projectStateWritesAppRequestBody()
				Header("content_type:Content-Type")
			}
			if response {
				projectStateWritesAppJSONResponse(200, projectStateRead)
			} else {
				Response(StatusNoContent)
			}
		})
	})
}

func projectStateWritesAppMarkDefault() {
	Method("mark_default", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_states_by_pk_mark_default")
		projectStateWritesAppPayload(true, false)
		Result(Empty)
		projectStateWritesAppErrors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/states/{pk}/mark-default/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(StatusNoContent)
		})
	})
}

var _ = Service("project_state_reads", func() {
	Description("Returns authenticated project workflow state read models.")

	Method("list", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("grouped", String)
			Required("slug", "project_id")
		})
		// Loom's untagged OneOf currently supports object branches only. This
		// endpoint returns either a top-level array or map, so Any preserves the
		// wire shape for generated servers and clients.
		Result(Any, func() { Meta("openapi:example", "false") })
		Error("unauthenticated", projectStateReadsUnauthenticated)
		Error("permission_denied", projectStateReadsPermissionDenied)
		Error("internal_error", projectStateReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/states/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("grouped")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("permission_denied", StatusForbidden)
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
		Result(projectStateRead)
		Error("unauthenticated", projectStateReadsUnauthenticated)
		Error("not_found", projectStateReadsDetailNotFound)
		Error("internal_error", projectStateReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/states/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("intake", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id")
		})
		Result(projectStateRead)
		Error("unauthenticated", projectStateReadsUnauthenticated)
		Error("permission_denied", projectStateReadsPermissionDenied)
		Error("not_found", projectStateReadsNotFound)
		Error("internal_error", projectStateReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/intake-state/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("permission_denied", StatusForbidden)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	projectStateWritesAppMethod("create", "post_api_workspaces_by_slug_projects_by_project_id_states", "/api/workspaces/{slug}/projects/{project_id}/states/", "POST", false, true)
	projectStateWritesAppMethod("update", "patch_api_workspaces_by_slug_projects_by_project_id_states_by_pk", "/api/workspaces/{slug}/projects/{project_id}/states/{pk}/", "PATCH", true, true)
	projectStateWritesAppMethod("delete", "delete_api_workspaces_by_slug_projects_by_project_id_states_by_pk", "/api/workspaces/{slug}/projects/{project_id}/states/{pk}/", "DELETE", true, false)
	projectStateWritesAppMarkDefault()
})
