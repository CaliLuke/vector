package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func workLogTarget(action string, detail bool) {
	Attribute("action", String, func() { Enum(action) })
	for _, key := range []string{"project_id", "workitem_id"} {
		Attribute(key, String, func() { Format(FormatUUID) })
		Required(key)
	}
	projectWorkspaceSelector()
	Required("action")
	if detail {
		Attribute("work_log_id", String, func() { Format(FormatUUID) })
		Required("work_log_id")
	}
	Meta("openapi:additionalProperties", "false")
}

var workLogList = Type("WorkLogList", func() {
	workLogTarget("list", false)
	Attribute("cursor", String)
	Attribute("per_page", Int, func() { Minimum(0); Maximum(1000) })
	Attribute("include_revision", Boolean)
})
var workLogCreate = Type("WorkLogCreate", func() {
	workLogTarget("create", false)
	Attribute("duration", Int64, "Minutes.", func() { Minimum(1); Maximum(2147483647) })
	Attribute("description", String)
	Required("duration")
	mutationExtensions(false)
})
var workLogUpdate = Type("WorkLogUpdate", func() {
	workLogTarget("update", true)
	Attribute("duration", Int64, "Minutes; zero or omission leaves unchanged.", func() { Minimum(0); Maximum(2147483647) })
	Attribute("description", String, "Empty or omitted leaves unchanged.")
	mutationExtensions(true)
})
var workLogDelete = Type("WorkLogDelete", func() { workLogTarget("delete", true); mutationExtensions(true) })
var workLogRecord = Type("WorkLogRecord", func() {
	for _, key := range []string{"id", "created_at", "updated_at", "description", "created_by", "updated_by", "project_id", "workspace_id", "logged_by"} {
		Attribute(key, String, func() { Nullable() })
		Required(key)
	}
	Attribute("duration", Int64, func() { Nullable() })
	Required("duration")
	Attribute("revision", String)
	Meta("openapi:additionalProperties", "false")
})
var workLogRecords = Type("WorkLogRecords", ArrayOf(workLogRecord))
var workLogReceipt = Type("WorkLogMutationReceipt", func() {
	Attribute("work_log_id", String)
	Attribute("revision", String)
	Attribute("result", workLogRecord, func() { Nullable() })
	Attribute("request_id", String)
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("work_log_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var workLogRequest = Type("WorkLogRequest", OneOf(workLogList, workLogCreate, workLogUpdate, workLogDelete), func() { Untagged() })
var workLogResult = Type("WorkLogResult", OneOf(workLogRecords, workLogRecord, workLogReceipt), func() { Untagged() })

func workLog() {
	Method("work_log", func() {
		Description("Time logged against a work item, in minutes. List returns a flat array with cursor/per-page selection. The author or a project administrator can change an entry. include_revision on list returns current revisions for guarded mutations.")
		Payload(workLogRequest)
		Result(workLogResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("work_log", "List, create, update and delete time entries.", mcp.ToolTitle("Work logs"))
	})
}
