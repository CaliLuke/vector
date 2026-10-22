// Package design defines the workspace create, update, and delete API.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/workspacelifecycle/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var sessionCookie = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane Django session cookie authentication.")
})

var rawResponse = Type("WorkspaceLifecycleRawJSON", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var errorResponse = Type("WorkspaceLifecycleError", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

var requestBody = MapOf(String, Any)

func workspacePayload(path bool) {
	Security(sessionCookie)
	Payload(func() {
		if path {
			Attribute("slug", String)
		}
		APIKey("PlaneSessionCookie", "planeSessionCookieCredential", String)
		if path {
			Required("slug")
		}
	})
}

func workspaceErrors() {
	for _, item := range []struct {
		name string
		code int
	}{
		{"status_400", 400}, {"status_401", 401}, {"status_403", 403},
		{"status_404", 404}, {"status_409", 409}, {"status_415", 415}, {"status_500", 500},
	} {
		Error(item.name, errorResponse, func() {
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

func workspaceWriteHTTP(method, path string, success int, body bool) {
	HTTP(func() {
		switch method {
		case "POST":
			POST(path)
		case "PATCH":
			PATCH(path)
		case "PUT":
			PUT(path)
		case "DELETE":
			DELETE(path)
		}
		Cookie("planeSessionCookieCredential:session-id")
		if method != "POST" {
			Param("slug")
		}
		if body {
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(requestBody, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, method == "POST")
		}
		if success == 204 {
			Response(StatusNoContent)
		} else {
			SkipResponseBodyEncodeDecode()
			Response(success, func() {
				Header("content_type:Content-Type")
				ContentType("application/json")
				OpenAPIBody(Any)
			})
		}
	})
}

var _ = API("Plane Workspace Lifecycle API", func() {
	Title("Plane Workspace Lifecycle API")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("workspace_lifecycle", func() {
	Description("Create, update, and delete workspaces through the authenticated session API.")
	HTTP(func() { RuntimeCORS() })

	Method("create_workspace", func() {
		Meta("openapi:operationId", "post_api_workspaces")
		workspacePayload(false)
		Result(rawResponse)
		workspaceErrors()
		workspaceWriteHTTP("POST", "/api/workspaces/", 201, true)
	})
	Method("patch_workspace", func() {
		Meta("openapi:operationId", "patch_api_workspaces_by_slug")
		workspacePayload(true)
		Result(rawResponse)
		workspaceErrors()
		workspaceWriteHTTP("PATCH", "/api/workspaces/{slug}/", 200, true)
	})
	Method("put_workspace", func() {
		Meta("openapi:operationId", "put_api_workspaces_by_slug")
		workspacePayload(true)
		Result(rawResponse)
		workspaceErrors()
		workspaceWriteHTTP("PUT", "/api/workspaces/{slug}/", 200, true)
	})
	Method("delete_workspace", func() {
		Meta("openapi:operationId", "delete_api_workspaces_by_slug")
		workspacePayload(true)
		Result(Empty)
		workspaceErrors()
		workspaceWriteHTTP("DELETE", "/api/workspaces/{slug}/", 204, false)
	})
})
