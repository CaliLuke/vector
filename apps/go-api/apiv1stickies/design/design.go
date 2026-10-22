// Package design is imported from the Django v1 Sticky OpenAPI contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/apiv1stickies/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"
import "strconv"

var apiKeyAuth = APIKeySecurity("ApiKeyAuth", func() {
	Description("Plane API token authentication.")
})

var stickyErrorResponse = Type("APIV1StickyErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

var rawResponseMetadata = Type("APIV1StickyRawResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

func stickyErrors() {
	for _, status := range []int{400, 401, 403, 404, 415, 429, 500} {
		name := "status_" + strconv.Itoa(status)
		Error(name, stickyErrorResponse, func() {
			switch status {
			case 429:
				Temporary()
			case 500:
				Fault()
			}
		})
	}
}

func stickyHTTPErrorResponses() {
	for _, status := range []int{400, 401, 403, 404, 415, 429, 500} {
		name := "status_" + strconv.Itoa(status)
		Response(name, status, func() {
			ContentType("application/json")
			ErrorNameFromStatus()
			Body("body")
		})
	}
}

func stickyJSONResponse(status int, schema any, description string) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Description(description)
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

func stickyRequestBody(schema any, required bool) {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(schema, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, required)
}

var stickyRequest = Type("StickyWriteRequest", func() {
	Attribute("background_color", String, func() { Nullable(); MaxLength(255) })
	Attribute("color", String, func() { Nullable(); MaxLength(255) })
	Attribute("created_by", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("deleted_at", String, func() { Nullable(); Format(FormatDateTime) })
	Attribute("description", Any)
	Attribute("description_html", String)
	Attribute("description_stripped", String, func() { Nullable() })
	Attribute("logo_props", Any)
	Attribute("name", String, func() { Nullable() })
	Attribute("sort_order", Float64)
	Attribute("updated_by", String, func() { Nullable(); Format(FormatUUID) })
})

var stickyResponse = Type("StickyResponse", func() {
	Attribute("background_color", String, func() { Nullable() })
	Attribute("color", String, func() { Nullable() })
	Attribute("created_at", String, func() { Format(FormatDateTime); ReadOnly() })
	Attribute("created_by", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("deleted_at", String, func() { Nullable(); Format(FormatDateTime) })
	Attribute("description", Any)
	Attribute("description_binary", String, func() { Nullable(); Format(FormatBase64) })
	Attribute("description_html", String)
	Attribute("description_stripped", String, func() { Nullable() })
	Attribute("id", String, func() { Format(FormatUUID); ReadOnly() })
	Attribute("logo_props", Any)
	Attribute("name", String, func() { Nullable() })
	Attribute("owner", String, func() { Format(FormatUUID); ReadOnly() })
	Attribute("sort_order", Float64)
	Attribute("updated_at", String, func() { Format(FormatDateTime); ReadOnly() })
	Attribute("updated_by", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("workspace", String, func() { Format(FormatUUID); ReadOnly() })
})

var stickyListResponse = Type("StickyListResponse", func() {
	Attribute("count", Int)
	Attribute("extra_stats", String, func() { Nullable() })
	Attribute("grouped_by", String, func() { Nullable() })
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(stickyResponse))
	Attribute("sub_grouped_by", String, func() { Nullable() })
	Attribute("total_count", Int)
	Attribute("total_pages", Int)
	Attribute("total_results", Int)
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
})

var _ = API("API v1 Stickies", func() {
	Title("API v1 Stickies")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

func stickyPayload(detail, query bool) {
	Security(apiKeyAuth)
	Payload(func() {
		Attribute("slug", String)
		if detail {
			// Keep lookup text opaque until API-key and workspace authorization
			// have run. DRF performs UUID coercion inside the view lookup.
			Attribute("pk", String)
		}
		if query {
			Attribute("cursor", String)
			Attribute("per_page", String)
			Attribute("query", String)
		}
		APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
		required := []string{"slug"}
		if detail {
			required = append(required, "pk")
		}
		Required(required...)
	})
}

var _ = Service("api_v1_stickies", func() {
	Description("Serves the Django API v1 Sticky contract.")

	Method("list", func() {
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_stickies(.{routeIndex})")
		stickyPayload(false, true)
		Result(rawResponseMetadata)
		stickyErrors()
		HTTP(func() {
			GET("/api/v1/workspaces/{slug}/stickies/")
			GET("/api/v1/workspaces/{slug}/stickies.json")
			GET("/api/v1/workspaces/{slug}/stickies.json/")
			Param("slug")
			Param("cursor")
			Param("per_page")
			Param("query")
			Header("apiKeyAuthCredential:X-Api-Key")
			stickyJSONResponse(200, stickyListResponse, "List of stickies")
			stickyHTTPErrorResponses()
		})
	})

	Method("create", func() {
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_stickies(.{routeIndex})")
		stickyPayload(false, false)
		Result(rawResponseMetadata)
		stickyErrors()
		HTTP(func() {
			POST("/api/v1/workspaces/{slug}/stickies/")
			POST("/api/v1/workspaces/{slug}/stickies.json")
			POST("/api/v1/workspaces/{slug}/stickies.json/")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			stickyRequestBody(stickyRequest, false)
			stickyJSONResponse(201, stickyResponse, "Sticky created")
			stickyHTTPErrorResponses()
		})
	})

	Method("list_options", func() {
		Meta("openapi:operationId", "options_api_v1_workspaces_by_slug_stickies(.{routeIndex})")
		stickyPayload(false, false)
		Result(rawResponseMetadata)
		stickyErrors()
		HTTP(func() {
			OPTIONS("/api/v1/workspaces/{slug}/stickies/")
			OPTIONS("/api/v1/workspaces/{slug}/stickies.json")
			OPTIONS("/api/v1/workspaces/{slug}/stickies.json/")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			stickyJSONResponse(200, Any, "Sticky list metadata")
			stickyHTTPErrorResponses()
		})
	})

	Method("delete", func() {
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_stickies_by_pk(.{routeIndex})")
		stickyPayload(true, false)
		stickyErrors()
		HTTP(func() {
			DELETE("/api/v1/workspaces/{slug}/stickies/{pk}/")
			DELETE("/api/v1/workspaces/{slug}/stickies/{pk}.json")
			DELETE("/api/v1/workspaces/{slug}/stickies/{pk}.json/")
			Param("slug")
			Param("pk")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() { Description("Resource deleted successfully") })
			stickyHTTPErrorResponses()
		})
	})

	Method("retrieve", func() {
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_stickies_by_pk(.{routeIndex})")
		stickyPayload(true, false)
		Result(rawResponseMetadata)
		stickyErrors()
		HTTP(func() {
			GET("/api/v1/workspaces/{slug}/stickies/{pk}/")
			GET("/api/v1/workspaces/{slug}/stickies/{pk}.json")
			GET("/api/v1/workspaces/{slug}/stickies/{pk}.json/")
			Param("slug")
			Param("pk")
			Header("apiKeyAuthCredential:X-Api-Key")
			stickyJSONResponse(200, stickyResponse, "Sticky")
			stickyHTTPErrorResponses()
		})
	})

	Method("detail_options", func() {
		Meta("openapi:operationId", "options_api_v1_workspaces_by_slug_stickies_by_pk(.{routeIndex})")
		stickyPayload(true, false)
		Result(rawResponseMetadata)
		stickyErrors()
		HTTP(func() {
			OPTIONS("/api/v1/workspaces/{slug}/stickies/{pk}/")
			OPTIONS("/api/v1/workspaces/{slug}/stickies/{pk}.json")
			OPTIONS("/api/v1/workspaces/{slug}/stickies/{pk}.json/")
			Param("slug")
			Param("pk")
			Header("apiKeyAuthCredential:X-Api-Key")
			stickyJSONResponse(200, Any, "Sticky detail metadata")
			stickyHTTPErrorResponses()
		})
	})

	Method("patch", func() {
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_stickies_by_pk(.{routeIndex})")
		stickyPayload(true, false)
		Result(rawResponseMetadata)
		stickyErrors()
		HTTP(func() {
			PATCH("/api/v1/workspaces/{slug}/stickies/{pk}/")
			PATCH("/api/v1/workspaces/{slug}/stickies/{pk}.json")
			PATCH("/api/v1/workspaces/{slug}/stickies/{pk}.json/")
			Param("slug")
			Param("pk")
			Header("apiKeyAuthCredential:X-Api-Key")
			stickyRequestBody(stickyRequest, false)
			stickyJSONResponse(200, stickyResponse, "Sticky")
			stickyHTTPErrorResponses()
		})
	})

	Method("update", func() {
		Meta("openapi:operationId", "put_api_v1_workspaces_by_slug_stickies_by_pk(.{routeIndex})")
		stickyPayload(true, false)
		Result(rawResponseMetadata)
		stickyErrors()
		HTTP(func() {
			PUT("/api/v1/workspaces/{slug}/stickies/{pk}/")
			PUT("/api/v1/workspaces/{slug}/stickies/{pk}.json")
			PUT("/api/v1/workspaces/{slug}/stickies/{pk}.json/")
			Param("slug")
			Param("pk")
			Header("apiKeyAuthCredential:X-Api-Key")
			stickyRequestBody(stickyRequest, false)
			stickyJSONResponse(200, stickyResponse, "Sticky")
			stickyHTTPErrorResponses()
		})
	})
})
