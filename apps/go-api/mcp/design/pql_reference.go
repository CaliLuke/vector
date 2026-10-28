package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func pqlReference() {
	Method("get_pql_reference", func() {
		Description("PQL syntax reference. Use brief for a compact reference or full for syntax, examples and functions.")
		Payload(func() {
			Attribute("detail", String, func() { Enum("brief", "full"); Default("full") })
			Meta("openapi:additionalProperties", "false")
		})
		Result(func() {
			Attribute("detail", String)
			Attribute("reference", String)
			Required("detail", "reference")
			Meta("openapi:additionalProperties", "false")
		})
		readErrors()
		Meta("mcp:annotation:readOnlyHint", "true")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:idempotentHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("get_pql_reference", "PQL fields, operators, functions and examples.", mcp.ToolTitle("PQL reference"))
	})
}
