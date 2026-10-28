package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

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

var workspaceRequest = Type("WorkspaceRequest", OneOf(workspaceRetrieve, workspaceGetFeatures, workspaceUpdateFeatures), func() { Untagged() })
var workspaceResult = Type("WorkspaceResult", OneOf(workspaceRecord, workspaceFeatureRecord, pageMutationReceipt), func() { Untagged() })

func workspace() {
	Method("workspace", func() {
		Description("The workspace this connection is bound to and its authentication mode. Personal-token connections return null for id and name.")
		Payload(workspaceRequest)
		Result(workspaceResult)
		readErrors()
		writeErrors()
		Meta("mcp:annotation:readOnlyHint", "false")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("workspace", "The connected workspace and its feature settings.", mcp.ToolTitle("Workspace settings"))
	})
}
