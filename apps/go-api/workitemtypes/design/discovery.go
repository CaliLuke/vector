package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var typeIdentity = Type("WorkItemTypeIdentity", func() {
	for _, name := range []string{"id", "name", "icon", "color"} {
		Attribute(name, String)
	}
	Attribute("active", Boolean)
	Attribute("default", Boolean)
	Attribute("level", Int)
	Required("id", "name", "icon", "color", "active", "default", "level")
})

var identityCatalog = Type("TypeIdentityCatalog", func() {
	Attribute("enabled", Boolean)
	Attribute("revision", Int64)
	Attribute("types", ArrayOf(typeIdentity))
	Required("enabled", "revision", "types")
})

func typeIdentityMethod() {
	discoveryMethod("TypeIdentities", "work-item-type-identities", identityCatalog)
}

// Discovery catalogs share the same authenticated scope and read-only error contract.
func discoveryMethod(name, resource string, catalog any) {
	Method(name, func() {
		SessionSecurity(session)
		Payload(func() {
			Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug")
		})
		Result(catalog)
		for _, code := range []int{400, 401, 403, 422, 500} {
			Error("status_"+strconv.Itoa(code), failure)
		}
		HTTP(func() {
			GET("/api/workspaces/{slug}/" + resource + "/")
			Param("slug")
			Param("project_id")
			Response(StatusOK)
			for _, code := range []int{400, 401, 403, 422, 500} {
				Response("status_"+strconv.Itoa(code), code)
			}
		})
	})
}
