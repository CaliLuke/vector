package design

import . "github.com/CaliLuke/loom/dsl"

var pageLifecycleWriteBase = Type("PageLifecycleWriteBase", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("collaboration_revision", Int64)
	Attribute("owned_by", String, func() { Format(FormatUUID) })
	Attribute("access", Int)
	Attribute("color", String)
	Attribute("parent", String, func() { Format(FormatUUID); Nullable() })
	Attribute("is_locked", Boolean)
	Attribute("archived_at", String, func() { Format(FormatDate); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("view_props", Any)
	Attribute("logo_props", Any)
	Required("id", "name", "owned_by", "access", "color", "parent", "is_locked", "archived_at", "workspace", "created_at", "updated_at", "created_by", "updated_by", "view_props", "logo_props")
})

var pageLifecycleCreate = Type("PageLifecycleCreate", func() {
	Extend(pageLifecycleWriteBase)
	Attribute("description_html", String)
	Attribute("is_favorite", Boolean)
	Attribute("label_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("project_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Required("description_html", "is_favorite", "label_ids", "project_ids")
})

var pageLifecycleUpdate = Type("PageLifecycleUpdate", func() {
	Extend(pageLifecycleWriteBase)
	Attribute("description_html", String)
	Required("description_html")
})

var pageLifecycleDuplicate = Type("PageLifecycleDuplicate", func() {
	Extend(pageLifecycleWriteBase)
	Attribute("description_html", String)
	Attribute("project_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Required("description_html", "project_ids")
})

var pageLifecycleArchiveResult = Type("PageLifecycleArchiveResult", func() {
	Attribute("archived_at", String)
	Required("archived_at")
})

var pageLifecycleDescriptionResult = Type("PageLifecycleDescriptionResult", func() {
	Attribute("message", String)
	Required("message")
})

var pageLifecycleResponseMetadata = Type("PageLifecycleResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var pageLifecycleError = Type("PageLifecycleError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("kind", "body")
})

var pageLifecycleRequestBody = MapOf(String, Any)

func pageLifecycleMethods() {
	pageLifecycleDetailMethod("create", "post_api_workspaces_by_slug_projects_by_project_id_pages", "/api/workspaces/{slug}/projects/{project_id}/pages/", "POST", 201, pageLifecycleCreate)
	pageLifecycleDetailMethod("update", "patch_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/", "PATCH", 200, pageLifecycleUpdate, true)
	pageLifecycleEmptyMethod("delete", "delete_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/", "DELETE")
	pageLifecycleEmptyMethod("access", "post_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_access", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/access/", "POST", true)
	pageLifecycleJSONMethod("archive", "post_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_archive", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/", "POST", 200, pageLifecycleArchiveResult, true, false)
	pageLifecycleEmptyMethod("unarchive", "delete_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_archive", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/", "DELETE")
	pageLifecycleJSONMethod("description_update", "patch_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_description", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/", "PATCH", 200, pageLifecycleDescriptionResult, true)
	pageLifecycleDetailMethod("duplicate", "post_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_duplicate", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/duplicate/", "POST", 201, pageLifecycleDuplicate, true)
	pageLifecycleEmptyMethod("lock", "post_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_lock", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/", "POST")
	pageLifecycleEmptyMethod("unlock", "delete_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_lock", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/", "DELETE")
	pageLifecycleOptionsMethod("create_options", "options_api_workspaces_by_slug_projects_by_project_id_pages", "/api/workspaces/{slug}/projects/{project_id}/pages/", false)
	pageLifecycleOptionsMethod("update_options", "options_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/", true)
	pageLifecycleOptionsMethod("description_options", "options_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_description", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/", true)
	pageLifecycleOptionsMethod("access_options", "options_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_access", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/access/", true)
	pageLifecycleOptionsMethod("archive_options", "options_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_archive", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/", true)
	pageLifecycleOptionsMethod("duplicate_options", "options_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_duplicate", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/duplicate/", true)
	pageLifecycleOptionsMethod("lock_options", "options_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_lock", "/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/", true)
}

func pageLifecycleOptionsMethod(name, operationID, route string, pageID bool) {
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		Security(planeSessionCookie)
		Payload(func() { pageLifecyclePayload(pageID) })
		Result(rawJSONResponseMetadata)
		pageReadErrors(true, false)
		HTTP(func() {
			OPTIONS(route)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			if pageID {
				Param("page_id")
			}
			rawJSONReadResponse(pageOptionsMetadata)
			pageReadResponses(true, false)
		})
	})
}

func pageLifecyclePayload(pageID bool) {
	stickySessionPayload()
	Attribute("slug", String)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	if pageID {
		Attribute("page_id", String, func() { Format(FormatUUID) })
		Required("slug", "project_id", "page_id")
		return
	}
	Required("slug", "project_id")
}

func pageLifecycleErrors() {
	for _, item := range []struct {
		name   string
		status int
	}{
		{"status_400", 400}, {"status_403", 403},
		{"status_404", 404}, {"status_405", 405}, {"status_409", 409}, {"status_415", 415}, {"status_500", 500},
	} {
		Error(item.name, pageLifecycleError, func() {
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
	Error("unauthenticated", pageReadsUnauthenticated)
	HTTP(func() { Response("unauthenticated", StatusUnauthorized) })
}

func pageLifecycleRawRequestBody() {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(pageLifecycleRequestBody, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
}

func pageLifecycleRawResponse(status int, schema any) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Header("content_type:Content-Type")
		OpenAPIBody(schema)
	})
}

func pageLifecyclePathPayload(pageID bool) {
	Security(planeSessionCookie)
	Payload(func() { pageLifecyclePayload(pageID) })
	Result(pageLifecycleResponseMetadata)
	pageLifecycleErrors()
}

func pageLifecycleDetailMethod(name, operationID, route, method string, status int, schema any, pageID ...bool) {
	hasPageID := len(pageID) > 0 && pageID[0]
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		pageLifecyclePathPayload(hasPageID)
		HTTP(func() {
			switch method {
			case "POST":
				POST(route)
			case "PATCH":
				PATCH(route)
			}
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			if hasPageID {
				Param("page_id")
			}
			pageLifecycleRawRequestBody()
			pageLifecycleRawResponse(status, schema)
		})
	})
}

func pageLifecycleJSONMethod(name, operationID, route, method string, status int, schema any, options ...bool) {
	hasPageID := len(options) > 0 && options[0]
	withBody := len(options) < 2 || options[1]
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		pageLifecyclePathPayload(hasPageID)
		HTTP(func() {
			if method == "POST" {
				POST(route)
			} else {
				PATCH(route)
			}
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			if hasPageID {
				Param("page_id")
			}
			if withBody {
				pageLifecycleRawRequestBody()
			}
			pageLifecycleRawResponse(status, schema)
		})
	})
}

func pageLifecycleEmptyMethod(name, operationID, route, method string, body ...bool) {
	withBody := len(body) > 0 && body[0]
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		Security(planeSessionCookie)
		Payload(func() { pageLifecyclePayload(true) })
		Result(Empty)
		pageLifecycleErrors()
		HTTP(func() {
			if method == "POST" {
				POST(route)
			} else {
				DELETE(route)
			}
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("page_id")
			if withBody {
				pageLifecycleRawRequestBody()
			}
			Response(StatusNoContent)
		})
	})
}
