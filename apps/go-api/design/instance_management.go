package design

import . "github.com/CaliLuke/loom/dsl"

var instanceManagementError = Type("InstanceManagementError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Attribute("retry_after", Int)
	Required("kind", "body")
})

var instanceManagementBody = MapOf(String, Any)

var instanceRootResponseMetadata = Type("InstanceRootResponseMetadata", func() {
	Attribute("content_type", String)
	Attribute("cache_control", String)
	Required("content_type", "cache_control")
})

var instanceAdminRedirectResponse = Type("InstanceAdminRedirectResponse", func() {
	Attribute("location", String)
	Attribute("content_type", String)
	Required("location", "content_type")
})

func instanceAdminRedirect(method, operationID, route string) {
	Method(method, func() {
		Meta("openapi:operationId", operationID)
		Payload(func() { instanceManagementBodyHeader() })
		Result(instanceAdminRedirectResponse)
		instanceManagementStatusError("internal_error", 500)
		HTTP(func() {
			POST(route)
			instanceManagementRequestBody()
			SkipResponseBodyEncodeDecode()
			Response(StatusFound, func() {
				Header("location:Location")
				Header("content_type:Content-Type")
				ContentType("text/html")
			})
		})
	})
}

func instanceManagementAdminPayload() {
	instanceAdminSessionPayload()
}

func instanceManagementRequestBody() {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(instanceManagementBody, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
	Header("content_type:Content-Type")
}

func instanceManagementBodyHeader() { Attribute("content_type", String) }

func instanceManagementRawResponse(schema any, status int) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

func instanceManagementStatusError(name string, status int) {
	Error(name, instanceManagementError, func() {
		if status == 500 {
			Fault()
		}
		if status == 429 {
			Temporary()
		}
	})
	HTTP(func() {
		Response(name, status, func() {
			ContentType("application/json")
			ErrorNameFromStatus()
			if status == 429 {
				Header("retry_after:Retry-After")
			}
			Body("body")
		})
	})
}

func instanceManagementErrors() {
	for _, item := range []struct {
		name   string
		status int
	}{
		{"bad_request", 400}, {"forbidden", 403}, {"not_found", 404},
		{"conflict", 409}, {"method_not_allowed", 405}, {"internal_error", 500},
	} {
		instanceManagementStatusError(item.name, item.status)
	}
	Error("unauthenticated", instanceAdminSessionUnauthenticated)
	HTTP(func() { Response("unauthenticated", StatusUnauthorized) })
}

func instanceManagementProtectedMethod(method, operationID, route string, body, detail bool) {
	Method(method, func() {
		Meta("openapi:operationId", operationID)
		Security(planeAdminSessionCookie)
		Payload(func() {
			instanceAdminSessionPayload()
			if detail {
				Attribute("pk", String, func() { Format(FormatUUID) })
				Required("pk")
			}
		})
		Result(rawJSONResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			switch method {
			case "GET":
				GET(route)
			case "POST":
				POST(route)
			case "PATCH":
				PATCH(route)
			case "DELETE":
				DELETE(route)
			}
			Cookie("session_id:admin-session-id")
			if detail {
				Param("pk")
			}
			if body {
				instanceManagementRequestBody()
			}
			instanceManagementRawResponse(Any, StatusOK)
		})
	})
}

var _ = Service("instance_management", func() {
	Description("Manages the Plane self-hosted instance configuration and administration.")
	HTTP(func() { RuntimeCORS() })
	instanceAdminRedirect("admin_sign_in", "post_api_instances_admins_sign_in", "/api/instances/admins/sign-in/")
	instanceAdminRedirect("admin_sign_up", "post_api_instances_admins_sign_up", "/api/instances/admins/sign-up/")
	instanceAdminRedirect("admin_sign_out", "post_api_instances_admins_sign_out", "/api/instances/admins/sign-out/")

	Method("root_get", func() {
		Meta("openapi:operationId", "get_api_instances")
		Payload(func() {})
		Result(instanceRootResponseMetadata)
		instanceManagementStatusError("throttled", 429)
		instanceManagementStatusError("internal_error", 500)
		HTTP(func() {
			GET("/api/instances/")
			SkipResponseBodyEncodeDecode()
			Response(StatusOK, func() {
				Header("content_type:Content-Type")
				Header("cache_control:Cache-Control")
				ContentType("application/json")
				OpenAPIBody(Any)
			})
		})
	})
	Method("root_patch", func() {
		Meta("openapi:operationId", "patch_api_instances")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload(); instanceManagementBodyHeader() })
		Result(rawJSONResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			PATCH("/api/instances/")
			Cookie("session_id:admin-session-id")
			instanceManagementRequestBody()
			instanceManagementRawResponse(Any, StatusOK)
		})
	})
	Method("admins_get", func() {
		Meta("openapi:operationId", "get_api_instances_admins")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload() })
		Result(rawJSONResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			GET("/api/instances/admins/")
			Cookie("session_id:admin-session-id")
			instanceManagementRawResponse(ArrayOf(Any), StatusOK)
		})
	})
	Method("admins_post", func() {
		Meta("openapi:operationId", "post_api_instances_admins")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload(); instanceManagementBodyHeader() })
		Result(rawJSONResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			POST("/api/instances/admins/")
			Cookie("session_id:admin-session-id")
			instanceManagementRequestBody()
			instanceManagementRawResponse(Any, StatusCreated)
		})
	})
	Method("admins_delete_collection", func() {
		Meta("openapi:operationId", "delete_api_instances_admins")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload() })
		instanceManagementErrors()
		HTTP(func() {
			DELETE("/api/instances/admins/")
			Cookie("session_id:admin-session-id")
		})
	})
	Method("admins_detail_get", func() {
		Meta("openapi:operationId", "get_api_instances_admins_by_pk")
		Security(planeAdminSessionCookie)
		Payload(func() {
			instanceAdminSessionPayload()
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("pk")
		})
		instanceManagementErrors()
		HTTP(func() {
			GET("/api/instances/admins/{pk}/")
			Cookie("session_id:admin-session-id")
			Param("pk")
		})
	})
	Method("admins_detail_delete", func() {
		Meta("openapi:operationId", "delete_api_instances_admins_by_pk")
		Security(planeAdminSessionCookie)
		Payload(func() {
			instanceAdminSessionPayload()
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("pk")
		})
		instanceManagementErrors()
		HTTP(func() {
			DELETE("/api/instances/admins/{pk}/")
			Cookie("session_id:admin-session-id")
			Param("pk")
			Response(StatusNoContent)
		})
	})
	Method("signup_visited", func() {
		Meta("openapi:operationId", "post_api_instances_admins_sign_up_screen_visited")
		instanceManagementStatusError("throttled", 429)
		Payload(func() {})
		instanceManagementStatusError("bad_request", 400)
		instanceManagementStatusError("internal_error", 500)
		HTTP(func() {
			POST("/api/instances/admins/sign-up-screen-visited/")
			Response(StatusNoContent)
		})
	})
	Method("config_get", func() {
		Meta("openapi:operationId", "get_api_instances_configurations")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload() })
		Result(rawJSONResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			GET("/api/instances/configurations/")
			Cookie("session_id:admin-session-id")
			instanceManagementRawResponse(ArrayOf(Any), StatusOK)
		})
	})
	Method("telemetry_get", func() {
		Meta("openapi:operationId", "get_api_instances_telemetry")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload() })
		Result(instanceRootResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			GET("/api/instances/telemetry/")
			Cookie("session_id:admin-session-id")
			SkipResponseBodyEncodeDecode()
			Response(StatusOK, func() {
				Header("content_type:Content-Type")
				Header("cache_control:Cache-Control")
				ContentType("application/json")
				OpenAPIBody(Any)
			})
		})
	})
	Method("config_patch", func() {
		Meta("openapi:operationId", "patch_api_instances_configurations")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload(); instanceManagementBodyHeader() })
		Result(rawJSONResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			PATCH("/api/instances/configurations/")
			Cookie("session_id:admin-session-id")
			instanceManagementRequestBody()
			instanceManagementRawResponse(ArrayOf(Any), StatusOK)
		})
	})
	Method("disable_email", func() {
		Meta("openapi:operationId", "delete_api_instances_configurations_disable_email_feature")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload() })
		instanceManagementErrors()
		HTTP(func() {
			DELETE("/api/instances/configurations/disable-email-feature/")
			Cookie("session_id:admin-session-id")
			Response(StatusOK)
		})
	})
	Method("email_check", func() {
		Meta("openapi:operationId", "post_api_instances_email_credentials_check")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload(); instanceManagementBodyHeader() })
		Result(rawJSONResponseMetadata)
		instanceManagementStatusError("bad_request", 400)
		Error("unauthenticated", instanceAdminSessionUnauthenticated)
		Error("forbidden", instanceAdminSessionForbidden)
		instanceManagementStatusError("internal_error", 500)
		HTTP(func() {
			POST("/api/instances/email-credentials-check/")
			Cookie("session_id:admin-session-id")
			instanceManagementRequestBody()
			instanceManagementRawResponse(Any, StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
		})
	})
	Method("workspaces_get", func() {
		Meta("openapi:operationId", "get_api_instances_workspaces")
		Security(planeAdminSessionCookie)
		Payload(func() {
			instanceAdminSessionPayload()
			Attribute("search", String)
			Attribute("per_page", String)
			Attribute("cursor", String)
		})
		Result(rawJSONResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			GET("/api/instances/workspaces/")
			Cookie("session_id:admin-session-id")
			Param("search")
			Param("per_page")
			Param("cursor")
			instanceManagementRawResponse(Any, StatusOK)
		})
	})
	Method("workspaces_post", func() {
		Meta("openapi:operationId", "post_api_instances_workspaces")
		Security(planeAdminSessionCookie)
		Payload(func() { instanceAdminSessionPayload(); instanceManagementBodyHeader() })
		Result(rawJSONResponseMetadata)
		instanceManagementErrors()
		HTTP(func() {
			POST("/api/instances/workspaces/")
			Cookie("session_id:admin-session-id")
			instanceManagementRequestBody()
			instanceManagementRawResponse(Any, StatusCreated)
		})
	})
})
