// Package design contains the six project intake/inbox write operations.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/projectintakewrites/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var ResponseMetadata = Type("ProjectIntakeWritesResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})
var ErrorBody = Type("ProjectIntakeWritesErrorResponse", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})
var SessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {})
var Session = SessionAuth("PlaneSession", func() {
	CookieTransport(SessionCookieSecurity, "", func() { CookieName("session-id") })
})

var IntakeWriteRequest = Type("IntakeWriteRequest", func() {
	Attribute("name", String, func() { MaxLength(255) })
	Attribute("description", String)
	Attribute("is_default", Boolean)
	Attribute("view_props", Any)
	Attribute("logo_props", Any)
})
var IntakeProjectDetail = Type("IntakeProjectDetail", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("name", String)
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("cover_image_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("logo_props", Any)
	Attribute("description", String)
	Required("id", "identifier", "name", "cover_image", "cover_image_url", "logo_props", "description")
})
var Intake = Type("Intake", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("description", String)
	Attribute("is_default", Boolean)
	Attribute("view_props", Any)
	Attribute("logo_props", Any)
	Attribute("project_detail", IntakeProjectDetail)
	Attribute("pending_issue_count", Int)
	Required("id", "created_at", "updated_at", "deleted_at", "created_by", "updated_by", "workspace", "project", "name", "description", "is_default", "view_props", "logo_props", "project_detail")
})

func rawRequest(schema any) {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(schema, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
}
func rawResponse(status int, schema any) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}
func pathPayload(detail bool) {
	Payload(func() {
		Attribute("slug", String)
		Attribute("project_id", String, func() { Format(FormatUUID) })
		if detail {
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "pk")
		} else {
			Required("slug", "project_id")
		}
	})
}
func writeErrors() {
	for _, status := range []int{400, 401, 403, 404, 405, 415, 500} {
		Error("status_"+itoa(status), ErrorBody, func() {
			if status == 500 {
				Fault()
			}
		})
	}
}
func httpWriteErrors() {
	for _, status := range []int{400, 401, 403, 404, 405, 415, 500} {
		Response("status_"+strconv.Itoa(status), status, func() {
			ContentType("application/json")
			ErrorNameFromStatus()
			Body("body")
		})
	}
}
func itoa(value int) string { return strconv.Itoa(value) }

var _ = API("Project Intake Writes", func() {
	Title("Project Intake Writes")
	Version("1.0.0")
	Meta("http:generate", "server")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("ProjectIntakeWrites", func() {
	HTTP(func() { RuntimeCORS() })
	Method("CreateInbox", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_inboxes")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/inboxes/\"")
		SessionSecurity(Session)
		pathPayload(false)
		Result(ResponseMetadata)
		writeErrors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/inboxes/")
			Param("slug")
			Param("project_id")
			rawRequest(IntakeWriteRequest)
			rawResponse(201, Intake)
			httpWriteErrors()
		})
	})
	Method("PatchInbox", func() {
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_inboxes_by_pk")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/\"")
		SessionSecurity(Session)
		pathPayload(true)
		Result(ResponseMetadata)
		writeErrors()
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			rawRequest(IntakeWriteRequest)
			rawResponse(200, Intake)
			httpWriteErrors()
		})
	})
	Method("DeleteInbox", func() {
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_inboxes_by_pk")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/\"")
		SessionSecurity(Session)
		pathPayload(true)
		Result(Empty)
		writeErrors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/projects/{project_id}/inboxes/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(204)
			httpWriteErrors()
		})
	})
	Method("CreateIntake", func() {
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_intakes")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/intakes/\"")
		SessionSecurity(Session)
		pathPayload(false)
		Result(ResponseMetadata)
		writeErrors()
		HTTP(func() {
			POST("/api/workspaces/{slug}/projects/{project_id}/intakes/")
			Param("slug")
			Param("project_id")
			rawRequest(IntakeWriteRequest)
			rawResponse(201, Intake)
			httpWriteErrors()
		})
	})
	Method("PatchIntake", func() {
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_intakes_by_pk")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/\"")
		SessionSecurity(Session)
		pathPayload(true)
		Result(ResponseMetadata)
		writeErrors()
		HTTP(func() {
			PATCH("/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			rawRequest(IntakeWriteRequest)
			rawResponse(200, Intake)
			httpWriteErrors()
		})
	})
	Method("DeleteIntake", func() {
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_intakes_by_pk")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/\"")
		SessionSecurity(Session)
		pathPayload(true)
		Result(Empty)
		writeErrors()
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/projects/{project_id}/intakes/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(204)
			httpWriteErrors()
		})
	})
})
