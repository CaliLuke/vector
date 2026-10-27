package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

var labelList = Type("LabelList", func() {
	Attribute("action", String, func() { Enum("list") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("cursor", String)
	Attribute("per_page", Int, "Zero or omitted uses 1000.", func() { Minimum(0); Maximum(1000) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	Meta("openapi:additionalProperties", "false")
})
var labelRetrieve = Type("LabelRetrieve", func() {
	Attribute("action", String, func() { Enum("retrieve") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("label_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id", "label_id")
	Meta("openapi:additionalProperties", "false")
})
var labelRecord = Type("LabelRecord", func() {
	Attribute("name", String)
	Required("name")
	for _, field := range []string{"id", "created_at", "updated_at", "deleted_at", "description", "color", "external_source", "external_id", "created_by", "updated_by", "workspace", "project", "parent"} {
		Attribute(field, String, func() { Nullable() })
		Required(field)
	}
	Attribute("sort_order", Float64, func() { Nullable() })
	Required("sort_order")
	Meta("openapi:additionalProperties", "false")
})
var labelPage = Type("LabelPage", func() {
	// The pinned toolkit envelope exposes seven fields, unlike SDK page models.
	Attribute("results", ArrayOf(labelRecord))
	Attribute("count", Int)
	Attribute("total_count", Int)
	Attribute("next_cursor", String)
	Attribute("prev_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_page_results", Boolean)
	Required("results", "count", "total_count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results")
	Meta("openapi:additionalProperties", "false")
})
var labelRequest = Type("LabelRequest", OneOf(labelList, labelRetrieve), func() { Untagged() })
var labelResult = Type("LabelResult", OneOf(labelPage, labelRecord), func() { Untagged() })

func label() {
	Method("label", func() {
		Description("List labels or retrieve a label in a project.")
		Payload(labelRequest)
		Result(labelResult)
		readErrors()
		Meta("mcp:annotation:readOnlyHint", "true")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("label", "Labels in a project. Use an advertised action.", mcp.ToolTitle("Labels"))
	})
}
