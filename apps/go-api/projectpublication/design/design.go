// Package design defines project deployment-board publication endpoints.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/projectpublication/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })
var requestObject = MapOf(String, Any)
var rawResponse = Type("ProjectPublicationRawJSON", func() {
	Attribute("content_type", String)
	Required("content_type")
})
var errorResponse = Type("ProjectPublicationErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func errorsFor(statuses ...int) {
	for _, status := range statuses {
		name := "status_" + strconv.Itoa(status)
		Error(name, errorResponse, func() {
			if status >= 500 {
				Fault()
			}
		})
		HTTP(func() {
			Response(name, status, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
		})
	}
}
func payload(detail bool) {
	Security(sessionCookie)
	Payload(func() {
		Attribute("slug", String)
		Attribute("project_id", String)
		if detail {
			Attribute("pk", String)
		}
		APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		Required("slug", "project_id")
		if detail {
			Required("pk")
		}
	})
}
func rawRequest() {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(requestObject, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
}
func rawJSONResponse() {
	SkipResponseBodyEncodeDecode()
	Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
}

var _ = API("Plane Project Publication API", func() {
	Title("Plane Project Publication API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})
var _ = Service("project_publication", func() {
	HTTP(func() { RuntimeCORS() })
	Method("create", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_project_deploy_boards")
		payload(false)
		Result(rawResponse)
		errorsFor(400, 401, 403, 404, 415, 500)
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/")
			Param("slug")
			Param("project_id")
			Cookie("planeSessionCookieCredential:session-id")
			rawRequest()
			rawJSONResponse()
		})
	})
	Method("update", func() {
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_project_deploy_boards_by_pk")
		payload(true)
		Result(rawResponse)
		errorsFor(400, 401, 403, 404, 415, 500)
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			Cookie("planeSessionCookieCredential:session-id")
			rawRequest()
			rawJSONResponse()
		})
	})
	Method("delete", func() {
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_project_deploy_boards_by_pk")
		payload(true)
		Result(Empty)
		errorsFor(401, 403, 404, 500)
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			Cookie("planeSessionCookieCredential:session-id")
			Response(204)
		})
	})
})
