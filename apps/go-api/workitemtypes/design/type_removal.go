package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var typeRemovalInput = Type("TypeRemovalInput", func() {
	Attribute("revision", Int64, func() { Minimum(0) })
	Attribute("type_id", String, func() { Format(FormatUUID) })
	Attribute("replacement_id", String, func() { Format(FormatUUID) })
	Attribute("review_id", String, func() { Format(FormatUUID) })
	Required("revision", "type_id", "replacement_id")
})
var typeRemovalReview = Type("TypeRemovalReview", func() {
	Attribute("id", String)
	Attribute("blocked_reason", String)
	for _, name := range []string{"items", "drafts", "projects", "preserved", "invalid_items"} {
		Attribute(name, Int)
	}
	Required("id", "blocked_reason", "items", "drafts", "projects", "preserved", "invalid_items")
})

func typeRemovalMethods() {
	for _, operation := range []struct {
		name, path string
		result     any
	}{
		{"PreviewTypeRemoval", "preview", typeRemovalReview},
		{"CommitTypeRemoval", "commit", catalog},
	} {
		Method(operation.name, func() {
			SessionSecurity(session)
			Payload(func() {
				Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
				Attribute("data", typeRemovalInput)
				Required("slug", "data")
			})
			Result(operation.result)
			for _, status := range []int{400, 401, 403, 404, 409, 422, 500} {
				Error("status_"+strconv.Itoa(status), failure)
			}
			HTTP(func() {
				POST("/api/workspaces/{slug}/work-item-types/removal/" + operation.path + "/")
				Param("slug")
				Body("data")
				Response(StatusOK)
				for _, status := range []int{400, 401, 403, 404, 409, 422, 500} {
					Response("status_"+strconv.Itoa(status), status)
				}
			})
		})
	}
}
