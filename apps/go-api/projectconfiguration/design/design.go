// Package design defines the app project-configuration API contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/projectconfiguration/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var apiKeyAuth = APIKeySecurity("ApiKeyAuth", func() { Description("Plane API token authentication.") })
var sessionAuth = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })

var rawMetadata = Type("ProjectConfigurationRawJSON", func() {
	Attribute("content_type", String)
	Required("content_type")
})
var errorResponse = Type("ProjectConfigurationErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func errors() {
	for _, status := range []int{400, 401, 403, 404, 409, 415, 429, 500} {
		name := "status_" + strconv.Itoa(status)
		Error(name, errorResponse, func() {
			if status == 429 {
				Temporary()
			}
			if status == 500 {
				Fault()
			}
		})
	}
}

func httpErrors() {
	for _, status := range []int{400, 401, 403, 404, 409, 415, 429, 500} {
		name := "status_" + strconv.Itoa(status)
		Response(name, status, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
	}
}

func jsonBody(schema any, required bool) {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(schema, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, required)
}

var requestObject = MapOf(String, Any)

func jsonResponse(status int) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(Any)
	})
}

var _ = API("Project Configuration", func() {
	Title("Project Configuration")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

func payload(api bool, path ...string) {
	if api {
		Security(apiKeyAuth)
	} else {
		Security(sessionAuth)
	}
	Payload(func() {
		Attribute("slug", String)
		Attribute("project_id", String)
		for _, name := range path {
			Attribute(name, String)
		}
		if api {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
		} else {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		}
		Required("slug", "project_id")
	})
}

var _ = Service("project_configuration", func() {
	Description("Serves selected project estimate and label configuration operations.")
	Method("create_estimate", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_estimates")
		payload(false)
		Result(rawMetadata)
		errors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/estimates/")
			Param("slug")
			Param("project_id")
			Cookie("planeSessionCookieCredential:session-id")
			jsonBody(requestObject, true)
			jsonResponse(200)
			httpErrors()
		})
	})
	Method("patch_estimate", func() {
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_estimates_by_estimate_id")
		payload(false, "estimate_id")
		Result(rawMetadata)
		errors()
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/")
			Param("slug")
			Param("project_id")
			Param("estimate_id")
			Cookie("planeSessionCookieCredential:session-id")
			jsonBody(requestObject, false)
			jsonResponse(200)
			httpErrors()
		})
	})
	Method("delete_estimate", func() {
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_estimates_by_estimate_id")
		payload(false, "estimate_id")
		errors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/")
			Param("slug")
			Param("project_id")
			Param("estimate_id")
			Cookie("planeSessionCookieCredential:session-id")
			Response(204)
			httpErrors()
		})
	})
	Method("create_estimate_point", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_estimates_by_estimate_id_estimate_points")
		payload(false, "estimate_id")
		Result(rawMetadata)
		errors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/")
			Param("slug")
			Param("project_id")
			Param("estimate_id")
			Cookie("planeSessionCookieCredential:session-id")
			jsonBody(requestObject, true)
			jsonResponse(200)
			httpErrors()
		})
	})
	Method("patch_estimate_point", func() {
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_estimates_by_estimate_id_estimate_points_by_estimate_point_id")
		payload(false, "estimate_id", "estimate_point_id")
		Result(rawMetadata)
		errors()
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/{estimate_point_id}/")
			Param("slug")
			Param("project_id")
			Param("estimate_id")
			Param("estimate_point_id")
			Cookie("planeSessionCookieCredential:session-id")
			jsonBody(requestObject, false)
			jsonResponse(200)
			httpErrors()
		})
	})
	Method("delete_estimate_point", func() {
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_estimates_by_estimate_id_estimate_points_by_estimate_point_id")
		payload(false, "estimate_id", "estimate_point_id")
		Result(rawMetadata)
		errors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/projects/{project_id}/estimates/{estimate_id}/estimate-points/{estimate_point_id}/")
			Param("slug")
			Param("project_id")
			Param("estimate_id")
			Param("estimate_point_id")
			Cookie("planeSessionCookieCredential:session-id")
			jsonBody(requestObject, false)
			jsonResponse(200)
			httpErrors()
		})
	})
	Method("list_project_estimates", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_project_estimates")
		payload(false)
		Result(rawMetadata)
		errors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/project-estimates/")
			Param("slug")
			Param("project_id")
			Cookie("planeSessionCookieCredential:session-id")
			jsonResponse(200)
			httpErrors()
		})
	})
	Method("create_api_label", func() {
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_labels")
		payload(true)
		Result(rawMetadata)
		errors()
		HTTP(func() {
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/labels/")
			Param("slug")
			Param("project_id")
			Header("apiKeyAuthCredential:X-Api-Key")
			jsonBody(Any, false)
			jsonResponse(201)
			httpErrors()
		})
	})
	Method("patch_api_label", func() {
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_labels_by_pk")
		payload(true, "pk")
		Result(rawMetadata)
		errors()
		HTTP(func() {
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/labels/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			Header("apiKeyAuthCredential:X-Api-Key")
			jsonBody(requestObject, false)
			jsonResponse(200)
			httpErrors()
		})
	})
	Method("delete_api_label", func() {
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_labels_by_pk")
		payload(true, "pk")
		errors()
		HTTP(func() {
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/labels/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204)
			httpErrors()
		})
	})
	Method("bulk_create_labels", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_bulk_create_labels")
		payload(false)
		Result(rawMetadata)
		errors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/bulk-create-labels/")
			Param("slug")
			Param("project_id")
			Cookie("planeSessionCookieCredential:session-id")
			jsonBody(requestObject, true)
			jsonResponse(201)
			httpErrors()
		})
	})
})
