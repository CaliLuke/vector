// Package design is reverse-generated from the selected operations in
// docs/go-backend-replacement/openapi/full-server.openapi.json. The source
// projection was produced with Loom v1.10.0-alpha.8; source-backed adjustments
// below preserve runtime details absent from the published schema.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/projectmembershiplifecycle/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var ImportedPatchedPlaneAPISerializersMemberProjectmemberserializerRequestRequest = Type("PatchedPlaneApiSerializersMemberProjectmemberserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for project members.")
	Attribute("member", String, func() {
		Format(FormatUUID)
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
})

var ImportedPatchedPlaneAppSerializersProjectProjectmemberadminserializerRequestRequest = Type("PatchedPlaneAppSerializersProjectProjectmemberadminserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("comment", String, func() {
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
	Attribute("is_active", Boolean)
	Attribute("preferences", Any, func() {
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("sort_order", Float64)
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
})

var ImportedPlaneAPISerializersMemberProjectmemberliteapiserializerResponse = Type("PlaneApiSerializersMemberProjectmemberliteapiserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Minimal ProjectMember representation for paginated member pickers/directories.")
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
	Attribute("role", Int, func() {
		Minimum(0)
		Maximum(32767)
		Meta("openapi:readOnly", "true")
	})
})

var ImportedPlaneAPISerializersMemberProjectmemberserializerRequestRequest = Type("PlaneApiSerializersMemberProjectmemberserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for project members.")
	Attribute("member", String, func() {
		Format(FormatUUID)
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Required("member")
})

var ImportedPlaneAPISerializersMemberProjectmemberserializerResponse = Type("PlaneApiSerializersMemberProjectmemberserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for project members.")
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("member", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("role", Int, func() {
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Required("member")
})

var ImportedPlaneAPISerializersUserUserliteserializerResponse = Type("PlaneApiSerializersUserUserliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Lightweight user serializer for minimal data transfer.\n\nProvides essential user information including names, avatar, and contact details\noptimized for member lists, assignee displays, and user references.")
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
})

var ImportedPlaneAppSerializersProjectProjectmemberadminserializerRequestRequest = Type("PlaneAppSerializersProjectProjectmemberadminserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("comment", String, func() {
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
	Attribute("is_active", Boolean)
	Attribute("preferences", Any, func() {
	})
	Attribute("role", ImportedRoleC37Enum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("sort_order", Float64)
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
})

var ProjectMembershipLegacyPatchResponse = Type("ProjectMembershipLegacyPatchResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("comment", String, func() {
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
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_active", Boolean)
	Attribute("member", ProjectMembershipLegacyUserLite, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("preferences", Any, func() {
	})
	Attribute("project", ImportedPlaneAppSerializersProjectProjectliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
	Attribute("role", Int, func() {
		Meta("openapi:format", "")
		Minimum(0)
		Maximum(32767)
	})
	Attribute("sort_order", Float64)
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
	Attribute("workspace", ImportedPlaneAppSerializersWorkspaceWorkspaceliteserializerResponse, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:readOnly", "true")
	})
})

var ProjectMembershipLegacyUserLite = Type("ProjectMembershipLegacyUserLite", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("avatar", String)
	Attribute("avatar_url", String, func() {
		Nullable()
		Meta("openapi:readOnly", "true")
	})
	Attribute("display_name", String, func() {
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
	Attribute("last_name", String, func() {
		MaxLength(255)
	})
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
})

var ImportedRestFrameworkSerializersPaginatedprojectmemberliteResponse = Type("RestFrameworkSerializersPaginatedprojectmemberliteResponse", func() {
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
	Attribute("next_cursor", String)
	Attribute("next_page_results", Boolean)
	Attribute("prev_cursor", String)
	Attribute("prev_page_results", Boolean)
	Attribute("results", ArrayOf(ImportedPlaneAPISerializersMemberProjectmemberliteapiserializerResponse))
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

var ImportedPlaneSession = SessionAuth("PlaneSession", func() {
	CookieTransport(ImportedPlaneSessionCookieSecurity, "", func() { CookieName("session-id") })
})

var _ = API("Project Membership Lifecycle", func() {
	Title("Plane Internal Django Migration API")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:Assets")
	Meta("openapi:tag:Assets:desc", "**File Upload & Presigned URLs**\n\nGenerate presigned URLs for direct file uploads to cloud storage. Handle user avatars, cover images, and generic project assets with secure upload workflows.\n\n*Key Features:*\n- Generate presigned URLs for S3 uploads\n- Support for user avatars and cover images\n- Generic asset upload for projects\n- File validation and size limits\n\n*Use Cases:* User profile images, project file uploads, secure direct-to-cloud uploads.")
	Meta("openapi:tag:Cycles")
	Meta("openapi:tag:Cycles:desc", "**Sprint & Development Cycles**\n\nCreate and manage development cycles (sprints) to organize work into time-boxed iterations. Track progress, assign work items, and monitor team velocity.\n\n*Key Features:*\n- Create and configure development cycles\n- Assign work items to cycles\n- Track cycle progress and completion\n- Generate cycle analytics and reports\n\n*Use Cases:* Sprint planning, iterative development, progress tracking, team velocity.")
	Meta("openapi:tag:Intake")
	Meta("openapi:tag:Intake:desc", "**Work Item Intake Queue**\n\nManage incoming work items through a dedicated intake queue for triage and review. Submit, update, and process work items before they enter the main project workflow.\n\n*Key Features:*\n- Submit work items to intake queue\n- Review and triage incoming work items\n- Update intake work item status and properties\n- Accept, reject, or modify work items before approval\n\n*Use Cases:* Work item triage, external submissions, quality review, approval workflows.")
	Meta("openapi:tag:Labels")
	Meta("openapi:tag:Labels:desc", "**Labels & Tags**\n\nCreate and manage labels to categorize and organize work items. Use color-coded labels for easy identification, filtering, and project organization.\n\n*Key Features:*\n- Create custom labels with colors and descriptions\n- Apply labels to work items for categorization\n- Filter and search by labels\n- Organize labels across projects\n\n*Use Cases:* Priority marking, feature categorization, bug classification, team organization.")
	Meta("openapi:tag:Members")
	Meta("openapi:tag:Members:desc", "**Team Member Management**\n\nManage team members, roles, and permissions within projects and workspaces. Control access levels and track member participation.\n\n*Key Features:*\n- Invite and manage team members\n- Assign roles and permissions\n- Control project and workspace access\n- Track member activity and participation\n\n*Use Cases:* Team setup, access control, role management, collaboration.")
	Meta("openapi:tag:Modules")
	Meta("openapi:tag:Modules:desc", "**Feature Modules**\n\nGroup related work items into modules for better organization and tracking. Plan features, track progress, and manage deliverables at a higher level.\n\n*Key Features:*\n- Create and organize feature modules\n- Group work items by module\n- Track module progress and completion\n- Manage module leads and assignments\n\n*Use Cases:* Feature planning, release organization, progress tracking, team coordination.")
	Meta("openapi:tag:Projects")
	Meta("openapi:tag:Projects:desc", "**Project Management**\n\nCreate and manage projects to organize your development work. Configure project settings, manage team access, and control project visibility.\n\n*Key Features:*\n- Create, update, and delete projects\n- Configure project settings and preferences\n- Manage team access and permissions\n- Control project visibility and sharing\n\n*Use Cases:* Project setup, team collaboration, access control, project configuration.")
	Meta("openapi:tag:States")
	Meta("openapi:tag:States:desc", "**Workflow States**\n\nDefine custom workflow states for work items to match your team's process. Configure state transitions and track work item progress through different stages.\n\n*Key Features:*\n- Create custom workflow states\n- Configure state transitions and rules\n- Track work item progress through states\n- Set state-based permissions and automation\n\n*Use Cases:* Custom workflows, status tracking, process automation, progress monitoring.")
	Meta("openapi:tag:Users")
	Meta("openapi:tag:Users:desc", "**Current User Information**\n\nGet information about the currently authenticated user including profile details and account settings.\n\n*Key Features:*\n- Retrieve current user profile\n- Access user account information\n- View user preferences and settings\n- Get authentication context\n\n*Use Cases:* Profile display, user context, account information, authentication status.")
	Meta("openapi:tag:Work Item Activity")
	Meta("openapi:tag:Work Item Activity:desc", "**Activity History & Search**\n\nView activity history and search for work items across the workspace. Get detailed activity logs and find work items using text search.\n\n*Key Features:*\n- View work item activity history\n- Search work items across workspace\n- Track changes and modifications\n- Filter search results by project\n\n*Use Cases:* Activity tracking, work item discovery, change history, workspace search.")
	Meta("openapi:tag:Work Item Attachments")
	Meta("openapi:tag:Work Item Attachments:desc", "**Work Item File Attachments**\n\nGenerate presigned URLs for uploading files directly to specific work items. Upload and manage attachments associated with work items.\n\n*Key Features:*\n- Generate presigned URLs for work item attachments\n- Upload files directly to work items\n- Retrieve and manage attachment metadata\n- Delete attachments from work items\n\n*Use Cases:* Screenshots, error logs, design files, supporting documents.")
	Meta("openapi:tag:Work Item Comments")
	Meta("openapi:tag:Work Item Comments:desc", "**Comments & Discussions**\n\nAdd comments and discussions to work items for team collaboration. Support threaded conversations, mentions, and rich text formatting.\n\n*Key Features:*\n- Add comments to work items\n- Thread conversations and replies\n- Mention users and trigger notifications\n- Rich text and markdown support\n\n*Use Cases:* Team discussions, progress updates, code reviews, decision tracking.")
	Meta("openapi:tag:Work Item Links")
	Meta("openapi:tag:Work Item Links:desc", "**External Links & References**\n\nLink work items to external resources like documentation, repositories, or design files. Maintain connections between work items and external systems.\n\n*Key Features:*\n- Add external URL links to work items\n- Validate and preview linked resources\n- Organize links by type and category\n- Track link usage and access\n\n*Use Cases:* Documentation links, repository connections, design references, external tools.")
	Meta("openapi:tag:Work Items")
	Meta("openapi:tag:Work Items:desc", "**Work Items & Tasks**\n\nCreate and manage work items like tasks, bugs, features, and user stories. The core entities for tracking work in your projects.\n\n*Key Features:*\n- Create, update, and manage work items\n- Assign to team members and set priorities\n- Track progress through workflow states\n- Set due dates, estimates, and relationships\n\n*Use Cases:* Bug tracking, task management, feature development, sprint planning.")
	Meta("openapi:tag:members")
	Meta("openapi:tag:projects")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

// The legacy create view consumes a bulk command instead of its default
// ModelSerializer annotation. Role values retain the source's integer coercion.
var ProjectMembershipBulkMember = Type("ProjectMembershipBulkMember", func() {
	Attribute("member_id", String, func() { Format(FormatUUID) })
	Attribute("role", Any)
	Required("member_id")
})
var ProjectMembershipBulkCreateRequest = Type("ProjectMembershipBulkCreateRequest", func() {
	Attribute("members", ArrayOf(ProjectMembershipBulkMember), func() { MinLength(1) })
	Required("members")
})

var ProjectMembershipRoleCreateResult = Type("ProjectMembershipRoleCreateResult", func() {
	Description("Legacy ProjectMemberRoleSerializer response returned by bulk member creation.")
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("role", Int)
	Attribute("member", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("original_role", Int)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Required("id", "role", "member", "project", "original_role", "created_at")
})

var membershipLifecycleRawJSONResponseMetadata = Type("MembershipLifecycleRawJSONResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var membershipLifecycleErrorResponseBody = Type("MembershipLifecycleErrorResponseBody", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("detail", Any)
	Attribute("member", Any)
	Attribute("role", Any)
	Attribute("non_field_errors", Any)
	Attribute("is_active", Any)
	Attribute("comment", Any)
	Attribute("sort_order", Any)
	Attribute("deleted_at", Any)
	Attribute("view_props", Any)
	Attribute("default_props", Any)
	Attribute("preferences", Any)
	Attribute("created_by", Any)
	Attribute("updated_by", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
	Required("name")
})

var membershipLifecycleHTTPErrorResponseBody = Type("MembershipLifecycleHTTPErrorResponseBody", func() {
	Attribute("detail", Any)
	Attribute("member", Any)
	Attribute("role", Any)
	Attribute("non_field_errors", Any)
	Attribute("is_active", Any)
	Attribute("comment", Any)
	Attribute("sort_order", Any)
	Attribute("deleted_at", Any)
	Attribute("view_props", Any)
	Attribute("default_props", Any)
	Attribute("preferences", Any)
	Attribute("created_by", Any)
	Attribute("updated_by", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
})

func membershipLifecycleErrors() {
	Error("status_400", membershipLifecycleErrorResponseBody)
	Error("status_401", membershipLifecycleErrorResponseBody)
	Error("status_403", membershipLifecycleErrorResponseBody)
	Error("status_404", membershipLifecycleErrorResponseBody)
	Error("status_405", membershipLifecycleErrorResponseBody)
	Error("status_406", membershipLifecycleErrorResponseBody)
	Error("status_415", membershipLifecycleErrorResponseBody)
	Error("status_429", membershipLifecycleErrorResponseBody, func() { Temporary() })
	Error("status_500", membershipLifecycleErrorResponseBody, func() { Fault() })
}

func membershipLifecycleHTTPErrorResponses() {
	Response("status_400", 400, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
	Response("status_401", 401, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
	Response("status_403", 403, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
	Response("status_404", 404, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
	Response("status_405", 405, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
	Response("status_406", 406, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
	Response("status_415", 415, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
	Response("status_429", 429, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
	Response("status_500", 500, func() {
		ContentType("application/json")
		ErrorNameFromStatus()
		Body(membershipLifecycleHTTPErrorResponseBody)
	})
}

var _ = Service("ProjectMembershipLifecycle", func() {
	HTTP(func() { RuntimeCORS() })

	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDMembers", func() {
		membershipLifecycleErrors()
		Description("Retrieve all users who are members of the specified project.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_members")
		Meta("openapi:summary", "List project members")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/members/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectMemberPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:members")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/members/")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("List of project members with their roles")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ArrayOf(ImportedPlaneAPISerializersUserUserliteserializerResponse))
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDMembers", func() {
		membershipLifecycleErrors()
		Description("Create a new project member")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_members")
		Meta("openapi:summary", "Create project member")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/members/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:members")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/members/")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersMemberProjectmemberserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				Description("Project member created")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ImportedPlaneAPISerializersMemberProjectmemberserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk", func() {
		membershipLifecycleErrors()
		Description("Delete a project member")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_members_by_pk")
		Meta("openapi:summary", "Delete project member")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			Meta("openapi:tag:members")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Project member deleted")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk", func() {
		membershipLifecycleErrors()
		Description("Retrieve a project member by ID.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_members_by_pk")
		Meta("openapi:summary", "Get project member")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectMemberPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:members")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Project member")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ImportedPlaneAPISerializersUserUserliteserializerResponse)
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDMembersByPk", func() {
		membershipLifecycleErrors()
		Description("Update a project member")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_members_by_pk")
		Meta("openapi:summary", "Update project member")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:members")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/members/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersMemberProjectmemberserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				Description("Project member updated")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ImportedPlaneAPISerializersMemberProjectmemberserializerResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersLite", func() {
		membershipLifecycleErrors()
		Description("Retrieve a paginated, lightweight list of project members for pickers and directories.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_project_members_lite")
		Meta("openapi:summary", "List project members (lite)")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/project-members-lite/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectMemberPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberLiteAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("cursor", String, func() {
				Description("Pagination cursor for getting next set of results")
				Meta("openapi:allowEmptyValue", "false")
			})
			Attribute("perPage", String, func() {
				Description("Number of results per page, parsed by the Django-compatible paginator")
				Meta("openapi:allowEmptyValue", "false")
				Meta("openapi:format", "")
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/project-members-lite/")
			Param("cursor")
			Param("perPage:per_page")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Paginated list of project members with minimal fields")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ImportedRestFrameworkSerializersPaginatedprojectmemberliteResponse)
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembers", func() {
		membershipLifecycleErrors()
		Description("Retrieve all users who are members of the specified project.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_project_members")
		Meta("openapi:summary", "List project members")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/project-members/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectMemberPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("List of project members with their roles")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ArrayOf(ImportedPlaneAPISerializersUserUserliteserializerResponse))
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembers", func() {
		membershipLifecycleErrors()
		Description("Create a new project member")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_project_members")
		Meta("openapi:summary", "Create project member")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/project-members/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberListCreateAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersMemberProjectmemberserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				Description("Project member created")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ImportedPlaneAPISerializersMemberProjectmemberserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPk", func() {
		membershipLifecycleErrors()
		Description("Delete a project member")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_project_members_by_pk")
		Meta("openapi:summary", "Delete project member")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			Meta("openapi:tag:projects")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(204, func() {
				Description("Project member deleted")
			})
		})
	})
	Method("GetAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPk", func() {
		membershipLifecycleErrors()
		Description("Retrieve a project member by ID.")
		Meta("openapi:operationId", "get_api_v1_workspaces_by_slug_projects_by_project_id_project_members_by_pk")
		Meta("openapi:summary", "Get project member")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectMemberPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			GET("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			Response(200, func() {
				Description("Project member")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ImportedPlaneAPISerializersUserUserliteserializerResponse)
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDProjectMembersByPk", func() {
		membershipLifecycleErrors()
		Description("Update a project member")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_project_members_by_pk")
		Meta("openapi:summary", "Update project member")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.utils.permissions.project.ProjectAdminPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.member.ProjectMemberDetailAPIEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Required("pk", "project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:projects")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/project-members/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Header("apiKeyAuthCredential:X-Api-Key")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersMemberProjectmemberserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				Description("Project member updated")
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ImportedPlaneAPISerializersMemberProjectmemberserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDMembers", func() {
		membershipLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_members")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/members/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.member.ProjectMemberViewSet\"")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:members")
			POST("/api/workspaces/{slug}/projects/{project_id}/members/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ProjectMembershipBulkCreateRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(201, func() {
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ArrayOf(ProjectMembershipRoleCreateResult))
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDMembersLeave", func() {
		membershipLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_members_leave")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/members/leave/\"")
		Meta("openapi:extension:x-plane-handler", "\"leave\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.member.ProjectMemberViewSet\"")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			Meta("openapi:tag:members")
			POST("/api/workspaces/{slug}/projects/{project_id}/members/leave/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(Any, []string{"*/*"}, false)
			Response(204, func() { Description("No response body") })
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDMembersByPk", func() {
		membershipLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_members_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.member.ProjectMemberViewSet\"")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			Meta("openapi:tag:members")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/members/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDMembersByPk", func() {
		membershipLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_members_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/members/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.project.member.ProjectMemberViewSet\"")
		SessionSecurity(ImportedPlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(membershipLifecycleRawJSONResponseMetadata)
		HTTP(func() {
			membershipLifecycleHTTPErrorResponses()
			SkipResponseBodyEncodeDecode()
			Meta("openapi:tag:members")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/members/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersProjectProjectmemberadminserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				Header("content_type:Content-Type")
				OpenAPIBody(ProjectMembershipLegacyPatchResponse)
			})
		})
	})
})
