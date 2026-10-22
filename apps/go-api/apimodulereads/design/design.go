package design

// Package design is reverse-generated from the scoped committed API v1 OpenAPI contract.
//go:generate go tool loom gen github.com/CaliLuke/go-plane/apimodulereads/design --output ..

import . "github.com/CaliLuke/loom/dsl"

var ImportedModuleStatusEnum = Type("ModuleStatusEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `backlog` - Backlog\n* `planned` - Planned\n* `in-progress` - In Progress\n* `paused` - Paused\n* `completed` - Completed\n* `cancelled` - Cancelled")
	Enum("backlog", "planned", "in-progress", "paused", "completed", "cancelled")
})

var ImportedPlaneAPISerializersIssueIssueserializerResponse = Type("PlaneApiSerializersIssueIssueserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Comprehensive work item serializer with full relationship management.\n\nHandles complete work item lifecycle including assignees, labels, validation,\nand related model updates. Supports dynamic field expansion and HTML content\nprocessing.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("completed_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
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
	Attribute("description_binary", Bytes, func() {
		Nullable()
		Meta("openapi:format", "byte")
		Meta("openapi:readOnly", "true")
	})
	Attribute("description_html", String)
	Attribute("estimate_point", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
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
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("point", Int, func() {
		Nullable()
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(12)
	})
	Attribute("priority", ImportedPriorityEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Title("Issue Priority")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
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
	Attribute("state", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("type", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("type_id", String, func() {
		Nullable()
		Format(FormatUUID)
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
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("name")
})

var ImportedPlaneAPISerializersModuleModuleliteserializerResponse = Type("PlaneApiSerializersModuleModuleliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Lightweight module serializer for minimal data transfer.\n\nProvides essential module information without computed metrics,\noptimized for list views and reference lookups.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
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
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("description_html", Any, func() {
	})
	Attribute("description_text", Any, func() {
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("members", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MaxLength(255)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
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
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
	})
	Required("name", "project", "workspace")
})

var ImportedPlaneAPISerializersModuleModuleserializerResponse = Type("PlaneApiSerializersModuleModuleserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Comprehensive module serializer with work item metrics and member management.\n\nProvides complete module data including work item counts by status, member\nrelationships, and progress tracking for feature-based project organization.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("backlog_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("cancelled_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("completed_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("description_html", Any, func() {
	})
	Attribute("description_text", Any, func() {
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MaxLength(255)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("started_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("total_issues", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Attribute("unstarted_issues", Int, func() {
		Meta("openapi:format", "")
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
		Meta("openapi:readOnly", "true")
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("name")
})

var ImportedPriorityEnum = Type("PriorityEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `urgent` - Urgent\n* `high` - High\n* `medium` - Medium\n* `low` - Low\n* `none` - None")
	Enum("urgent", "high", "medium", "low", "none")
})

var ImportedRestFrameworkSerializersPaginatedarchivedmoduleresponseResponse = Type("RestFrameworkSerializersPaginatedarchivedmoduleresponseResponse", func() {
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
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersModuleModuleserializerResponse))
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

var ImportedRestFrameworkSerializersPaginatedmoduleissueresponseResponse = Type("RestFrameworkSerializersPaginatedmoduleissueresponseResponse", func() {
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
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIssueIssueserializerResponse))
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

var ImportedRestFrameworkSerializersPaginatedmoduleliteresponseResponse = Type("RestFrameworkSerializersPaginatedmoduleliteresponseResponse", func() {
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
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersModuleModuleliteserializerResponse))
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

var ImportedRestFrameworkSerializersPaginatedmoduleresponseResponse = Type("RestFrameworkSerializersPaginatedmoduleresponseResponse", func() {
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
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersModuleModuleserializerResponse))
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

var _ = API("API Module Reads", func() {
	Title("API Module Reads")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:APIModuleReads")
})

var apiModuleReadsRawJSONResponseMetadata = Type("APIModuleReadsRawJSONResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

func apiModuleReadsRawJSONResponse(schema any, description string) {
	SkipResponseBodyEncodeDecode()
	Response(200, func() {
		Description(description)
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

var apiModuleReadsErrorResponseBody = Type("APIModuleReadsErrorResponseBody", func() {
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

var apiModuleReadsHTTPErrorResponseBody = Type("APIModuleReadsHTTPErrorResponseBody", func() {
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
})

func apiModuleReadsErrors() {
	Error("status_400", apiModuleReadsErrorResponseBody)
	Error("status_401", apiModuleReadsErrorResponseBody)
	Error("status_403", apiModuleReadsErrorResponseBody)
	Error("status_404", apiModuleReadsErrorResponseBody)
	Error("status_405", apiModuleReadsErrorResponseBody)
	Error("status_406", apiModuleReadsErrorResponseBody)
	Error("status_429", apiModuleReadsErrorResponseBody, func() { Temporary() })
	Error("status_500", apiModuleReadsErrorResponseBody, func() { Fault() })
}

func apiModuleReadsHTTPErrorResponses() {
	Response("status_400", 400, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiModuleReadsHTTPErrorResponseBody)
	})
	Response("status_401", 401, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiModuleReadsHTTPErrorResponseBody)
	})
	Response("status_403", 403, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiModuleReadsHTTPErrorResponseBody)
	})
	Response("status_404", 404, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiModuleReadsHTTPErrorResponseBody)
	})
	Response("status_405", 405, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiModuleReadsHTTPErrorResponseBody)
	})
	Response("status_406", 406, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiModuleReadsHTTPErrorResponseBody)
	})
	Response("status_429", 429, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiModuleReadsHTTPErrorResponseBody)
	})
	Response("status_500", 500, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiModuleReadsHTTPErrorResponseBody)
	})
}

var _ = Service("APIModuleReads", func() {
	HTTP(func() { RuntimeCORS() })
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModules", func() {
		Description("Retrieve all modules that have been archived in the project.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_archived_modules")
		Meta("openapi:summary", "List archived modules")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/archived-modules/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleArchiveUnarchiveAPIEndpoint\"")
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
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Pagination input interpreted by Django-compatible pagination (default and maximum: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(apiModuleReadsRawJSONResponseMetadata)
		apiModuleReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APIModuleReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/archived-modules/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiModuleReadsRawJSONResponse(ImportedRestFrameworkSerializersPaginatedarchivedmoduleresponseResponse, "Paginated list of archived modules")
			apiModuleReadsHTTPErrorResponses()
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesLite", func() {
		Description("Retrieve a paginated, lightweight list of modules in a project for pickers and references.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_modules_lite")
		Meta("openapi:summary", "List modules (lite)")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/modules-lite/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleListLiteAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Pagination input interpreted by Django-compatible pagination (default and maximum: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(apiModuleReadsRawJSONResponseMetadata)
		apiModuleReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APIModuleReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/modules-lite/")
			Param("cursor")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiModuleReadsRawJSONResponse(ImportedRestFrameworkSerializersPaginatedmoduleliteresponseResponse, "Paginated list of modules with minimal fields")
			apiModuleReadsHTTPErrorResponses()
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDModules", func() {
		Description("Retrieve all modules in a project.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_modules")
		Meta("openapi:summary", "List modules")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/modules/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleListCreateAPIEndpoint\"")
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
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Pagination input interpreted by Django-compatible pagination (default and maximum: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(apiModuleReadsRawJSONResponseMetadata)
		apiModuleReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APIModuleReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/modules/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiModuleReadsRawJSONResponse(ImportedRestFrameworkSerializersPaginatedmoduleresponseResponse, "Paginated list of modules")
			apiModuleReadsHTTPErrorResponses()
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleIssues", func() {
		Description("Retrieve all work items assigned to a module with detailed information.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_issues")
		Meta("openapi:summary", "List module work items")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleIssueListCreateAPIEndpoint\"")
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
			Attribute("module_id", String, func() {
				Description("Module ID")
				Format(FormatUUID)
			})
			Attribute("orderBy", String, func() {
				Description("Field to order results by. Prefix with '-' for descending order")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Pagination input interpreted by Django-compatible pagination (default and maximum: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("module_id", "project_id", "slug")
		})
		Result(apiModuleReadsRawJSONResponseMetadata)
		apiModuleReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APIModuleReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-issues/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("module_id")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiModuleReadsRawJSONResponse(ImportedRestFrameworkSerializersPaginatedmoduleissueresponseResponse, "Paginated list of module work items")
			apiModuleReadsHTTPErrorResponses()
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk", func() {
		Description("Retrieve details of a specific module.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_pk")
		Meta("openapi:summary", "Retrieve module")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("fields", String)
			Attribute("expand", String)
			Attribute("pk", String, func() {
				Description("Module ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(apiModuleReadsRawJSONResponseMetadata)
		apiModuleReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APIModuleReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/")
			Param("fields")
			Param("expand")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiModuleReadsRawJSONResponse(ImportedPlaneAPISerializersModuleModuleserializerResponse, "Module")
			apiModuleReadsHTTPErrorResponses()
		})
	})
})
