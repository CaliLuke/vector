package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	. "github.com/CaliLuke/loom/dsl"
)

func memberWorkspaceSelector() {
	Attribute("workspace_slug", String, "Optional workspace selector; must match an authenticated workspace header.", func() { MaxLength(255) })
}

var memberListWorkspace = Type("MemberListWorkspace", func() {
	Description("List workspace members with case-insensitive substring filters combined with AND.")
	Attribute("action", String, func() { Enum("list_workspace") })
	for _, name := range []string{"first_name", "last_name", "email", "display_name", "role_slug", "cursor", "order_by"} {
		Attribute(name, String)
	}
	Attribute("is_active", Boolean, "Filter membership activity; omit or null to include both.", func() { Nullable() })
	Attribute("is_bot", Boolean, "Filter bot accounts; omit or null to include both.", func() { Nullable() })
	Attribute("per_page", Int, "Page size; zero or omitted uses 100.", func() { Minimum(0); Maximum(1000) })
	memberWorkspaceSelector()
	Required("action")
	Meta("openapi:additionalProperties", "false")
})
var memberListRoles = Type("MemberListRoles", func() {
	Description("List existing Vector roles. Slugs are unique within namespace, not globally.")
	Attribute("action", String, func() { Enum("list_roles") })
	Attribute("namespace", String, "Omit or empty for both namespaces.", func() { Enum("", "workspace", "project") })
	Attribute("cursor", String)
	Attribute("per_page", Int, "Page size; zero or omitted uses 20.", func() { Minimum(0); Maximum(1000) })
	memberWorkspaceSelector()
	Required("action")
	Meta("openapi:additionalProperties", "false")
})
var memberRetrieveRole = Type("MemberRetrieveRole", func() {
	Attribute("action", String, func() { Enum("retrieve_role") })
	Attribute("role_id", String, func() { Format(FormatUUID) })
	memberWorkspaceSelector()
	Required("action", "role_id")
	Meta("openapi:additionalProperties", "false")
})

// Directory entries are a typed carrier for two action-specific projections.
// Member rows emit exactly the SDK identity/membership fields; role rows emit
// only id/name/slug/namespace. Unset nullable fields remain absent on the wire.
// https://github.com/CaliLuke/loom/issues/1036 tracks nested untagged unions.
// Replace this carrier with an entry union when the minimum released generator
// supports that composition and both-protocol empty/nonempty page tests pass.
var memberDirectoryEntry = Type("MemberDirectoryEntry", func() {

	Attribute("id", String, func() { Nullable(); Format(FormatUUID) })
	for _, name := range []string{"first_name", "last_name", "email", "avatar", "avatar_url", "display_name", "role_slug", "name", "slug"} {
		Attribute(name, String, func() { Nullable() })
	}
	Attribute("role", Int, func() { Nullable() })
	Attribute("is_active", Boolean, func() { Nullable() })
	Attribute("is_bot", Boolean, func() { Nullable() })
	Attribute("namespace", String, func() { Nullable(); Enum("workspace", "project") })
	Required("id")
	Meta("openapi:additionalProperties", "false")
})

// Both directory actions have the same empty-page wire shape. A single page
// result avoids an ambiguous outer union while keeping every entry field typed.
var memberDirectoryPage = Type("MemberDirectoryPage", func() {
	membertypes.MemberPageFields()
	Attribute("results", ArrayOf(memberDirectoryEntry))
})
