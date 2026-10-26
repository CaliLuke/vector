package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var formulaAuthoringInput = Type("FormulaAuthoringInput", func() {
	Attribute("revision", Int64, func() { Minimum(0) })
	Attribute("definition", propertyDefinition)
	for _, name := range []string{"type_id", "project_id", "issue_id"} {
		Attribute(name, String, func() { Format(FormatUUID) })
	}
	Required("revision", "definition", "type_id")
})
var formulaAuthoringResult = Type("FormulaAuthoringResult", func() {
	Attribute("valid", Boolean)
	Attribute("validation_error", String)
	Attribute("active", Boolean)
	Attribute("evaluated_at", String, func() { Format(FormatDateTime) })
	Attribute("contexts", ArrayOf(propertyChoice))
	Attribute("result", formulaResult)
	Required("valid", "validation_error", "active", "evaluated_at", "contexts")
})

func formulaAuthoringMethod() {
	Method("PreviewFormula", func() {
		SessionSecurity(session)
		Payload(func() {
			Attribute("slug", String, func() { MinLength(1); MaxLength(255) })
			Attribute("data", formulaAuthoringInput)
			Required("slug", "data")
		})
		Result(formulaAuthoringResult)
		for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
			Error("status_"+strconv.Itoa(code), failure)
		}
		HTTP(func() {
			POST("/api/workspaces/{slug}/work-item-properties/formula-preview/")
			Param("slug")
			Body("data")
			Response(StatusOK)
			for _, code := range []int{400, 401, 403, 404, 409, 422, 500} {
				Response("status_"+strconv.Itoa(code), code)
			}
		})
	})
}
