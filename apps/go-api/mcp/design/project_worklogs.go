package design

import . "github.com/CaliLuke/loom/dsl"

var projectWorklogSummary = Type("ProjectWorklogSummary", func() {
	Attribute("action", String, func() { Enum("worklog_summary") })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	Meta("openapi:additionalProperties", "false")
})
var projectWorklogTotal = Type("ProjectWorklogTotal", func() {
	Attribute("issue_id", String)
	Attribute("duration", Int64, "Total seconds logged for the visible work item.")
	Required("issue_id", "duration")
	Meta("openapi:additionalProperties", "false")
})
var projectWorklogTotals = Type("ProjectWorklogTotals", ArrayOf(projectWorklogTotal))
