package main

import (
	"net/http"

	usersettingssvr "github.com/CaliLuke/go-plane/gen/http/user_settings/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func userSettingsCompatibilityMiddleware(next http.Handler) http.Handler {
	return djangoProtocolCompatibilityMiddleware(func(string) string { return "GET, HEAD, OPTIONS" })(next)
}

func mountUserSettingsRoutes(pending pendingReplacementEndpoints, mux loomhttp.Muxer) {
	if pending.userSettings != nil {
		server := usersettingssvr.New(pending.userSettings, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userSettingsCompatibilityMiddleware)
		usersettingssvr.Mount(mux, server)
		mountProtocolHandler(mux, http.MethodHead, "/api/users/me/settings/", djangoProtocolHeadHandler(userSettingsCompatibilityMiddleware, server.Get))
		mountProtocolHandler(mux, http.MethodOptions, "/api/users/me/settings/", djangoProtocolOptionsHandler(userSettingsCompatibilityMiddleware, "User Settings Endpoint", false))
	}
}
