package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func workitemLinkTarget(action string, detail bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("workitem_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id", "workitem_id")
	if detail {
		Attribute("link_id", String, func() { Format(FormatUUID) })
		Required("link_id")
	}
	Meta("openapi:additionalProperties", "false")
}

var workitemLinkRetrieve = Type("WorkitemLinkRetrieve", func() { workitemLinkTarget("retrieve", true); Attribute("include_revision", Boolean) })
var workitemLinkCreate = Type("WorkitemLinkCreate", func() {
	workitemLinkTarget("create", false)
	Attribute("url", String)
	Attribute("title", String)
	mutationExtensions(false)
	Required("url")
})
var workitemLinkUpdate = Type("WorkitemLinkUpdate", func() {
	workitemLinkTarget("update", true)
	Attribute("url", String)
	Attribute("title", String)
	mutationExtensions(true)
})
var workitemLinkDelete = Type("WorkitemLinkDelete", func() { workitemLinkTarget("delete", true); mutationExtensions(true) })
var workitemLinkPage = Type("WorkitemLinkPage", func() {
	membertypes.MemberPageFields()
	Attribute("results", ArrayOf(workitemLinkRecord))
	Meta("openapi:additionalProperties", "false")
})
var workitemLinkReceipt = Type("WorkitemLinkMutationReceipt", func() {
	Attribute("link_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("result", workitemLinkRecord, func() { Nullable() })
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("link_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var workitemLinkRequest = Type("WorkitemLinkRequest", OneOf(workitemReadList, workitemLinkRetrieve, workitemLinkCreate, workitemLinkUpdate, workitemLinkDelete), func() { Untagged() })
var workitemLinkResult = Type("WorkitemLinkResult", OneOf(workitemLinkPage, workitemLinkRecord, workitemLinkReceipt), func() { Untagged() })

func workitemLink() {
	Method("workitem_link", func() {
		Description("External links for a visible work item. Use an advertised action.")
		Payload(workitemLinkRequest)
		Result(workitemLinkResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("workitem_link", "Links attached to a work item. Use an advertised action.", mcp.ToolTitle("Work item links"))
	})
}
