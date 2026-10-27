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
	project()
	cycle()
	module()
	label()
	workitemComment()
	workitemLink()
	workitemActivity()
	milestone()
})

// Shared error names must have one consistent remedy across resource methods.
func readErrors() {
	for _, failure := range []struct{ name, message string }{
		{"unauthorized", "Authentication is no longer valid."},
		{"forbidden", "Access is no longer allowed."},
		{"invalid_params", "The action or workspace selector is invalid."},
		{"not_found", "The requested resource is no longer available."},
		{"internal_error", "The requested operation could not be completed."},
	} {
		Error(failure.name, func() { Remedy(func() { SafeMessage(failure.message) }) })
	}
}
