package design

import . "github.com/CaliLuke/loom/dsl"

var projectIdentifier = Type("ProjectIdentifier", func() {
	Attribute("id", Int64)
	Attribute("name", String)
	Attribute("project", String, func() { Format(FormatUUID) })
	Required("id", "name", "project")
})

var projectIdentifierCollection = Type("ProjectIdentifierCollection", func() {
	Attribute("exists", Int)
	Attribute("identifiers", ArrayOf(projectIdentifier))
	Required("exists", "identifiers")
})

var projectIdentifierDelete = Type("ProjectIdentifierDelete", func() {
	Attribute("name", String)
	Required("name")
})

var projectIdentifierUnauthenticated = Type("ProjectIdentifierUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})
var projectIdentifierForbidden = Type("ProjectIdentifierForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})
var projectIdentifierInvalid = Type("ProjectIdentifierInvalid", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})
var projectIdentifierConflict = Type("ProjectIdentifierConflict", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})
var projectIdentifierInternal = Type("ProjectIdentifierInternal", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("project_identifiers", func() {
	Description("Checks and removes unused project identifiers in a workspace.")
	Method("get", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("name", String)
			Required("slug", "name")
		})
		Result(projectIdentifierCollection)
		Error("unauthenticated", projectIdentifierUnauthenticated)
		Error("forbidden", projectIdentifierForbidden)
		Error("invalid", projectIdentifierInvalid)
		Error("internal_error", projectIdentifierInternal, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/project-identifiers/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("name")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("invalid", StatusBadRequest)
			Response("internal_error", StatusInternalServerError)
		})
	})
	Method("delete", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("body", projectIdentifierDelete)
			Required("slug", "body")
		})
		Error("unauthenticated", projectIdentifierUnauthenticated)
		Error("forbidden", projectIdentifierForbidden)
		Error("invalid", projectIdentifierInvalid)
		Error("conflict", projectIdentifierConflict)
		Error("internal_error", projectIdentifierInternal, func() { Fault() })
		HTTP(func() {
			DELETE("/api/workspaces/{slug}/project-identifiers/")
			Cookie("session_id:session-id")
			Param("slug")
			Body("body")
			Response(StatusNoContent)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("invalid", StatusBadRequest)
			Response("conflict", StatusBadRequest)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
