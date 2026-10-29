package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

var workspaceList = Type("WorkspaceList", func() {
	Attribute("action", String, func() { Enum("list") })
	Required("action")
	Meta("openapi:additionalProperties", "false")
})
var accessibleWorkspace = Type("AccessibleWorkspace", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("slug", String)
	Required("id", "name", "slug")
	Meta("openapi:additionalProperties", "false")
})
var workspaceListResult = Type("WorkspaceListResult", func() {
	Attribute("results", ArrayOf(accessibleWorkspace))
	Required("results")
	Meta("openapi:additionalProperties", "false")
})

var workspaceRetrieve = Type("WorkspaceRetrieve", func() {
	Attribute("action", String, func() { Enum("retrieve") })
	projectWorkspaceSelector()
	Required("action")
	Meta("openapi:additionalProperties", "false")
})
var workspaceRecord = Type("WorkspaceRecord", func() {
	Attribute("slug", String)
	Attribute("id", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("name", String, func() { Nullable() })
	Attribute("connected_via", String, func() { Enum("oauth", "api_key_header") })
	Required("slug", "id", "name", "connected_via")
	Meta("openapi:additionalProperties", "false")
})

var workspaceRequest = Type("WorkspaceRequest", OneOf(workspaceList, workspaceRetrieve, workspaceGetFeatures, workspaceUpdateFeatures), func() { Untagged() })
var workspaceResult = Type("WorkspaceResult", OneOf(workspaceListResult, workspaceRecord, workspaceFeatureRecord, pageMutationReceipt), func() { Untagged() })

func workspace() {
	Method("workspace", func() {
		Description("List accessible workspaces without a workspace selector, then select workspace_slug per scoped call. Retrieve returns the selected workspace and authentication mode. Personal-token retrieval preserves null id and name for compatibility.")
		Payload(workspaceRequest)
		Result(workspaceResult)
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("workspace", "List accessible workspaces, retrieve a selected workspace, and manage its feature settings.", mcp.ToolTitle("Workspace settings"))
	})
}
