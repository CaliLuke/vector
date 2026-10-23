// Package design declares workspace search endpoints.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/workspacesearch/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })
var rawMetadata = Type("WorkspaceSearchRawJSON", func() { Attribute("content_type", String); Required("content_type") })
var errorResponse = Type("WorkspaceSearchErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func authPayload(path, query []string) {
	Security(sessionCookie)
	Payload(func() {
		for _, name := range path {
			Attribute(name, String)
		}
		for _, name := range query {
			Attribute(name, String)
		}
		APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		Required(path...)
	})
}
func errorContracts() {
	Error("status_400", errorResponse)
	Error("status_401", errorResponse)
	Error("status_403", errorResponse)
	Error("status_500", errorResponse, func() { Fault() })
}
func errorResponses() {
	for _, status := range []int{400, 401, 403, 500} {
		name := "status_" + strconv.Itoa(status)
		Response(name, status, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
	}
}

var _ = API("Plane Workspace Search API", func() {
	Title("Plane Workspace Search API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	HTTP(func() { RuntimeCORS(); Consumes("application/json"); Produces("application/json") })
})
var _ = Service("workspace_search", func() {
	Description("Workspace search read endpoints.")
	HTTP(func() { RuntimeCORS() })
	Method("global_search", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_search")
		authPayload([]string{"slug"}, []string{"search", "entities", "workspace_search", "project_id"})
		Result(rawMetadata)
		errorContracts()
		HTTP(func() {
			GET("/api/workspaces/{slug}/search/")
			Param("slug")
			Param("search")
			Param("entities")
			Param("workspace_search")
			Param("project_id")
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			errorResponses()
		})
	})
	Method("entity_search", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_entity_search")
		authPayload([]string{"slug"}, []string{"query", "query_type", "count", "project_id"})
		Result(rawMetadata)
		errorContracts()
		HTTP(func() {
			GET("/api/workspaces/{slug}/entity-search/")
			Param("slug")
			Param("query")
			Param("query_type")
			Param("count")
			Param("project_id")
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			errorResponses()
		})
	})
})
