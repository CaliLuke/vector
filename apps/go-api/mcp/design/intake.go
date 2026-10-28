package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func intakeTarget(action string, item bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Required("action", "project_id")
	if item {
		Attribute("workitem_id", String, "Underlying issue ID, not the intake record ID.", func() { Format(FormatUUID) })
		Required("workitem_id")
	}
	projectWorkspaceSelector()
	Meta("openapi:additionalProperties", "false")
}

var intakeList = Type("IntakeList", func() {
	intakeTarget("list", false)
	Attribute("cursor", String)
	Attribute("per_page", Int, func() { Minimum(0); Maximum(1000) })
	Attribute("include_revision", Boolean)
})
var intakeRetrieve = Type("IntakeRetrieve", func() { intakeTarget("retrieve", true); Attribute("include_revision", Boolean) })
var intakeCreate = Type("IntakeCreate", func() {
	intakeTarget("create", false)
	Attribute("name", String, func() { MinLength(1); MaxLength(255) })
	Attribute("description_html", String)
	Attribute("priority", String, func() { Enum("", "urgent", "high", "medium", "low", "none") })
	Required("name")
	mutationExtensions(false)
})
var intakeUpdate = Type("IntakeUpdate", func() {
	intakeTarget("update", true)
	Attribute("status", Int, "-2 pending, -1 declined, 0 snoozed, 1 accepted, 2 duplicate.", func() { Enum(-2, -1, 0, 1, 2) })
	Attribute("snoozed_till", String, "Required when status is 0.")
	Attribute("duplicate_to", String, "Visible work-item UUID; required when status is 2.")
	Attribute("source", String, func() { MaxLength(255) })
	Attribute("source_email", String)
	mutationExtensions(true)
})
var intakeDelete = Type("IntakeDelete", func() { intakeTarget("delete", true); mutationExtensions(true) })
var intakeRecord = Type("IntakeRecord", func() {
	nullableStrings("id", "inbox", "created_at", "updated_at", "deleted_at", "snoozed_till", "source", "source_email", "external_source", "external_id", "created_by", "updated_by", "project", "workspace", "intake", "issue", "duplicate_to")
	Attribute("issue_detail", workitemReadRecord, func() { Nullable() })
	// Nullable enum carrier until https://github.com/CaliLuke/loom/issues/1037
	// is released. The native intake validator enforces the five statuses.
	Attribute("status", Int, func() { Nullable() })
	Attribute("extra", Any, func() { Nullable() })
	Attribute("revision", String)
	Required("issue_detail", "status", "extra")
	Meta("openapi:additionalProperties", "false")
})
var intakePage = Type("IntakePage", func() {
	toolkitPageFields()
	Attribute("results", ArrayOf(intakeRecord))
	Meta("openapi:additionalProperties", "false")
})
var intakeReceipt = Type("IntakeMutationReceipt", func() {
	Attribute("workitem_id", String)
	Attribute("revision", String)
	Attribute("result", intakeRecord, func() { Nullable() })
	Attribute("request_id", String)
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("workitem_id", "revision", "result")
	Meta("openapi:additionalProperties", "false")
})
var intakeRequest = Type("IntakeRequest", OneOf(intakeList, intakeRetrieve, intakeCreate, intakeUpdate, intakeDelete), func() { Untagged() })
var intakeResult = Type("IntakeResult", OneOf(intakePage, intakeRecord, intakeReceipt), func() { Untagged() })

func intake() {
	Method("intake", func() {
		Description("Project intake queue. workitem_id is the underlying issue ID. Members submit items; project administrators triage them. Snoozing requires snoozed_till and duplicate status requires duplicate_to. Deleting an accepted submission removes its intake association and keeps the work item. Optional receipts and revision guards support safe retries.")
		Payload(intakeRequest)
		Result(intakeResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("intake", "List, retrieve, create, triage and delete intake submissions.", mcp.ToolTitle("Intake queue"))
	})
}
