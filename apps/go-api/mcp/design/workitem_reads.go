package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func nullableStrings(fields ...string) {
	for _, field := range fields {
		Attribute(field, String, func() { Nullable() })
		Required(field)
	}
}

var workitemCommentRecord = Type("WorkitemCommentRecord", func() {
	Attribute("revision", String)
	nullableStrings("id", "created_at", "updated_at", "deleted_at", "comment_stripped", "comment_html", "external_source", "external_id", "edited_at", "created_by", "updated_by", "project", "workspace", "issue", "actor", "parent", "description")
	Attribute("is_member", Boolean, func() { Nullable() })
	Required("is_member")
	Attribute("attachments", ArrayOf(String), func() { Nullable() })
	Required("attachments")
	// https://github.com/CaliLuke/loom/issues/1037: restore Enum after nullable
	// union matching agrees with record validation. Projection enforces values.
	Attribute("access", String, func() { Nullable() })
	Required("access")
	Meta("openapi:additionalProperties", "false")
})
var workitemLinkRecord = Type("WorkitemLinkRecord", func() {
	Attribute("revision", String)
	Attribute("url", String)
	Required("url")
	nullableStrings("id", "created_at", "updated_at", "deleted_at", "title", "created_by", "updated_by", "project", "workspace", "issue")
	Attribute("metadata", Any, func() { Nullable() })
	Required("metadata")
	Meta("openapi:additionalProperties", "false")
})
var workitemActivityRecord = Type("WorkitemActivityRecord", func() {
	Attribute("project", String)
	Attribute("workspace", String)
	Required("project", "workspace")
	nullableStrings("id", "created_at", "updated_at", "deleted_at", "verb", "field", "old_value", "new_value", "comment", "old_identifier", "new_identifier", "issue", "issue_comment", "actor")
	Attribute("attachments", ArrayOf(String), func() { Nullable() })
	Required("attachments")
	Attribute("epoch", Int64, func() { Nullable() })
	Required("epoch")
	Meta("openapi:additionalProperties", "false")
})

var workitemReadList = Type("WorkitemReadList", func() {
	Attribute("action", String, func() { Enum("list") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("workitem_id", String, func() { Format(FormatUUID) })
	Attribute("cursor", String)
	Attribute("per_page", Int, "Zero or omitted uses 1000.", func() { Minimum(0); Maximum(1000) })
	projectWorkspaceSelector()
	Required("action", "project_id", "workitem_id")
	Meta("openapi:additionalProperties", "false")
})

func workitemReadTool(name, title, prefix, target string, record expr.UserType, compact bool) func() {

	detail := Type(prefix+"Retrieve", func() {
		Attribute("action", String, func() { Enum("retrieve") })
		Attribute("project_id", String, func() { Format(FormatUUID) })
		Attribute("workitem_id", String, func() { Format(FormatUUID) })
		Attribute(target, String, func() { Format(FormatUUID) })
		projectWorkspaceSelector()
		Required("action", "project_id", "workitem_id", target)
		Meta("openapi:additionalProperties", "false")
	})
	page := Type(prefix+"Page", func() {
		if compact {
			toolkitPageFields()
		} else {
			membertypes.MemberPageFields()
		}
		Attribute("results", ArrayOf(record))
		Meta("openapi:additionalProperties", "false")
	})
	request := Type(prefix+"Request", OneOf(workitemReadList, detail), func() { Untagged() })
	result := Type(prefix+"Result", OneOf(page, record), func() { Untagged() })
	return func() {
		Method(name, func() {
			Description(title + " for a visible work item. List records or retrieve a record.")
			Payload(request)
			Result(result)
			readErrors()
			Meta("mcp:annotation:readOnlyHint", "true")
			Meta("mcp:annotation:destructiveHint", "false")
			Meta("mcp:annotation:openWorldHint", "false")
			mcp.Tool(name, title+". Use an advertised action.", mcp.ToolTitle(title))
		})
	}
}

// The pinned toolkit exposes seven page fields, unlike the full SDK page model.
func toolkitPageFields() {
	Attribute("count", Int)
	Attribute("total_count", Int)
	Attribute("next_cursor", String)
	Attribute("prev_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_page_results", Boolean)
	Required("results", "count", "total_count", "next_cursor", "prev_cursor", "next_page_results", "prev_page_results")
}

var workitemActivity = workitemReadTool("workitem_activity", "Work item activity", "WorkitemActivity", "activity_id", workitemActivityRecord, true)
