// Package design contains the reverse-imported app label write contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/project_label_writes/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var LabelPatchRequest = Type("LabelPatchRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("color", String, func() {
		MaxLength(255)
	})
	Attribute("name", String, func() {
		MinLength(1)
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("sort_order", Float64)
})

var LabelRequest = Type("LabelRequest", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("color", String, func() {
		MaxLength(255)
	})
	Attribute("name", String, func() {
		MinLength(1)
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Format(FormatUUID)
	})
	Attribute("sort_order", Float64)
	Required("name")
})

var Label = Type("Label", func() {
	Meta("openapi:typename:canonical", "true")
	Attribute("color", String, func() {
		MaxLength(255)
	})
	Attribute("id", String, func() {
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
		Meta("openapi:readOnly", "true")
	})
	Attribute("name", String, func() {
		MaxLength(255)
	})
	Attribute("parent", String, func() {
		Nullable()
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	})
	Attribute("project_id", String, func() {
		Nullable()
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
		Meta("openapi:readOnly", "true")
	})
	Attribute("sort_order", Float64)
	Attribute("workspace_id", String, func() {
		Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
		Meta("openapi:readOnly", "true")
	})
	Required("parent", "name", "color", "id", "project_id", "workspace_id", "sort_order")
})

var ImportedPlaneSessionCookieSecurity = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane session authentication without CSRF enforcement.")
})

var _ = API("Plane Internal Django Migration API", func() {
	Meta("http:generate", "all")
	Title("Plane Internal Django Migration API")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:projects")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("ProjectLabelWrites", func() {
	HTTP(func() { RuntimeCORS() })
	Method("Create", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "post_api_workspaces_by_slug_projects_by_project_id_issue_labels")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"POST /api/workspaces/{slug}/projects/{project_id}/issue-labels/\"")
		Meta("openapi:extension:x-plane-handler", "\"create\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectBasePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.label.LabelViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("project_id", String, func() {
				Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
			})
			Attribute("slug", String)
			Required("project_id", "slug")
		})
		labelErrors(true)
		Result(Label)
		HTTP(func() {
			Meta("openapi:tag:projects")
			POST("/api/workspaces/{slug}/projects/{project_id}/issue-labels/")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(LabelRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(201, func() {
				ContentType("application/json")
				OpenAPIBody(Label)
			})
		})
	})
	Method("Delete", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "delete_api_workspaces_by_slug_projects_by_project_id_issue_labels_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"DELETE /api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"destroy\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectBasePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.label.LabelViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
			})
			Attribute("project_id", String, func() {
				Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		labelErrors()
		Result(Empty)
		HTTP(func() {
			Meta("openapi:tag:projects")
			DELETE("/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			Response(204, func() {
				Description("No response body")
			})
		})
	})
	Method("Patch", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "patch_api_workspaces_by_slug_projects_by_project_id_issue_labels_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PATCH /api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"partial_update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectBasePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.label.LabelViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
			})
			Attribute("project_id", String, func() {
				Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		labelErrors(true)
		Result(Label)
		HTTP(func() {
			Meta("openapi:tag:projects")
			PATCH("/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(LabelPatchRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(Label)
			})
		})
	})
	Method("Update", func() {
		Description("This enables timezone conversion according\nto the user set timezone")
		Meta("openapi:operationId", "put_api_workspaces_by_slug_projects_by_project_id_issue_labels_by_pk")
		Meta("openapi:summary", "")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.authentication.session.BaseSessionAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"PUT /api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/\"")
		Meta("openapi:extension:x-plane-handler", "\"update\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"plane.app.permissions.project.ProjectBasePermission\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.app.views.issue.label.LabelViewSet\"")
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("pk", String, func() {
				Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
			})
			Attribute("project_id", String, func() {
				Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
			})
			Attribute("slug", String)
			Required("pk", "project_id", "slug")
		})
		labelErrors(true)
		Result(Label)
		HTTP(func() {
			Meta("openapi:tag:projects")
			PUT("/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/")
			Param("pk")
			Param("project_id")
			Param("slug")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(LabelRequest, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true)
			Response(200, func() {
				ContentType("application/json")
				OpenAPIBody(Label)
			})
		})
	})
	Method("CollectionOptions", func() {
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("slug", String)
			Attribute("project_id", String, func() { Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$") })
			Required("slug", "project_id")
		})
		Result(LabelMetadata)
		labelErrors()
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/issue-labels/")
			Param("slug")
			Param("project_id")
			Response(StatusOK, func() { ContentType("application/json") })
		})
	})
	Method("DetailOptions", func() {
		SessionSecurity(PlaneSession)
		Payload(func() {
			Attribute("slug", String)
			Attribute("project_id", String, func() { Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$") })
			Attribute("pk", String, func() { Pattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$") })
			Required("slug", "project_id", "pk")
		})
		Result(LabelMetadata)
		labelErrors()
		HTTP(func() {
			OPTIONS("/api/workspaces/{slug}/projects/{project_id}/issue-labels/{pk}/")
			Param("slug")
			Param("project_id")
			Param("pk")
			Response(StatusOK, func() { ContentType("application/json") })
		})
	})

})

var PlaneSession = SessionAuth("PlaneSession", func() {
	CookieTransport(ImportedPlaneSessionCookieSecurity, "", func() { CookieName("session-id") })
})

var LabelError = Type("LabelError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	labelErrorFields()
	Required("kind")
})
var LabelHTTPError = Type("LabelHTTPError", labelErrorFields)

func labelErrorFields() {
	Attribute("detail", String)
	Attribute("error", String, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("name", ArrayOf(String), func() { Meta("struct:field:name", "NameErrors") })
	Attribute("color", ArrayOf(String))
	Attribute("parent", ArrayOf(String))
	Attribute("sort_order", ArrayOf(String))
	Attribute("non_field_errors", ArrayOf(String))
}

// HTML renderer failures originate in the shared Django compatibility boundary.
// The generated contract records this additional response representation.
func labelErrors(rendererFailure ...bool) {
	for _, status := range []struct {
		name string
		code int
	}{
		{"status_400", 400}, {"status_401", 401}, {"status_403", 403}, {"status_404", 404},
		{"status_405", 405}, {"status_406", 406}, {"status_415", 415}, {"status_500", 500},
	} {
		Error(status.name, LabelError, func() {
			if status.code == 500 {
				Fault()
			}
		})
		HTTP(func() {
			Response(status.name, status.code, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body(LabelHTTPError)
				if status.code == 500 && len(rendererFailure) > 0 && rendererFailure[0] {
					Description("JSON application failure, or Django HTML server error if rendering a saved nonfinite sort_order fails.")
					Meta("openapi:response:extension:x-plane-renderer-failure", `{"content_type":"text/html; charset=utf-8","schema":{"type":"string"},"boundary":"shared Django renderer compatibility"}`)
				}
			})
		})
	}
}

var LabelMetadataField = Type("LabelMetadataField", func() {
	Attribute("type", String)
	Attribute("required", Boolean)
	Attribute("read_only", Boolean)
	Attribute("label", String)
	Attribute("max_length", Int)
	Required("type", "required", "read_only", "label")
})
var LabelMetadata = Type("LabelMetadata", func() {
	Attribute("name", String)
	Attribute("description", String)
	Attribute("renders", ArrayOf(String))
	Attribute("parses", ArrayOf(String))
	Attribute("actions", MapOf(String, MapOf(String, LabelMetadataField)))
	Required("name", "description", "renders", "parses")
})
