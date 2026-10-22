// Package design defines the reverse-generated work-item activity read contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/activityreads/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

const djangoURLUUIDPattern = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"

var activityErrorResponseBody = Type("ActivityErrorResponseBody", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
	Required("name")
})

var activityHTTPErrorResponseBody = Type("ActivityHTTPErrorResponseBody", func() {
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
})

var ImportedPlaneAPISerializersIssueIssueactivityserializerResponse = Type("PlaneApiSerializersIssueIssueactivityserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for work item activity and change history.\n\nTracks and represents work item modifications, state changes,\nand user interactions for audit trails and activity feeds.")
	Attribute("actor", Any, func() {
		Nullable()
	})
	Attribute("attachments", ArrayOf(String, func() {
		Format(FormatURI)
		MaxLength(200)
	}), func() {
		MaxLength(10)
	})
	Attribute("comment", String)
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("epoch", Float64, func() {
		Nullable()
	})
	Attribute("field", String, func() {
		Nullable()
		Title("Field Name")
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", Any, func() {
		Nullable()
	})
	Attribute("issue_comment", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("new_identifier", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("new_value", String, func() {
		Nullable()
	})
	Attribute("old_identifier", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("old_value", String, func() {
		Nullable()
	})
	Attribute("project", Any)
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("verb", String, func() {
		Title("Action")
		MaxLength(255)
	})
	Attribute("workspace", Any)
	Required("project", "workspace")
})

var ImportedPlaneAppSerializersIssueIssueactivityserializerResponse = Type("PlaneAppSerializersIssueIssueactivityserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("actor", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("actor_detail", ImportedPlaneAppSerializersUserUserliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("attachments", ArrayOf(String, func() {
		Format(FormatURI)
		MaxLength(200)
	}), func() {
		MaxLength(10)
	})
	Attribute("comment", String)
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("epoch", Float64, func() {
		Nullable()
	})
	Attribute("field", String, func() {
		Nullable()
		Title("Field Name")
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("issue", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("issue_comment", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("issue_detail", ImportedPlaneAppSerializersIssueIssueflatserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("new_identifier", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("new_value", String, func() {
		Nullable()
	})
	Attribute("old_identifier", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("old_value", String, func() {
		Nullable()
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
	})
	Attribute("project_detail", ImportedPlaneAppSerializersProjectProjectliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("source_data", Any, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("verb", String, func() {
		Title("Action")
		MaxLength(255)
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
	})
	Attribute("workspace_detail", ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Required("project", "workspace")
})

var ImportedPlaneAppSerializersIssueIssueflatserializerResponse = Type("PlaneAppSerializersIssueIssueflatserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_draft", Boolean)
	Attribute("name", String, func() {
		Title("Issue Name")
		MaxLength(255)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("sequence_id", Int, func() {
		Title("Issue Sequence ID")
		Meta("openapi:format", "")
		Minimum(-2.147483648e+09)
		Maximum(2.147483647e+09)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Required("name")
})

var ImportedPlaneAppSerializersProjectProjectliteserializerResponse = Type("PlaneAppSerializersProjectProjectliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("cover_image", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("cover_image_url", String, func() {
		Nullable()
		Format(FormatURIReference)
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("Project Description")
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("identifier", String, func() {
		Title("Project Identifier")
		Meta("openapi:readOnly", "true")
	})
	Attribute("logo_props", Any, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Project Name")
		Meta("openapi:readOnly", "true")
	})
})

var ImportedPlaneAppSerializersUserUserliteserializerResponse = Type("PlaneAppSerializersUserUserliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("avatar", String)
	Attribute("avatar_url", String, func() {
		Nullable()
		Format(FormatURIReference)
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		MaxLength(255)
	})
	Attribute("first_name", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_bot", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("last_name", String, func() {
		MaxLength(255)
	})
})

var ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse = Type("PlaneAppSerializersWorkspaceWorkspaceliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("logo_url", String, func() {
		Nullable()
		Format(FormatURIReference)
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Workspace Name")
		Meta("openapi:readOnly", "true")
	})
	Attribute("slug", String, func() {
		Meta("openapi:readOnly", "true")
		Pattern("^[-a-zA-Z0-9_]+$")
	})
})

var ImportedPriorityEnum = Type("PriorityEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `urgent` - Urgent\n* `high` - High\n* `medium` - Medium\n* `low` - Low\n* `none` - None")
	Enum("urgent", "high", "medium", "low", "none")
})

var ImportedRestFrameworkSerializersPaginatedissueactivitydetailresponseResponse = Type("RestFrameworkSerializersPaginatedissueactivitydetailresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", String, func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIssueIssueactivityserializerResponse))
	Attribute("sub_grouped_by", String, func() {
		Nullable()
	})
	Attribute("total_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
})

var ImportedRestFrameworkSerializersPaginatedissueactivityresponseResponse = Type("RestFrameworkSerializersPaginatedissueactivityresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", String, func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIssueIssueactivityserializerResponse))
	Attribute("sub_grouped_by", String, func() {
		Nullable()
	})
	Attribute("total_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
})

var ImportedRestFrameworkSerializersUseractivitypageResponse = Type("RestFrameworkSerializersUseractivitypageResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", MapOf(String, Any), func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedPlaneAppSerializersIssueIssueactivityserializerResponse))
	Attribute("sub_grouped_by", String, func() {
		Nullable()
	})
	Attribute("total_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
})

var ImportedAPIKeyAuthSecurity = APIKeySecurity("ApiKeyAuth", func() {
	Description("Plane API token authentication.")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var ImportedPlaneSession = SessionAuth("PlaneSession", func() {
	CookieTransport(ImportedPlaneSessionCookieSecurity, "", func() {
		CookieName("session-id")
	})
})

var activityRawJSONResponseMetadata = Type("ActivityRawJSONResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var activityOptions = Type("ActivityOptions", func() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("renders", ArrayOf(String))
	Attribute("parses", ArrayOf(String))
	Required("name", "description", "renders", "parses")
})

func activityRawJSONResponse(schema any, description string) {
	SkipResponseBodyEncodeDecode()
	Response(200, func() {
		if description != "" {
			Description(description)
		}
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

var _ = API("Plane Internal Django Migration API", func() {
	Title("Plane Internal Django Migration API")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("http:generate", "server")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:work-item-activities")
})

var _ = Service("PlaneInternalDjangoMigrationAPI", func() {
	HTTP(func() {
		RuntimeCORS()
	})
	Method("GetAPIUsersMeActivities", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_users_me_activities")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/users/me/activities/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.user.base.UserActivityEndpoint\"")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
			Attribute("cursor", String)
			Attribute("orderBy", String)
			Attribute("perPage", Int, func() { Meta("openapi:format", "") })
		})
		Result(activityRawJSONResponseMetadata)
		Error("status_400", activityErrorResponseBody)
		Error("status_401", activityErrorResponseBody)
		Error("status_500", activityErrorResponseBody, func() { Fault() })
		HTTP(func() {
			Meta("openapi:tag:work-item-activities")
			GET("/api/users/me/activities/")
			Param("cursor")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			activityRawJSONResponse(ImportedRestFrameworkSerializersUseractivitypageResponse, "")

			Response("status_400", 400, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(activityHTTPErrorResponseBody) })
			Response("status_401", 401, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(activityHTTPErrorResponseBody) })
			Response("status_500", 500, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
		})
	})

	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivities", func() {
		Description("Retrieve all activities for a work item. Supports filtering by activity type and date range.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueActivityListAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", Int, func() {
				Description("Number of results per page (default: 1000, max: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(activityRawJSONResponseMetadata)
		Error("status_400", activityErrorResponseBody)
		Error("status_401", activityErrorResponseBody)
		Error("status_403", activityErrorResponseBody)
		Error("status_404", activityErrorResponseBody)
		Error("status_500", activityErrorResponseBody, func() { Fault() })
		Error("status_429", activityErrorResponseBody, func() { Temporary() })
		HTTP(func() {
			Meta("openapi:tag:work-item-activities")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			activityRawJSONResponse(ImportedRestFrameworkSerializersPaginatedissueactivityresponseResponse, "Paginated list of issue activities")
			Response("status_400", 400, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_401", 401, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_403", 403, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_404", 404, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})

			Response("status_500", 500, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_429", 429, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDIssuesByIssueIDActivitiesByPk", func() {
		Description("Retrieve details of a specific activity.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_activities_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueActivityDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", Int, func() {
				Description("Number of results per page (default: 1000, max: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("pk", String, func() {
				Description("Activity ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(activityRawJSONResponseMetadata)
		Error("status_400", activityErrorResponseBody)
		Error("status_401", activityErrorResponseBody)
		Error("status_403", activityErrorResponseBody)
		Error("status_404", activityErrorResponseBody)
		Error("status_500", activityErrorResponseBody, func() { Fault() })
		Error("status_429", activityErrorResponseBody, func() { Temporary() })
		HTTP(func() {
			Meta("openapi:tag:work-item-activities")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/activities/{pk}/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			activityRawJSONResponse(ImportedPlaneAPISerializersIssueIssueactivityserializerResponse, "Paginated list of work item activities")
			Response("status_400", 400, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_401", 401, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_403", 403, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_404", 404, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})

			Response("status_500", 500, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_429", 429, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivities", func() {
		Description("Retrieve all activities for a work item. Supports filtering by activity type and date range.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/activities/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueActivityListAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", Int, func() {
				Description("Number of results per page (default: 1000, max: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "project_id", "slug")
		})
		Result(activityRawJSONResponseMetadata)
		Error("status_400", activityErrorResponseBody)
		Error("status_401", activityErrorResponseBody)
		Error("status_403", activityErrorResponseBody)
		Error("status_404", activityErrorResponseBody)
		Error("status_500", activityErrorResponseBody, func() { Fault() })
		Error("status_429", activityErrorResponseBody, func() { Temporary() })
		HTTP(func() {
			Meta("openapi:tag:work-item-activities")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/activities/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			activityRawJSONResponse(ImportedRestFrameworkSerializersPaginatedissueactivityresponseResponse, "Paginated list of issue activities")
			Response("status_400", 400, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_401", 401, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_403", 403, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_404", 404, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})

			Response("status_500", 500, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_429", 429, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDWorkItemsByIssueIDActivitiesByPk", func() {
		Description("Retrieve details of a specific activity.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_work_items_by_issue_id_activities_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/activities/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.IssueActivityDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("expand", String, func() {
				Description("Comma-separated list of related fields to expand in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("fields", String, func() {
				Description("Comma-separated list of fields to include in response")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("issue_id", String, func() {
				Description("Issue ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", Int, func() {
				Description("Number of results per page (default: 1000, max: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("pk", String, func() {
				Description("Activity ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Pattern(djangoURLUUIDPattern)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("issue_id", "pk", "project_id", "slug")
		})
		Result(activityRawJSONResponseMetadata)
		Error("status_400", activityErrorResponseBody)
		Error("status_401", activityErrorResponseBody)
		Error("status_403", activityErrorResponseBody)
		Error("status_404", activityErrorResponseBody)
		Error("status_500", activityErrorResponseBody, func() { Fault() })
		Error("status_429", activityErrorResponseBody, func() { Temporary() })
		HTTP(func() {
			Meta("openapi:tag:work-item-activities")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/work-items/{issue_id}/activities/{pk}/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("issue_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			activityRawJSONResponse(ImportedPlaneAPISerializersIssueIssueactivityserializerResponse, "Paginated list of work item activities")
			Response("status_400", 400, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_401", 401, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_403", 403, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_404", 404, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})

			Response("status_500", 500, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
			Response("status_429", 429, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(activityHTTPErrorResponseBody)
			})
		})
	})
	Method("GetAPIUsersMeActivitiesOptions", func() {
		Meta("openapi:operationId", "options_api_users_me_activities")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
		})
		Result(activityOptions)
		Error("status_401", activityErrorResponseBody)
		HTTP(func() {
			OPTIONS("/api/users/me/activities/")
			Response(200)
			Response("status_401", 401, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(activityHTTPErrorResponseBody) })
		})
	})
})
