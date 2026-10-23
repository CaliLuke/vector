// Package design defines member-specific custom view creation endpoints.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/customviewcreation/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })
var errorResponse = Type("CustomViewCreationErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func payload(pathAttrs ...string) {
	Security(sessionCookie)
	Payload(func() {
		Attribute("slug", String)
		for _, attr := range pathAttrs {
			Attribute(attr, String)
		}
		APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		Required(append([]string{"slug"}, pathAttrs...)...)
	})
}

func errorsFor(statuses ...int) {
	for _, status := range statuses {
		name := "status_" + strconv.Itoa(status)
		Error(name, errorResponse, func() {
			if status >= 500 {
				Fault()
			}
		})
		HTTP(func() {
			Response(name, status, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body("body")
			})
		})
	}
}

func rawRequest() {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(MapOf(String, Any), []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
}

var _ = API("Plane Custom View Creation API", func() {
	Title("Plane Custom View Creation API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("custom_view_creation", func() {
	HTTP(func() { RuntimeCORS() })
	Method("create_project_view", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_project_views")
		payload("project_id")
		Result(Empty)
		errorsFor(400, 401, 403, 404, 415, 500)
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/project-views/")
			Param("slug")
			Param("project_id")
			Cookie("planeSessionCookieCredential:session-id")
			rawRequest()
			Response(204)
		})
	})
	Method("create_workspace_view", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_workspace_views")
		payload()
		Result(Empty)
		errorsFor(400, 401, 404, 415, 500)
		HTTP(func() {
			POST("/api/workspaces/{slug}/workspace-views/")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			rawRequest()
			Response(204)
		})
	})
})
