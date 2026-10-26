package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var propertyRestore = Type("ItemPropertyRestore", func() {
	Attribute("property_id", String, func() { Format(FormatUUID) })
	Attribute("version_id", String, func() { Format(FormatUUID) })
	Required("property_id", "version_id")
})

var itemEditCommand = Type("ItemEditInput", func() {
	Attribute("mode", String, func() { Enum("edit", "draft", "publish") })
	Attribute("item_id", String, func() { Format(FormatUUID) })
	Attribute("project_id", String, func() { Pattern(`^(|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`) })
	Attribute("revision", Int64, func() { Minimum(0) })
	Attribute("changes", MapOf(String, Any), func() { MaxLength(32) })
	Attribute("expected_source_version", String, func() { Pattern(`^[a-f0-9]{64}$`) })
	Attribute("restore", propertyRestore)
	Attribute("review_id", String, func() { Format(FormatUUID) })
	Required("mode", "item_id", "project_id", "revision", "changes")
})
var itemEditFailure = Type("ItemEditError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("message", String)
	Attribute("fields", MapOf(String, Any))
	Required("kind", "message")
})

func itemEditMethods() {
	itemEditSourceMethod()
	for _, action := range []string{"PreviewItemChange", "CommitItemChange"} {
		Method(action, func() {
			SessionSecurity(session)
			Payload(func() {
				Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
				Attribute("data", itemEditCommand)
				Required("slug", "data")
			})
			Result(Any)
			for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
				if code == 401 {
					Error("status_401", failure)
				} else {
					Error("status_"+strconv.Itoa(code), itemEditFailure)
				}
			}
			HTTP(func() {
				path := "/api/workspaces/{slug}/work-item-edits/preview/"
				if action == "CommitItemChange" {
					path = "/api/workspaces/{slug}/work-item-edits/commit/"
				}
				POST(path)
				Param("slug")
				Body("data")
				Response(StatusOK)
				for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
					Response("status_"+strconv.Itoa(code), code)
				}
			})
		})
	}
}

func itemEditSourceMethod() {
	Method("ReadItemEditSource", func() {
		SessionSecurity(session)
		Payload(func() {
			Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
			Attribute("mode", String, func() { Enum("edit", "draft") })
			Attribute("item_id", String, func() { Format(FormatUUID) })
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug", "mode", "item_id")
		})
		Result(Any)
		for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
			if code == 401 {
				Error("status_401", failure)
			} else {
				Error("status_"+strconv.Itoa(code), itemEditFailure)
			}
		}
		HTTP(func() {
			GET("/api/workspaces/{slug}/work-item-edits/source/")
			for _, param := range []string{"slug", "mode", "item_id", "project_id"} {
				Param(param)
			}
			Response(StatusOK)
			for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
				Response("status_"+strconv.Itoa(code), code)
			}
		})
	})
}
