package design

import . "github.com/CaliLuke/loom/dsl"

var userNotificationPreference = Type("UserNotificationPreference", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("created_at", String, func() { Format(FormatDateTime) })
	Attribute("updated_at", String, func() { Format(FormatDateTime) })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("user", String, func() { Format(FormatUUID) })
	Attribute("workspace", String, func() { Format(FormatUUID); Nullable() })
	Attribute("project", String, func() { Format(FormatUUID); Nullable() })
	Attribute("property_change", Boolean)
	Attribute("state_change", Boolean)
	Attribute("comment", Boolean)
	Attribute("mention", Boolean)
	Attribute("issue_completed", Boolean)
	Required("id", "created_at", "updated_at", "user", "property_change", "state_change", "comment", "mention", "issue_completed")
})

var userNotificationPreferenceError = Type("UserNotificationPreferenceError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var userNotificationPreferenceUnauthenticated = Type("UserNotificationPreferenceUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var userNotificationPreferenceInternalError = Type("UserNotificationPreferenceInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var userNotificationPreferenceValidationError = Type("UserNotificationPreferenceValidationError", func() {
	Attribute("property_change", ArrayOf(String))
	Attribute("state_change", ArrayOf(String))
	Attribute("comment", ArrayOf(String))
	Attribute("mention", ArrayOf(String))
	Attribute("issue_completed", ArrayOf(String))
	Attribute("created_by", ArrayOf(String))
	Attribute("updated_by", ArrayOf(String))
	Attribute("deleted_at", ArrayOf(String))
	Attribute("user", ArrayOf(String))
	Attribute("workspace", ArrayOf(String))
	Attribute("project", ArrayOf(String))
	Required()
})

var userNotificationPreferencePatch = Type("UserNotificationPreferencePatch", func() {
	Attribute("created_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("updated_by", String, func() { Format(FormatUUID); Nullable() })
	Attribute("deleted_at", String, func() { Format(FormatDateTime); Nullable() })
	Attribute("user", String, func() { Format(FormatUUID); Nullable() })
	Attribute("workspace", String, func() { Format(FormatUUID); Nullable() })
	Attribute("project", String, func() { Format(FormatUUID); Nullable() })
	Attribute("property_change", Any, func() { Nullable() })
	Attribute("state_change", Any, func() { Nullable() })
	Attribute("comment", Any, func() { Nullable() })
	Attribute("mention", Any, func() { Nullable() })
	Attribute("issue_completed", Any, func() { Nullable() })
})

var _ = Service("user_notification_preferences", func() {
	Description("Reads and updates the authenticated user's notification preferences.")
	Method("get", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload() })
		Result(userNotificationPreference)
		Error("unauthenticated", userNotificationPreferenceUnauthenticated)
		Error("not_found", userNotificationPreferenceError)
		Error("validation", userNotificationPreferenceValidationError)
		Error("internal_error", userNotificationPreferenceInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/users/me/notification-preferences/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("validation", StatusBadRequest)
			Response("internal_error", StatusInternalServerError)
		})
	})
	Method("patch", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("body", userNotificationPreferencePatch)
		})
		Result(userNotificationPreference)
		Error("unauthenticated", userNotificationPreferenceUnauthenticated)
		Error("not_found", userNotificationPreferenceError)
		Error("validation", userNotificationPreferenceValidationError)
		Error("internal_error", userNotificationPreferenceInternalError, func() { Fault() })
		HTTP(func() {
			PATCH("/api/users/me/notification-preferences/")
			Cookie("session_id:session-id")
			Body("body")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("validation", StatusBadRequest)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
