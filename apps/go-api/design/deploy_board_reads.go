package design

import . "github.com/CaliLuke/loom/dsl"

var deployBoardProjectDetail = Type("DeployBoardProjectDetail", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("identifier", String)
	Attribute("name", String)
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("cover_image_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Attribute("logo_props", Any)
	Attribute("description", String)
	Required("id", "identifier", "name", "cover_image", "cover_image_url", "logo_props", "description")
})

var deployBoardWorkspaceDetail = Type("DeployBoardWorkspaceDetail", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("name", String)
	Attribute("slug", String)
	Attribute("logo_url", String, func() { Nullable(); Meta("loom:vet:ignore", "string-format") })
	Required("id", "name", "slug", "logo_url")
})

// The collection action serializes a single model without many=True. An absent
// row exposes the serializer's 12 unsaved-model defaults and omits database-
// generated and relationship fields.
var deployBoardCollectionRead = Type("DeployBoardCollectionRead", func() {
	deployBoardReadAttributes(true)
	Required(
		"created_by", "deleted_at", "entity_identifier", "entity_name", "intake",
		"is_activity_enabled", "is_comments_enabled", "is_disabled",
		"is_reactions_enabled", "is_votes_enabled", "updated_by", "view_props",
	)
})

var deployBoardRead = Type("DeployBoardRead", func() {
	deployBoardReadAttributes(false)
	Required(
		"anchor", "created_at", "created_by", "deleted_at", "entity_identifier",
		"entity_name", "id", "intake", "is_activity_enabled", "is_comments_enabled",
		"is_disabled", "is_reactions_enabled", "is_votes_enabled", "project",
		"project_details", "updated_at", "updated_by", "view_props", "workspace",
		"workspace_detail",
	)
})

func deployBoardReadAttributes(nullableViewProps bool) {
	Attribute("anchor", String)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("entity_identifier", String, func() { Format(FormatUUID); Nullable() })
	Attribute("entity_name", String, func() { MaxLength(30); Nullable() })
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("intake", String, func() { Format(FormatUUID); Nullable() })
	Attribute("is_activity_enabled", Boolean)
	Attribute("is_comments_enabled", Boolean)
	Attribute("is_disabled", Boolean)
	Attribute("is_reactions_enabled", Boolean)
	Attribute("is_votes_enabled", Boolean)
	Attribute("project", String, func() { Format(FormatUUID); Nullable() })
	Attribute("project_details", deployBoardProjectDetail, func() { Nullable() })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	if nullableViewProps {
		Attribute("view_props", Any, func() { Nullable() })
	} else {
		Attribute("view_props", Any)
	}
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("workspace_detail", deployBoardWorkspaceDetail)
}

var deployBoardReadsUnauthenticated = Type("DeployBoardReadsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var deployBoardReadsForbidden = Type("DeployBoardReadsForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var deployBoardReadsNotFound = Type("DeployBoardReadsNotFound", func() {
	Attribute("detail", String)
	Required("detail")
})

var deployBoardReadsInternalError = Type("DeployBoardReadsInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("deploy_board_reads", func() {
	Description("Returns authenticated project deploy-board read models.")
	HTTP(func() {
		RuntimeCORS()
	})

	Method("collection", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_project_deploy_boards")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("search", String)
			Required("slug", "project_id")
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", deployBoardReadsUnauthenticated)
		Error("forbidden", deployBoardReadsForbidden)
		Error("internal_error", deployBoardReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/")
			rawJSONReadResponse(deployBoardCollectionRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("search")
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("detail", func() {
		Meta("openapi:operationId", "get_api_workspaces_by_slug_projects_by_project_id_project_deploy_boards_by_pk")
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("slug", String)
			Attribute("project_id", String, func() { Format(FormatUUID) })
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("slug", "project_id", "pk")
		})
		Result(rawJSONResponseMetadata)
		Error("unauthenticated", deployBoardReadsUnauthenticated)
		Error("forbidden", deployBoardReadsForbidden)
		Error("not_found", deployBoardReadsNotFound)
		Error("internal_error", deployBoardReadsInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/workspaces/{slug}/projects/{project_id}/project-deploy-boards/{pk}/")
			rawJSONReadResponse(deployBoardRead)
			Cookie("session_id:session-id")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
