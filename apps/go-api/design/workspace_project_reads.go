package design

import . "github.com/CaliLuke/loom/dsl"

var workspaceProjectWorkspace = Type("WorkspaceProjectWorkspace", func() {
	Attribute("background_color", String)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("logo", String, func() { Nullable() })
	Attribute("logo_asset", String, func() { Format(FormatUUID); Nullable() })
	Attribute("logo_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("name", String)
	Attribute("organization_size", String, func() { Nullable() })
	Attribute("owner", String, func() { Format(FormatUUID) })
	Attribute("slug", String)
	Attribute("timezone", String)
	Attribute("total_members", Int)
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Required("background_color", "created_at", "created_by", "deleted_at", "id", "logo", "logo_asset", "logo_url", "name", "organization_size", "owner", "slug", "timezone", "total_members", "updated_at", "updated_by")
})

var workspaceProjectUserWorkspace = Type("WorkspaceProjectUserWorkspace", func() {
	Extend(workspaceProjectWorkspace)
	Attribute("role", Int)
	Required("role")
})

var workspaceProjectCompactProject = Type("WorkspaceProjectCompactProject", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("identifier", String)
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("logo_props", Any)
	Attribute("member_role", Int, func() { Nullable() })
	Attribute("intake_count", Int)
	Attribute("archived_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("cycle_view", Boolean)
	Attribute("issue_views_view", Boolean)
	Attribute("module_view", Boolean)
	Attribute("page_view", Boolean)
	Attribute("inbox_view", Boolean)
	Attribute("guest_view_all_features", Boolean)
	Attribute("project_lead", String, func() { Format(FormatUUID); Nullable() })
	Attribute("network", Int, func() { Enum(0, 2) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "name", "identifier", "sort_order", "logo_props", "member_role", "intake_count", "archived_at", "workspace", "cycle_view", "issue_views_view", "module_view", "page_view", "inbox_view", "guest_view_all_features", "project_lead", "network", "created_at", "updated_at", "created_by", "updated_by")
})

var workspaceProjectProject = Type("WorkspaceProjectProject", func() {
	Attribute("anchor", String, func() { Nullable() })
	Attribute("archive_in", Int)
	Attribute("archived_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("close_in", Int)
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("cover_image_asset", String, func() { Format(FormatUUID); Nullable() })
	Attribute("cover_image_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("cycle_view", Boolean)
	Attribute("default_assignee", String, func() { Format(FormatUUID); Nullable() })
	Attribute("default_state", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("description", String)
	Attribute("description_html", Any)
	Attribute("description_text", Any)
	Attribute("emoji", String, func() { Nullable() })
	Attribute("estimate", String, func() { Format(FormatUUID); Nullable() })
	Attribute("external_id", String, func() { Nullable() })
	Attribute("external_source", String, func() { Nullable() })
	Attribute("guest_view_all_features", Boolean)
	Attribute("icon_prop", Any)
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("inbox_view", Boolean)
	Attribute("intake_view", Boolean)
	Attribute("is_favorite", Boolean)
	Attribute("is_issue_type_enabled", Boolean)
	Attribute("is_time_tracking_enabled", Boolean)
	Attribute("issue_views_view", Boolean)
	Attribute("logo_props", Any)
	Attribute("member_role", Int, func() { Nullable() })
	Attribute("members", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("module_view", Boolean)
	Attribute("name", String)
	Attribute("network", Int, func() { Enum(0, 2) })
	Attribute("next_work_item_sequence", Int)
	Attribute("page_view", Boolean)
	Attribute("project_lead", String, func() { Format(FormatUUID); Nullable() })
	Attribute("sort_order", Float64, func() { Nullable() })
	Attribute("timezone", String)
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Required("anchor", "archive_in", "archived_at", "close_in", "cover_image", "cover_image_asset", "cover_image_url", "created_at", "created_by", "cycle_view", "default_assignee", "default_state", "deleted_at", "description", "description_html", "description_text", "emoji", "estimate", "external_id", "external_source", "guest_view_all_features", "icon_prop", "id", "identifier", "inbox_view", "intake_view", "is_favorite", "is_issue_type_enabled", "is_time_tracking_enabled", "issue_views_view", "logo_props", "member_role", "members", "module_view", "name", "network", "next_work_item_sequence", "page_view", "project_lead", "sort_order", "timezone", "updated_at", "updated_by", "workspace")
})

var workspaceProjectStats = Type("WorkspaceProjectStats", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("total_issues", Int)
	Attribute("completed_issues", Int)
	Attribute("total_members", Int)
	Attribute("total_cycles", Int)
	Attribute("total_modules", Int)
	Required("id")
})

var workspaceProjectUnauthenticated = Type("WorkspaceProjectUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceProjectForbidden = Type("WorkspaceProjectForbidden", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceProjectNotFound = Type("WorkspaceProjectNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceProjectDetailNotFound = Type("WorkspaceProjectDetailNotFound", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceProjectConflict = Type("WorkspaceProjectConflict", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceProjectBadRequest = Type("WorkspaceProjectBadRequest", func() {
	Attribute("detail", String)
	Required("detail")
})

var workspaceProjectOwnerBadRequest = Type("WorkspaceProjectOwnerBadRequest", func() {
	Attribute("owner", ArrayOf(String))
	Required("owner")
})

var workspaceProjectValidationError = Type("WorkspaceProjectValidationError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceProjectInternalError = Type("WorkspaceProjectInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var workspaceProjectOptions = Type("WorkspaceProjectOptions", func() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("renders", ArrayOf(String))
	Attribute("parses", ArrayOf(String))
	Required("name", "description", "renders", "parses")
})

var rawJSONResponseMetadata = Type("RawJSONResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

func workspaceProjectReadPayload() {
	stickySessionPayload()
	Attribute("slug", String)
	Required("slug")
}

func workspaceProjectReadErrors() {
	Error("unauthenticated", workspaceProjectUnauthenticated)
	Error("forbidden", workspaceProjectForbidden)
	Error("internal_error", workspaceProjectInternalError, func() { Fault() })
}

func rawJSONReadResponse(schema any) {
	SkipResponseBodyEncodeDecode()
	Response(StatusOK, func() {
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

func workspaceProjectReadResponses() {
	Response("unauthenticated", StatusUnauthorized)
	Response("forbidden", StatusForbidden)
	Response("internal_error", StatusInternalServerError)
}

var _ = Service("workspace_project_reads", func() {
	Description("Returns authenticated workspace and project core read models.")
	HTTP(func() {
		RuntimeCORS()
	})

	Method("user_workspaces", func() {
		Meta("openapi:operationId", "get_api_users_me_workspaces")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("owner", String)
			Attribute("search", String)
			Attribute("fields", String)
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", workspaceProjectUnauthenticated)
		Error("internal_error", workspaceProjectInternalError, func() { Fault() })
		Error("owner_bad_request", workspaceProjectOwnerBadRequest)
		HTTP(func() {
			GET("/api/users/me/workspaces/")
			rawJSONReadResponse(ArrayOf(workspaceProjectUserWorkspace))
			Cookie("session_id:session-id")
			Param("owner")
			Param("search")
			Param("fields")
			Response("unauthenticated", StatusUnauthorized)
			Response("internal_error", StatusInternalServerError)
			Response("owner_bad_request", StatusBadRequest)
		})
	})

	Method("workspaces", func() {
		Meta("openapi:operationId", "get_api_workspaces")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("owner", String)
			Attribute("search", String)
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", workspaceProjectUnauthenticated)
		Error("internal_error", workspaceProjectInternalError, func() { Fault() })
		Error("owner_bad_request", workspaceProjectOwnerBadRequest)
		HTTP(func() {
			GET("/api/workspaces/")
			rawJSONReadResponse(ArrayOf(workspaceProjectWorkspace))
			Cookie("session_id:session-id")
			Param("owner")
			Param("search")
			Response("unauthenticated", StatusUnauthorized)
			Response("internal_error", StatusInternalServerError)
			Response("owner_bad_request", StatusBadRequest)
		})
	})

	Method("workspace", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug")
		Security(planeSessionCookie)
		Payload(func() {
			workspaceProjectReadPayload()
			Attribute("owner", String)
			Attribute("search", String)
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", workspaceProjectUnauthenticated)
		Error("not_found", workspaceProjectDetailNotFound)
		Error("internal_error", workspaceProjectInternalError, func() { Fault() })
		Error("owner_bad_request", workspaceProjectOwnerBadRequest)
		HTTP(func() {
			GET("/api/workspaces/{slug}/")
			rawJSONReadResponse(workspaceProjectWorkspace)
			Cookie("session_id:session-id")
			Param("slug")
			Param("owner")
			Param("search")
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
			Response("owner_bad_request", StatusBadRequest)
		})
	})

	Method("project_stats", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_project_stats")
		Security(planeSessionCookie)
		Payload(func() {
			workspaceProjectReadPayload()
			Attribute("fields", String)
			Attribute("project_ids", String)
			Attribute("search", String)
		})
		Result(rawJSONResponseMetadata)
		workspaceProjectReadErrors()
		Error("validation_error", workspaceProjectValidationError)
		HTTP(func() {
			GET("/api/workspaces/{slug}/project-stats/")
			rawJSONReadResponse(ArrayOf(workspaceProjectStats))
			Cookie("session_id:session-id")
			Param("slug")
			Param("fields")
			Param("project_ids")
			Param("search")
			workspaceProjectReadResponses()
			Response("validation_error", StatusBadRequest)
		})
	})

	Method("projects", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects")
		Security(planeSessionCookie)
		Payload(func() {
			workspaceProjectReadPayload()
			Attribute("search", String)
		})
		Result(rawJSONResponseMetadata)
		workspaceProjectReadErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/")
			rawJSONReadResponse(ArrayOf(workspaceProjectCompactProject))
			Cookie("session_id:session-id")
			Param("slug")
			Param("search")
			workspaceProjectReadResponses()
		})
	})

	Method("project_details", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_details")
		Security(planeSessionCookie)
		Payload(func() {
			workspaceProjectReadPayload()
			Attribute("fields", String)
			Attribute("per_page", String)
			Attribute("cursor", String)
			Attribute("order_by", String)
			Attribute("search", String)
		})
		// Django returns a plain array unless both per_page and cursor are set;
		// the paginated variant is an object envelope. Any preserves both shapes.
		Result(rawJSONResponseMetadata)
		workspaceProjectReadErrors()
		Error("bad_request", workspaceProjectBadRequest)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/details/")
			rawJSONReadResponse(Any)
			Cookie("session_id:session-id")
			Param("slug")
			Param("fields")
			Param("per_page")
			Param("cursor")
			Param("order_by")
			Param("search")
			workspaceProjectReadResponses()
			Response("bad_request", StatusBadRequest)
		})
	})

	Method("project", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			workspaceProjectReadPayload()
			Attribute("pk", String)
			Required("pk")
		})
		Result(rawJSONResponseMetadata)
		workspaceProjectReadErrors()
		Error("not_found", workspaceProjectNotFound)
		Error("conflict", workspaceProjectConflict)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{pk}/")
			rawJSONReadResponse(workspaceProjectProject)
			Cookie("session_id:session-id")
			Param("slug")
			Param("pk")
			workspaceProjectReadResponses()
			Response("not_found", StatusNotFound)
			Response("conflict", StatusConflict)
		})
	})

	Method("user_workspaces_options", func() {
		Meta("openapi:operationId", "options_api_users_me_workspaces")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
		})
		Result(workspaceProjectOptions)
		Error("unauthenticated", workspaceProjectUnauthenticated)
		HTTP(func() {
			OPTIONS("/api/users/me/workspaces/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
		})
	})

	Method("project_stats_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_project_stats")
		Security(planeSessionCookie)
		Payload(func() {
			workspaceProjectReadPayload()
		})
		Result(workspaceProjectOptions)
		Error("unauthenticated", workspaceProjectUnauthenticated)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/project-stats/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
		})
	})

	Method("project_details_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_projects_details")
		Security(planeSessionCookie)
		Payload(func() {
			workspaceProjectReadPayload()
		})
		Result(workspaceProjectOptions)
		Error("unauthenticated", workspaceProjectUnauthenticated)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/details/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
		})
	})
})
