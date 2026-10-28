package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func projectEstimateTarget(action string, estimate, point bool) {
	Attribute("action", String, func() { Enum(action) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	projectWorkspaceSelector()
	Required("action", "project_id")
	if estimate {
		Attribute("estimate_id", String, func() { Format(FormatUUID) })
		Required("estimate_id")
	}
	if point {
		Attribute("estimate_point_id", String, func() { Format(FormatUUID) })
		Required("estimate_point_id")
	}
	Meta("openapi:additionalProperties", "false")
}
func estimateTextFields() {
	for _, key := range []string{"description", "external_source", "external_id"} {
		Attribute(key, String)
	}
}

var projectEstimateRetrieve = Type("ProjectEstimateRetrieve", func() { projectEstimateTarget("retrieve", false, false); Attribute("include_revision", Boolean) })
var projectEstimateCreate = Type("ProjectEstimateCreate", func() {
	projectEstimateTarget("create", false, false)
	Attribute("name", String)
	Required("name")
	estimateTextFields()
	Attribute("type", String, "categories, points, or time; empty uses categories.")
	Attribute("last_used", Boolean, func() { Nullable() })
	mutationExtensions(true)
})
var projectEstimateUpdate = Type("ProjectEstimateUpdate", func() {
	projectEstimateTarget("update", false, false)
	Attribute("name", String)
	estimateTextFields()
	mutationExtensions(true)
})
var projectEstimateDelete = Type("ProjectEstimateDelete", func() { projectEstimateTarget("delete", false, false); mutationExtensions(true) })
var projectEstimateLink = Type("ProjectEstimateLink", func() { projectEstimateTarget("link", true, false); mutationExtensions(true) })
var projectEstimateListPoints = Type("ProjectEstimateListPoints", func() { projectEstimateTarget("list_points", true, false) })
var projectEstimateCreatePoints = Type("ProjectEstimateCreatePoints", func() {
	projectEstimateTarget("create_points", true, false)
	Attribute("points", String, "Nonempty JSON array of point objects; key zero is valid.")
	Required("points")
	mutationExtensions(true)
})
var projectEstimateUpdatePoint = Type("ProjectEstimateUpdatePoint", func() {
	projectEstimateTarget("update_point", true, true)
	Attribute("value", String)
	Attribute("key", Int64, func() { Nullable(); Minimum(0); Maximum(2147483647) })
	estimateTextFields()
	mutationExtensions(true)
})
var projectEstimateDeletePoint = Type("ProjectEstimateDeletePoint", func() { projectEstimateTarget("delete_point", true, true); mutationExtensions(true) })

func estimateRecordFields() {
	for _, key := range []string{"id", "description", "external_source", "external_id", "created_at", "updated_at", "created_by", "updated_by", "project", "workspace"} {
		Attribute(key, String, func() { Nullable() })
		Required(key)
	}
	Meta("openapi:additionalProperties", "false")
}

var estimateRecord = Type("EstimateRecord", func() {
	estimateRecordFields()
	Attribute("name", String)
	Required("name")
	Attribute("type", String, func() { Nullable() })
	Attribute("last_used", Boolean, func() { Nullable() })
	Required("type", "last_used")
	Attribute("revision", String)
})
var estimatePointRecord = Type("EstimatePointRecord", func() {
	estimateRecordFields()
	Attribute("value", String)
	Required("value")
	Attribute("estimate", String, func() { Nullable() })
	Attribute("key", Int64, func() { Nullable() })
	Required("estimate", "key")
})
var estimatePoints = Type("EstimatePoints", ArrayOf(estimatePointRecord))
var projectEstimateReceipt = Type("ProjectEstimateMutationReceipt", func() {
	Description("Optional receipt. Revision covers the project's estimate catalogue. Exactly one result field is populated; deletions have null estimate/point and empty points. Linking uses the project mutation receipt and project revision.")
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("revision", String)
	Attribute("estimate", estimateRecord, func() { Nullable() })
	Attribute("point", estimatePointRecord, func() { Nullable() })
	Attribute("points", ArrayOf(estimatePointRecord))
	Attribute("request_id", String, func() { Format(FormatUUID) })
	Attribute("replay_expires_at", String)
	Attribute("replayed", Boolean)
	Required("project_id", "revision", "estimate", "point", "points")
	Meta("openapi:additionalProperties", "false")
})
var projectEstimateRequest = Type("ProjectEstimateRequest", OneOf(projectEstimateRetrieve, projectEstimateCreate, projectEstimateUpdate, projectEstimateDelete, projectEstimateLink, projectEstimateListPoints, projectEstimateCreatePoints, projectEstimateUpdatePoint, projectEstimateDeletePoint), func() { Untagged() })
var projectEstimateResult = Type("ProjectEstimateResult", OneOf(estimateRecord, estimatePointRecord, estimatePoints, projectDetail, projectEstimateReceipt, projectMutationReceipt), func() { Untagged() })

func projectEstimate() {
	Method("project_estimate", func() {
		Description("A project's selected estimate and its sizing points. Create the estimate, link it, then add points. Use retrieve with include_revision for a catalogue revision; link uses project revisions.")
		Payload(projectEstimateRequest)
		Result(projectEstimateResult, func() { Nullable() })
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "true")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("project_estimate", "Manage the project's estimate system and points using an advertised action.", mcp.ToolTitle("Project estimates"))
	})
}
