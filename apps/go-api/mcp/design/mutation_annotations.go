package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

func mutationTool(name, title, description string) {
	Meta("mcp:annotation:readOnlyHint", "false")
	Meta("mcp:annotation:destructiveHint", "true")
	Meta("mcp:annotation:openWorldHint", "false")
	mcp.Tool(name, description, mcp.ToolTitle(title))
}
