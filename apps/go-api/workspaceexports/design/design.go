// Package design defines workspace export endpoints.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/workspaceexports/design --output ..

package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })
var rawResponse = Type("WorkspaceExportsRawJSON", func() { Attribute("content_type", String); Required("content_type") })
var errorResponse = Type("WorkspaceExportsErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func sessionPayload(query []string) {
	Security(sessionCookie)
	Payload(func() {
		Attribute("slug", String)
		for _, name := range query {
			Attribute(name, String)
		}
		APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		Required("slug")
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
	}
}
func httpErrors(statuses ...int) {
	for _, status := range statuses {
		name := "status_" + strconv.Itoa(status)
		Response(name, status, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
	}
}
func rawResponseHTTP() {
	SkipResponseBodyEncodeDecode()
	Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
}

var _ = API("Plane Workspace Exports API", func() {
	Title("Plane Workspace Exports API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("workspace_exports", func() {
	Description("Workspace issue and analytics exports.")
	HTTP(func() { RuntimeCORS() })
	Method("export_analytics", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_export_analytics")
		sessionPayload(nil)
		Result(rawResponse)
		errorsFor(400, 401, 403, 415, 500)
		HTTP(func() {
			POST("/api/workspaces/{slug}/export-analytics/")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(MapOf(String, Any), []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			rawResponseHTTP()
			httpErrors(400, 401, 403, 415, 500)
		})
	})
	Method("get_issue_exports", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_export_issues")
		sessionPayload([]string{"per_page", "cursor", "order_by"})
		Result(rawResponse)
		errorsFor(400, 401, 403, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/export-issues/")
			Param("slug")
			Param("per_page")
			Param("cursor")
			Param("order_by")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponseHTTP()
			httpErrors(400, 401, 403, 500)
		})
	})
	Method("create_issue_export", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_export_issues")
		sessionPayload(nil)
		Result(rawResponse)
		errorsFor(400, 401, 403, 415, 500)
		HTTP(func() {
			POST("/api/workspaces/{slug}/export-issues/")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(MapOf(String, Any), []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			rawResponseHTTP()
			httpErrors(400, 401, 403, 415, 500)
		})
	})
})
