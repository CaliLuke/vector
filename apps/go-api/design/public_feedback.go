package design

import . "github.com/CaliLuke/loom/dsl"

var publicFeedbackActor = Type("PublicFeedbackActor", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("first_name", String, func() { Nullable() })
	Attribute("last_name", String, func() { Nullable() })
	Attribute("avatar", String, func() { Nullable() })
	Attribute("avatar_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("display_name", String, func() { Nullable() })
	Required("id", "first_name", "last_name", "avatar", "avatar_url", "display_name")
})

var publicFeedbackVoteItem = Type("PublicFeedbackVoteItem", func() {
	Attribute("vote", Int)
	Attribute("actor_details", publicFeedbackActor)
	Required("vote", "actor_details")
})

var publicFeedbackReactionItem = Type("PublicFeedbackReactionItem", func() {
	Attribute("reaction", String)
	Attribute("actor_details", publicFeedbackActor)
	Required("reaction", "actor_details")
})

var publicFeedbackIssueCollectionItem = Type("PublicFeedbackIssueCollectionItem", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("state_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("sort_order", Float64)
	Attribute("estimate_point", String, func() { Format(FormatUUID); Nullable() })
	Attribute("priority", String, func() { Nullable() })
	Attribute("start_date", String, func() { Format(FormatDate); Nullable() })
	Attribute("target_date", String, func() { Format(FormatDate); Nullable() })
	Attribute("sequence_id", Int)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("parent_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("cycle_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("state__group", String)
	Attribute("assignee_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("label_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("module_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("vote_items", ArrayOf(publicFeedbackVoteItem), func() { Nullable() })
	Attribute("reaction_items", ArrayOf(publicFeedbackReactionItem), func() { Nullable() })
	Required("id", "name", "state_id", "sort_order", "estimate_point", "priority", "start_date", "target_date", "sequence_id", "project_id", "parent_id", "cycle_id", "created_by", "state__group", "assignee_ids", "label_ids", "module_ids", "vote_items", "reaction_items")
})

var publicFeedbackIssueDetail = Type("PublicFeedbackIssueDetail", func() {
	Extend(publicFeedbackIssueCollectionItem)
	Attribute("description_json", Any)
	Attribute("description_html", String)
	Attribute("description_stripped", String, func() { Nullable() })
	Attribute("description_binary", Bytes, func() { Nullable(); Meta("openapi:format", "binary") })
	Required("description_json", "description_html", "description_stripped", "description_binary")
})

var publicFeedbackIssuePage = Type("PublicFeedbackIssuePage", func() {
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
	Attribute("results", Any)
	Required("grouped_by", "sub_grouped_by", "total_count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results", "count", "total_pages", "total_results", "extra_stats", "results")
})

var publicFeedbackIntakeCard = Type("PublicFeedbackIntakeCard", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("duplicate_to", String, func() { Format(FormatUUID); Nullable() })
	Attribute("snoozed_till", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("source", String)
	Attribute("status", Int)
	Required("id", "duplicate_to", "snoozed_till", "source", "status")
})

var publicFeedbackIntakeProject = Type("PublicFeedbackIntakeProject", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("name", String)
	Attribute("description", String)
	Attribute("logo_props", Any)
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("cover_image_url", String, func() {
		Nullable()
		Description("Nullable legacy serializer value from cover_image_asset.asset_url or the raw cover_image field; the source does not require URI syntax.")
		Meta("loom:vet:ignore", "string-format")
	})
	Required("id", "identifier", "name", "description", "logo_props", "cover_image", "cover_image_url")
})

var publicFeedbackIntakeState = Type("PublicFeedbackIntakeState", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("group", String)
	Attribute("color", String)
	Required("id", "name", "group", "color")
})

var publicFeedbackIntakeIssue = Type("PublicFeedbackIntakeIssue", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("description_html", String)
	Attribute("description_json", Any)
	Attribute("description_stripped", String, func() { Nullable() })
	Attribute("description_binary", Bytes, func() { Nullable(); Meta("openapi:format", "binary") })
	Attribute("priority", String, func() { Nullable() })
	Attribute("point", Any, func() { Nullable() })
	Attribute("estimate_point", String, func() { Format(FormatUUID); Nullable() })
	Attribute("sequence_id", Int)
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() { Format(FormatDate); Nullable() })
	Attribute("target_date", String, func() { Format(FormatDate); Nullable() })
	Attribute("completed_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("archived_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Attribute("state", String, func() { Format(FormatUUID); Nullable() })
	Attribute("state_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("parent", Any, func() { Nullable() })
	Attribute("parent_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("type", String, func() { Format(FormatUUID); Nullable() })
	Attribute("external_id", String, func() { Nullable() })
	Attribute("external_source", String, func() { Nullable() })
	Attribute("is_draft", Boolean)
	Attribute("assignee_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("assignees", ArrayOf(Any))
	Attribute("assignee_details", ArrayOf(Any))
	Attribute("label_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("labels", ArrayOf(Any))
	Attribute("label_details", ArrayOf(Any))
	Attribute("issue_intake", ArrayOf(publicFeedbackIntakeCard))
	Attribute("project_detail", publicFeedbackIntakeProject)
	Attribute("state_detail", publicFeedbackIntakeState, func() { Nullable() })
	Attribute("sub_issues_count", Int)
})

var publicFeedbackVote = Type("PublicFeedbackVote", func() {
	Attribute("issue", String, func() { Format(FormatUUID) })
	Attribute("vote", Int)
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("actor", String, func() { Format(FormatUUID) })
	Attribute("actor_detail", Any)
	Required("issue", "vote", "workspace", "project", "actor", "actor_detail")
})

var publicFeedbackVoteCreated = Type("PublicFeedbackVoteCreated", func() {
	Attribute("issue", String, func() { Format(FormatUUID) })
	Attribute("vote", Any)
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("actor", String, func() { Format(FormatUUID) })
	Attribute("actor_detail", Any)
	Required("issue", "vote", "workspace", "project", "actor", "actor_detail")
})

var publicFeedbackDirectoryProject = Type("PublicFeedbackDirectoryProject", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("name", String)
	Attribute("description", String)
	Attribute("emoji", String, func() { Nullable() })
	Attribute("icon_prop", Any, func() { Nullable() })
	Attribute("cover_image", String, func() { Nullable() })
	Required("id", "identifier", "name", "description", "emoji", "icon_prop", "cover_image")
})

var publicFeedbackResponseMetadata = Type("PublicFeedbackResponseMetadata", func() {
	Attribute("content_type", String)
})

var publicFeedbackError = Type("PublicFeedbackError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("kind", "body")
})

var publicFeedbackRequestBody = MapOf(String, Any)

func publicFeedbackCookiePayload() {
	APIKey("plane_session_cookie", "session_id", String)
}

func publicFeedbackIssueQueryPayload() {
	Attribute("search", String)
	Attribute("order_by", String)
	Attribute("per_page", String)
	Attribute("cursor", String)
	Attribute("group_by", String)
	Attribute("sub_group_by", String)
	Attribute("state", String)
	Attribute("state_group", String)
	Attribute("estimate_point", String)
	Attribute("priority", String)
	Attribute("assignees", String)
	Attribute("labels", String)
	Attribute("created_by", String)
	Attribute("parent", String)
	Attribute("mentions", String)
	Attribute("name", String)
	Attribute("created_at", String)
	Attribute("project", String)
	Attribute("logged_by", String)
	Attribute("updated_at", String)
	Attribute("start_date", String)
	Attribute("target_date", String)
	Attribute("completed_at", String)
	Attribute("cycle", String)
	Attribute("module", String)
	Attribute("intake_status", String)
	Attribute("inbox_status", String)
	Attribute("sub_issue", String)
	Attribute("subscriber", String)
	Attribute("start_target_date", String)
	Attribute("type", String)
}

func publicFeedbackIssueQueryParams() {
	for _, name := range []string{
		"search", "order_by", "per_page", "cursor", "group_by", "sub_group_by",
		"state", "state_group", "estimate_point", "priority", "assignees", "labels", "created_by",
		"parent", "mentions", "name", "created_at", "project", "logged_by", "updated_at", "start_date",
		"target_date", "completed_at", "cycle", "module", "intake_status", "inbox_status", "sub_issue",
		"subscriber", "start_target_date", "type",
	} {
		Param(name)
	}
}

func publicFeedbackJSONResponse(status int, schema any, nullable bool) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Header("content_type:Content-Type")
		if nullable {
			OpenAPIBody(schema, func() { Nullable() })
		} else {
			OpenAPIBody(schema)
		}
	})
}

func publicFeedbackJSONErrors() {
	for _, item := range []struct {
		name string
		code int
	}{
		{"status_400", 400}, {"status_401", 401}, {"status_403", 403}, {"status_404", 404},
		{"status_405", 405}, {"status_406", 406}, {"status_413", 413}, {"status_415", 415},
		{"status_429", 429}, {"status_500", 500},
	} {
		Error(item.name, publicFeedbackError, func() {
			if item.code == 500 {
				Fault()
			}
			if item.code == 429 {
				Temporary()
			}
		})
		HTTP(func() {
			Response(item.name, item.code, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body("body")
			})
		})
	}
}

func publicFeedbackRawRequestBody() {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(publicFeedbackRequestBody, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
}

var _ = Service("public_feedback", func() {
	Description("Serves public issue, intake, vote, and project-board directory routes.")

	Method("issue_list", func() {
		Meta("openapi:operationId", "get_api_public_anchor_by_anchor_issues")
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("anchor", String)
			publicFeedbackIssueQueryPayload()
			Required("anchor")
		})
		Result(publicFeedbackResponseMetadata)
		publicFeedbackJSONErrors()
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/issues/")
			Cookie("session_id:session-id")
			Param("anchor")
			publicFeedbackIssueQueryParams()
			publicFeedbackJSONResponse(200, publicFeedbackIssuePage, false)
		})
	})

	Method("issue_detail", func() {
		Meta("openapi:operationId", "get_api_public_anchor_by_anchor_issues_by_issue_id")
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("anchor", String)
			Attribute("issue_id", String)
			Required("anchor", "issue_id")
		})
		Result(publicFeedbackResponseMetadata)
		publicFeedbackJSONErrors()
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/issues/{issue_id}/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("issue_id")
			publicFeedbackJSONResponse(200, publicFeedbackIssueDetail, true)
		})
	})

	Method("vote_list", func() {
		Meta("openapi:operationId", "get_api_public_anchor_by_anchor_issues_by_issue_id_votes")
		Security(planeSessionCookie)
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("anchor", String)
			Attribute("issue_id", String)
			Attribute("search", String)
			Required("anchor", "issue_id")
		})
		Result(publicFeedbackResponseMetadata)
		publicFeedbackJSONErrors()
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/issues/{issue_id}/votes/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("issue_id")
			Param("search")
			publicFeedbackJSONResponse(200, ArrayOf(publicFeedbackVote), false)
		})
	})

	Method("vote_create", func() {
		Meta("openapi:operationId", "post_api_public_anchor_by_anchor_issues_by_issue_id_votes")
		Security(planeSessionCookie)
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("anchor", String)
			Attribute("issue_id", String)
			Required("anchor", "issue_id")
		})
		Result(publicFeedbackResponseMetadata)
		publicFeedbackJSONErrors()
		HTTP(func() {
			POST("/api/public/anchor/{anchor}/issues/{issue_id}/votes/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("issue_id")
			publicFeedbackRawRequestBody()
			publicFeedbackJSONResponse(201, publicFeedbackVoteCreated, false)
		})
	})

	Method("vote_delete", func() {
		Meta("openapi:operationId", "delete_api_public_anchor_by_anchor_issues_by_issue_id_votes")
		Security(planeSessionCookie)
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("anchor", String)
			Attribute("issue_id", String)
			Required("anchor", "issue_id")
		})
		Result(Empty)
		publicFeedbackJSONErrors()
		HTTP(func() {
			DELETE("/api/public/anchor/{anchor}/issues/{issue_id}/votes/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("issue_id")
			Response(StatusNoContent)
		})
	})

	for _, path := range []struct {
		name string
		url  string
	}{
		{"inbox", "/api/public/anchor/{anchor}/intakes/{intake_id}/inbox-issues/"},
		{"intake", "/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/"},
	} {
		collectionPath := path.url
		alias := path.name
		Method(alias+"_list", func() {
			Meta("openapi:operationId", "get_api_public_anchor_by_anchor_intakes_by_intake_id_"+alias+"_issues")
			Security(planeSessionCookie)
			Payload(func() {
				publicFeedbackCookiePayload()
				Attribute("anchor", String)
				Attribute("intake_id", String)
				publicFeedbackIssueQueryPayload()
				Required("anchor", "intake_id")
			})
			Result(publicFeedbackResponseMetadata)
			publicFeedbackJSONErrors()
			HTTP(func() {
				GET(collectionPath)
				Cookie("session_id:session-id")
				Param("anchor")
				Param("intake_id")
				publicFeedbackIssueQueryParams()
				publicFeedbackJSONResponse(200, ArrayOf(publicFeedbackIntakeIssue), false)
			})
		})
		Method(alias+"_create", func() {
			operation := "post_api_public_anchor_by_anchor_intakes_by_intake_id_" + alias + "_issues"
			Meta("openapi:operationId", operation)
			Security(planeSessionCookie)
			Payload(func() {
				publicFeedbackCookiePayload()
				Attribute("anchor", String)
				Attribute("intake_id", String)
				Required("anchor", "intake_id")
			})
			Result(publicFeedbackResponseMetadata)
			publicFeedbackJSONErrors()
			HTTP(func() {
				POST(collectionPath)
				Cookie("session_id:session-id")
				Param("anchor")
				Param("intake_id")
				publicFeedbackRawRequestBody()
				publicFeedbackJSONResponse(200, publicFeedbackIntakeIssue, false)
			})
		})
	}

	Method("intake_detail", func() {
		Meta("openapi:operationId", "get_api_public_anchor_by_anchor_intakes_by_intake_id_intake_issues_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("anchor", String)
			Attribute("intake_id", String)
			Attribute("pk", String)
			Required("anchor", "intake_id", "pk")
		})
		Result(publicFeedbackResponseMetadata)
		publicFeedbackJSONErrors()
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("intake_id")
			Param("pk")
			publicFeedbackJSONResponse(200, publicFeedbackIntakeIssue, false)
		})
	})

	Method("intake_update", func() {
		Meta("openapi:operationId", "patch_api_public_anchor_by_anchor_intakes_by_intake_id_intake_issues_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("anchor", String)
			Attribute("intake_id", String)
			Attribute("pk", String)
			Required("anchor", "intake_id", "pk")
		})
		Result(publicFeedbackResponseMetadata)
		publicFeedbackJSONErrors()
		HTTP(func() {
			PATCH("/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("intake_id")
			Param("pk")
			publicFeedbackRawRequestBody()
			publicFeedbackJSONResponse(200, publicFeedbackIntakeIssue, false)
		})
	})

	Method("intake_delete", func() {
		Meta("openapi:operationId", "delete_api_public_anchor_by_anchor_intakes_by_intake_id_intake_issues_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("anchor", String)
			Attribute("intake_id", String)
			Attribute("pk", String)
			Required("anchor", "intake_id", "pk")
		})
		Result(Empty)
		publicFeedbackJSONErrors()
		HTTP(func() {
			DELETE("/api/public/anchor/{anchor}/intakes/{intake_id}/intake-issues/{pk}/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("intake_id")
			Param("pk")
			Response(StatusNoContent)
		})
	})

	Method("workspace_project_boards", func() {
		Meta("openapi:operationId", "get_api_public_workspaces_by_slug_project_boards")
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			publicFeedbackCookiePayload()
			Attribute("slug", String)
			Attribute("search", String)
			Required("slug")
		})
		Result(publicFeedbackResponseMetadata)
		publicFeedbackJSONErrors()
		HTTP(func() {
			GET("/api/public/workspaces/{slug}/project-boards/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("search")
			publicFeedbackJSONResponse(200, ArrayOf(publicFeedbackDirectoryProject), false)
		})
	})
})
