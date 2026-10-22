package page_lifecycle

import . "github.com/CaliLuke/loom/dsl"

var ImportedAccessEnum = Type("AccessEnum", Int, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `0` - Public\n* `1` - Private")
	Meta("openapi:format", "")
	Enum(0, 1)
})

var ImportedPatchedPlaneAppSerializersPagePagebinaryupdateserializerRequestRequest = Type("PatchedPlaneAppSerializersPagePagebinaryupdateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("description_binary", String)
	Attribute("description_html", String)
	Attribute("description_json", Any, func() {
	})
	Description("Serializer for updating page binary description with validation")
})

var ImportedPatchedPlaneAppSerializersPagePageserializerRequestRequest = Type("PatchedPlaneAppSerializersPagePageserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("access", ImportedAccessEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum("0")
		Maximum("32767")
	})
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("color", String, func() {
		MaxLength(255)
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("is_locked", Boolean)
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("labels", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String)
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("project_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
})

var ImportedPlaneAppSerializersPagePagedetailserializerResponse = Type("PlaneAppSerializersPagePagedetailserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("access", ImportedAccessEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum("0")
		Maximum("32767")
	})
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("color", String, func() {
		MaxLength(255)
	})
	Attribute("created_at", String, func() {
		Meta("openapi:readOnly", "true")
		Format(FormatDateTime)
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("description_html", String)
	Attribute("id", String, func() {
		Meta("openapi:readOnly", "true")
		Format(FormatUUID)
	})
	Attribute("is_favorite", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_locked", Boolean)
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String)
	Attribute("owned_by", String, func() {
		Meta("openapi:readOnly", "true")
		Format(FormatUUID)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("project_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Meta("openapi:readOnly", "true")
		Format(FormatDateTime)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Meta("openapi:readOnly", "true")
		Format(FormatUUID)
	})
	Required("description_html")
})

var ImportedPlaneAppSerializersPagePageserializerRequestRequest = Type("PlaneAppSerializersPagePageserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("access", ImportedAccessEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum("0")
		Maximum("32767")
	})
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("color", String, func() {
		MaxLength(255)
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("is_locked", Boolean)
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("labels", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String)
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("project_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
})

var ImportedPlaneAppSerializersPagePageserializerResponse = Type("PlaneAppSerializersPagePageserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("access", ImportedAccessEnum, func() {
		Meta("openapi:allOf:reference", "true")
		Meta("openapi:format", "")
		Minimum("0")
		Maximum("32767")
	})
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("color", String, func() {
		MaxLength(255)
	})
	Attribute("created_at", String, func() {
		Meta("openapi:readOnly", "true")
		Format(FormatDateTime)
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("id", String, func() {
		Meta("openapi:readOnly", "true")
		Format(FormatUUID)
	})
	Attribute("is_favorite", Boolean, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("is_locked", Boolean)
	Attribute("label_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("logo_props", Any, func() {
	})
	Attribute("name", String)
	Attribute("owned_by", String, func() {
		Meta("openapi:readOnly", "true")
		Format(FormatUUID)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("project_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}))
	Attribute("updated_at", String, func() {
		Title("Last Modified At")
		Meta("openapi:readOnly", "true")
		Format(FormatDateTime)
	})
	Attribute("updated_by", String, func() {
		Nullable()
		Title("Last Modified By")
		Format(FormatUUID)
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Meta("openapi:readOnly", "true")
		Format(FormatUUID)
	})
})

var ImportedRestFrameworkSerializersPagedescriptionupdateresponseResponse = Type("RestFrameworkSerializersPagedescriptionupdateresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("message", String)
	Required("message")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var _ = API("Plane Internal Django Migration API", func() {
	Title("Plane Internal Django Migration API")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:pages")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("PlaneInternalDjangoMigrationAPI", func() {
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDPages", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_pages")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/pages/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersPagePageserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:pages")
			POST("/api/workspaces/{slug}/projects/{project_id}/pages/")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersPagePageserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(201, func() {
				ContentType("application/json")
				Body(ImportedPlaneAppSerializersPagePageserializerResponse)
				OpenAPIBody(ImportedPlaneAppSerializersPagePageserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDPagesByPageID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:pages")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDPagesByPageID", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersPagePageserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:pages")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersPagePageserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				Body(ImportedPlaneAppSerializersPagePageserializerResponse)
				OpenAPIBody(ImportedPlaneAppSerializersPagePageserializerResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDPagesByPageIDAccess", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_access")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/access/\"")
		Meta("openapi:extension:x-plane-handler", "\"access\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersPagePageserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:pages")
			POST("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/access/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersPagePageserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				Body(ImportedPlaneAppSerializersPagePageserializerResponse)
				OpenAPIBody(ImportedPlaneAppSerializersPagePageserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDPagesByPageIDArchive", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"unarchive\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:pages")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDPagesByPageIDArchive", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"archive\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersPagePageserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:pages")
			POST("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/archive/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersPagePageserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				Body(ImportedPlaneAppSerializersPagePageserializerResponse)
				OpenAPIBody(ImportedPlaneAppSerializersPagePageserializerResponse)
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDPagesByPageIDDescription", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_description")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PagesDescriptionViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersPagedescriptionupdateresponseResponse)
		HTTP(func() {
			Meta("openapi:tag:pages")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/description/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersPagePagebinaryupdateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				Body(ImportedRestFrameworkSerializersPagedescriptionupdateresponseResponse)
				OpenAPIBody(ImportedRestFrameworkSerializersPagedescriptionupdateresponseResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDPagesByPageIDDuplicate", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_duplicate")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/duplicate/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageDuplicateEndpoint\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersPagePagedetailserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:pages")
			POST("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/duplicate/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(201, func() {
				ContentType("application/json")
				Body(ImportedPlaneAppSerializersPagePagedetailserializerResponse)
				OpenAPIBody(ImportedPlaneAppSerializersPagePagedetailserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDPagesByPageIDLock", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_lock")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/\"")
		Meta("openapi:extension:x-plane-handler", "\"unlock\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:pages")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDPagesByPageIDLock", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_pages_by_page_id_lock")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/\"")
		Meta("openapi:extension:x-plane-handler", "\"lock\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.page.ProjectPagePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.page.base.PageViewSet\"")
		Security(ImportedPlaneSessionCookieSecurity)
		Payload(func() {
			Attribute("page_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("")
				Meta("openapi:description:schema", "")
			})
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Required("page_id", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersPagePageserializerResponse)
		HTTP(func() {
			Meta("openapi:tag:pages")
			POST("/api/workspaces/{slug}/projects/{project_id}/pages/{page_id}/lock/")
			Param("page_id")
			Param("project_id")
			Param("slug")
			Cookie("planeSessionCookieCredential:session-id")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersPagePageserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				Body(ImportedPlaneAppSerializersPagePageserializerResponse)
				OpenAPIBody(ImportedPlaneAppSerializersPagePageserializerResponse)
			})
		})
	})
})
