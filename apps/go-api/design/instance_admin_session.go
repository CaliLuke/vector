package design

import . "github.com/CaliLuke/loom/dsl"

var instanceAdminUser = Type("InstanceAdminUser", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("avatar", String)
	Attribute("avatar_url", String, func() { Format(FormatURI); Nullable() })
	Attribute("cover_image", String, func() { Nullable() })
	Attribute("date_joined", String, func() { Format(FormatDateTime) })
	Attribute("display_name", String)
	Attribute("email", String, func() { Format(FormatEmail); Nullable() })
	Attribute("first_name", String)
	Attribute("last_name", String)
	Attribute("is_active", Boolean)
	Attribute("is_bot", Boolean)
	Attribute("is_email_verified", Boolean)
	Attribute("user_timezone", String)
	Attribute("username", String)
	Attribute("is_password_autoset", Boolean)
	Required("id", "avatar", "avatar_url", "cover_image", "date_joined", "display_name", "email", "first_name", "last_name", "is_active", "is_bot", "is_email_verified", "user_timezone", "username", "is_password_autoset")
})

var instanceAdminSessionUnauthenticated = Type("InstanceAdminSessionUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var instanceAdminSessionForbidden = Type("InstanceAdminSessionForbidden", func() {
	Attribute("detail", String)
	Required("detail")
})

var instanceAdminSessionInternalError = Type("InstanceAdminSessionInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("instance_admin_session", func() {
	Method("me", func() {
		Security(planeAdminSessionCookie)
		Payload(func() {
			APIKey("plane_admin_session_cookie", "session_id", String)
		})
		Result(instanceAdminUser)
		Error("unauthenticated", instanceAdminSessionUnauthenticated)
		Error("forbidden", instanceAdminSessionForbidden)
		Error("internal_error", instanceAdminSessionInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/instances/admins/me/")
			Cookie("session_id:admin-session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("forbidden", StatusForbidden)
			Response("internal_error", StatusInternalServerError)
		})
	})
	Method("session", func() {
		Payload(func() {
			Attribute("session_id", String)
		})
		Result(func() {
			Attribute("is_authenticated", Boolean)
			Attribute("user", instanceAdminUser, func() { Meta("openapi:extension:x-nullable", "true") })
			Required("is_authenticated")
		})
		Error("internal_error", instanceAdminSessionInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/instances/admins/session/")
			Cookie("session_id:admin-session-id")
			Response(StatusOK)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
