// Package design defines the external integrations API contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/externalintegrations/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })

var rawMetadata = Type("ExternalIntegrationRawJSON", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var errorResponse = Type("ExternalIntegrationErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

var requestObject = MapOf(String, Any)

func errorsFor(statuses ...int) {
	for _, status := range statuses {
		name := "status_" + strconv.Itoa(status)
		Error(name, errorResponse, func() {
			if status >= 500 {
				Fault()
			}
		})
	}
}

func httpErrors(statuses ...int) {
	for _, status := range statuses {
		name := "status_" + strconv.Itoa(status)
		Response(name, status, func() {
			ContentType("application/json")
			ErrorNameFromStatus()
			Body("body")
		})
	}
}

func unsplashStatuses() []int {
	statuses := make([]int, 0, 300)
	for status := 300; status <= 599; status++ {
		statuses = append(statuses, status)
	}
	return statuses
}

func unsplashErrors() {
	for _, status := range unsplashStatuses() {
		Error("status_"+strconv.Itoa(status), errorResponse, func() {
			if status == 429 || status == 502 || status == 503 {
				Temporary()
			}
			if status >= 500 {
				Fault()
			}
		})
	}
}

func sessionPayload(path ...string) {
	Security(sessionCookie)
	Payload(func() {
		for _, name := range path {
			Attribute(name, String)
		}
		APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		if len(path) > 0 {
			Required(path...)
		}
	})
}

var _ = API("Plane External Integrations API", func() {
	Title("Plane External Integrations API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("external_integrations", func() {
	Description("Provides workspace-scoped AI assistance and Unsplash proxy access.")
	HTTP(func() { RuntimeCORS() })
	Method("unsplash", func() {
		Meta("openapi:operationId", "get_api_unsplash")
		Security(sessionCookie)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("query", String)
			Attribute("page", String)
			Attribute("per_page", String)
		})
		Result(rawMetadata)
		unsplashErrors()
		HTTP(func() {
			GET("/api/unsplash/")
			Cookie("planeSessionCookieCredential:session-id")
			Param("query")
			Param("page")
			Param("per_page")
			SkipResponseBodyEncodeDecode()
			Response(200, func() {
				Header("content_type:Content-Type")
				ContentType("application/json")
				OpenAPIBody(Any)
			})
			httpErrors(unsplashStatuses()...)
		})
	})
	Method("workspace_ai_assistant", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_ai_assistant")
		sessionPayload("slug")
		Result(rawMetadata)
		errorsFor(400, 401, 403, 404, 415, 500)
		HTTP(func() {
			POST("/api/workspaces/{slug}/ai-assistant/")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			SkipResponseBodyEncodeDecode()
			OpenAPIRequestBodyTypes(requestObject, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				Header("content_type:Content-Type")
				ContentType("application/json")
				OpenAPIBody(Any)
			})
			httpErrors(400, 401, 403, 404, 415, 500)
		})
	})
	Method("project_ai_assistant", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_ai_assistant")
		sessionPayload("slug", "project_id")
		Result(rawMetadata)
		errorsFor(400, 401, 403, 404, 415, 500)
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/ai-assistant/")
			Param("slug")
			Param("project_id")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			SkipResponseBodyEncodeDecode()
			OpenAPIRequestBodyTypes(requestObject, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				Header("content_type:Content-Type")
				ContentType("application/json")
				OpenAPIBody(Any)
			})
			httpErrors(400, 401, 403, 404, 415, 500)
		})
	})
})
