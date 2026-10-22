package design

import . "github.com/CaliLuke/loom/dsl"

var userInstanceAdminResult = Type("UserInstanceAdminResult", func() {
	Attribute("is_instance_admin", Boolean)
	Required("is_instance_admin")
})

var userInstanceAdminUnauthenticated = Type("UserInstanceAdminUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var userInstanceAdminInternalError = Type("UserInstanceAdminInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("user_instance_admin", func() {
	Description("Reports whether the authenticated user is an instance administrator.")

	Method("retrieve", func() {
		Security(planeSessionCookie)
		Payload(func() { stickySessionPayload() })
		Result(userInstanceAdminResult)
		Error("unauthenticated", userInstanceAdminUnauthenticated)
		Error("internal_error", userInstanceAdminInternalError, func() { Fault() })
		HTTP(func() {
			GET("/api/users/me/instance-admin/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
