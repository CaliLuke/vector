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
	Attribute("include_revision", Boolean)
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
	Attribute("revision", String)
	Attribute("sort_order", Float64, func() { Nullable() })
	Required("sort_order")
	Meta("openapi:additionalProperties", "false")
})
var labelPage = Type("LabelPage", func() {
	toolkitPageFields()
	Attribute("results", ArrayOf(labelRecord))
	Meta("openapi:additionalProperties", "false")
})
var labelRequest = Type("LabelRequest", OneOf(labelList, labelRetrieve, labelCreate, labelUpdate, labelDelete), func() { Untagged() })
var labelResult = Type("LabelResult", OneOf(labelPage, labelRecord, labelReceipt), func() { Untagged() })

func label() {
	Method("label", func() {
		Description("Read and manage labels in a project.")
		Payload(labelRequest)
		Result(labelResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Error("conflict", func() { Remedy(func() { SafeMessage("The operation conflicts with an existing resource.") }) })
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("label", "Labels in a project. Use an advertised action.", mcp.ToolTitle("Labels"))
	})
}

func labelMutationTarget(action string, detail bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	if detail {
		Attribute("label_id", String, func() { Format(FormatUUID) })
		Required("label_id")
	}
	Meta("openapi:additionalProperties", "false")
}
func labelMutationFields() {
	for _, field := range []string{"name", "color", "description", "parent", "external_source", "external_id"} {
		Attribute(field, String)
	}
	Attribute("sort_order", Float64, func() { Nullable() })
}

var labelCreate = Type("LabelCreate", func() {
	labelMutationTarget("create", false)
	labelMutationFields()
	mutationExtensions(false)
	Required("name")
})
var labelUpdate = Type("LabelUpdate", func() { labelMutationTarget("update", true); labelMutationFields(); mutationExtensions(true) })
var labelDelete = Type("LabelDelete", func() { labelMutationTarget("delete", true); mutationExtensions(true) })
var labelReceipt = Type("LabelMutationReceipt", func() {
	Attribute("label_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("result", labelRecord, func() { Nullable() })
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("label_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
