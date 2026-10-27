// Package design owns Vector's remote MCP projection.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/mcp/design --output ..
package design

import (
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

var _ = API("Vector MCP", func() {
	Title("Vector MCP")
	Version("1.0.0")
})

// Add only implemented actions after their authentication, authorization and
// reliability prerequisites pass.
var _ = Service("VectorMCP", func() {
	Description("Remote access to Vector.")
	mcp.MCP("vector", "1.0.0")
	member()
})
