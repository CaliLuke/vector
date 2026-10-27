package design

import (
	membertypes "github.com/CaliLuke/go-plane/invitationsmembership/design/types"
	. "github.com/CaliLuke/loom/dsl"
)

func memberDirectoryFilterAttributes() {
	for _, name := range []string{"first_name", "last_name", "email", "display_name", "role_slug", "order_by", "is_active", "is_bot"} {
		// Keep raw boolean spelling until authenticated application validation.
		Attribute(name, String)
	}
}

func memberDirectoryFilterParams() {
	for _, name := range []string{"first_name", "last_name", "email", "display_name", "role_slug", "order_by", "is_active", "is_bot"} {
		Param(name)
	}
}

func memberRoleMethods() {
	Method("ListMemberRoles", func() {
		Description("List the workspace's existing workspace and project role definitions.")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("slug", String)
			Attribute("namespace", String)
			Attribute("cursor", String)
			Attribute("perPage", String, "Page size, default 20 and maximum 1000.")
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("slug")
		})
		Result(membertypes.RolesPage)
		HTTP(func() {
			invitationsMembershipAPIKeyHTTPResponses()
			GET("/api/v1/workspaces/{slug}/roles/")
			Param("slug")
			Param("namespace")
			Param("cursor")
			Param("perPage:per_page")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(StatusOK)
		})
	})
	Method("RetrieveMemberRole", func() {
		Description("Retrieve a role definition by its workspace-scoped identifier.")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("slug", String)
			Attribute("role_id", String)
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("slug", "role_id")
		})
		Result(membertypes.RoleDefinition)
		HTTP(func() {
			invitationsMembershipAPIKeyHTTPResponses()
			GET("/api/v1/workspaces/{slug}/roles/{role_id}/")
			Param("slug")
			Param("role_id")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(StatusOK)
		})
	})
}
