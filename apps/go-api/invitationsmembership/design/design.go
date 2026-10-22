// Package design contains the reverse-imported source contracts for the
// invitation and workspace membership route family. Runtime behavior remains
// source-fixture driven.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/invitationsmembership/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var ImportedPatchedPlaneAPISerializersInviteWorkspaceinviteserializerRequestRequest = Type("PatchedPlaneApiSerializersInviteWorkspaceinviteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for workspace invites.")
	Attribute("email", String, func() {
		MinLength(1)
		MaxLength(255)
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
})

var ImportedPatchedPlaneAppSerializersWorkspaceWorkspacememberadminserializerRequestRequest = Type("PatchedPlaneAppSerializersWorkspaceWorkspacememberadminserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("company_role", String, func() {
		Nullable()
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("default_props", Any, func() {
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("explored_features", Any, func() {
	})
	Attribute("getting_started_checklist", Any, func() {
	})
	Attribute("is_active", Boolean)
	Attribute("issue_props", Any, func() {
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("tips", Any, func() {
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
	})
})

var ImportedPatchedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerRequestRequest = Type("PatchedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("accepted", Boolean)
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
})

var ImportedPlaneAPISerializersInviteWorkspaceinviteserializerRequestRequest = Type("PlaneApiSerializersInviteWorkspaceinviteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for workspace invites.")
	Attribute("email", String, func() {
		MinLength(1)
		MaxLength(255)
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Required("email")
})

var ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse = Type("PlaneApiSerializersInviteWorkspaceinviteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for workspace invites.")
	Attribute("accepted", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("email", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("responded_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Required("email")
	Required("created_at", "email", "id", "updated_at")
})

var ImportedPlaneAPISerializersMemberWorkspacememberliteapiserializerResponse = Type("PlaneApiSerializersMemberWorkspacememberliteapiserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Minimal WorkspaceMember representation for paginated member pickers/directories.")
	Attribute("avatar", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("avatar_url", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("email", String, func() {
		Format(FormatEmail)
		Meta("openapi:readOnly", "true")
	})
	Attribute("first_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_active", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_bot", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("last_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Required("display_name", "email", "first_name", "id", "last_name")
})

var ImportedPlaneAPISerializersMemberWorkspacememberuserliteserializerResponse = Type("PlaneApiSerializersMemberWorkspacememberuserliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("The existing workspace roster adds a role to the API user projection.")
	Attribute("avatar", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("avatar_url", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("email", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("first_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("last_name", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", Int, func() {
		Meta("openapi:format", "")
		Meta("openapi:readOnly", "true")
	})
	Required("avatar", "display_name", "first_name", "id", "last_name")
})

var ImportedPlaneAppSerializersProjectProjectliteserializerResponse = Type("PlaneAppSerializersProjectProjectliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("cover_image", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("cover_image_url", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("Project Description")
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("identifier", String, func() {
		Title("Project Identifier")
		Meta("openapi:readOnly", "true")
	})
	Attribute("logo_props", Any, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Project Name")
		Meta("openapi:readOnly", "true")
	})
	Required("description", "id", "identifier", "name")
})

var ImportedPlaneAppSerializersProjectProjectmemberinvitepublicserializerResponse = Type("PlaneAppSerializersProjectProjectmemberinvitepublicserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Safe read-only serializer for the public project invite GET endpoint.\n\nIntentionally excludes ``email`` and ``token`` so that an unauthenticated\ncaller cannot retrieve the invitee's email address or the acceptance token\n(GHSA-2r58-hgv7-635q).")
	Attribute("accepted", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("message", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", ImportedPlaneAppSerializersProjectProjectliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("responded_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Required("id")
})

var ImportedPlaneAppSerializersProjectProjectmemberinviteserializerRequestRequest = Type("PlaneAppSerializersProjectProjectmemberinviteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("accepted", Boolean)
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("email", String, func() {
		MinLength(1)
		MaxLength(255)
	})
	Attribute("message", String, func() {
		Nullable()
		MinLength(1)
	})
	Attribute("responded_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("token", String, func() {
		MinLength(1)
		MaxLength(255)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Required("email", "token")
})

var ImportedPlaneAppSerializersProjectProjectmemberinviteserializerResponse = Type("PlaneAppSerializersProjectProjectmemberinviteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("accepted", Boolean)
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("email", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("message", String, func() {
		Nullable()
	})
	Attribute("project", ImportedPlaneAppSerializersProjectProjectliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("responded_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("token", String, func() {
		MaxLength(255)
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("workspace", ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Required("email", "token")
	Required("created_at", "email", "id", "token", "updated_at")
})

var ImportedPlaneAppSerializersUserUseradminliteserializerResponse = Type("PlaneAppSerializersUserUseradminliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("avatar", String, func() {})
	Attribute("avatar_url", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
		MaxLength(255)
	})
	Attribute("email", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("first_name", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_bot", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("last_login_medium", String, func() {
		MaxLength(20)
	})
	Attribute("last_name", String, func() {
		MaxLength(255)
	})
	Required("display_name", "first_name", "id", "last_name")
	Required("avatar")
})

var ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse = Type("PlaneAppSerializersWorkspaceWorkspaceliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("logo_url", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Workspace Name")
		Meta("openapi:readOnly", "true")
	})
	Attribute("slug", String, func() {
		Meta("openapi:readOnly", "true")
		Pattern("^[-a-zA-Z0-9_]+$")
	})
	Required("id", "name", "slug")
})

var ImportedPlaneAppSerializersWorkspaceWorkspacememberadminserializerRequestRequest = Type("PlaneAppSerializersWorkspaceWorkspacememberadminserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("company_role", String, func() {
		Nullable()
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("default_props", Any, func() {
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("explored_features", Any, func() {
	})
	Attribute("getting_started_checklist", Any, func() {
	})
	Attribute("is_active", Boolean)
	Attribute("issue_props", Any, func() {
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("tips", Any, func() {
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
	})
	Required("workspace")
})

var ImportedPlaneAppSerializersWorkspaceWorkspacememberadminserializerResponse = Type("PlaneAppSerializersWorkspaceWorkspacememberadminserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("company_role", String, func() {
		Nullable()
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("default_props", Any, func() {
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("explored_features", Any, func() {
	})
	Attribute("getting_started_checklist", Any, func() {
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_active", Boolean)
	Attribute("issue_props", Any, func() {
	})
	Attribute("member", ImportedPlaneAppSerializersUserUseradminliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("tips", Any, func() {
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
	})
	Required("workspace")
	Required("created_at", "id", "updated_at", "workspace")
})

var ImportedPlaneAppSerializersWorkspaceWorkspacememberinvitepublicserializerResponse = Type("PlaneAppSerializersWorkspaceWorkspacememberinvitepublicserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Safe read-only serializer for the public workspace invite GET endpoint.\n\nIntentionally excludes ``token`` and ``invite_link`` so that an\nunauthenticated caller cannot retrieve the acceptance token and use it to\nhijack an invitation (GHSA-86mg-259g-pwgg / GHSA-gf48-p6jp-cwc4).")
	Attribute("accepted", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("email", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("message", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("responded_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("workspace", ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Required("created_at", "email", "id", "updated_at")
})

var ImportedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerRequestRequest = Type("PlaneAppSerializersWorkspaceWorkspacememberinviteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("accepted", Boolean)
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
})

var ImportedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse = Type("PlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("accepted", Boolean)
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("email", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("invite_link", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("message", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("responded_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("token", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("workspace", ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Required("created_at", "email", "id", "invite_link", "token", "updated_at")
})

var ImportedRestFrameworkSerializersInvitationresponserequestRequestRequest = Type("RestFrameworkSerializersInvitationresponserequestRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("accepted", Boolean, func() {
		Default(false)
	})
	Attribute("token", String, func() {
		MinLength(1)
	})
	Required("token")
})

var ImportedRestFrameworkSerializersMessageresponseResponse = Type("RestFrameworkSerializersMessageresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("message", String, func() {})
	Required("message")
})

var ImportedRestFrameworkSerializersPaginatedworkspacememberliteResponse = Type("RestFrameworkSerializersPaginatedworkspacememberliteResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("extra_stats", String, func() {
		Nullable()
	})
	Attribute("grouped_by", String, func() {
		Nullable()
	})
	Attribute("next_cursor", String, func() {})
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String, func() {})
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersMemberWorkspacememberliteapiserializerResponse))
	Attribute("sub_grouped_by", String, func() {
		Nullable()
	})
	Attribute("total_count", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_pages", Int, func() {
		Meta("openapi:format", "")
	})
	Attribute("total_results", Int, func() {
		Meta("openapi:format", "")
	})
	Required("count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results")
})

var ImportedRoleC37Enum = Type("RoleC37Enum", Int, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `20` - Admin\n* `15` - Member\n* `5` - Guest")
	Meta("openapi:format", "")
	Enum(20, 15, 5)
})

var ImportedAPIKeyAuthSecurity = APIKeySecurity("ApiKeyAuth", func() {
	Description("Plane API token authentication.")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var _ = API("Invitations and Membership API", func() {
	Title("Invitations and Membership API")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:invitations")
	Meta("openapi:tag:workspaces")
	Meta("openapi:tag:members")
	Meta("openapi:tag:projects")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var invitationsMembershipAPIKeyError = Type("InvitationsMembershipAPIKeyError", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("code", Any)
	Attribute("email", Any)
	Attribute("role", Any)
	Attribute("non_field_errors", Any)
	Attribute("workspace_users", Any)
	Attribute("is_active", Any)
	Attribute("created_by", Any)
	Attribute("updated_by", Any)
	Attribute("deleted_at", Any)
	Attribute("workspace", Any)
	Attribute("company_role", Any)
	Attribute("accepted", Any)
	Attribute("view_props", Any)
	Attribute("default_props", Any)
	Attribute("issue_props", Any)
	Attribute("getting_started_checklist", Any)
	Attribute("tips", Any)
	Attribute("explored_features", Any)
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Required("name")
})

var invitationsMembershipHTTPError = Type("InvitationsMembershipHTTPError", func() {
	Attribute("code", Any)
	Attribute("email", Any)
	Attribute("role", Any)
	Attribute("non_field_errors", Any)
	Attribute("workspace_users", Any)
	Attribute("is_active", Any)
	Attribute("created_by", Any)
	Attribute("updated_by", Any)
	Attribute("deleted_at", Any)
	Attribute("workspace", Any)
	Attribute("company_role", Any)
	Attribute("accepted", Any)
	Attribute("view_props", Any)
	Attribute("default_props", Any)
	Attribute("issue_props", Any)
	Attribute("getting_started_checklist", Any)
	Attribute("tips", Any)
	Attribute("explored_features", Any)
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
})

func invitationsMembershipAPIKeyErrors() {
	Error("status_400", invitationsMembershipAPIKeyError)
	Error("status_401", invitationsMembershipAPIKeyError)
	Error("status_403", invitationsMembershipAPIKeyError)
	Error("status_404", invitationsMembershipAPIKeyError)
	Error("status_409", invitationsMembershipAPIKeyError)
	Error("status_415", invitationsMembershipAPIKeyError)
	Error("status_429", invitationsMembershipAPIKeyError, func() { Temporary() })
	Error("status_500", invitationsMembershipAPIKeyError, func() { Fault() })
}

func invitationsMembershipAPIKeyHTTPResponses() {
	Response("status_400", 400, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(invitationsMembershipHTTPError)
	})
	Response("status_401", 401, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(invitationsMembershipHTTPError)
	})
	Response("status_403", 403, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(invitationsMembershipHTTPError)
	})
	Response("status_404", 404, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(invitationsMembershipHTTPError)
	})
	Response("status_409", 409, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(invitationsMembershipHTTPError)
	})
	Response("status_415", 415, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(invitationsMembershipHTTPError)
	})
	Response("status_429", 429, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(invitationsMembershipHTTPError)
	})
	Response("status_500", 500, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(invitationsMembershipHTTPError)
	})
}

// The source's custom create actions consume batches rather than model
// serializer fields. Raw decoding preserves the source's validation precedence.
var invitationEntryRequest = Type("InvitationEntryRequest", func() {
	Attribute("email", String, func() { Format(FormatEmail) })
	Attribute("role", Any, func() {
		Description("Invitation role. Project invitations accept canonical choices 5, 15, or 20; workspace invitations preserve Django integer coercion and the caller role cap.")
	})
	Required("email")
})
var invitationBatchRequest = Type("InvitationBatchRequest", func() {
	Attribute("emails", ArrayOf(invitationEntryRequest), func() { MinLength(1) })
	Required("emails")
})
var workspaceInvitationSelectionRequest = Type("WorkspaceInvitationSelectionRequest", func() {
	Attribute("invitations", ArrayOf(String, func() { Format(FormatUUID) }))
})
var projectMembershipSelectionRequest = Type("ProjectMembershipSelectionRequest", func() {
	Attribute("project_ids", ArrayOf(String, func() { Format(FormatUUID) }))
})

var _ = Service("InvitationsMembership", func() {
	invitationsMembershipAPIKeyErrors()
	Method("GetAPIUsersMeWorkspacesInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_users_me_workspaces_invitations")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/users/me/workspaces/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.UserWorkspaceInvitationsViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		})
		Result(ArrayOf(ImportedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse))
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/users/me/workspaces/invitations/")
			Param("search")
			Cookie("planeSessionCookieCredential:session-id")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIUsersMeWorkspacesInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_users_me_workspaces_invitations")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/users/me/workspaces/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.UserWorkspaceInvitationsViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			POST("/api/users/me/workspaces/invitations/")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(workspaceInvitationSelectionRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIUsersMeWorkspacesBySlugProjectsInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_users_me_workspaces_by_slug_projects_invitations")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/users/me/workspaces/{slug}/projects/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.invite.UserProjectInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("slug")
		})
		Result(ArrayOf(ImportedPlaneAppSerializersProjectProjectmemberinviteserializerResponse))
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/users/me/workspaces/{slug}/projects/invitations/")
			Param("search")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIUsersMeWorkspacesBySlugProjectsInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_users_me_workspaces_by_slug_projects_invitations")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/users/me/workspaces/{slug}/projects/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.invite.UserProjectInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("slug")
		})
		Result(ImportedRestFrameworkSerializersMessageresponseResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			POST("/api/users/me/workspaces/{slug}/projects/invitations/")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(projectMembershipSelectionRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("List all workspace invites for a workspace")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_invitations")
		Meta("openapi:summary", "List workspace invites")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.workspace.WorkspaceOwnerPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("slug")
		})
		Result(ArrayOf(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse))
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/v1/workspaces/{slug}/invitations/")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Workspace invites")
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Create a workspace invite")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_invitations")
		Meta("openapi:summary", "Create workspace invite")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.workspace.WorkspaceOwnerPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("slug")
		})
		Result(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			POST("/api/v1/workspaces/{slug}/invitations/")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				Description("Workspace invite")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Delete a workspace invite")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_invitations_by_pk")
		Meta("openapi:summary", "Delete workspace invite")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.workspace.WorkspaceOwnerPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Description("Workspace invite ID")
				Meta("openapi:format", "uuid")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			DELETE("/api/v1/workspaces/{slug}/invitations/{pk}/")
			Param("pk")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Workspace invite deleted")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Get a workspace invite by ID")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_invitations_by_pk")
		Meta("openapi:summary", "Get workspace invite")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.workspace.WorkspaceOwnerPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Description("Workspace invite ID")
				Meta("openapi:format", "uuid")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "slug")
		})
		Result(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/v1/workspaces/{slug}/invitations/{pk}/")
			Param("pk")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Workspace invite")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse)
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Update a workspace invite")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_invitations_by_pk")
		Meta("openapi:summary", "Update workspace invite")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.workspace.WorkspaceOwnerPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Description("Workspace invite ID")
				Meta("openapi:format", "uuid")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "slug")
		})
		Result(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			PATCH("/api/v1/workspaces/{slug}/invitations/{pk}/")
			Param("pk")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersInviteWorkspaceinviteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				Description("Workspace invite")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse)
			})
		})
	})
	Method("PutAPIV1WorkspacesBySlugInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Endpoint for creating, listing and deleting workspace invites.")
		Meta("openapi:operationId", "put_api_v1_workspaces_by_slug_invitations_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/v1/workspaces/{slug}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.workspace.WorkspaceOwnerPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Description("A UUID string identifying this Workspace Member Invite.")
				Meta("openapi:format", "uuid")
			})
			Attribute("slug", String)
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "slug")
		})
		Result(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			PUT("/api/v1/workspaces/{slug}/invitations/{pk}/")
			Param("pk")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersInviteWorkspaceinviteserializerResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugMembersLite", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Retrieve a paginated, lightweight list of workspace members for pickers and directories.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_members_lite")
		Meta("openapi:summary", "List workspace members (lite)")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/members-lite/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.workspace.WorkSpaceAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.WorkspaceMemberLiteAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", Int, func() {
				Description("Number of results per page (default: 20, max: 100)")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("slug")
		})
		Result(ImportedRestFrameworkSerializersPaginatedworkspacememberliteResponse, func() {
			Meta("openapi:allOf:reference", "true")
			Example("PaginatedWorkspaceMembers(Lite)", func() {
				Meta("openapi:example:summary", "Paginated Workspace Members (Lite)")
				Value(Val{"count": 20, "extra_stats": nil, "grouped_by": "state", "next_cursor": "20:1:0", "next_page_results": true, "prev_cursor": "20:0:0", "prev_page_results": false, "results": []any{Val{"created_at": "2024-01-15T12:00:00Z", "id": "550e8400-e29b-41d4-a716-446655440000", "name": "Sample Item"}}, "sub_grouped_by": "priority", "total_count": 150, "total_pages": 8, "total_results": 150})
			})
		})
		HTTP(func() {
			Meta("openapi:tag:workspaces")
			GET("/api/v1/workspaces/{slug}/members-lite/")
			Param("cursor")
			Param("perPage:per_page")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of workspace members with minimal fields")
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedworkspacememberliteResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugMembers", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Retrieve all users who are members of the specified workspace.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_members")
		Meta("openapi:summary", "List workspace members")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/members/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.workspace.WorkSpaceAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.WorkspaceMemberAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("slug")
		})
		Result(ArrayOf(ImportedPlaneAPISerializersMemberWorkspacememberuserliteserializerResponse))
		HTTP(func() {
			Meta("openapi:tag:members")
			GET("/api/v1/workspaces/{slug}/members/")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("List of workspace members with their roles")
				ContentType("application/json")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Endpoint for creating, listing and  deleting workspaces")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_invitations")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.workspace.WorkSpaceAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("slug")
		})
		Result(ArrayOf(ImportedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse))
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/workspaces/{slug}/invitations/")
			Param("search")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Endpoint for creating, listing and  deleting workspaces")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_invitations")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.workspace.WorkSpaceAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("slug")
		})
		Result(ImportedRestFrameworkSerializersMessageresponseResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			POST("/api/workspaces/{slug}/invitations/")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(invitationBatchRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Endpoint for creating, listing and  deleting workspaces")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_invitations_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.workspace.WorkSpaceAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			DELETE("/api/workspaces/{slug}/invitations/{pk}/")
			Param("pk")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Endpoint for creating, listing and  deleting workspaces")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_invitations_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.workspace.WorkSpaceAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "slug")
		})
		Result(ImportedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/workspaces/{slug}/invitations/{pk}/")
			Param("pk")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse)
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("Endpoint for creating, listing and  deleting workspaces")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_invitations_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.workspace.WorkSpaceAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.WorkspaceInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "slug")
		})
		Result(ImportedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			PATCH("/api/workspaces/{slug}/invitations/{pk}/")
			Param("pk")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersWorkspaceWorkspacememberinviteserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugInvitationsByPkJoin", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_invitations_by_pk_join")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/invitations/{pk}/join/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.AllowAny\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.WorkspaceJoinEndpoint\"")
		Security()
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "slug")
		})
		Result(ImportedPlaneAppSerializersWorkspaceWorkspacememberinvitepublicserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/workspaces/{slug}/invitations/{pk}/join/")
			Param("pk")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersWorkspaceWorkspacememberinvitepublicserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugInvitationsByPkJoin", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_invitations_by_pk_join")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/invitations/{pk}/join/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.AllowAny\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.invite.WorkspaceJoinEndpoint\"")
		Security()
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "slug")
		})
		Result(ImportedRestFrameworkSerializersMessageresponseResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			POST("/api/workspaces/{slug}/invitations/{pk}/join/")
			Param("pk")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersInvitationresponserequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugMembersLeave", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_members_leave")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/members/leave/\"")
		Meta("openapi:extension:x-plane-handler", "\"leave\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.member.WorkSpaceMemberViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("slug")
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:members")
			POST("/api/workspaces/{slug}/members/leave/")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersWorkspaceWorkspacememberadminserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugMembersByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_members_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.member.WorkSpaceMemberViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:members")
			DELETE("/api/workspaces/{slug}/members/{pk}/")
			Param("pk")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugMembersByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_members_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.workspace.member.WorkSpaceMemberViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "slug")
		})
		Result(ImportedPlaneAppSerializersWorkspaceWorkspacememberadminserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:members")
			PATCH("/api/workspaces/{slug}/members/{pk}/")
			Param("pk")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersWorkspaceWorkspacememberadminserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersWorkspaceWorkspacememberadminserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_invitations")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"list\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.invite.ProjectInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("search", String, func() {
				Description("A search term.")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("project_id", "slug")
		})
		Result(ArrayOf(ImportedPlaneAppSerializersProjectProjectmemberinviteserializerResponse))
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/workspaces/{slug}/projects/{project_id}/invitations/")
			Param("project_id")
			Param("search")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(200, func() {
				ContentType("application/json")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDInvitations", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_invitations")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/invitations/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.invite.ProjectInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersMessageresponseResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			POST("/api/workspaces/{slug}/projects/{project_id}/invitations/")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(invitationBatchRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_invitations_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.invite.ProjectInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/invitations/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDInvitationsByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_invitations_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/invitations/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"retrieve\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.invite.ProjectInvitationsViewset\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersProjectProjectmemberinviteserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:invitations")
			GET("/api/workspaces/{slug}/projects/{project_id}/invitations/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersProjectProjectmemberinviteserializerResponse)
			})
		})
	})
	Method("GetAPIWorkspacesBySlugProjectsByProjectIDJoinByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_join_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/workspaces/{slug}/projects/{project_id}/join/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.AllowAny\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.invite.ProjectJoinEndpoint\"")
		Security()
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersProjectProjectmemberinvitepublicserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:projects")
			GET("/api/workspaces/{slug}/projects/{project_id}/join/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersProjectProjectmemberinvitepublicserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDJoinByPk", func() {
		HTTP(func() { invitationsMembershipAPIKeyHTTPResponses() })
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_join_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/join/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.AllowAny\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.invite.ProjectJoinEndpoint\"")
		Security()
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersMessageresponseResponse)
		HTTP(func() {
			Meta("openapi:tag:projects")
			POST("/api/workspaces/{slug}/projects/{project_id}/join/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedRestFrameworkSerializersInvitationresponserequestRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersMessageresponseResponse)
			})
		})
	})
})
