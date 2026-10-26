// Package design defines the workspace work item type library.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/workitemtypes/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var cookie = APIKeySecurity("WorkItemTypesCookie", func() {})
var session = SessionAuth("WorkItemTypesSession", func() { CookieTransport(cookie, "", func() { CookieName("session-id") }) })
var failure = Type("TypeError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("message", String)
	Attribute("review_again", Boolean)
	Required("kind", "message")
})
var item = Type("WorkItemType", func() {
	for _, name := range []string{"id", "name", "description", "icon", "color"} {
		Attribute(name, String)
	}
	for _, name := range []string{"active", "default", "mandatory"} {
		Attribute(name, Boolean)
	}
	Attribute("level", Int)
	Attribute("usage", Int)
	Required("id", "name", "description", "icon", "color", "active", "default", "mandatory", "level", "usage")
})
var catalog = Type("Catalog", func() {
	Attribute("enabled", Boolean)
	Attribute("can_manage", Boolean)
	Attribute("revision", Int64)
	Attribute("project_count", Int)
	Attribute("types", ArrayOf(item))
	Required("enabled", "can_manage", "revision", "project_count", "types")
})
var input = Type("TypeInput", func() {
	Attribute("revision", Int64, func() { Minimum(0) })
	Attribute("name", String, func() { MinLength(1); MaxLength(255) })
	Attribute("description", String, func() { MaxLength(10000) })
	Attribute("icon", String, func() { Pattern(`^[A-Za-z][A-Za-z0-9]{0,63}$`) })
	Attribute("color", String, func() { Pattern(`^#[0-9a-fA-F]{6}$`) })
	Attribute("active", Boolean)
	Attribute("mandatory", Boolean)
	Required("revision", "name", "description", "icon", "color", "active", "mandatory")
})
var revision = Type("Revision", func() { Attribute("revision", Int64, func() { Minimum(0) }); Required("revision") })
var _ = API("Vector Work Item Types", func() {
	Title("Vector Work Item Types")
	Version("1.0")
	Meta("http:generate", "server")
	HTTP(func() { Consumes("application/json"); Produces("application/json") })
})
var _ = Service("WorkItemTypes", func() {
	HTTP(func() { RuntimeCORS() })
	projectMethods()
	typeIdentityMethod()
	propertyDiscoveryMethod()
	releaseMethods()
	propertyMethods()
	itemSchemaMethod()
	formulaAuthoringMethod()
	hierarchyMethods()
	itemEditMethods()
	typeRemovalMethods()
	for _, action := range []string{"Catalog", "Enable", "Create", "Update", "Delete"} {
		Method(action, func() {
			SessionSecurity(session)
			Payload(func() {
				Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
				Required("slug")
				if action == "Update" || action == "Delete" {
					Attribute("id", String, func() { Format(FormatUUID) })
					Required("id")
				}
				if action == "Create" || action == "Update" {
					Attribute("data", input)
					Required("data")
				} else if action != "Catalog" {
					Attribute("data", revision)
					Required("data")
				}
			})
			Result(catalog)
			for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
				Error("status_"+strconv.Itoa(code), failure)
			}
			HTTP(func() {
				base := "/api/workspaces/{slug}/work-item-types/"
				switch action {
				case "Catalog":
					GET(base)
				case "Enable":
					POST(base + "enable/")
				case "Create":
					POST(base)
				case "Update":
					PUT(base + "{id}/")
				case "Delete":
					POST(base + "{id}/delete/")
				}
				Param("slug")
				if action == "Update" || action == "Delete" {
					Param("id")
				}
				if action != "Catalog" {
					Body("data")
				}
				Response(StatusOK)
				for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
					Response("status_"+strconv.Itoa(code), code)
				}
			})
		})
	}
})
