//go:generate go tool loom gen github.com/CaliLuke/go-plane/modulelifecycle/design --output ..

package design

import . "github.com/CaliLuke/loom/dsl"

var ImportedModuleStatusEnum = Type("ModuleStatusEnum", String, func() {
	Meta("openapi:typename:canonical", "true")
	Description("* `backlog` - Backlog\n* `planned` - Planned\n* `in-progress` - In Progress\n* `paused` - Paused\n* `completed` - Completed\n* `cancelled` - Cancelled")
	Enum("backlog", "planned", "in-progress", "paused", "completed", "cancelled")
})

var moduleLifecycleError = Type("ModuleLifecycleError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("kind", "body")
})

func moduleLifecycleErrors() {
	for _, item := range []struct {
		name string
		code int
	}{
		{"status_400", 400}, {"status_401", 401}, {"status_403", 403},
		{"status_404", 404}, {"status_405", 405}, {"status_409", 409},
		{"status_415", 415}, {"status_429", 429}, {"status_500", 500},
	} {
		Error(item.name, moduleLifecycleError, func() {
			if item.code == 429 {
				Temporary()
			}
			if item.code == 500 {
				Fault()
			}
		})
		HTTP(func() {
			Response(item.name, item.code, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body("body")
			})
		})
	}
}

var ImportedPatchedPlaneAPISerializersModuleModuleupdateserializerRequestRequest = Type("PatchedPlaneApiSerializersModuleModuleupdateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for updating modules with enhanced validation and member management.\n\nExtends module creation with update-specific validations including\nmember reassignment, name conflict checking,\nand relationship management for module modifications.")
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("members", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
})

var ImportedPatchedPlaneAppSerializersModuleModulelinkserializerRequestRequest = Type("PatchedPlaneAppSerializersModuleModulelinkserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("metadata", Any, func() {
	})
	Attribute("title", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("url", String, func() {
		Format(FormatURI)
		MinLength(1)
		MaxLength(200)
	})
})

var ImportedPatchedPlaneAppSerializersModuleModulewriteserializerRequestRequest = Type("PatchedPlaneAppSerializersModuleModulewriteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("description_html", Any, func() {
		Nullable()
	})
	Attribute("description_text", Any, func() {
		Nullable()
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("lead_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("member_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("view_props", Any, func() {
	})
})

var ImportedPlaneAPISerializersModuleModulecreateserializerRequestRequest = Type("PlaneApiSerializersModuleModulecreateserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Serializer for creating modules with member validation and date checking.\n\nHandles module creation including member assignment validation, date range\nverification, and duplicate name prevention for feature-based\nproject organization setup.")
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("members", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Required("name")
})

var ImportedPlaneAPISerializersModuleModuleserializerResponse = Type("PlaneApiSerializersModuleModuleserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("The API v1 create response serialized by ModuleSerializer.")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("description_html", Any, func() {
		Nullable()
	})
	Attribute("description_text", Any, func() {
		Nullable()
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("members", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MaxLength(255)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
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
		Meta("openapi:readOnly", "true")
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("archived_at", "created_at", "created_by", "deleted_at", "description", "description_html", "description_text", "external_id", "external_source", "id", "lead", "logo_props", "members", "name", "project", "sort_order", "start_date", "status", "target_date", "updated_at", "updated_by", "view_props", "workspace")
})

var ModuleUpdateSerializerResponse = Type("ModuleUpdateSerializerResponse", func() {
	Description("API v1 PATCH response from ModuleUpdateSerializer; this intentionally omits the broader ModuleSerializer projection.")
	Attribute("description", String)
	Attribute("external_id", String, func() { Nullable(); MaxLength(255) })
	Attribute("external_source", String, func() { Nullable(); MaxLength(255) })
	Attribute("lead", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("name", String, func() { MaxLength(255) })
	Attribute("start_date", String, func() { Nullable(); Format(FormatDate) })
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() { Nullable(); Format(FormatDate) })
	Required("description", "external_id", "external_source", "lead", "name", "start_date", "status", "target_date")
})

var AppModuleComputedResponse = Type("AppModuleComputedResponse", func() {
	Description("Legacy app create and PATCH projection returned from the module queryset, with actor-local timestamps and computed issue/member values.")
	Attribute("backlog_issues", Int)
	Attribute("cancelled_issues", Int)
	Attribute("completed_estimate_points", Float64)
	Attribute("completed_issues", Int)
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("description", String)
	Attribute("description_html", Any, func() { Nullable() })
	Attribute("description_text", Any, func() { Nullable() })
	Attribute("external_id", String, func() { Nullable(); MaxLength(255) })
	Attribute("external_source", String, func() { Nullable(); MaxLength(255) })
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("is_favorite", Boolean)
	Attribute("lead_id", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("logo_props", Any)
	Attribute("member_ids", ArrayOf(String, func() { Format(FormatUUID) }))
	Attribute("name", String, func() { MaxLength(255) })
	Attribute("project_id", String, func() { Format(FormatUUID) })
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() { Nullable(); Format(FormatDate) })
	Attribute("started_issues", Int)
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() { Nullable(); Format(FormatDate) })
	Attribute("total_estimate_points", Float64)
	Attribute("total_issues", Int)
	Attribute("unstarted_issues", Int)
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("view_props", Any)
	Attribute("workspace_id", String, func() { Format(FormatUUID) })
	Required("backlog_issues", "cancelled_issues", "completed_estimate_points", "completed_issues", "created_at", "description", "description_html", "description_text", "external_id", "external_source", "id", "is_favorite", "lead_id", "logo_props", "member_ids", "name", "project_id", "sort_order", "start_date", "started_issues", "status", "target_date", "total_estimate_points", "total_issues", "unstarted_issues", "updated_at", "view_props", "workspace_id")
})

var ImportedPlaneAppSerializersModuleModulelinkserializerRequestRequest = Type("PlaneAppSerializersModuleModulelinkserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("metadata", Any, func() {
	})
	Attribute("title", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("url", String, func() {
		Format(FormatURI)
		MinLength(1)
		MaxLength(200)
	})
	Required("url")
})

var ImportedPlaneAppSerializersModuleModulelinkserializerResponse = Type("PlaneAppSerializersModuleModulelinkserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("created_at", String, func() {
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("created_by", String, func() {
		Nullable()
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("metadata", Any, func() {
	})
	Attribute("module", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("title", String, func() {
		Nullable()
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
		Meta("openapi:readOnly", "true")
	})
	Attribute("url", String, func() {
		Format(FormatURI)
		MaxLength(200)
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("url")
})

// The create response carries a Location header and the serializer body also
// contains its URL field. Keep those values separate in the transport DTO so
// mapping one field to a header cannot remove the other from JSON.
var AppModuleLinkCreateResponse = Type("AppModuleLinkCreateResponse", func() {
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("created_by", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("deleted_at", String, func() { Nullable(); Format(FormatDateTime) })
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("metadata", Any)
	Attribute("module", String, func() { Format(FormatUUID) })
	Attribute("project", String, func() { Format(FormatUUID) })
	Attribute("title", String, func() { Nullable(); MaxLength(255) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_by", String, func() { Nullable(); Format(FormatUUID) })
	Attribute("url", String, func() { Format(FormatURI); MaxLength(200) })
	Attribute("workspace", String, func() { Format(FormatUUID) })
	Attribute("location", String, func() { Format(FormatURI) })
	Required("created_at", "created_by", "deleted_at", "id", "metadata", "module", "project", "title", "updated_at", "updated_by", "url", "workspace", "location")
})

var ImportedPlaneAppSerializersModuleModulewriteserializerRequestRequest = Type("PlaneAppSerializersModuleModulewriteserializerRequestRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("description_html", Any, func() {
	})
	Attribute("description_text", Any, func() {
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("lead_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("member_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:writeOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MinLength(1)
		MaxLength(255)
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("view_props", Any, func() {
	})
	Required("name")
})

var ImportedPlaneAppSerializersModuleModulewriteserializerResponse = Type("PlaneAppSerializersModuleModulewriteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", String, func() {
		Nullable()
		Format(FormatDateTime)
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
	Attribute("deleted_at", String, func() {
		Nullable()
		Format(FormatDateTime)
		Meta("openapi:readOnly", "true")
	})
	Attribute("description", String, func() {
		Title("Module Description")
	})
	Attribute("description_html", Any, func() {
		Nullable()
	})
	Attribute("description_text", Any, func() {
		Nullable()
	})
	Attribute("external_id", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("external_source", String, func() {
		Nullable()
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("lead", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("lead_id", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("logo_props", Any, func() {
	})
	Attribute("members", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("member_ids", ArrayOf(String, func() {
		Format(FormatUUID)
	}), func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		Title("Module Name")
		MaxLength(255)
	})
	Attribute("project", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64)
	Attribute("start_date", String, func() {
		Nullable()
		Format(FormatDate)
	})
	Attribute("status", ImportedModuleStatusEnum)
	Attribute("target_date", String, func() {
		Nullable()
		Format(FormatDate)
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
		Meta("openapi:readOnly", "true")
	})
	Attribute("view_props", Any, func() {
	})
	Attribute("workspace", String, func() {
		Format(FormatUUID)
		Meta("openapi:readOnly", "true")
	})
	Required("archived_at", "created_at", "created_by", "deleted_at", "description", "description_html", "description_text", "external_id", "external_source", "id", "lead", "lead_id", "logo_props", "member_ids", "members", "name", "project", "sort_order", "start_date", "status", "target_date", "updated_at", "updated_by", "view_props", "workspace")
})

var ImportedRestFrameworkSerializersArchiveresponseResponse = Type("RestFrameworkSerializersArchiveresponseResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("archived_at", ImportedPythonDateTimeString)
	Required("archived_at")
})

var ImportedPythonDateTimeString = Type("PlanePythonDateTimeString", String, func() {
	Pattern(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d{1,6})?\+00:00$`)
})

var ImportedAPIKeyAuthSecurity = APIKeySecurity("ApiKeyAuth", func() {
	Description("Plane API token authentication.")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var PlaneSession = SessionAuth("PlaneSession", func() {
	CookieTransport(ImportedPlaneSessionCookieSecurity, "", func() { CookieName("session-id") })
})

var _ = API("Module Lifecycle API", func() {
	Title("Module Lifecycle API")
	Description("Django and API v1 module lifecycle operations.")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:projects")
	Meta("openapi:tag:modules")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("ModuleLifecycleAPI", func() {
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDArchivedModulesByPkUnarchive", func() {
		moduleLifecycleErrors()
		Description("Restore an archived module to active status, making it available for regular use.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_archived_modules_by_pk_unarchive")
		Meta("openapi:summary", "Unarchive module")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Security(ImportedAPIKeyAuthSecurity)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/unarchive/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleArchiveUnarchiveAPIEndpoint\"")
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("pk", String, func() {
				Description("Module ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:projects")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/unarchive/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDModules", func() {
		moduleLifecycleErrors()
		Description("Create a new project module with specified name, description, and timeline.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_modules")
		Meta("openapi:summary", "Create module")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Security(ImportedAPIKeyAuthSecurity)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/modules/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleListCreateAPIEndpoint\"")
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("project_id", "slug")
		})
		Result(ImportedPlaneAPISerializersModuleModuleserializerResponse)
		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:modules")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/modules/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAPISerializersModuleModulecreateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true, func() {
				Meta("openapi:allOf:reference", "true")
				Example("ModuleCreateSerializer", func() {
					Description("Example request for creating a module")
					Value(Val{"description": "New module description", "end_date": "2021-01-31", "external_id": "1234567890", "external_source": "github", "name": "New Module", "start_date": "2021-01-01"})
				})
			})
			Response(201, func() {
				Description("Module created")
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAPISerializersModuleModuleserializerResponse)
			})
		})
	})
	Method("DeleteAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk", func() {
		moduleLifecycleErrors()
		Description("Permanently remove a module and all its associated issue relationships.")
		Meta("openapi:operationId", "delete_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_pk")
		Meta("openapi:summary", "Delete module")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Security(ImportedAPIKeyAuthSecurity)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleDetailAPIEndpoint\"")
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("pk", String, func() {
				Description("Module ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:modules")
			DELETE("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("Resource deleted successfully")
			})
		})
	})
	Method("PatchAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPk", func() {
		moduleLifecycleErrors()
		Description("Modify an existing module's properties like name, description, status, or timeline.")
		Meta("openapi:operationId", "patch_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_pk")
		Meta("openapi:summary", "Update module")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Security(ImportedAPIKeyAuthSecurity)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"patch\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleDetailAPIEndpoint\"")
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("pk", String, func() {
				Description("Module ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("pk", "project_id", "slug")
		})
		Result(ModuleUpdateSerializerResponse)
		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:modules")
			PATCH("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAPISerializersModuleModuleupdateserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false, func() {
				Meta("openapi:allOf:reference", "true")
				Example("ModuleUpdateSerializer", func() {
					Description("Example request for updating a module")
					Value(Val{"description": "Updated module description", "end_date": "2021-01-31", "external_id": "1234567890", "external_source": "github", "name": "Updated Module", "start_date": "2021-01-01"})
				})
			})
			Response(200, func() {
				Description("Module updated successfully")
				ContentType("application/json")
				OpenAPIBody(ModuleUpdateSerializerResponse)
			})
		})
	})
	Method("PostAPIV1WorkspacesBySlugProjectsByProjectIDModulesByPkArchive", func() {
		moduleLifecycleErrors()
		Description("Move a module to archived status for historical tracking.")
		Meta("openapi:operationId", "post_api_v1_workspaces_by_slug_projects_by_project_id_modules_by_pk_archive")
		Meta("openapi:summary", "Archive module")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Security(ImportedAPIKeyAuthSecurity)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.module.ModuleArchiveUnarchiveAPIEndpoint\"")
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
			Attribute("pk", String, func() {
				Description("Module ID")
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Description("Project ID")
				Format(FormatUUID)
			})
			Attribute("slug", String, func() {
				Description("Workspace slug")
			})
			Required("pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Header("apiKeyAuthCredential:X-Api-Key")
			Meta("openapi:tag:modules")
			POST("/api/v1/workspaces/{slug}/projects/{project_id}/modules/{pk}/archive/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedModules", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_archived_modules")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/archived-modules/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.archive.ModuleArchiveUnarchiveEndpoint\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:projects")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/archived-modules/")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDArchivedModules", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_archived_modules")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/archived-modules/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.archive.ModuleArchiveUnarchiveEndpoint\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersArchiveresponseResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:projects")
			POST("/api/workspaces/{slug}/projects/{project_id}/archived-modules/")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersArchiveresponseResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDArchivedModulesByPk", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_archived_modules_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.archive.ModuleArchiveUnarchiveEndpoint\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
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
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:projects")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDArchivedModulesByPk", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_archived_modules_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.archive.ModuleArchiveUnarchiveEndpoint\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersArchiveresponseResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:projects")
			POST("/api/workspaces/{slug}/projects/{project_id}/archived-modules/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersArchiveresponseResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDModules", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_modules")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/modules/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.base.ModuleViewSet\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		Result(AppModuleComputedResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			POST("/api/workspaces/{slug}/projects/{project_id}/modules/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersModuleModulewriteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(AppModuleComputedResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchive", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"delete\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.archive.ModuleArchiveUnarchiveEndpoint\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("module_id", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/")
			Param("module_id")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDArchive", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_archive")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/\"")
		Meta("openapi:extension:x-plane-handler", "\"post\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.archive.ModuleArchiveUnarchiveEndpoint\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("module_id", "project_id", "slug")
		})
		Result(ImportedRestFrameworkSerializersArchiveresponseResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			POST("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/archive/")
			Param("module_id")
			Param("project_id")
			Param("slug")
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedRestFrameworkSerializersArchiveresponseResponse)
			})
		})
	})
	Method("PostAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinks", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_links")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.base.ModuleLinkViewSet\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("module_id", "project_id", "slug")
		})
		Result(AppModuleLinkCreateResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			POST("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/")
			Param("module_id")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersModuleModulelinkserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				Header("location:Location")
				Body(func() {
					Attribute("created_at")
					Attribute("created_by")
					Attribute("deleted_at")
					Attribute("id")
					Attribute("metadata")
					Attribute("module")
					Attribute("project")
					Attribute("title")
					Attribute("updated_at")
					Attribute("updated_by")
					Attribute("url")
					Attribute("workspace")
				})
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPk", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.base.ModuleLinkViewSet\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("module_id", "pk", "project_id", "slug")
		})
		Result(Empty)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/")
			Param("module_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPk", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.base.ModuleLinkViewSet\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("module_id", "pk", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersModuleModulelinkserializerResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/")
			Param("module_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersModuleModulelinkserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersModuleModulelinkserializerResponse)
			})
		})
	})
	Method("PutAPIWorkspacesBySlugProjectsByProjectIDModulesByModuleIDModuleLinksByPk", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_modules_by_module_id_module_links_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectEntityPermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.base.ModuleLinkViewSet\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("module_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("module_id", "pk", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersModuleModulelinkserializerResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			PUT("/api/workspaces/{slug}/projects/{project_id}/modules/{module_id}/module-links/{pk}/")
			Param("module_id")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersModuleModulelinkserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersModuleModulelinkserializerResponse)
			})
		})
	})
	Method("DeleteAPIWorkspacesBySlugProjectsByProjectIDModulesByPk", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_modules_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/modules/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.base.ModuleViewSet\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
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
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("PatchAPIWorkspacesBySlugProjectsByProjectIDModulesByPk", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_modules_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/modules/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.base.ModuleViewSet\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(AppModuleComputedResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPatchedPlaneAppSerializersModuleModulewriteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(AppModuleComputedResponse)
			})
		})
	})
	Method("PutAPIWorkspacesBySlugProjectsByProjectIDModulesByPk", func() {
		moduleLifecycleErrors()
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_modules_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		SessionSecurity(PlaneSession)
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/modules/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.module.base.ModuleViewSet\"")
		Payload(func() {
			APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
			Attribute("pk", String, func() {
				Format(FormatUUID)
			})
			Attribute("project_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		Result(ImportedPlaneAppSerializersModuleModulewriteserializerResponse)
		HTTP(func() {
			Cookie("planeSessionCookieCredential:session-id")
			Meta("openapi:tag:modules")
			PUT("/api/workspaces/{slug}/projects/{project_id}/modules/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(ImportedPlaneAppSerializersModuleModulewriteserializerRequestRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(ImportedPlaneAppSerializersModuleModulewriteserializerResponse)
			})
		})
	})
})
