// Package linkdesign owns browser account-linking HTTP contracts.
package design

//go:generate go tool loom gen github.com/CaliLuke/go-plane/mcpoauthlinks/design --output ..

import (
	"strconv"

	. "github.com/CaliLuke/loom/dsl"
)

var cookie = APIKeySecurity("MCPOAuthLinkCookie", func() {})
var session = SessionAuth("MCPOAuthLinkSession", func() { CookieTransport(cookie, "", func() { CookieName("session-id") }) })

var failure = Type("LinkError", func() {
	ErrorName("kind", String, func() { Meta("struct:tag:json", "-") })
	Attribute("message", String)
	Required("kind", "message")
})

var pending = Type("PendingLink", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("subject", String)
	Attribute("expires_at", String, func() { Format(FormatDateTime) })
	Required("id", "subject", "expires_at")
})

var status = Type("LinkStatus", func() {
	Attribute("issuer", String)
	Attribute("connected", Boolean)
	Attribute("subject", String, func() { Nullable() })
	Attribute("pending", pending, func() { Nullable() })
	Required("issuer", "connected", "subject", "pending")
})

var _ = API("Vector MCP Account Linking", func() {
	Title("Vector MCP Account Linking")
	Version("1.0")
	Meta("http:generate", "server")
	HTTP(func() { Produces("application/json") })
})

var _ = Service("mcp_oauth_links", func() {
	HTTP(func() { RuntimeCORS() })
	Method("Status", func() {
		SessionSecurity(session)
		Payload(func() { Attribute("proof", String) })
		Result(status)
		linkErrors()
		HTTP(func() {
			GET("/api/users/me/mcp-link/")
			Cookie("proof:__Host-vector-mcp-link")
			Response(StatusOK)
			linkHTTPFailures()
		})
	})
	Method("Start", func() {
		SessionSecurity(session)
		Result(func() {
			Attribute("authorization_url", String)
			Attribute("set_cookie", String)
			Required("authorization_url", "set_cookie")
		})
		linkErrors()
		HTTP(func() {
			POST("/api/users/me/mcp-link/start/")
			Response(StatusOK, func() { Header("set_cookie:Set-Cookie") })
			linkHTTPFailures()
		})
	})
	Method("Confirm", func() {
		SessionSecurity(session)
		Payload(func() {
			Attribute("proof", String)
			Attribute("id", String, func() { Format(FormatUUID) })
			Required("id")
		})
		Result(func() { Attribute("set_cookie", String); Required("set_cookie") })
		linkErrors()
		HTTP(func() {
			POST("/api/users/me/mcp-link/confirm/")
			Cookie("proof:__Host-vector-mcp-link")
			Response(StatusNoContent, func() { Header("set_cookie:Set-Cookie") })
			linkHTTPFailures()
		})
	})
	Method("Disconnect", func() {
		SessionSecurity(session)
		Result(func() { Attribute("set_cookie", String); Required("set_cookie") })
		linkErrors()
		HTTP(func() {
			DELETE("/api/users/me/mcp-link/")
			Response(StatusNoContent, func() { Header("set_cookie:Set-Cookie") })
			linkHTTPFailures()
		})
	})
	Method("Callback", func() {
		Payload(func() {
			Attribute("state", String, func() { MinLength(43); MaxLength(43) })
			Attribute("code", String, func() { MaxLength(8192) })
			Attribute("error", String, func() { MaxLength(255) })
			Attribute("iss", String, func() { MaxLength(2048) })
			Required("state")
		})
		Result(func() { Attribute("location", String); Attribute("set_cookie", String); Required("location") })
		linkErrors()
		HTTP(func() {
			POST("/auth/mcp/callback/")
			FormRequest()
			Response(StatusSeeOther, func() { Header("location:Location"); Header("set_cookie:Set-Cookie") })
			linkHTTPFailures()
		})
	})
})

func linkErrors() {
	for _, code := range []int{400, 401, 403, 409, 503} {
		Error("status_"+strconv.Itoa(code), failure)
	}
}
func linkHTTPFailures() {
	for _, code := range []int{400, 401, 403, 409, 503} {
		Response("status_"+strconv.Itoa(code), code)
	}
}
