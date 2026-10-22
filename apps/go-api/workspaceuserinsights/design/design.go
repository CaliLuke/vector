// Package design defines workspace user profile and activity insights.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/workspaceuserinsights/design --output ..

package design

import (
	. "github.com/CaliLuke/loom/dsl"
	"strconv"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })
var rawResponse = Type("WorkspaceUserInsightsRawJSON", func() {
	Attribute("content_type", String)
	Required("content_type")
})
var csvResponse = Type("WorkspaceUserInsightsCSV", func() {
	Attribute("content_disposition", String)
	Required("content_disposition")
})
var errorResponse = Type("WorkspaceUserInsightsErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func sessionPayload(query []string) {
	Security(sessionCookie)
	Payload(func() {
		Attribute("slug", String)
		Attribute("user_id", String)
		for _, name := range query {
			Attribute(name, String)
		}
		APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		Required("slug", "user_id")
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
func rawHTTP(contentType string) {
	SkipResponseBodyEncodeDecode()
	Response(200, func() { Header("content_type:Content-Type"); ContentType(contentType); OpenAPIBody(Any) })
}
func rawCSVHTTP() {
	SkipResponseBodyEncodeDecode()
	Response(200, func() {
		ContentType("text/csv")
		OpenAPIBody(String)
		Header("content_disposition:Content-Disposition")
	})
}

var _ = API("Plane Workspace User Insights API", func() {
	Title("Plane Workspace User Insights API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json", "text/csv")
	})
})
var _ = Service("workspace_user_insights", func() {
	Description("Workspace-scoped user profile, activity, statistics, and activity CSV export.")
	HTTP(func() { RuntimeCORS() })
	Method("activity", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_user_activity_by_user_id")
		sessionPayload([]string{"project", "order_by", "per_page", "cursor"})
		Result(rawResponse)
		errorsFor(400, 401, 403, 404, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/user-activity/{user_id}/")
			Param("slug")
			Param("user_id")
			Param("project")
			Param("order_by")
			Param("per_page")
			Param("cursor")
			Cookie("planeSessionCookieCredential:session-id")
			rawHTTP("application/json")
			httpErrors(400, 401, 403, 404, 500)
		})
	})
	Method("export_activity", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_user_activity_by_user_id_export")
		sessionPayload(nil)
		Result(csvResponse)
		errorsFor(400, 401, 403, 404, 415, 500)
		HTTP(func() {
			POST("/api/workspaces/{slug}/user-activity/{user_id}/export/")
			Param("slug")
			Param("user_id")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(MapOf(String, Any), []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			rawCSVHTTP()
			httpErrors(400, 401, 403, 404, 415, 500)
		})
	})
	Method("profile", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_user_profile_by_user_id")
		sessionPayload(nil)
		Result(rawResponse)
		errorsFor(401, 404, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/user-profile/{user_id}/")
			Param("slug")
			Param("user_id")
			Cookie("planeSessionCookieCredential:session-id")
			rawHTTP("application/json")
			httpErrors(401, 404, 500)
		})
	})
	Method("stats", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_user_stats_by_user_id")
		sessionPayload([]string{"state", "state_group", "estimate_point", "priority", "parent", "labels", "assignees", "mentions", "created_by", "logged_by", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "type", "project", "cycle", "module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date"})
		Result(rawResponse)
		errorsFor(400, 401, 404, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/user-stats/{user_id}/")
			Param("slug")
			Param("user_id")
			for _, q := range []string{"state", "state_group", "estimate_point", "priority", "parent", "labels", "assignees", "mentions", "created_by", "logged_by", "name", "created_at", "updated_at", "start_date", "target_date", "completed_at", "type", "project", "cycle", "module", "intake_status", "inbox_status", "sub_issue", "subscriber", "start_target_date"} {
				Param(q)
			}
			Cookie("planeSessionCookieCredential:session-id")
			rawHTTP("application/json")
			httpErrors(400, 401, 404, 500)
		})
	})
})
