package design

import . "github.com/CaliLuke/loom/dsl"

var userProfileFlagResult = Type("UserProfileFlagResult", func() {
	Attribute("message", String)
	Required("message")
})

var userProfileFlagUnauthenticated = Type("UserProfileFlagUnauthenticated", func() {
	Attribute("detail", String)
	Required("detail")
})

var userProfileFlagInternalError = Type("UserProfileFlagInternalError", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var userProfileFlagNotFound = Type("UserProfileFlagNotFound", func() {
	Attribute("error", String, func() { Meta("struct:field:name", "Message") })
	Required("error")
})

var _ = Service("user_profile_flags", func() {
	Description("Updates the authenticated user's onboarding flags.")

	Method("update_onboarded", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("is_onboarded", Boolean)
		})
		Result(userProfileFlagResult)
		Error("unauthenticated", userProfileFlagUnauthenticated)
		Error("not_found", userProfileFlagNotFound)
		Error("internal_error", userProfileFlagInternalError, func() { Fault() })
		HTTP(func() {
			PATCH("/api/users/me/onboard/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})

	Method("update_tour_completed", func() {
		Security(planeSessionCookie)
		Payload(func() {
			stickySessionPayload()
			Attribute("is_tour_completed", Boolean)
		})
		Result(userProfileFlagResult)
		Error("unauthenticated", userProfileFlagUnauthenticated)
		Error("not_found", userProfileFlagNotFound)
		Error("internal_error", userProfileFlagInternalError, func() { Fault() })
		HTTP(func() {
			PATCH("/api/users/me/tour-completed/")
			Cookie("session_id:session-id")
			Response(StatusOK)
			Response("unauthenticated", StatusUnauthorized)
			Response("not_found", StatusNotFound)
			Response("internal_error", StatusInternalServerError)
		})
	})
})
