// Package design defines Plane's form-based authentication redirects.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/authredirects/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var authForm = Type("AuthRedirectForm", func() {
	Attribute("email", String)
	Attribute("password", String)
	Attribute("code", String)
	Attribute("next_path", String)
})

var redirect = Type("AuthRedirectResponse", func() {
	Attribute("location", String)
	Attribute("content_type", String)
	Required("location", "content_type")
})

var authError = Type("AuthRedirectError", func() {
	ErrorName("name", String, func() { Meta("struct:tag:json", "-") })
	Attribute("body", Any)
	Required("name", "body")
})

func postRedirect(name, operationID, route string) {
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		Meta("openapi:extension:x-plane-contract-id", "\"POST "+route+"\"")
		Result(redirect)
		Error("status_500", authError, func() { Fault() })
		HTTP(func() {
			POST(route)
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(authForm, []string{"application/x-www-form-urlencoded", "multipart/form-data"}, false)
			SkipResponseBodyEncodeDecode()
			Response(302, func() {
				Header("location:Location")
				Header("content_type:Content-Type")
			})
			Response("status_500", 500, func() {
				ContentType("text/html")
				ErrorNameFromStatus()
				Body("body")
			})
		})
	})
}

func postResetRedirect(name, operationID, route string) {
	Method(name, func() {
		Meta("openapi:operationId", operationID)
		Meta("openapi:extension:x-plane-contract-id", "\"POST "+route+"\"")
		Payload(func() {
			Attribute("uidb64", String)
			Attribute("token", String)
			Required("uidb64", "token")
		})
		Result(redirect)
		Error("status_500", authError, func() { Fault() })
		HTTP(func() {
			POST(route)
			Param("uidb64")
			Param("token")
			SkipRequestBodyEncodeDecode()
			OpenAPIRequestBodyTypes(authForm, []string{"application/x-www-form-urlencoded", "multipart/form-data"}, false)
			SkipResponseBodyEncodeDecode()
			Response(302, func() {
				Header("location:Location")
				Header("content_type:Content-Type")
			})
			Response("status_500", 500, func() {
				ContentType("text/html")
				ErrorNameFromStatus()
				Body("body")
			})
		})
	})
}

var _ = API("Plane Authentication Redirects", func() {
	Title("Plane Authentication Redirects")
	Version("1.0.0")
	Meta("openapi:version", "3.1")
	Meta("openapi:example", "false")
	Meta("http:generate", "server")
	HTTP(func() {
		Consumes("application/x-www-form-urlencoded", "multipart/form-data")
		Produces("text/html")
	})
})

var _ = Service("AuthRedirects", func() {
	HTTP(func() { RuntimeCORS() })
	postRedirect("EmailSignIn", "post_auth_sign_in", "/auth/sign-in/")
	postRedirect("EmailSignUp", "post_auth_sign_up", "/auth/sign-up/")
	postRedirect("SpaceEmailSignIn", "post_auth_spaces_sign_in", "/auth/spaces/sign-in/")
	postRedirect("SpaceEmailSignUp", "post_auth_spaces_sign_up", "/auth/spaces/sign-up/")
	postRedirect("SignOut", "post_auth_sign_out", "/auth/sign-out/")
	postRedirect("SpaceSignOut", "post_auth_spaces_sign_out", "/auth/spaces/sign-out/")
	postRedirect("MagicSignIn", "post_auth_magic_sign_in", "/auth/magic-sign-in/")
	postRedirect("MagicSignUp", "post_auth_magic_sign_up", "/auth/magic-sign-up/")
	postRedirect("SpaceMagicSignIn", "post_auth_spaces_magic_sign_in", "/auth/spaces/magic-sign-in/")
	postRedirect("SpaceMagicSignUp", "post_auth_spaces_magic_sign_up", "/auth/spaces/magic-sign-up/")
	postResetRedirect("ResetPassword", "post_auth_reset_password_by_uidb64_by_token", "/auth/reset-password/{uidb64}/{token}/")
	postResetRedirect("SpaceResetPassword", "post_auth_spaces_reset_password_by_uidb64_by_token", "/auth/spaces/reset-password/{uidb64}/{token}/")
})
