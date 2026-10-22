package design

import . "github.com/CaliLuke/loom/dsl"

var projectMembershipUser = Type("ProjectMembershipUser", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("first_name", String)
	Attribute("last_name", String)
	Attribute("avatar", String)
	Attribute("avatar_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("is_bot", Boolean)
	Attribute("display_name", String)
	Attribute("email", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("last_login_medium", String)
	Required("id", "first_name", "last_name", "avatar", "avatar_url", "is_bot", "display_name")
})

var workspaceMembershipDirectory = Type("WorkspaceMembershipDirectory", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("member", projectMembershipUser)
	Attribute("role", Int)
	Attribute("is_active", Boolean)
	Attribute("company_role", String, func() { Nullable() })
	Attribute("view_props", Any)
	Attribute("default_props", Any)
	Attribute("issue_props", Any)
	Attribute("explored_features", Any)
	Attribute("getting_started_checklist", Any)
	Attribute("tips", Any)
	Required(
		"id", "created_at", "updated_at", "created_by", "updated_by", "deleted_at", "workspace", "member",
		"role", "is_active", "company_role", "view_props", "default_props", "issue_props", "explored_features",
		"getting_started_checklist", "tips",
	)
})

var workspaceProjectMembershipRole = Type("WorkspaceProjectMembershipRole", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("role", Int)
	Attribute("member", String, func() { Format(FormatUUID); Nullable() })
	Attribute("original_role", Int)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Required("id", "role", "member", "original_role", "created_at")
})

var projectMembershipRole = Type("ProjectMembershipRole", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("role", Int)
	Attribute("member", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("original_role", Int)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Required("id", "role", "member", "project", "original_role", "created_at")
})

var projectMembershipGuestDirectory = Type("ProjectMembershipGuestDirectory", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("role", Int)
	Attribute("member", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("original_role", Int)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Required("id", "role", "member", "project", "original_role", "created_at")
})

var projectMembershipFullDirectory = Type("ProjectMembershipFullDirectory", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("role", Int)
	Attribute("member", projectMembershipUser)
	Attribute("project", projectMembershipProject)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("comment", String, func() { Nullable() })
	Attribute("view_props", Any)
	Attribute("default_props", Any)
	Attribute("preferences", Any)
	Attribute("sort_order", Float64)
	Attribute("is_active", Boolean)
	Attribute("workspace", projectMembershipWorkspace)
	Required(
		"id", "role", "member", "project", "created_at", "updated_at", "created_by", "updated_by",
		"deleted_at", "comment", "view_props", "default_props", "preferences", "sort_order", "is_active", "workspace",
	)
})

var projectMembershipDirectory = OneOf(projectMembershipGuestDirectory, projectMembershipFullDirectory)

var projectMembershipPreference = Type("ProjectMembershipPreference", func() {
	Attribute("preferences", Any)
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("member_id", String, func() { Format(FormatUUID) })
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Required("preferences", "project_id", "member_id", "workspace_id")
})

var projectMembershipProject = Type("ProjectMembershipProject", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("name", String)
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("cover_image_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("logo_props", Any)
	Attribute("description", String)
	Required("id", "identifier", "name", "cover_image", "cover_image_url", "logo_props", "description")
})

var projectMembershipWorkspace = Type("ProjectMembershipWorkspace", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("slug", String)
	Attribute("logo_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Required("id", "name", "slug", "logo_url")
})

var projectMembership = Type("ProjectMembership", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("comment", String, func() { Nullable() })
	Attribute("role", Int, func() { Enum(5, 15, 20) })
	Attribute("view_props", Any)
	Attribute("default_props", Any)
	Attribute("preferences", Any)
	Attribute("sort_order", Float64)
	Attribute("is_active", Boolean)
	Attribute("member", projectMembershipUser)
	Attribute("project", projectMembershipProject)
	Attribute("workspace", projectMembershipWorkspace)
	Required(
		"id", "created_at", "updated_at", "created_by", "updated_by", "deleted_at", "comment", "role",
		"view_props", "default_props", "preferences", "sort_order", "is_active", "member", "project", "workspace",
	)
})

var projectMembershipReadsUnauthenticated = Type("ProjectMembershipReadsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var projectMembershipReadsForbidden = Type("ProjectMembershipReadsForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var projectMembershipReadsPermissionDenied = Type("ProjectMembershipReadsPermissionDenied", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var projectMembershipReadsNotFound = Type("ProjectMembershipReadsNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var projectMembershipReadsInternalError = Type("ProjectMembershipReadsInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("project_membership_reads", func() {
	Description("Returns authenticated workspace and project membership read models.")

	Method("membership", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id")
		})
		Result(projectMembership)
		Error("unauthenticated", projectMembershipReadsUnauthenticated)
		Error("not_found", projectMembershipReadsNotFound)
		Error("internal_error", projectMembershipReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/project-members/me/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("roles", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Required("slug")
		})
		Result(MapOf(String, Int))
		Error("unauthenticated", projectMembershipReadsUnauthenticated)
		Error("forbidden", projectMembershipReadsForbidden)
		Error("internal_error", projectMembershipReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/users/me/workspaces/{slug}/project-roles/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("workspace_members", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("search", String)
			Required("slug")
		})
		Result(ArrayOf(workspaceMembershipDirectory))
		Error("unauthenticated", projectMembershipReadsUnauthenticated)
		Error("permission_denied", projectMembershipReadsPermissionDenied)
		Error("internal_error", projectMembershipReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/members/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("search")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("permission_denied", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("workspace_member", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("pk", String, func() { Format(FormatUUID) })
			Attribute("search", String)
			Required("slug", "pk")
		})
		Result(workspaceMembershipDirectory)
		Error("unauthenticated", projectMembershipReadsUnauthenticated)
		Error("permission_denied", projectMembershipReadsPermissionDenied)
		Error("not_found", projectMembershipReadsNotFound)
		Error("internal_error", projectMembershipReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/members/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("pk")
			Param("search")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("permission_denied", StatusForbidden)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("workspace_project_members", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Required("slug")
		})
		Result(MapOf(String, ArrayOf(workspaceProjectMembershipRole)))
		Error("unauthenticated", projectMembershipReadsUnauthenticated)
		Error("forbidden", projectMembershipReadsForbidden)
		Error("internal_error", projectMembershipReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/project-members/")
			Cookie("session_id:session-id")
			Param("slug")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("project_members", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id")
		})
		Result(ArrayOf(projectMembershipRole))
		Error("unauthenticated", projectMembershipReadsUnauthenticated)
		Error("permission_denied", projectMembershipReadsPermissionDenied)
		Error("internal_error", projectMembershipReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/members/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("permission_denied", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("project_member", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "pk")
		})
		Result(projectMembershipDirectory, func() { Untagged() })
		Error("unauthenticated", projectMembershipReadsUnauthenticated)
		Error("permission_denied", projectMembershipReadsPermissionDenied)
		Error("not_found", projectMembershipReadsNotFound)
		Error("internal_error", projectMembershipReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/members/{pk}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("permission_denied", StatusForbidden)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("preference", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("member_id", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "member_id")
		})
		Result(projectMembershipPreference)
		Error("unauthenticated", projectMembershipReadsUnauthenticated)
		Error("permission_denied", projectMembershipReadsPermissionDenied)
		Error("not_found", projectMembershipReadsNotFound)
		Error("internal_error", projectMembershipReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/preferences/member/{member_id}/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("member_id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("permission_denied", StatusForbidden)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
