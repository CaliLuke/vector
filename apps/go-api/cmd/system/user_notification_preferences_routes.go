package main

import (
	"context"
	"net/http"

	usernotificationpreferencessvr "github.com/CaliLuke/go-plane/gen/http/user_notification_preferences/server"
	usernotificationpreferences "github.com/CaliLuke/go-plane/gen/user_notification_preferences"
	loomhttp "github.com/CaliLuke/loom/http"
)

func userNotificationPreferencesCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setUserNotificationPreferencesHeaders(response.Header())
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func userNotificationPreferencesHeadHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		userNotificationPreferencesCompatibilityMiddleware(next).ServeHTTP(workspaceSlugHeadWriter{response}, request)
	})
}

func userNotificationPreferencesOptionsHandler(response http.ResponseWriter, _ *http.Request) {
	setUserNotificationPreferencesHeaders(response.Header())
	writeStickyJSON(response, http.StatusOK, map[string]any{
		"name":        "User Notification Preference Endpoint",
		"description": "",
		"renders":     []string{"application/json"},
		"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
	})
}

func setUserNotificationPreferencesHeaders(headers http.Header) {
	headers.Set("Allow", "GET, PATCH, HEAD, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func mountUserNotificationPreferencesRoutes(ctx context.Context, notificationEndpoints *usernotificationpreferences.Endpoints, mux loomhttp.Muxer) {
	if notificationEndpoints != nil {
		server := usernotificationpreferencessvr.New(notificationEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		server.Use(userNotificationPreferencesCompatibilityMiddleware)
		usernotificationpreferencessvr.Mount(mux, server)
		mountProtocolHandler(mux, "HEAD", "/api/users/me/notification-preferences/", userNotificationPreferencesHeadHandler(server.Get))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/notification-preferences/", http.HandlerFunc(userNotificationPreferencesOptionsHandler))
		for _, mount := range server.Mounts {
			logHTTPRouteMount(ctx, "user_notification_preferences", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
