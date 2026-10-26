package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

// Project catalogs share authenticated read/update routes and the same error contract.
func projectCatalogMethods(readName, writeName, resource string, catalog, input any) {
	for _, action := range []string{readName, writeName} {
		Method(action, func() {
			SessionSecurity(session)
			Payload(func() {
				Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
				Attribute("project_id", String, func() { Format(FormatUUID) })
				Required("slug", "project_id")
				if action == writeName {
					Attribute("data", input)
					Required("data")
				}
			})
			Result(catalog)
			for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
				Error("status_"+strconv.Itoa(code), failure)
			}
			HTTP(func() {
				path := "/api/workspaces/{slug}/projects/{project_id}/" + resource + "/"
				if action == writeName {
					POST(path)
					Body("data")
				} else {
					GET(path)
				}
				Param("slug")
				Param("project_id")
				Response(StatusOK)
				for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
					Response("status_"+strconv.Itoa(code), code)
				}
			})
		})
	}
}
