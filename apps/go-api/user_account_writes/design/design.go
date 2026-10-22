// Package design defines the authenticated current-user write contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/user_account_writes/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var userAccountSessionCookie = APIKeySecurity("user_account_writes_session_cookie", func() {
	Description("Plane app session cookie.")
})

func userAccountSessionPayload() {
	APIKey("user_account_writes_session_cookie", "session_id", String)
}

var userAccountWritesResponseMetadata = Type("UserAccountWritesResponseMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var userAccountWritesError = Type("UserAccountWritesError", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Attribute("retry_after", Int)
	Required("name", "body")
})

var userAccountPatchRequest = Type("UserAccountPatchRequest", func() {
	Attribute("mobile_number", String, func() { Nullable() })
	Attribute("display_name", String)
	Attribute("first_name", String)
	Attribute("last_name", String)
	Attribute("avatar", String)
	Attribute("avatar_asset", String, func() { Format(FormatUUID); Nullable() })
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("cover_image_asset", String, func() { Format(FormatUUID); Nullable() })
	Attribute("last_location", String)
	Attribute("created_location", String)
	Attribute("is_password_expired", Boolean)
	Attribute("is_password_reset_required", Boolean)
	Attribute("bot_type", String, func() { Nullable() })
	Attribute("user_timezone", String)
	Attribute("is_email_valid", Boolean)
	Attribute("masked_at", String, func() { Format(FormatDateTime); Nullable() })
})

var userAccountEmailUpdateRequest = Type("UserAccountEmailUpdateRequest", func() {
	Attribute("email", String)
	Attribute("code", String)
})

var userAccountEmailCodeRequest = Type("UserAccountEmailCodeRequest", func() {
	Attribute("email", String)
})

func userAccountRawResponse(schema any, status int) {
	SkipResponseBodyEncodeDecode()
	Response(status, func() {
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(schema)
	})
}

func userAccountRequestBody(schema any, required bool) {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(schema, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, required)
	Header("content_type:Content-Type")
}

func userAccountCommonErrors() {
	Error("unauthenticated", userAccountWritesError)
	Error("bad_request", userAccountWritesError)
	Error("rate_limited", userAccountWritesError, func() { Temporary() })
	Error("unsupported_media_type", userAccountWritesError)
	Error("internal_error", userAccountWritesError, func() { Fault() })
	HTTP(func() {
		Response("unauthenticated", StatusUnauthorized, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
		Response("bad_request", StatusBadRequest, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
		Response("unsupported_media_type", StatusUnsupportedMediaType, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
		Response("rate_limited", StatusTooManyRequests, func() {
			ContentType("application/json")
			ErrorNameFromStatus()
			Header("retry_after:Retry-After")
			Body("body")
		})
		Response("internal_error", StatusInternalServerError, func() { ContentType("application/json"); ErrorNameFromStatus(); Body("body") })
	})
}

var _ = API("Plane User Account Writes API", func() {
	Meta("http:generate", "all")
	Title("Plane User Account Writes API")
	Description("Authenticated operations for the current user's account.")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("user_account_writes", func() {
	Description("Updates or deactivates the authenticated user's account.")
	HTTP(func() { RuntimeCORS() })

	Method("patch_me", func() {
		Meta("openapi:operationId", "patch_api_users_me")
		Security(userAccountSessionCookie)
		Payload(func() { userAccountSessionPayload(); Attribute("content_type", String) })
		Result(userAccountWritesResponseMetadata)
		userAccountCommonErrors()
		HTTP(func() {
			PATCH("/api/users/me/")
			Cookie("session_id:session-id")
			userAccountRequestBody(userAccountPatchRequest, false)
			userAccountRawResponse(Any, StatusOK)
		})
	})

	Method("deactivate", func() {
		Meta("openapi:operationId", "delete_api_users_me")
		Security(userAccountSessionCookie)
		Payload(func() { userAccountSessionPayload() })
		userAccountCommonErrors()
		HTTP(func() {
			DELETE("/api/users/me/")
			Cookie("session_id:session-id")
			SkipResponseBodyEncodeDecode()
			Response(StatusNoContent)
		})
	})

	Method("update_email", func() {
		Meta("openapi:operationId", "patch_api_users_me_email")
		Security(userAccountSessionCookie)
		Payload(func() { userAccountSessionPayload(); Attribute("content_type", String) })
		Result(userAccountWritesResponseMetadata)
		userAccountCommonErrors()
		HTTP(func() {
			PATCH("/api/users/me/email/")
			Cookie("session_id:session-id")
			userAccountRequestBody(userAccountEmailUpdateRequest, true)
			userAccountRawResponse(Any, StatusOK)
		})
	})

	Method("generate_email_code", func() {
		Meta("openapi:operationId", "post_api_users_me_email_generate_code")
		Security(userAccountSessionCookie)
		Payload(func() { userAccountSessionPayload(); Attribute("content_type", String) })
		Result(userAccountWritesResponseMetadata)
		userAccountCommonErrors()
		HTTP(func() {
			POST("/api/users/me/email/generate-code/")
			Cookie("session_id:session-id")
			userAccountRequestBody(userAccountEmailCodeRequest, true)
			userAccountRawResponse(Any, StatusOK)
		})
	})
})
