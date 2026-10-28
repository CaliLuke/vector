package design

//go:generate go tool loom gen github.com/CaliLuke/go-plane/apistatereads/design --output ..

import (
	"fmt"

	. "github.com/CaliLuke/loom/dsl"
)

var ImportedGroupEnum = Type("GroupEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `backlog` - Backlog\n* `unstarted` - Unstarted\n* `started` - Started\n* `completed` - Completed\n* `cancelled` - Cancelled\n* `triage` - Triage")
	Enum("backlog", "unstarted", "started", "completed", "cancelled", "triage")
})

var ImportedPlaneAPISerializersStateStateserializerResponse = Type("PlaneApiSerializersStateStateserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for work item states with default state management.\n\nHandles state creation and updates including default state validation\nand automatic default state switching for workflow management.")
	Attribute("color", String, func() {
		Title("State Color")
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
	Attribute("default", Boolean)
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("State Description")
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("group", ImportedGroupEnum)
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_triage", Boolean)
	Attribute("name", String, func() {
		Title("State Name")
		MaxLength(255)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sequence", Float64)
	Attribute("slug", String, func() {
		Meta("openapi:readOnly", "true")
		Pattern("^[-a-zA-Z0-9_]+$")
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
	Required("color", "name")
})

var ImportedRestFrameworkSerializersPaginatedstateresponseResponse = Type("RestFrameworkSerializersPaginatedstateresponseResponse", func() {
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
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersStateStateserializerResponse))
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

var _ = API("API State Reads", func() {
	Title("API State Reads")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:APIStateReads")
})

var apiStateReadsRawJSONResponseMetadata = Type("APIStateReadsRawJSONResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

func apiStateReadsRawJSONResponse(schema any, description string) {
	SkipResponseBodyEncodeDecode()
	Response(200, func() {
		Description(description)
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

var apiStateReadsErrorResponseBody = Type("APIStateReadsErrorResponseBody", func() {
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

var apiStateReadsHTTPErrorResponseBody = Type("APIStateReadsHTTPErrorResponseBody", func() {
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
})

var apiStateWritesErrorResponseBody = Type("APIStateWritesErrorResponseBody", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("kind", "body")
})

func apiStateReadsErrors() {
	Error("status_400", apiStateReadsErrorResponseBody)
	Error("status_401", apiStateReadsErrorResponseBody)
	Error("status_403", apiStateReadsErrorResponseBody)
	Error("status_404", apiStateReadsErrorResponseBody)
	Error("status_405", apiStateReadsErrorResponseBody)
	Error("status_406", apiStateReadsErrorResponseBody)
	Error("status_429", apiStateReadsErrorResponseBody, func() { Temporary() })
	Error("status_500", apiStateReadsErrorResponseBody, func() { Fault() })
}

func apiStateReadsHTTPErrorResponses() {
	Response("status_400", 400, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiStateReadsHTTPErrorResponseBody)
	})
	Response("status_401", 401, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiStateReadsHTTPErrorResponseBody)
	})
	Response("status_403", 403, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiStateReadsHTTPErrorResponseBody)
	})
	Response("status_404", 404, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiStateReadsHTTPErrorResponseBody)
	})
	Response("status_405", 405, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiStateReadsHTTPErrorResponseBody)
	})
	Response("status_406", 406, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiStateReadsHTTPErrorResponseBody)
	})
	Response("status_429", 429, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiStateReadsHTTPErrorResponseBody)
	})
	Response("status_500", 500, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(apiStateReadsHTTPErrorResponseBody)
	})
}

func apiStateWritesPayload(stateID bool) {
	Security(ImportedAPIKeyAuthSecurity)
	Payload(func() {
		Attribute("project_id", String, func() { Format(FormatUUID) })
		Attribute("slug", String)
		Attribute("fields", String, func() { Meta("openapi:allowEmptyValue", "false") })
		Attribute("expand", String, func() { Meta("openapi:allowEmptyValue", "false") })
		Attribute("content_type", String)
		if stateID {
			Attribute("state_id", String, func() { Format(FormatUUID) })
		}
		APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
		required := []string{"project_id", "slug"}
		if stateID {
			required = append(required, "state_id")
		}
		Required(required...)
	})
}

var apiStateWritesRawRequest = MapOf(String, Any)

func apiStateWritesRequestBody() {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(apiStateWritesRawRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
}

func apiStateWritesJSONResponse(status int, schema any, description string) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
		Description(description)
	})
}

func apiStateWritesErrors() {
	for _, status := range []int{400, 401, 403, 404, 405, 406, 409, 415, 429, 500} {
		name := fmt.Sprintf("status_%d", status)
		Error(name, apiStateWritesErrorResponseBody, func() {
			if status == 429 {
				Temporary()
			}
			if status == 500 {
				Fault()
			}
		})
	}
}

func apiStateWritesHTTPErrorResponses() {
	for _, status := range []int{400, 401, 403, 404, 405, 406, 409, 415, 429, 500} {
		name := fmt.Sprintf("status_%d", status)
		Response(name, status, func() {
			ContentType("application/json")
			ErrorNameFromStatus()
			Body("body")
		})
	}
}

func apiStateWritesMethod(name, operationID, route, method string, stateID bool, response bool) {
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\""+method+" "+route+"\"")
		Meta("openapi:extension:x-plane-handler", "\""+map[string]string{"POST": "post", "PATCH": "patch", "DELETE": "delete"}[method]+"\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		view := "plane.api.views.state.StateDetailAPIEndpoint"
		if !stateID {
			view = "plane.api.views.state.StateListCreateAPIEndpoint"
		}
		Meta("openapi:extension:x-plane-view", "\""+view+"\"")
		apiStateWritesPayload(stateID)
		if response {
			Result(apiStateReadsRawJSONResponseMetadata)
		} else {
			Result(Empty)
		}
		apiStateWritesErrors()
		HTTP(func() {
			Meta("openapi:tag:APIStateReads")
			switch method {
			case "POST":
				POST(route)
			case "PATCH":
				PATCH(route)
			case "DELETE":
				DELETE(route)
			}
			Param("project_id")
			Param("slug")
			Param("fields")
			Param("expand")
			if stateID {
				Param("state_id")
			}
			Header("apiKeyAuthCredential:X-Api-Key")
			Header("content_type:Content-Type")
			if method == "POST" || method == "PATCH" {
				apiStateWritesRequestBody()
			}
			if response {
				apiStateWritesJSONResponse(200, ImportedPlaneAPISerializersStateStateserializerResponse, "Workflow state after mutation")
			} else {
				Response(StatusNoContent)
			}
			apiStateWritesHTTPErrorResponses()
		})
	})
}

var _ = Service("APIStateReads", func() {
	HTTP(func() { RuntimeCORS() })
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDStates", func() {
		Description("Retrieve workflow states for a project using the API v1 serializer and pagination contract.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_states")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/states/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.state.StateListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() { Meta("openapi:allowEmptyValue", "false") })
			Attribute("expand", String, func() { Meta("openapi:allowEmptyValue", "false") })
			Attribute("fields", String, func() { Meta("openapi:allowEmptyValue", "false") })
			Attribute("perPage", String, func() {
				Description("Pagination input interpreted by Django-compatible pagination (default and maximum: 1000)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("slug", String)
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(apiStateReadsRawJSONResponseMetadata)
		apiStateReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APIStateReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/states/")
			Param("cursor")
			Param("expand")
			Param("fields")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiStateReadsRawJSONResponse(ImportedRestFrameworkSerializersPaginatedstateresponseResponse, "Paginated workflow states")
			apiStateReadsHTTPErrorResponses()
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID", func() {
		Description("Retrieve one workflow state using the API v1 serializer.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_states_by_state_id")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/states/{state_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.state.StateDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("slug", String)
			Attribute("state_id", String, func() { Format(FormatUUID) })
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug", "state_id")
		})
		Result(apiStateReadsRawJSONResponseMetadata)
		apiStateReadsErrors()
		HTTP(func() {
			Meta("openapi:tag:APIStateReads")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/states/{state_id}/")
			Param("project_id")
			Param("slug")
			Param("state_id")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiStateReadsRawJSONResponse(ImportedPlaneAPISerializersStateStateserializerResponse, "Workflow state")
			apiStateReadsHTTPErrorResponses()
		})
	})
	apiStateWritesMethod("CreateAPIV1WorkspacesBySlugProjectsByProjectIDStates", "post_api_v1_workspaces_by_slug_projects_by_project_id_states", "/api/v1/workspaces/{slug}/projects/{project_id}/states/", "POST", false, true)
	apiStateWritesMethod("UpdateAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID", "patch_api_v1_workspaces_by_slug_projects_by_project_id_states_by_state_id", "/api/v1/workspaces/{slug}/projects/{project_id}/states/{state_id}/", "PATCH", true, true)
	apiStateWritesMethod("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDStatesByStateID", "delete_api_v1_workspaces_by_slug_projects_by_project_id_states_by_state_id", "/api/v1/workspaces/{slug}/projects/{project_id}/states/{state_id}/", "DELETE", true, false)
})
