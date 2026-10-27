package design

import (
	identity "github.com/CaliLuke/go-plane/apiuseridentity/design/types"
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
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

var memberListProject = Type("MemberListProject", func() {
	Description("List the members of a project the authenticated user can access.")
	Attribute("action", String, func() { Enum("list_project") })
	Attribute("project_id", String, "Project identifier.", func() { Format(FormatUUID) })
	Attribute("workspace_slug", String, "Optional workspace selector; must match the authenticated workspace.", func() { MaxLength(255) })
	Required("action", "project_id")
	Meta("openapi:additionalProperties", "false")
})

var membershipIdentity = Type("MemberIdentityWithRole", func() {
	Description("An API workspace or project member normalized to the pinned SDK's ProjectMember model.")
	Extend(memberIdentity)
	Attribute("role", Int, func() { Nullable() })
	Attribute("role_slug", String, func() { Nullable() })
	Attribute("is_active", Boolean, func() { Nullable() })
	Attribute("is_bot", Boolean, func() { Nullable() })
	Required("role", "role_slug", "is_active", "is_bot")
})

var projectMembers = Type("ProjectMembers", ArrayOf(membershipIdentity))

var memberRequest = Type("MemberRequest", OneOf(memberMe, memberListProject, memberListWorkspace, memberListRoles, memberRetrieveRole), func() { Untagged() })
var memberResult = Type("MemberResult", OneOf(memberIdentity, projectMembers, memberDirectoryPage, membertypes.RoleDefinition), func() { Untagged() })

func member() {
	Method("member", func() {
		Description("Read authorized identities, workspace and project members, and role definitions.")
		Payload(memberRequest)
		Result(memberResult)
		for _, failure := range []struct{ name, message string }{
			{"unauthorized", "Authentication is no longer valid."},
			{"forbidden", "Access is no longer allowed."},
			{"invalid_params", "The action or workspace selector is invalid."},
			{"not_found", "The requested resource is no longer available."},
			{"internal_error", "The member information could not be read."},
		} {
			Error(failure.name, func() { Remedy(func() { SafeMessage(failure.message) }) })
		}
		Meta("mcp:annotation:readOnlyHint", "true")
		Meta("mcp:annotation:destructiveHint", "false")
		Meta("mcp:annotation:openWorldHint", "false")
		mcp.Tool("member", "Read identities, members and role definitions using the selected action.", mcp.ToolTitle("Members and roles"))
	})
}
