package design

import . "github.com/CaliLuke/loom/dsl"

var publicProjectLabel = Type("PublicProjectLabel", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("color", String)
	Attribute("parent", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "name", "color", "parent")
})

var publicProjectState = Type("PublicProjectState", func() {
	Attribute("name", String)
	Attribute("group", String)
	Attribute("color", String)
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("sequence", Float64)
	Required("name", "group", "color", "id", "sequence")
})

var publicProjectCollectionItem = Type("PublicProjectCollectionItem", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Required("id", "name")
})

var publicProjectMember = Type("PublicProjectMember", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("member", String, func() { Format(FormatUUID); Nullable() })
	Attribute("member__display_name", String, func() { Nullable() })
	Attribute("member__avatar", String, func() { Nullable() })
	Required("id", "member", "member__display_name", "member__avatar")
})

var publicProjectMeta = Type("PublicProjectMeta", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("name", String)
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("icon_prop", Any, func() { Nullable() })
	Attribute("emoji", String, func() { Nullable() })
	Attribute("description", String)
	Required("id", "identifier", "name", "cover_image", "icon_prop", "emoji", "description")
})

var publicProjectSettingsProject = Type("PublicProjectSettingsProject", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("name", String)
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("cover_image_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("logo_props", Any)
	Attribute("description", String)
	Required("id", "identifier", "name", "cover_image", "cover_image_url", "logo_props", "description")
})

var publicProjectSettingsWorkspace = Type("PublicProjectSettingsWorkspace", func() {
	Attribute("name", String)
	Attribute("slug", String)
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("logo_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Required("name", "slug", "id", "logo_url")
})

var publicProjectSettings = Type("PublicProjectSettings", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("project_details", publicProjectSettingsProject, func() { Nullable() })
	Attribute("workspace_detail", publicProjectSettingsWorkspace)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("entity_identifier", String, func() { Format(FormatUUID); Nullable() })
	Attribute("entity_name", String, func() { Nullable() })
	Attribute("anchor", String)
	Attribute("is_comments_enabled", Boolean)
	Attribute("is_reactions_enabled", Boolean)
	Attribute("is_votes_enabled", Boolean)
	Attribute("view_props", Any)
	Attribute("is_activity_enabled", Boolean)
	Attribute("is_disabled", Boolean)
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID); Nullable() })
	Attribute("intake", String, func() { Format(FormatUUID); Nullable() })
	Required("id", "project_details", "workspace_detail", "created_at", "updated_at", "deleted_at", "entity_identifier", "entity_name", "anchor", "is_comments_enabled", "is_reactions_enabled", "is_votes_enabled", "view_props", "is_activity_enabled", "is_disabled", "created_by", "updated_by", "workspace", "project", "intake")
})

var publicProjectMetadataNotFound = Type("PublicProjectMetadataNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var publicProjectMetadataInternalError = Type("PublicProjectMetadataInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("public_project_metadata_reads", func() {
	Description("Returns public project cycles, labels, members, metadata, modules, settings, and workflow states for a deploy-board anchor.")

	Method("cycles", func() {
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			stickySessionPayload()
			Attribute("anchor", String)
			Attribute("search", String)
			Required("anchor")
		})
		Result(ArrayOf(publicProjectCollectionItem))
		Error("not_found", publicProjectMetadataNotFound)
		Error("internal_error", publicProjectMetadataInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/cycles/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("search")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("labels", func() {
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			stickySessionPayload()
			Attribute("anchor", String)
			Attribute("search", String)
			Required("anchor")
		})
		Result(ArrayOf(publicProjectLabel))
		Error("not_found", publicProjectMetadataNotFound)
		Error("internal_error", publicProjectMetadataInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/labels/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("search")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("modules", func() {
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			stickySessionPayload()
			Attribute("anchor", String)
			Attribute("search", String)
			Required("anchor")
		})
		Result(ArrayOf(publicProjectCollectionItem))
		Error("not_found", publicProjectMetadataNotFound)
		Error("internal_error", publicProjectMetadataInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/modules/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("search")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("members", func() {
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			stickySessionPayload()
			Attribute("anchor", String)
			Attribute("search", String)
			Required("anchor")
		})
		Result(ArrayOf(publicProjectMember))
		Error("not_found", publicProjectMetadataNotFound)
		Error("internal_error", publicProjectMetadataInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/members/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("search")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("meta", func() {
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			stickySessionPayload()
			Attribute("anchor", String)
			Attribute("search", String)
			Required("anchor")
		})
		Result(publicProjectMeta)
		Error("not_found", publicProjectMetadataNotFound)
		Error("internal_error", publicProjectMetadataInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/meta/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("search")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("settings", func() {
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			stickySessionPayload()
			Attribute("anchor", String)
			Attribute("search", String)
			Required("anchor")
		})
		Result(publicProjectSettings)
		Error("not_found", publicProjectMetadataNotFound)
		Error("internal_error", publicProjectMetadataInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/settings/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("search")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("project_anchor", func() {
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() {
				Pattern(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
			})
			Attribute("search", String)
			Required("slug", "project_id")
		})
		Result(publicProjectSettings)
		Error("not_found", publicProjectMetadataNotFound)
		Error("internal_error", publicProjectMetadataInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/public/workspaces/{slug}/projects/{project_id}/anchor/")
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("search")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("states", func() {
		Security(planeSessionCookie)
		Security()
		Payload(func() {
			stickySessionPayload()
			Attribute("anchor", String)
			Attribute("search", String)
			Required("anchor")
		})
		Result(ArrayOf(publicProjectState))
		Error("not_found", publicProjectMetadataNotFound)
		Error("internal_error", publicProjectMetadataInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/public/anchor/{anchor}/states/")
			Cookie("session_id:session-id")
			Param("anchor")
			Param("search")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
