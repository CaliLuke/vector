package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func attachmentTarget(actions []any, detail bool) {
	Attribute("action", String, func() { Enum(actions...) })
	for _, key := range []string{"project_id", "workitem_id"} {
		Attribute(key, String, func() { Format(FormatUUID) })
		Required(key)
	}
	projectWorkspaceSelector()
	Required("action")
	if detail {
		Attribute("attachment_id", String, func() { Format(FormatUUID) })
		Required("attachment_id")
	}
	Meta("openapi:additionalProperties", "false")
}

var attachmentList = Type("AttachmentList", func() { attachmentTarget([]any{"list"}, false); Attribute("include_revision", Boolean) })
var attachmentRead = Type("AttachmentRead", func() { attachmentTarget([]any{"read", "download_url"}, true); Attribute("include_revision", Boolean) })
var attachmentUpload = Type("AttachmentUpload", func() {
	attachmentTarget([]any{"upload_from_url"}, false)
	Attribute("url", String)
	Attribute("name", String)
	Required("url")
	mutationExtensions(false)
})
var attachmentDelete = Type("AttachmentDelete", func() { attachmentTarget([]any{"delete"}, true); mutationExtensions(true) })
var attachmentRecord = Type("AttachmentRecord", func() {
	Attribute("asset", String)
	Required("asset")
	for _, key := range []string{"id", "created_at", "updated_at", "deleted_at", "entity_type", "entity_identifier", "external_id", "external_source", "created_by", "updated_by", "user", "workspace", "draft_issue", "project", "issue", "comment", "page", "name", "content_type"} {
		Attribute(key, String, func() { Nullable() })
		Required(key)
	}
	for _, key := range []string{"attributes", "storage_metadata"} {
		Attribute(key, Any, func() { Nullable() })
		Required(key)
	}
	for _, key := range []string{"is_deleted", "is_archived", "is_uploaded"} {
		Attribute(key, Boolean, func() { Nullable() })
		Required(key)
	}
	Attribute("size", Int64, func() { Nullable() })
	Required("size")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var attachmentRecords = Type("AttachmentRecords", ArrayOf(attachmentRecord))
var attachmentDownload = Type("AttachmentDownload", func() {
	Attribute("download_url", String)
	Attribute("attachment_id", String)
	Attribute("name", String)
	Attribute("revision", String)
	Required("download_url", "attachment_id", "name")
	Meta("openapi:additionalProperties", "false")
})
var attachmentInline = Type("AttachmentInline", func() {
	Attribute("kind", String, func() { Enum("text", "image") })
	Attribute("mime_type", String)
	Attribute("text", String)
	Attribute("data", Bytes)
	Required("kind", "mime_type")
	Meta("openapi:additionalProperties", "false")
})
var attachmentReceipt = Type("AttachmentMutationReceipt", func() {
	Attribute("attachment_id", String)
	Attribute("revision", String)
	Attribute("result", attachmentRecord, func() { Nullable() })
	Attribute("request_id", String)
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("attachment_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var attachmentRequest = Type("WorkitemAttachmentRequest", OneOf(attachmentList, attachmentRead, attachmentUpload, attachmentDelete), func() { Untagged() })
var attachmentResult = Type("WorkitemAttachmentResult", OneOf(attachmentRecords, attachmentRecord, attachmentDownload, attachmentInline, attachmentReceipt), func() { Untagged() })

func workitemAttachment() {
	Method("workitem_attachment", func() {
		Description("List, read, download, import and delete work item attachments. Inline images support PNG/JPEG/GIF/WEBP up to 5 MB; text supports TXT/MD/CSV/HTML/XML/YAML/JSON up to 1 MB. URL imports require public HTTP(S), reject private addresses, and accept files up to 5 MB. Upload bytes are durable before object storage delivery. Use request_id to recover an interrupted import. include_revision returns current revisions for guarded deletion.")
		Payload(attachmentRequest)
		Result(attachmentResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "true")
		mcp.Tool("workitem_attachment", "Files attached to a work item, with inline image/text reads.", mcp.ToolTitle("Work item attachments"))
	})
}
