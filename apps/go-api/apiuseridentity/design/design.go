package design

//go:generate go tool loom gen github.com/CaliLuke/go-plane/apiuseridentity/design --output ..

import . "github.com/CaliLuke/loom/dsl"

var ImportedPlaneAPISerializersUserUserliteserializerResponse = Type("PlaneApiSerializersUserUserliteserializerResponse", func() {
	Meta("openapi:typename:canonical", "true")
	Description("Lightweight user serializer for minimal data transfer.\n\nProvides essential user information including names, avatar, and contact details\noptimized for member lists, assignee displays, and user references.")
	Attribute("avatar", String, func() {
		Meta("openapi:readOnly", "true")
	})
	Attribute("avatar_url", String, func() {
		// Source fixtures include null and omission for users without a workspace.
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

var ImportedAPIKeyAuthSecurity = APIKeySecurity("ApiKeyAuth", func() {
	Description("Plane API token authentication.")
})

var apiUserIdentityRawJSONResponseMetadata = Type("APIUserIdentityRawJSONResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

func apiUserIdentityRawJSONResponse(schema any, description string) {
	SkipResponseBodyEncodeDecode()
	Response(200, func() {
		Description(description)
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

var apiUserIdentityErrorBody = Type("APIUserIdentityErrorBody", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
	Required("name")
})

var apiUserIdentityHTTPErrorBody = Type("APIUserIdentityHTTPErrorBody", func() {
	Attribute("detail", Any)
	Attribute("error", Any, func() { Meta("struct:field:name", "ErrorValue") })
	Attribute("error_code", Any)
	Attribute("error_message", Any)
	Attribute("code", Any)
	Attribute("message", Any)
	Attribute("status", Any)
})

func apiUserIdentityErrors() {
	Error("status_400", apiUserIdentityErrorBody)
	Error("status_401", apiUserIdentityErrorBody)
	Error("status_403", apiUserIdentityErrorBody)
	Error("status_404", apiUserIdentityErrorBody)
	Error("status_405", apiUserIdentityErrorBody)
	Error("status_406", apiUserIdentityErrorBody)
	Error("status_429", apiUserIdentityErrorBody, func() { Temporary() })
	Error("status_500", apiUserIdentityErrorBody, func() { Fault() })
}

func apiUserIdentityHTTPErrorResponses() {
	Response("status_400", 400, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(apiUserIdentityHTTPErrorBody) })
	Response("status_401", 401, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(apiUserIdentityHTTPErrorBody) })
	Response("status_403", 403, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(apiUserIdentityHTTPErrorBody) })
	Response("status_404", 404, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(apiUserIdentityHTTPErrorBody) })
	Response("status_405", 405, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(apiUserIdentityHTTPErrorBody) })
	Response("status_406", 406, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(apiUserIdentityHTTPErrorBody) })
	Response("status_429", 429, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(apiUserIdentityHTTPErrorBody) })
	Response("status_500", 500, func() { ContentType("application/json"); ErrorNameFromStatus(); Body(apiUserIdentityHTTPErrorBody) })
}

var _ = API("API User Identity", func() {
	Title("API User Identity")
	Description("Internal full-server migration schema. Route inventory and behavior fixtures remain authoritative. This artifact does not change the public /api/v1 schema.")
	Version("1.0.0")
	Meta("openapi:example", "false")
	Meta("openapi:version", "3.1")
	Meta("openapi:tag:users")
})

var _ = Service("APIUserIdentity", func() {
	HTTP(func() {
		RuntimeCORS()
	})
	Method("GetAPIV1UsersMe", func() {
		Description("Retrieve the authenticated user's profile information including basic details.")
		Meta("openapi:operationId", "get_api_v1_users_me")
		Meta("openapi:summary", "Get current user")
		Meta("openapi:extension:x-plane-authentication-classes", "[\"plane.api.middleware.api_authentication.APIKeyAuthentication\"]")
		Meta("openapi:extension:x-plane-authority", "\"route-inventory-and-behavior-fixtures\"")
		Meta("openapi:extension:x-plane-contract-id", "\"GET /api/v1/users/me/\"")
		Meta("openapi:extension:x-plane-handler", "\"get\"")
		Meta("openapi:extension:x-plane-permission-classes", "[\"rest_framework.permissions.IsAuthenticated\"]")
		Meta("openapi:extension:x-plane-view", "\"plane.api.views.user.UserEndpoint\"")
		Security(ImportedAPIKeyAuthSecurity)
		Payload(func() {
			APIKey("ApiKeyAuth", "apiKeyAuthCredential", String)
		})
		Result(apiUserIdentityRawJSONResponseMetadata)
		apiUserIdentityErrors()
		HTTP(func() {
			Meta("openapi:tag:users")
			GET("/api/v1/users/me/")
			Header("apiKeyAuthCredential:X-Api-Key")
			apiUserIdentityRawJSONResponse(ImportedPlaneAPISerializersUserUserliteserializerResponse, "Current user profile")
			apiUserIdentityHTTPErrorResponses()
		})
	})
})
