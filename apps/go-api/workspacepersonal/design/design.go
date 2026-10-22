// Package design defines personal workspace dashboard endpoints.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/workspacepersonal/design --output ..

package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })

var rawResponse = Type("PersonalDashboardRawJSON", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var errorResponse = Type("PersonalDashboardErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func sessionPayload(path, query []string) {
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

var _ = API("Plane Personal Dashboard API", func() {
	Title("Plane Personal Dashboard API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() { RuntimeCORS(); Consumes("application/json"); Produces("application/json") })
})

var _ = Service("personal_dashboard", func() {
	Description("Authenticated user's workspace dashboard and activity graphs.")
	HTTP(func() { RuntimeCORS() })
	Method("activity_graph", func() {
		Meta("openapi:operationId", "get_api_users_me_workspaces_by_slug_activity_graph")
		sessionPayload([]string{"slug"}, nil)
		Result(rawResponse)
		errorsFor(400, 401, 500)
		HTTP(func() {
			GET("/api/users/me/workspaces/{slug}/activity-graph/")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 500)
		})
	})
	Method("dashboard", func() {
		Meta("openapi:operationId", "get_api_users_me_workspaces_by_slug_dashboard")
		sessionPayload([]string{"slug"}, []string{"month"})
		Result(rawResponse)
		errorsFor(400, 401, 500)
		HTTP(func() {
			GET("/api/users/me/workspaces/{slug}/dashboard/")
			Param("slug")
			Param("month")
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 500)
		})
	})
	Method("issues_completed_graph", func() {
		Meta("openapi:operationId", "get_api_users_me_workspaces_by_slug_issues_completed_graph")
		sessionPayload([]string{"slug"}, []string{"month"})
		Result(rawResponse)
		errorsFor(400, 401, 500)
		HTTP(func() {
			GET("/api/users/me/workspaces/{slug}/issues-completed-graph/")
			Param("slug")
			Param("month")
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 500)
		})
	})
})
