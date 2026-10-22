package design

import . "github.com/CaliLuke/loom/dsl"

var intakeInboxDraftIssue = Type("IntakeInboxDraftIssue", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String, func() { Nullable() })
	Attribute("state_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("sort_order", Float64)
	Attribute("completed_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("estimate_point", String, func() { Format(FormatUUID); Nullable() })
	Attribute("priority", String, func() { Enum("urgent", "high", "medium", "low", "none") })
	Attribute("start_date", String, func() { Format(FormatDate); Nullable() })
	Attribute("target_date", String, func() { Format(FormatDate); Nullable() })
	Attribute("project_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("parent_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("cycle_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("module_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("label_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("assignee_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("type_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("description_html", String)
	Required("id", "name", "state_id", "sort_order", "completed_at", "estimate_point", "priority", "start_date", "target_date", "project_id", "parent_id", "cycle_id", "module_ids", "label_ids", "assignee_ids", "created_at", "updated_at", "created_by", "updated_by", "type_id", "description_html")
})

var intakeInboxIssueLite = Type("IntakeInboxIssueLite", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("priority", String, func() { Enum("urgent", "high", "medium", "low", "none") })
	Attribute("sequence_id", Int)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("label_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "name", "priority", "sequence_id", "project_id", "created_at", "label_ids", "created_by")
})

var intakeInboxDuplicateIssueLite = Type("IntakeInboxDuplicateIssueLite", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("priority", String, func() { Enum("urgent", "high", "medium", "low", "none") })
	Attribute("sequence_id", Int)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "name", "priority", "sequence_id", "project_id", "created_at", "created_by")
})

var intakeInboxIssueListItem = Type("IntakeInboxIssueListItem", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("status", Int)
	Attribute("duplicate_to", String, func() { Format(FormatUUID); Nullable() })
	Attribute("snoozed_till", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("source", String, func() { Nullable() })
	Attribute("issue", intakeInboxIssueLite)
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "status", "duplicate_to", "snoozed_till", "source", "issue", "created_by")
})

var intakeInboxIssueDetailIssue = Type("IntakeInboxIssueDetailIssue", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("state_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("sort_order", Float64)
	Attribute("completed_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("estimate_point", String, func() { Format(FormatUUID); Nullable() })
	Attribute("priority", String, func() { Enum("urgent", "high", "medium", "low", "none") })
	Attribute("start_date", String, func() { Format(FormatDate); Nullable() })
	Attribute("target_date", String, func() { Format(FormatDate); Nullable() })
	Attribute("sequence_id", Int)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("parent_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("label_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("assignee_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("is_draft", Boolean)
	Attribute("archived_at", String, func() { Format(FormatDate); Nullable() })
	Attribute("description_html", String)
	Required("id", "name", "state_id", "sort_order", "completed_at", "estimate_point", "priority", "start_date", "target_date", "sequence_id", "project_id", "parent_id", "label_ids", "assignee_ids", "created_at", "updated_at", "created_by", "updated_by", "is_draft", "archived_at", "description_html")
})

var intakeInboxIssueDetail = Type("IntakeInboxIssueDetail", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("status", Int)
	Attribute("duplicate_to", String, func() { Format(FormatUUID); Nullable() })
	Attribute("snoozed_till", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("duplicate_issue_detail", intakeInboxDuplicateIssueLite, func() { Nullable() })
	Attribute("source", String, func() { Nullable() })
	Attribute("issue", intakeInboxIssueDetailIssue)
	Required("id", "status", "duplicate_to", "snoozed_till", "duplicate_issue_detail", "source", "issue")
})

var intakeInboxProjectLite = Type("IntakeInboxProjectLite", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("name", String)
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("cover_image_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("logo_props", Any)
	Attribute("description", String)
	Required("id", "identifier", "name", "cover_image", "cover_image_url", "logo_props", "description")
})

var intakeInbox = Type("IntakeInbox", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("description", String)
	Attribute("is_default", Boolean)
	Attribute("view_props", Any)
	Attribute("logo_props", Any)
	Attribute("project_detail", intakeInboxProjectLite)
	Attribute("pending_issue_count", Int)
	Required("id", "created_at", "updated_at", "deleted_at", "created_by", "updated_by", "workspace", "project", "name", "description", "is_default", "view_props", "logo_props", "project_detail", "pending_issue_count")
})

var intakeInboxDescriptionVersionSummary = Type("IntakeInboxDescriptionVersionSummary", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("issue", String, func() { Format(FormatUUID) })
	Attribute("last_saved_at", String, func() { Format(FormatDateTime) })
	Attribute("owned_by", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "workspace", "project", "issue", "last_saved_at", "owned_by", "created_at", "updated_at", "created_by", "updated_by")
})

var intakeInboxDescriptionVersionDetail = Type("IntakeInboxDescriptionVersionDetail", func() {
	Extend(intakeInboxDescriptionVersionSummary)
	Attribute("description_binary", Bytes, func() {
		Nullable()
		Meta("openapi:format", "binary")
	})
	Attribute("description_html", String)
	Attribute("description_stripped", String, func() { Nullable() })
	Attribute("description_json", Any)
	Required("description_binary", "description_html", "description_stripped", "description_json")
})

var intakeInboxIssuePage = Type("IntakeInboxIssuePage", func() {
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
	Attribute("results", ArrayOf(intakeInboxIssueListItem))
	Required("grouped_by", "sub_grouped_by", "total_count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results", "count", "total_pages", "total_results", "extra_stats", "results")
})

var intakeInboxDraftPage = Type("IntakeInboxDraftPage", func() {
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
	Attribute("results", ArrayOf(intakeInboxDraftIssue))
	Required("grouped_by", "sub_grouped_by", "total_count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results", "count", "total_pages", "total_results", "extra_stats", "results")
})

var intakeInboxDescriptionVersionPage = Type("IntakeInboxDescriptionVersionPage", func() {
	Attribute("prev_cursor", String)
	Attribute("cursor", String)
	Attribute("next_cursor", String, func() { Nullable() })
	Attribute("prev_page_results", Boolean)
	Attribute("next_page_results", Boolean)
	Attribute("page_count", Int)
	Attribute("total_results", Int)
	Attribute("total_pages", Int)
	Attribute("results", ArrayOf(intakeInboxDescriptionVersionSummary))
	Required("prev_cursor", "cursor", "next_cursor", "prev_page_results", "next_page_results", "page_count", "total_results", "total_pages", "results")
})

var intakeInboxReadsUnauthenticated = Type("IntakeInboxReadsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var intakeInboxReadsPermissionDenied = Type("IntakeInboxReadsPermissionDenied", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var intakeInboxReadsBadRequest = Type("IntakeInboxReadsBadRequest", func() {
	Attribute("detail", String)
	Required("detail")
})

var intakeInboxReadsInvalidDetail = Type("IntakeInboxReadsInvalidDetail", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var intakeInboxReadsMetadata = Type("IntakeInboxReadsMetadata", func() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("renders", ArrayOf(String))
	Attribute("parses", ArrayOf(String))
	Required("name", "description", "renders", "parses")
})

var intakeInboxReadsNotFound = Type("IntakeInboxReadsNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var intakeInboxReadsInternalError = Type("IntakeInboxReadsInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var intakeInboxReadsDetailNotFound = Type("IntakeInboxReadsDetailNotFound", func() {
	Attribute("detail", String)
	Required("detail")
})

func intakeInboxProjectPayload() {
	stickySessionPayload()
	Attribute("slug", String)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Required("slug", "project_id")
}

func intakeInboxIssueFilterPayload() {
	Attribute("cursor", String)
	Attribute("per_page", String)
	Attribute("order_by", String)
	Attribute("search", String)
	Attribute("status", String)
	for _, name := range []string{"state", "state_group", "estimate_point", "priority", "parent", "labels", "assignees", "mentions", "created_by", "logged_by", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "type", "project", "cycle", "module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date"} {
		Attribute(name, String)
	}
}

func intakeInboxIssueFilterParams() {
	for _, name := range []string{"cursor", "per_page", "order_by", "search", "status", "state", "state_group", "estimate_point", "priority", "parent", "labels", "assignees", "mentions", "created_by", "logged_by", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "type", "project", "cycle", "module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date"} {
		Param(name)
	}
}

func intakeInboxStandardErrors() {
	Error("unauthenticated", intakeInboxReadsUnauthenticated)
	Error("permission_denied", intakeInboxReadsPermissionDenied)
	Error("bad_request", intakeInboxReadsBadRequest)
	Error("invalid_detail", intakeInboxReadsInvalidDetail)
	Error("not_found", intakeInboxReadsNotFound)
	Error("internal_error", intakeInboxReadsInternalError, func() { Fault() })
}

func intakeInboxDetailErrors() {
	Error("unauthenticated", intakeInboxReadsUnauthenticated)
	Error("permission_denied", intakeInboxReadsPermissionDenied)
	Error("bad_request", intakeInboxReadsBadRequest)
	Error("invalid_detail", intakeInboxReadsInvalidDetail)
	Error("detail_not_found", intakeInboxReadsDetailNotFound)
	Error("internal_error", intakeInboxReadsInternalError, func() { Fault() })
}

func intakeInboxHTTPResponses() {
	Response("unauthenticated", StatusUnauthorized)
	Response("permission_denied", StatusForbidden)
	Response("bad_request", StatusBadRequest)
	Response("invalid_detail", StatusBadRequest)
	Response("not_found", StatusNotFound)
	Response("internal_error", StatusInternalServerError)
}

func intakeInboxDetailHTTPResponses() {
	Response("unauthenticated", StatusUnauthorized)
	Response("permission_denied", StatusForbidden)
	Response("bad_request", StatusBadRequest)
	Response("invalid_detail", StatusBadRequest)
	Response("detail_not_found", StatusNotFound)
	Response("internal_error", StatusInternalServerError)
}

var _ = Service("intake_inbox_reads", func() {
	Description("Returns authenticated workspace draft, project intake, inbox, and description-version read models.")
	HTTP(func() {
		RuntimeCORS()
	})

	Method("draft_issue_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_draft_issues")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			intakeInboxIssueFilterPayload()
			Required("slug")
		})
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/draft-issues/")
			rawJSONReadResponse(intakeInboxDraftPage)
			Cookie("session_id:session-id")
			Param("slug")
			intakeInboxIssueFilterParams()
			intakeInboxHTTPResponses()
		})
	})

	Method("draft_issue_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_draft_issues_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "pk")
		})
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/draft-issues/{pk}/")
			rawJSONReadResponse(intakeInboxDraftIssue)
			Cookie("session_id:session-id")
			Param("slug")
			Param("pk")
			intakeInboxHTTPResponses()
		})
	})

	Method("inbox_issue_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_inbox_issues")
		Security(planeSessionCookie)
		Payload(func() { intakeInboxProjectPayload(); intakeInboxIssueFilterPayload() })
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/")
			rawJSONReadResponse(intakeInboxIssuePage)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			intakeInboxIssueFilterParams()
			intakeInboxHTTPResponses()
		})
	})

	Method("inbox_issue_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_inbox_issues_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			intakeInboxProjectPayload()
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("pk")
		})
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/inbox-issues/{pk}/")
			rawJSONReadResponse(intakeInboxIssueDetail)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			intakeInboxHTTPResponses()
		})
	})

	Method("inbox_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_inboxes")
		Security(planeSessionCookie)
		Payload(func() { intakeInboxProjectPayload(); Attribute("search", String) })
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/inboxes/")
			// Django returns either the intake object or an empty object when none exists.
			rawJSONReadResponse(Any)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("search")
			intakeInboxHTTPResponses()
		})
	})

	Method("inbox_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_inboxes_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			intakeInboxProjectPayload()
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("pk")
		})
		Result(rawJSONResponseMetadata)
		intakeInboxDetailErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/")
			rawJSONReadResponse(intakeInbox)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			intakeInboxDetailHTTPResponses()
		})
	})

	Method("intake_issue_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_intake_issues")
		Security(planeSessionCookie)
		Payload(func() { intakeInboxProjectPayload(); intakeInboxIssueFilterPayload() })
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/intake-issues/")
			rawJSONReadResponse(intakeInboxIssuePage)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			intakeInboxIssueFilterParams()
			intakeInboxHTTPResponses()
		})
	})

	Method("intake_issue_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_intake_issues_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			intakeInboxProjectPayload()
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("pk")
		})
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/intake-issues/{pk}/")
			rawJSONReadResponse(intakeInboxIssueDetail)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			intakeInboxHTTPResponses()
		})
	})

	Method("description_version_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_intake_work_items_by_work_item_id_description_versions")
		Security(planeSessionCookie)
		Payload(func() {
			intakeInboxProjectPayload()
			Attribute("work_item_id", String, func() { Format(FormatUUID) })
			Attribute("cursor", String)
			Required("work_item_id")
		})
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/")
			rawJSONReadResponse(intakeInboxDescriptionVersionPage)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Param("cursor")
			intakeInboxHTTPResponses()
		})
	})

	Method("description_version_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_intake_work_items_by_work_item_id_description_versions_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			intakeInboxProjectPayload()
			Attribute("work_item_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("work_item_id", "pk")
		})
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/")
			rawJSONReadResponse(intakeInboxDescriptionVersionDetail)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Param("pk")
			intakeInboxHTTPResponses()
		})
	})

	Method("description_version_list_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_projects_by_project_id_intake_work_items_by_work_item_id_description_versions")
		Security(planeSessionCookie)
		Payload(func() {
			intakeInboxProjectPayload()
			Attribute("work_item_id", String, func() { Format(FormatUUID) })
			Required("work_item_id")
		})
		Result(intakeInboxReadsMetadata)
		Error("unauthenticated", intakeInboxReadsUnauthenticated)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
		})
	})

	Method("description_version_detail_options", func() {
		Meta("openapi:operationId", "options_api_workspaces_by_slug_projects_by_project_id_intake_work_items_by_work_item_id_description_versions_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			intakeInboxProjectPayload()
			Attribute("work_item_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("work_item_id", "pk")
		})
		Result(intakeInboxReadsMetadata)
		Error("unauthenticated", intakeInboxReadsUnauthenticated)
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/intake-work-items/{work_item_id}/description-versions/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Param("pk")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
		})
	})

	Method("intake_list", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_intakes")
		Security(planeSessionCookie)
		Payload(func() { intakeInboxProjectPayload(); Attribute("search", String) })
		Result(rawJSONResponseMetadata)
		intakeInboxStandardErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/intakes/")
			// Django returns either the intake object or an empty object when none exists.
			rawJSONReadResponse(Any)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("search")
			intakeInboxHTTPResponses()
		})
	})

	Method("intake_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_intakes_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			intakeInboxProjectPayload()
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("pk")
		})
		Result(rawJSONResponseMetadata)
		intakeInboxDetailErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/")
			rawJSONReadResponse(intakeInbox)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			intakeInboxDetailHTTPResponses()
		})
	})
})
