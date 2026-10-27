package design

import (
	identity "github.com/CaliLuke/go-plane/apiuseridentity/design/types"
	mcp "github.com/CaliLuke/loom-mcp/v2/dsl"
	. "github.com/CaliLuke/loom/dsl"
)

var memberMe = Type("MemberMe", func() {
	Description("Read the authenticated user's identity.")
	Attribute("action", String, func() { Enum("me") })
	Attribute("workspace_slug", String, "Optional workspace selector; must match the authenticated workspace.", func() { MaxLength(255) })
	Required("action")
	Meta("openapi:additionalProperties", "false")
})

var memberIdentity = Type("MemberIdentity", func() {
	Description("The existing API identity, normalized to the pinned SDK's UserLite result.")
	Reference(identity.UserLite)
	fields := []string{"id", "first_name", "last_name", "email", "avatar", "avatar_url", "display_name"}
	for _, name := range fields {
		Attribute(name, func() { Nullable() })
	}
	Required(fields...)
})

func member() {
	Method("member", func() {
		Description("Read the authenticated user's identity.")
		Payload(memberMe)
		Result(memberIdentity)
		for _, failure := range []struct{ name, message string }{
			{"unauthorized", "Authentication is no longer valid."},
			{"forbidden", "Access is no longer allowed."},
			{"invalid_params", "The action or workspace selector is invalid."},
			{"not_found", "The user is no longer available."},
			{"internal_error", "The identity could not be read."},
		} {
			Error(failure.name, func() { Remedy(func() { SafeMessage(failure.message) }) })
		}
		Meta("mcp:annotation:readOnlyHint", "true")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("member", "Members and roles. Available action: me — the authenticated user.", mcp.ToolTitle("Members and roles"))
	})
}
