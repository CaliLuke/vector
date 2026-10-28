package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func workitemCommentTarget(action string, detail bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("workitem_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id", "workitem_id")
	if detail {
		Attribute("comment_id", String, func() { Format(FormatUUID) })
		Required("comment_id")
	}
	Meta("openapi:additionalProperties", "false")
}

var workitemCommentRetrieve = Type("WorkitemCommentRetrieve", func() { workitemCommentTarget("retrieve", true); Attribute("include_revision", Boolean) })
var workitemCommentCreate = Type("WorkitemCommentCreate", func() {
	workitemCommentTarget("create", false)
	for _, field := range []string{"comment_html", "access", "external_source", "external_id"} {
		Attribute(field, String)
	}
	mutationExtensions(false)
	Required("comment_html")
})
var workitemCommentUpdate = Type("WorkitemCommentUpdate", func() {
	workitemCommentTarget("update", true)
	for _, field := range []string{"comment_html", "access", "external_source", "external_id"} {
		Attribute(field, String)
	}
	mutationExtensions(true)
})
var workitemCommentDelete = Type("WorkitemCommentDelete", func() { workitemCommentTarget("delete", true); mutationExtensions(true) })
var workitemCommentPage = Type("WorkitemCommentPage", func() {
	membertypes.MemberPageFields()
	Attribute("results", ArrayOf(workitemCommentRecord))
	Meta("openapi:additionalProperties", "false")
})
var workitemCommentReceipt = Type("WorkitemCommentMutationReceipt", func() {
	Attribute("comment_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("result", workitemCommentRecord, func() { Nullable() })
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("comment_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var workitemCommentRequest = Type("WorkitemCommentRequest", OneOf(workitemReadList, workitemCommentRetrieve, workitemCommentCreate, workitemCommentUpdate, workitemCommentDelete), func() { Untagged() })
var workitemCommentResult = Type("WorkitemCommentResult", OneOf(workitemCommentPage, workitemCommentRecord, workitemCommentReceipt), func() { Untagged() })

func workitemComment() {
	Method("workitem_comment", func() {
		Description("Comments for a visible work item. Use an advertised action.")
		Payload(workitemCommentRequest)
		Result(workitemCommentResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("workitem_comment", "Comments attached to a work item. Use an advertised action.", mcp.ToolTitle("Work item comments"))
	})
}
