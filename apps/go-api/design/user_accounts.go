package design

import . "github.com/CaliLuke/loom/dsl"

var userAccount = Type("UserAccount", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("user", String, func() { Format(FormatUUID) })
	Attribute("provider_account_id", String)
	Attribute("provider", String, func() { Enum("google", "github", "gitlab") })
	Attribute("access_token", String)
	Attribute("access_token_expired_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("refresh_token", String, func() { Nullable() })
	Attribute("refresh_token_expired_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("last_connected_at", String, func() { Format(FormatDateTime) })
	Attribute("id_token", String)
	Attribute("metadata", Any)
	Required("id", "created_at", "updated_at", "user", "provider_account_id", "provider", "access_token", "last_connected_at", "id_token", "metadata")
})

var userAccountUnauthenticated = Type("UserAccountUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var userAccountNotFound = Type("UserAccountNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var userAccountInternalError = Type("UserAccountInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("user_accounts", func() {
	Description("Lists, retrieves, and disconnects the authenticated user's linked accounts.")

	Method("list", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload() })
		Result(ArrayOf(userAccount))
		Error("unauthenticated", userAccountUnauthenticated)
		Error("internal_error", userAccountInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/users/me/accounts/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("delete_collection", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload() })
		Error("unauthenticated", userAccountUnauthenticated)
		Error("internal_error", userAccountInternalError, func() { Fault() })
		HTTP(func() {
			DELETE("/api/users/me/accounts/")
			Cookie("session_id:session-id")
			Response(StatusNoContent)
			Response("unauthenticated", StatusUnauthorized)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("retrieve", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("pk")
		})
		Result(userAccount)
		Error("unauthenticated", userAccountUnauthenticated)
		Error("not_found", userAccountNotFound)
		Error("internal_error", userAccountInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/users/me/accounts/{pk}/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("delete", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("pk", String, func() { Format(FormatUUID) })
			Required("pk")
		})
		Error("unauthenticated", userAccountUnauthenticated)
		Error("not_found", userAccountNotFound)
		Error("internal_error", userAccountInternalError, func() { Fault() })
		HTTP(func() {
			DELETE("/api/users/me/accounts/{pk}/")
			Cookie("session_id:session-id")
			Response(StatusNoContent)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
