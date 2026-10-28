// Package design defines the selected public and authenticated account-authentication API.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/accountauthentication/design --output ..
package design

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var accountAuthenticationSession = APIKeySecurity("PlaneSessionCookie", func() {
	Description("Plane application session cookie authentication.")
})

var accountAuthenticationRawResponse = Type("AccountAuthenticationRawResponse", func() {
	Attribute("content_type", String)
	Required("content_type")
})

var accountAuthenticationError = Type("AccountAuthenticationError", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Attribute("retry_after", Int)
	Required("name", "body")
})

var accountAuthenticationRequest = MapOf(String, Any)

func accountAuthenticationErrors(authenticated bool) {
	for _, status := range []int{400, 415, 429, 500} {
		name := "status_" + strconv.Itoa(status)
		Error(name, accountAuthenticationError, func() {
			if status == 429 {
				Temporary()
			}
			if status == 500 {
				Fault()
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
	if authenticated {
		Error("unauthenticated", accountAuthenticationError)
		HTTP(func() {
			Response("unauthenticated", 401, func() {
				ContentType("application/json")
				ErrorNameFromStatus()
				Body("body")
			})
		})
	}
}

func accountAuthenticationBody(schema any) {
	SkipRequestBodyEncodeDecode()
	OpenAPIRequestBodyTypes(schema, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)
	Header("content_type:Content-Type")
}

func accountAuthenticationResponse() {
	SkipResponseBodyEncodeDecode()
	Response(200, func() {
		Header("content_type:Content-Type")
		ContentType("application/json")
		OpenAPIBody(Any)
	})
}

var _ = API("Account Authentication", func() {
	Title("Account Authentication")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	HTTP(func() {
		RuntimeCORS()
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("account_authentication", func() {
	Description("Handles account email discovery, password management, and authentication-code requests.")
	Method("email_check", func() {
		Meta("openapi:operationId", "post_auth_email_check")
		Payload(func() {
			Attribute("content_type", String)
		})
		Result(accountAuthenticationRawResponse)
		accountAuthenticationErrors(false)
		HTTP(func() {
			POST("/auth/email-check/")
			accountAuthenticationBody(accountAuthenticationRequest)
			accountAuthenticationResponse()
		})
	})
	Method("email_check_space", func() {
		Meta("openapi:operationId", "post_auth_spaces_email_check")
		Payload(func() {
			Attribute("content_type", String)
		})
		Result(accountAuthenticationRawResponse)
		accountAuthenticationErrors(false)
		HTTP(func() {
			POST("/auth/spaces/email-check/")
			accountAuthenticationBody(accountAuthenticationRequest)
			accountAuthenticationResponse()
		})
	})
	Method("forgot_password", func() {
		Meta("openapi:operationId", "post_auth_forgot_password")
		Payload(func() {
			Attribute("content_type", String)
		})
		Result(accountAuthenticationRawResponse)
		accountAuthenticationErrors(false)
		HTTP(func() {
			POST("/auth/forgot-password/")
			accountAuthenticationBody(accountAuthenticationRequest)
			accountAuthenticationResponse()
		})
	})
	Method("forgot_password_space", func() {
		Meta("openapi:operationId", "post_auth_spaces_forgot_password")
		Payload(func() {
			Attribute("content_type", String)
		})
		Result(accountAuthenticationRawResponse)
		accountAuthenticationErrors(false)
		HTTP(func() {
			POST("/auth/spaces/forgot-password/")
			accountAuthenticationBody(accountAuthenticationRequest)
			accountAuthenticationResponse()
		})
	})
	Method("magic_generate", func() {
		Meta("openapi:operationId", "post_auth_magic_generate")
		Payload(func() {
			Attribute("content_type", String)
		})
		Result(accountAuthenticationRawResponse)
		accountAuthenticationErrors(false)
		HTTP(func() {
			POST("/auth/magic-generate/")
			accountAuthenticationBody(accountAuthenticationRequest)
			accountAuthenticationResponse()
		})
	})
	Method("magic_generate_space", func() {
		Meta("openapi:operationId", "post_auth_spaces_magic_generate")
		Payload(func() {
			Attribute("content_type", String)
		})
		Result(accountAuthenticationRawResponse)
		accountAuthenticationErrors(false)
		HTTP(func() {
			POST("/auth/spaces/magic-generate/")
			accountAuthenticationBody(accountAuthenticationRequest)
			accountAuthenticationResponse()
		})
	})
	Method("change_password", func() {
		Meta("openapi:operationId", "post_auth_change_password")
		Security(accountAuthenticationSession)
		Payload(func() {
			APIKey("PlaneSessionCookie", "session_id", String)
			Attribute("content_type", String)
		})
		Result(accountAuthenticationRawResponse)
		accountAuthenticationErrors(true)
		HTTP(func() {
			POST("/auth/change-password/")
			Cookie("session_id:session-id")
			accountAuthenticationBody(accountAuthenticationRequest)
			accountAuthenticationResponse()
		})
	})
	Method("set_password", func() {
		Meta("openapi:operationId", "post_auth_set_password")
		Security(accountAuthenticationSession)
		Payload(func() {
			APIKey("PlaneSessionCookie", "session_id", String)
			Attribute("content_type", String)
		})
		Result(accountAuthenticationRawResponse)
		accountAuthenticationErrors(true)
		HTTP(func() {
			POST("/auth/set-password/")
			Cookie("session_id:session-id")
			accountAuthenticationBody(accountAuthenticationRequest)
			accountAuthenticationResponse()
		})
	})
})
