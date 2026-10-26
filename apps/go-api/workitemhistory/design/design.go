// Package design defines the issue history, metadata, and version read routes.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/workitemhistory/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() { Description("Plane session cookie authentication.") })
var responseMetadata = Type("WorkItemHistoryResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})
var errorBody = Type("WorkItemHistoryErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func sessionPayload(pathAttributes []string, queryAttributes ...string) {
	Security(sessionCookie)
	Payload(func() {
		Attribute("slug", String)
		Attribute("project_id", String, func() { Format(FormatUUID) })
		for _, name := range pathAttributes {
			Attribute(name, String, func() { Format(FormatUUID) })
		}
		for _, name := range queryAttributes {
			Attribute(name, String)
		}
		APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		Required(append([]string{"slug", "project_id"}, pathAttributes...)...)
	})
}

func declareErrors() {
	for _, status := range []int{400, 401, 403, 404, 500} {
		name := "status_" + strconv.Itoa(status)
		Error(name, errorBody, func() {
			if status >= 500 {
				Fault()
			}
		})
	}
}

func rawResponse() {
	SkipResponseBodyEncodeDecode()
	Response(200, func() {
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(Any)
	})
}

func httpErrors() {
	for _, status := range []int{400, 401, 403, 404, 500} {
		Response("status_"+strconv.Itoa(status), status, func() {
			ContentType("application/json")
			ErrorNameFromStatus()
			Body("body")
		})
	}
}

var _ = API("Plane Work Item History API", func() {
	Title("Plane Work Item History API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("work_item_history", func() {
	Description("Issue activity history, metadata, and version reads.")
	HTTP(func() { RuntimeCORS() })
	Method("history", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_history")
		sessionPayload([]string{"issue_id"}, "created_at__gt", "activity_type")
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/history/")
			Param("slug")
			Param("project_id")
			Param("issue_id")
			Param("created_at__gt")
			Param("activity_type")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})
	Method("meta", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_meta")
		sessionPayload([]string{"issue_id"})
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/meta/")
			Param("slug")
			Param("project_id")
			Param("issue_id")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})
	Method("versions", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_versions")
		sessionPayload([]string{"issue_id"}, "cursor")
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/")
			Param("slug")
			Param("project_id")
			Param("issue_id")
			Param("cursor")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})
	Method("version_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_versions_by_pk")
		sessionPayload([]string{"issue_id", "pk"})
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/issues/{issue_id}/versions/{pk}/")
			Param("slug")
			Param("project_id")
			Param("issue_id")
			Param("pk")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})
	Method("description_versions", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_work_items_by_work_item_id_description_versions")
		sessionPayload([]string{"work_item_id"}, "cursor")
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Param("cursor")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})
	Method("description_version_detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_work_items_by_work_item_id_description_versions_by_pk")
		sessionPayload([]string{"work_item_id", "pk"})
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/description-versions/{pk}/")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Param("pk")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})
	Method("property_history", func() {
		sessionPayload([]string{"work_item_id"})
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/property-history/")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})

	Method("property_versions", func() {
		sessionPayload([]string{"work_item_id", "property_id"}, "before")
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/properties/{property_id}/versions/")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Param("property_id")
			Param("before")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})

	Method("property_version_detail", func() {
		sessionPayload([]string{"work_item_id", "property_id", "pk"})
		Result(responseMetadata)
		declareErrors()
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/work-items/{work_item_id}/properties/{property_id}/versions/{pk}/")
			Param("slug")
			Param("project_id")
			Param("work_item_id")
			Param("property_id")
			Param("pk")
			Cookie("planeSessionCookieCredential:session-id")
			rawResponse()
			httpErrors()
		})
	})

})
