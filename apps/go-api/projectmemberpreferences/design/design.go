// Package design defines project member preference updates.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/projectmemberpreferences/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var Preferences = Type("Preferences", func() {
	Attribute("preferences", Any)
	Required("preferences")
})

var PreferenceError = Type("PreferenceError", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

var PreferenceHTTPError = Type("PreferenceHTTPError", func() {
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "Message") })
})

var SessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {})
var Session = SessionAuth("PlaneSession", func() {
	CookieTransport(SessionCookieSecurity, "", func() { CookieName("session-id") })
})

var _ = API("Project Member Preferences", func() {
	Title("Project Member Preferences")
	Version("1.0.0")
	Meta("http:generate", "server")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("ProjectMemberPreferences", func() {
	HTTP(func() { RuntimeCORS() })
	Method("Patch", func() {
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_preferences_member_by_member_id")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/preferences/member/{member_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		SessionSecurity(Session)
		Payload(func() {
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("member_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "member_id")
		})
		Result(Preferences)
		Error("status_400", PreferenceError)
		Error("status_401", PreferenceError)
		Error("status_403", PreferenceError)
		Error("status_404", PreferenceError)
		Error("status_415", PreferenceError)
		Error("status_500", PreferenceError, func() { Fault() })
		HTTP(func() {
			Meta("openapi:tag:projects")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/preferences/member/{member_id}/")
			Param("slug")
			Param("project_id")
			Param("member_id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(MapOf(String, Any), []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() { ContentType("application/json"); Body(Preferences) })
			for _, status := range []int{400, 401, 403, 404, 415, 500} {
				Response("status_"+strconv.Itoa(status), status, func() {
					ContentType("application/json")
					ErrorNameFromStatus()
					Body("body")
				})
			}
		})
	})
})
