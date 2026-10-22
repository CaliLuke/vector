// Package design defines the public Go API contract.
//
//go:generate go tool loom gen github.com/CaliLuke/go-plane/design --output ..
package design

import . "github.com/CaliLuke/loom/dsl"

var csrfTokenResult = Type("CSRFTokenResult", func() {
	Field(1, "csrf_token", String, func() {
		MinLength(64)
		MaxLength(64)
		Pattern("^[a-zA-Z0-9]+$")
	})
	Field(2, "csrf_cookie", String, func() {
		MinLength(32)
		MaxLength(32)
		Pattern("^[a-zA-Z0-9]+$")
	})
	Required("csrf_token", "csrf_cookie")
})

var csrfTokenOptionsResult = Type("CSRFTokenOptionsResult", func() {
	Field(1, "name", String)
	Field(2, "description", String)
	Field(3, "renders", ArrayOf(String))
	Field(4, "parses", ArrayOf(String))
	Required("name", "description", "renders", "parses")
})

var _ = API("system", func() {
	HTTP(func() {
		Consumes("application/json", "application/x-www-form-urlencoded", "multipart/form-data")
		Produces("application/json")
	})
})

var _ = Service("csrf_token", func() {
	Description("Issues Django-compatible CSRF tokens and endpoint metadata.")
	HTTP(func() {
		RuntimeCORS()
	})

	Method("csrf_token", func() {
		Description("Returns a freshly masked CSRF token backed by the client's CSRF cookie secret.")
		Security()
		SessionSecurity(planeDjangoSession)
		Payload(func() {
			Field(1, "csrf_cookie", String)
		})
		Result(csrfTokenResult)

		HTTP(func() {
			GET("/auth/get-csrf-token/")
			Cookie("csrf_cookie:csrftoken")
			Response(StatusOK, func() {
				Cookie("csrf_cookie:csrftoken", String, func() {
					MinLength(32)
					MaxLength(32)
					Pattern("^[a-zA-Z0-9]+$")
				})
				CookieMaxAge(31_449_600)
				CookiePath("/")
				CookieHTTPOnly()
				CookieSameSite(CookieSameSiteLax)
			})
		})
	})

	Method("csrf_token_options", func() {
		Description("Describes the CSRF token endpoint's supported representations.")
		Security()
		SessionSecurity(planeDjangoSession)
		Payload(func() {
			Field(1, "format", String)
		})
		Result(csrfTokenOptionsResult)
		HTTP(func() {
			OPTIONS("/auth/get-csrf-token/")
			Param("format")
			Response(StatusOK)
		})
	})
})

var _ = Service("system", func() {
	Description("Reports the state of the Go backend process.")

	Method("live", func() {
		Description("Reports whether the process can serve requests.")
		Result(func() {
			Field(1, "status", String, "Process state", func() {
				Example("ok")
			})
			Required("status")
		})

		HTTP(func() {
			GET("/health/live")
			Response(StatusOK)
		})
	})

	Method("ready", func() {
		Description("Reports whether the process can query its PostgreSQL database.")
		Result(func() {
			Field(1, "status", String, "Database state", func() {
				Example("ok")
			})
			Required("status")
		})

		HTTP(func() {
			GET("/health/ready")
			Response(StatusOK)
		})
	})
})
