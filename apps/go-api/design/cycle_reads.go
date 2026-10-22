package design

import . "github.com/CaliLuke/loom/dsl"

var cycleRead = Type("CycleRead", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("description", String)
	Attribute("start_date", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("end_date", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("owned_by_id", String, func() { Format(FormatUUID) })
	Attribute("view_props", Any)
	Attribute("sort_order", Float64)
	Attribute("external_source", String, func() { Nullable() })
	Attribute("external_id", String, func() { Nullable() })
	Attribute("progress_snapshot", Any)
	Attribute("logo_props", Any)
	Attribute("is_favorite", Boolean)
	Attribute("total_issues", Int)
	Attribute("cancelled_issues", Int)
	Attribute("completed_issues", Int)
	Attribute("started_issues", Int)
	Attribute("unstarted_issues", Int)
	Attribute("backlog_issues", Int)
	Attribute("assignee_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("status", String, func() { Enum("CURRENT", "UPCOMING", "COMPLETED", "DRAFT") })
	Attribute("version", Int)
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("archived_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("sub_issues", Int)
	Attribute("backlog_estimate_points", Float64)
	Attribute("unstarted_estimate_points", Float64)
	Attribute("started_estimate_points", Float64)
	Attribute("cancelled_estimate_points", Float64)
	Attribute("completed_estimate_points", Float64)
	Attribute("total_estimate_points", Float64)
	Attribute("estimate_distribution", Any)
	Attribute("distribution", Any)
	Required(
		"id", "workspace_id", "project_id", "name", "description", "start_date", "end_date",
		"owned_by_id", "view_props", "sort_order", "external_source", "external_id", "progress_snapshot",
	)
})

var cycleIssueProjection = Type("CycleIssueProjection", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("state_id", String, func() { Format(FormatUUID) })
	Attribute("sort_order", Float64)
	Attribute("completed_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("estimate_point", String, func() { Format(FormatUUID); Nullable() })
	Attribute("priority", String, func() { Nullable() })
	Attribute("start_date", String, func() { Nullable() })
	Attribute("target_date", String, func() { Nullable() })
	Attribute("sequence_id", Int)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("parent_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("cycle_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("sub_issues_count", Int)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("attachment_count", Int)
	Attribute("link_count", Int)
	Attribute("is_draft", Boolean)
	Attribute("archived_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("state__group", String, func() { Enum("backlog", "unstarted", "started", "completed", "cancelled", "triage") })
	Attribute("assignee_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("label_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("module_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Required(
		"id", "name", "state_id", "sort_order", "completed_at", "estimate_point", "priority", "start_date",
		"target_date", "sequence_id", "project_id", "parent_id", "cycle_id", "sub_issues_count", "created_at",
		"updated_at", "created_by", "updated_by", "attachment_count", "link_count", "is_draft", "archived_at",
		"state__group", "assignee_ids", "label_ids", "module_ids",
	)
})

var cycleIssuePage = Type("CycleIssuePage", func() {
	Attribute("grouped_by", String, func() { Nullable() })
	Attribute("sub_grouped_by", String, func() { Nullable() })
	Attribute("total_count", Int)
	Attribute("next_cursor", String)
	Attribute("prev_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_page_results", Boolean)
	Attribute("count", Int)
	Attribute("total_pages", Int)
	Attribute("total_results", Int)
	Attribute("extra_stats", Any, func() { Nullable() })
	// Grouping changes results from an array into one or two levels of maps.
	Attribute("results", Any)
	Required(
		"grouped_by", "sub_grouped_by", "total_count", "next_cursor", "prev_cursor",
		"next_page_results", "prev_page_results", "count", "total_pages", "total_results", "extra_stats", "results",
	)
})

var cycleIssueRead = Type("CycleIssueRead", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("cycle", String, func() { Format(FormatUUID) })
	Attribute("issue", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("sub_issues_count", Int)
	Attribute("issue_detail", Any)
	Required(
		"id", "workspace", "project", "cycle", "issue", "created_at", "updated_at",
		"created_by", "updated_by", "deleted_at", "sub_issues_count", "issue_detail",
	)
})

var cycleProgress = Type("CycleProgress", func() {
	Attribute("backlog_estimate_points", Float64)
	Attribute("unstarted_estimate_points", Float64)
	Attribute("started_estimate_points", Float64)
	Attribute("cancelled_estimate_points", Float64)
	Attribute("completed_estimate_points", Float64)
	Attribute("total_estimate_points", Float64)
	Attribute("backlog_issues", Int)
	Attribute("unstarted_issues", Int)
	Attribute("started_issues", Int)
	Attribute("cancelled_issues", Int)
	Attribute("completed_issues", Int)
	Attribute("total_issues", Int)
	Required(
		"backlog_estimate_points", "unstarted_estimate_points", "started_estimate_points",
		"cancelled_estimate_points", "completed_estimate_points", "total_estimate_points",
		"backlog_issues", "unstarted_issues", "started_issues", "cancelled_issues", "completed_issues", "total_issues",
	)
})

var cycleIssueAnalyticsAssignee = Type("CycleIssueAnalyticsAssignee", func() {
	Attribute("display_name", String, func() { Nullable() })
	Attribute("assignee_id", String, func() { Nullable() })
	Attribute("avatar_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("total_issues", Int)
	Attribute("completed_issues", Int)
	Attribute("pending_issues", Int)
	Required("display_name", "assignee_id", "avatar_url", "total_issues", "completed_issues", "pending_issues")
})

var cycleSnapshotAnalyticsAssignee = Type("CycleSnapshotAnalyticsAssignee", func() {
	Attribute("first_name", String, func() { Nullable() })
	Attribute("last_name", String, func() { Nullable() })
	Attribute("display_name", String, func() { Nullable() })
	Attribute("assignee_id", String, func() { Nullable() })
	Attribute("avatar_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("total_issues", Int)
	Attribute("completed_issues", Int)
	Attribute("pending_issues", Int)
	Required("first_name", "last_name", "display_name", "assignee_id", "avatar_url", "total_issues", "completed_issues", "pending_issues")
})

var cyclePointAnalyticsAssignee = Type("CyclePointAnalyticsAssignee", func() {
	Attribute("display_name", String, func() { Nullable() })
	Attribute("assignee_id", String, func() { Nullable() })
	Attribute("avatar_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("total_estimates", Float64, func() { Nullable() })
	Attribute("completed_estimates", Float64, func() { Nullable() })
	Attribute("pending_estimates", Float64, func() { Nullable() })
	Required("display_name", "assignee_id", "avatar_url", "total_estimates", "completed_estimates", "pending_estimates")
})

var cycleIssueAnalyticsLabel = Type("CycleIssueAnalyticsLabel", func() {
	Attribute("label_name", String, func() { Nullable() })
	Attribute("color", String, func() { Nullable() })
	Attribute("label_id", String, func() { Nullable() })
	Attribute("total_issues", Int)
	Attribute("completed_issues", Int)
	Attribute("pending_issues", Int)
	Required("label_name", "color", "label_id", "total_issues", "completed_issues", "pending_issues")
})

var cyclePointAnalyticsLabel = Type("CyclePointAnalyticsLabel", func() {
	Attribute("label_name", String, func() { Nullable() })
	Attribute("color", String, func() { Nullable() })
	Attribute("label_id", String, func() { Nullable() })
	Attribute("total_estimates", Float64, func() { Nullable() })
	Attribute("completed_estimates", Float64, func() { Nullable() })
	Attribute("pending_estimates", Float64, func() { Nullable() })
	Required("label_name", "color", "label_id", "total_estimates", "completed_estimates", "pending_estimates")
})

var cycleAnalytics = Type("CycleAnalytics", func() {
	Attribute("assignees", ArrayOf(Any))
	Attribute("labels", ArrayOf(Any))
	// Values are numeric through today and null for future dates.
	Attribute("completion_chart", MapOf(String, Any))
	Required("assignees", "labels", "completion_chart")
})

var cycleReadsMetadata = Type("CycleReadsMetadata", func() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("renders", ArrayOf(String))
	Attribute("parses", ArrayOf(String))
	Required("name", "description", "renders", "parses")
})

var cycleReadsUnauthenticated = Type("CycleReadsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var cycleReadsForbidden = Type("CycleReadsForbidden", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var cycleReadsWorkspaceForbidden = Type("CycleReadsWorkspaceForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var cycleReadsNotFound = Type("CycleReadsNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

func cycleReadsIssueBadRequestShape() {
	Attribute("error", String, func() { Meta("struct:field:name", "ErrorMessage") })
	Attribute("detail", String)
	Attribute("message", String)
	Attribute("code", String)
	Attribute("errors", Any, func() { Nullable() })
}

var cycleReadsBadRequest = Type("CycleReadsBadRequest", func() {
	cycleReadsIssueBadRequestShape()
})

var cycleReadsInvalidPagination = Type("CycleReadsInvalidPagination", func() {
	cycleReadsIssueBadRequestShape()
})

var cycleReadsInternalError = Type("CycleReadsInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var cycleReadsInvalidGrouping = Type("CycleReadsInvalidGrouping", func() {
	cycleReadsIssueBadRequestShape()
})

var cycleReadsParseError = Type("CycleReadsParseError", func() {
	cycleReadsIssueBadRequestShape()
})

var cycleReadsFilterError = Type("CycleReadsFilterError", func() {
	cycleReadsIssueBadRequestShape()
})

func cycleProjectPayload(idName string) {
	stickySessionPayload()
	Attribute("slug", String)

	Attribute("project_id", String, func() { Format(FormatUUID) })
	if idName != "" {
		Attribute(idName, String, func() { Format(FormatUUID) })
		Required("slug", "project_id", idName)
		return
	}
	Required("slug", "project_id")
}

func cycleReadErrors(notFound bool, badRequest bool) {
	Error("unauthenticated", cycleReadsUnauthenticated)
	Error("forbidden", cycleReadsForbidden)
	if notFound {
		Error("not_found", cycleReadsNotFound)
	}
	if badRequest {
		Error("bad_request", cycleReadsBadRequest)
	}
	Error("internal_error", cycleReadsInternalError, func() { Fault() })
}

func cycleReadResponses(notFound bool, badRequest bool) {
	Response("unauthenticated", StatusUnauthorized)
	Response("forbidden", StatusForbidden)
	if notFound {
		Response("not_found", StatusNotFound)
	}
	if badRequest {
		Response("bad_request", StatusBadRequest)
	}
	Response("internal_error", StatusInternalServerError)
}

func cycleIssueFilterPayload() {
	for _, name := range []string{
		"state", "state_group", "estimate_point", "priority", "assignees", "labels", "created_by", "logged_by", "parent",
		"mentions", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "project", "cycle",
		"module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date", "type", "filters",
	} {
		Attribute(name, String)
	}
}

var _ = Service("cycle_reads", func() {
	Description("Returns authenticated workspace and project cycle read models.")
	HTTP(func() {
		RuntimeCORS()
	})

	Method("workspace_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_cycles")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Required("slug")
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", cycleReadsUnauthenticated)
		Error("workspace_forbidden", cycleReadsWorkspaceForbidden)
		Error("internal_error", cycleReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/cycles/")
			rawJSONReadResponse(ArrayOf(cycleRead))
			Cookie("session_id:session-id")
			Param("slug")
			Response("unauthenticated", StatusUnauthorized)
			Response("workspace_forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("archived_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_archived_cycles")
		Security(planeSessionCookie)
		Payload(func() { cycleProjectPayload("") })
		Result(rawJSONResponseMetadata)
		cycleReadErrors(false, false)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/")
			rawJSONReadResponse(ArrayOf(cycleRead))
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			cycleReadResponses(false, false)
		})
	})

	Method("archived_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_archived_cycles_by_pk")
		Security(planeSessionCookie)
		Payload(func() { cycleProjectPayload("pk") })
		Result(rawJSONResponseMetadata)
		cycleReadErrors(true, false)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/archived-cycles/{pk}/")
			rawJSONReadResponse(cycleRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			cycleReadResponses(true, false)
		})
	})

	Method("list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_cycles")
		Security(planeSessionCookie)
		Payload(func() {
			cycleProjectPayload("")
			Attribute("cycle_view", String)
		})
		Result(rawJSONResponseMetadata)
		cycleReadErrors(false, false)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/cycles/")
			rawJSONReadResponse(ArrayOf(cycleRead))
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("cycle_view")
			cycleReadResponses(false, false)
		})
	})

	Method("analytics", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_analytics")
		Security(planeSessionCookie)
		Payload(func() {
			cycleProjectPayload("cycle_id")
			Attribute("type", String)
		})
		Result(rawJSONResponseMetadata)
		cycleReadErrors(true, true)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/")
			rawJSONReadResponse(cycleAnalytics)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("cycle_id")
			Param("type")
			cycleReadResponses(true, true)
		})
	})

	Method("archive_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_archive")
		Security(planeSessionCookie)
		Payload(func() { cycleProjectPayload("cycle_id") })
		Result(rawJSONResponseMetadata)
		cycleReadErrors(true, false)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/archive/")
			rawJSONReadResponse(cycleRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("cycle_id")
			cycleReadResponses(true, false)
		})
	})

	Method("issues", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues")
		Security(planeSessionCookie)
		Payload(func() {
			cycleProjectPayload("cycle_id")
			Attribute("order_by", String)
			Attribute("group_by", String)
			Attribute("sub_group_by", String)
			Attribute("cursor", String)
			Attribute("per_page", String)
			cycleIssueFilterPayload()
		})
		Result(rawJSONResponseMetadata)
		cycleReadErrors(false, true)
		Error("invalid_pagination", cycleReadsInvalidPagination)
		Error("invalid_grouping", cycleReadsInvalidGrouping)
		Error("parse_error", cycleReadsParseError)
		Error("filter_error", cycleReadsFilterError)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/")
			rawJSONReadResponse(cycleIssuePage)
			Cookie("session_id:session-id")
			for _, name := range []string{
				"slug", "project_id", "cycle_id", "order_by", "group_by", "sub_group_by", "cursor", "per_page",
				"state", "state_group", "estimate_point", "priority", "assignees", "labels", "created_by", "logged_by", "parent",
				"mentions", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "project", "cycle",
				"module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date", "type", "filters",
			} {
				Param(name)
			}
			cycleReadResponses(false, true)
			Response("invalid_pagination", StatusBadRequest)
			Response("invalid_grouping", StatusBadRequest)
			Response("parse_error", StatusBadRequest)
			Response("filter_error", StatusBadRequest)
		})
	})

	Method("issue_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_cycle_issues_by_issue_id")
		Security(planeSessionCookie)
		Payload(func() {
			cycleProjectPayload("cycle_id")
			Attribute("issue_id", String, func() { Format(FormatUUID) })
			Required("issue_id")
		})
		Result(rawJSONResponseMetadata)
		cycleReadErrors(true, false)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/cycle-issues/{issue_id}/")
			rawJSONReadResponse(cycleIssueRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("cycle_id")
			Param("issue_id")
			cycleReadResponses(true, false)
		})
	})

	Method("progress", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_progress")
		Security(planeSessionCookie)
		Payload(func() { cycleProjectPayload("cycle_id") })
		Result(rawJSONResponseMetadata)
		cycleReadErrors(true, false)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/")
			rawJSONReadResponse(cycleProgress)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("cycle_id")
			cycleReadResponses(true, false)
		})
	})

	Method("detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_cycles_by_pk")
		Security(planeSessionCookie)
		Payload(func() { cycleProjectPayload("pk") })
		Result(rawJSONResponseMetadata)
		cycleReadErrors(true, false)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/cycles/{pk}/")
			rawJSONReadResponse(cycleRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			cycleReadResponses(true, false)
		})
	})

	Method("workspace_list_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_cycles")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Required("slug")
		})
		Result(cycleReadsMetadata)
		Error("unauthenticated", cycleReadsUnauthenticated)
		Error("workspace_forbidden", cycleReadsWorkspaceForbidden)
		Error("internal_error", cycleReadsInternalError, func() { Fault() })
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/cycles/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("workspace_forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("analytics_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_analytics")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("cycle_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "cycle_id")
		})
		Result(cycleReadsMetadata)
		Error("unauthenticated", cycleReadsUnauthenticated)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/analytics/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("cycle_id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
		})
	})

	Method("progress_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_projects_by_project_id_cycles_by_cycle_id_progress")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("cycle_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "cycle_id")
		})
		Result(cycleReadsMetadata)
		Error("unauthenticated", cycleReadsUnauthenticated)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/cycles/{cycle_id}/progress/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("cycle_id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
		})
	})
})
