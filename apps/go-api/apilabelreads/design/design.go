package design

// Reverse-generated from the committed two-operation API label contract.
//go:generate go tool loom gen github.com/CaliLuke/go-plane/apilabelreads/design --output ..

import . "github.com/CaliLuke/loom/dsl"

var ImportedPlaneAPISerializersIssueLabelserializerResponse = Type("PlaneApiSerializersIssueLabelserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Full serializer for work item labels with complete metadata.\n\nProvides comprehensive label information including hierarchical relationships,\nvisual properties, and organizational data for work item tagging.")
	Attribute("color", String, func() {
		MaxLength(255)
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
	Attribute("description", String)
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
	Attribute("name", String, func() {
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("project", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64)
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

var ImportedRestFrameworkSerializersPaginatedlabelresponseResponse = Type("RestFrameworkSerializersPaginatedlabelresponseResponse", func() {
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
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersIssueLabelserializerResponse))
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

var ImportedDjangoCsrfCookieSecurity = APIKeySecurity("DjangoCsrfCookie", func() {
	Description("Django CSRF cookie for unsafe standard-session requests.")
})

var ImportedDjangoCsrfHeaderSecurity = APIKeySecurity("DjangoCsrfHeader", func() {
	Description("Django CSRF header for unsafe standard-session requests.")
})

var ImportedDjangoSessionCookieSecurity = APIKeySecurity("DjangoSessionCookie", func() {
	Description("Standard Django session authentication.")
})

var ImportedPlaneAdminSessionCookieSecurity = APIKeySecurity("PlaneAdminSessionCookie", func() {
	Description("Plane instance-administration session authentication without CSRF enforcement.")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var _ = API("API Label Reads", func() {
	Title("API Label Reads")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:APILabelReads")
})

var apiLabelReadsRawJSONResponseMetadata = Type("APILabelReadsRawJSONResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

func apiLabelReadsRawJSONResponse(schema any, description string) {
	SkipResponseBodyEncodeDecode()
	Response(200, func() {
		Description(description)
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

var apiLabelReadsErrorResponseBody = Type("APILabelReadsErrorResponseBody", func() {
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

var apiLabelReadsHTTPErrorResponseBody = Type("APILabelReadsHTTPErrorResponseBody", func() {
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
})

func apiLabelReadsErrors() {
	Error("status_400", apiLabelReadsErrorResponseBody)
	Error("status_401", apiLabelReadsErrorResponseBody)
	Error("status_403", apiLabelReadsErrorResponseBody)
	Error("status_404", apiLabelReadsErrorResponseBody)
	Error("status_405", apiLabelReadsErrorResponseBody)
	Error("status_406", apiLabelReadsErrorResponseBody)
	Error("status_429", apiLabelReadsErrorResponseBody, func() { Temporary() })
	Error("status_500", apiLabelReadsErrorResponseBody, func() { Fault() })
}

func apiLabelReadsHTTPErrorResponses() {
	Response("status_400", 400, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiLabelReadsHTTPErrorResponseBody)
	})
	Response("status_401", 401, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiLabelReadsHTTPErrorResponseBody)
	})
	Response("status_403", 403, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiLabelReadsHTTPErrorResponseBody)
	})
	Response("status_404", 404, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiLabelReadsHTTPErrorResponseBody)
	})
	Response("status_405", 405, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiLabelReadsHTTPErrorResponseBody)
	})
	Response("status_406", 406, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiLabelReadsHTTPErrorResponseBody)
	})
	Response("status_429", 429, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiLabelReadsHTTPErrorResponseBody)
	})
	Response("status_500", 500, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiLabelReadsHTTPErrorResponseBody)
	})
}

var _ = Service("APILabelReads", func() {
	HTTP(func() { RuntimeCORS() })
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDLabels", func() {
		Description("Retrieve all labels in a project. Supports filtering by name and color.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_labels")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/labels/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectMemberPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.LabelListCreateAPIEndpoint\"")
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
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(apiLabelReadsRawJSONResponseMetadata)
		apiLabelReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APILabelReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/labels/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("orderBy:order_by")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiLabelReadsRawJSONResponse(ImportedRestFrameworkSerializersPaginatedlabelresponseResponse, "Labels")
			apiLabelReadsHTTPErrorResponses()
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDLabelsByPk", func() {
		Description("Retrieve details of a specific label.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_labels_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/labels/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectMemberPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.issue.LabelDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Description("Label ID")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(apiLabelReadsRawJSONResponseMetadata)
		apiLabelReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APILabelReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/labels/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiLabelReadsRawJSONResponse(ImportedPlaneAPISerializersIssueLabelserializerResponse, "Labels")
			apiLabelReadsHTTPErrorResponses()
		})
	})
})
