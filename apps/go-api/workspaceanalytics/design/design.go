// Package design defines workspace analytics endpoints.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/workspaceanalytics/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })

var rawMetadata = Type("WorkspaceAnalyticsRawJSON", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var errorResponse = Type("WorkspaceAnalyticsErrorResponse", func() {
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

func sessionPayload(path []string, query []string) {
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

func queryParams(names []string) {
	for _, name := range names {
		Param(name)
	}
}

var _ = API("Plane Workspace Analytics API", func() {
	Title("Plane Workspace Analytics API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json")
		Produces("application/json")
	})
})

var _ = Service("workspace_analytics", func() {
	Description("Workspace and project analytics read endpoints.")
	HTTP(func() { RuntimeCORS() })

	Method("advance_analytics", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_advance_analytics")
		sessionPayload([]string{"slug"}, []string{"date_filter", "project_ids", "tab"})
		Result(rawMetadata)
		errorsFor(400, 401, 403, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/advance-analytics/")
			Param("slug")
			queryParams([]string{"date_filter", "project_ids", "tab"})
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() {
				Header("content_type:Content-Type")
				ContentType("application/json")
				OpenAPIBody(Any)
			})
			httpErrors(400, 401, 403, 500)
		})
	})
	Method("advance_analytics_stats", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_advance_analytics_stats")
		sessionPayload([]string{"slug"}, []string{"date_filter", "project_ids", "type"})
		Result(rawMetadata)
		errorsFor(400, 401, 403, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/advance-analytics-stats/")
			Param("slug")
			queryParams([]string{"date_filter", "project_ids", "type"})
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 403, 500)
		})
	})
	Method("advance_analytics_charts", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_advance_analytics_charts")
		sessionPayload([]string{"slug"}, []string{"date_filter", "project_ids", "type", "x_axis", "group_by"})
		Result(rawMetadata)
		errorsFor(400, 401, 403, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/advance-analytics-charts/")
			Param("slug")
			queryParams([]string{"date_filter", "project_ids", "type", "x_axis", "group_by"})
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 403, 500)
		})
	})
	Method("project_advance_analytics", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_advance_analytics")
		sessionPayload([]string{"slug", "project_id"}, []string{"date_filter", "project_ids", "cycle_id", "module_id"})
		Result(rawMetadata)
		errorsFor(400, 401, 403, 404, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/advance-analytics/")
			Param("slug")
			Param("project_id")
			queryParams([]string{"date_filter", "project_ids", "cycle_id", "module_id"})
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 403, 404, 500)
		})
	})
	Method("project_advance_analytics_stats", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_advance_analytics_stats")
		sessionPayload([]string{"slug", "project_id"}, []string{"date_filter", "project_ids", "type", "cycle_id", "module_id"})
		Result(rawMetadata)
		errorsFor(400, 401, 403, 404, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/advance-analytics-stats/")
			Param("slug")
			Param("project_id")
			queryParams([]string{"date_filter", "project_ids", "type", "cycle_id", "module_id"})
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 403, 404, 500)
		})
	})
	Method("project_advance_analytics_charts", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_advance_analytics_charts")
		sessionPayload([]string{"slug", "project_id"}, []string{"date_filter", "project_ids", "type", "cycle_id", "module_id", "x_axis", "group_by"})
		Result(rawMetadata)
		errorsFor(400, 401, 403, 404, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/advance-analytics-charts/")
			Param("slug")
			Param("project_id")
			queryParams([]string{"date_filter", "project_ids", "type", "cycle_id", "module_id", "x_axis", "group_by"})
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 403, 404, 500)
		})
	})
	Method("analytics", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_analytics")
		analyticsParams := append([]string{"x_axis", "y_axis", "segment"}, analyticsFilterParams...)
		sessionPayload([]string{"slug"}, analyticsParams)
		Result(rawMetadata)
		errorsFor(400, 401, 403, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/analytics/")
			Param("slug")
			queryParams(analyticsParams)
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 403, 500)
		})
	})
	Method("default_analytics", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_default_analytics")
		sessionPayload([]string{"slug"}, analyticsFilterParams)
		Result(rawMetadata)
		errorsFor(400, 401, 403, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/default-analytics/")
			Param("slug")
			queryParams(analyticsFilterParams)
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 403, 500)
		})
	})
	Method("saved_analytic_view", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_saved_analytic_view_by_analytic_id")
		sessionPayload([]string{"slug", "analytic_id"}, []string{"segment"})
		Result(rawMetadata)
		errorsFor(400, 401, 403, 404, 500)
		HTTP(func() {
			GET("/api/workspaces/{slug}/saved-analytic-view/{analytic_id}/")
			Param("slug")
			Param("analytic_id")
			queryParams([]string{"segment"})
			Cookie("planeSessionCookieCredential:session-id")
			SkipResponseBodyEncodeDecode()
			Response(200, func() { Header("content_type:Content-Type"); ContentType("application/json"); OpenAPIBody(Any) })
			httpErrors(400, 401, 403, 404, 500)
		})
	})
})

var analyticsFilterParams = []string{
	"state", "state_group", "estimate_point", "priority", "parent", "labels", "assignees",
	"mentions", "created_by", "logged_by", "name", "created_at", "updated_at", "start_date",
	"target_date", "completed_at", "type", "project", "cycle", "module", "intake_status",
	"inbox_status", "sub_issue", "subscriber", "start_target_date",
}
