package design

import . "github.com/CaliLuke/loom/dsl"

var userSettingsWorkspace = Type("UserSettingsWorkspace", func() {
	Attribute("last_workspace_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("last_workspace_slug", String, func() { Nullable() })
	Attribute("last_workspace_name", String, func() { Nullable() })
	Attribute("last_workspace_logo", String, func() { Nullable() })
	Attribute("fallback_workspace_id", String, func() { Format(FormatUUID); Nullable() })
	Attribute("fallback_workspace_slug", String, func() { Nullable() })
	Attribute("invites", Int)
	Required("last_workspace_id", "last_workspace_slug", "fallback_workspace_id", "fallback_workspace_slug", "invites")
})

var userSettings = Type("UserSettings", func() {
	Attribute("id", String, func() { Format(FormatUUID) })
	Attribute("email", String, func() { Meta("loom:vet:ignore", "string-format") })
	Attribute("workspace", userSettingsWorkspace)
	Required("id", "email", "workspace")
})

var userSettingsUnauthenticated = Type("UserSettingsUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})
var userSettingsInternal = Type("UserSettingsInternal", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("user_settings", func() {
	Description("Returns the authenticated user's workspace selection and pending invite count.")
	Method("get", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload() })
		Result(userSettings)
		Error("unauthenticated", userSettingsUnauthenticated)
		Error("internal_error", userSettingsInternal, func() { Fault() })
		HTTP(func() {
			GET("/api/users/me/settings/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
