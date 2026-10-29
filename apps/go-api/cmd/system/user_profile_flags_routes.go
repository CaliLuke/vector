package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	userprofileflagssvr "github.com/CaliLuke/go-plane/gen/http/user_profile_flags/server"
	userprofileflags "github.com/CaliLuke/go-plane/gen/user_profile_flags"
	loomhttp "github.com/CaliLuke/loom/http"
)

func userProfileFlagsCompatibilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setUserProfileFlagsHeaders(response.Header())
		body, err := io.ReadAll(request.Body)
		if err != nil {
			writeStickyJSON(response, http.StatusBadRequest, map[string]string{"detail": "JSON parse error."})
			return
		}
		body, status, validation := normalizeUserProfileFlagsBody(request.URL.Path, body)
		if validation != nil {
			writeStickyJSON(response, status, validation)
			return
		}
		request.Header.Set("Content-Type", "application/json")
		request.Body = io.NopCloser(bytes.NewReader(body))
		request.ContentLength = int64(len(body))
		next.ServeHTTP(workspaceSlugResponseWriter{response}, request)
	})
}

func normalizeUserProfileFlagsBody(path string, body []byte) ([]byte, int, map[string]string) {
	if len(bytes.TrimSpace(body)) == 0 {
		return []byte("{}"), 0, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		if !json.Valid(body) {
			return nil, http.StatusBadRequest, map[string]string{"detail": "JSON parse error - " + err.Error()}
		}
		return nil, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"}
	}
	if fields == nil {
		return nil, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"}
	}
	field := "is_onboarded"
	if path == "/api/users/me/tour-completed/" {
		field = "is_tour_completed"
	}
	raw, exists := fields[field]
	if !exists {
		return body, 0, nil
	}
	value, status, validation := djangoBooleanValue(raw)
	if validation != nil {
		return nil, status, validation
	}
	fields[field], _ = json.Marshal(value)
	normalized, err := json.Marshal(fields)
	if err != nil {
		return nil, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"}
	}
	return normalized, 0, nil
}

func setUserProfileFlagsHeaders(headers http.Header) {
	headers.Set("Allow", "PATCH, OPTIONS")
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

func userProfileFlagsHeadHandler(response http.ResponseWriter, _ *http.Request) {
	setUserProfileFlagsHeaders(response.Header())
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusMethodNotAllowed)
}

func userProfileFlagsOptionsHandler(name string) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		setUserProfileFlagsHeaders(response.Header())
		writeStickyJSON(response, http.StatusOK, map[string]any{
			"name":        name,
			"description": "",
			"renders":     []string{"application/json"},
			"parses":      []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		})
	})
}

func mountUserProfileFlagsRoutes(ctx context.Context, userProfileFlagsEndpoints *userprofileflags.Endpoints, mux loomhttp.Muxer) {
	if userProfileFlagsEndpoints != nil {
		userProfileFlagsServer := userprofileflagssvr.New(userProfileFlagsEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errorHandler, nil)
		userProfileFlagsServer.Use(userProfileFlagsCompatibilityMiddleware)
		userprofileflagssvr.Mount(mux, userProfileFlagsServer)
		mountProtocolHandler(mux, "HEAD", "/api/users/me/onboard/", http.HandlerFunc(userProfileFlagsHeadHandler))
		mountProtocolHandler(mux, "HEAD", "/api/users/me/tour-completed/", http.HandlerFunc(userProfileFlagsHeadHandler))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/onboard/", userProfileFlagsOptionsHandler("Update User On Boarded Endpoint"))
		mountProtocolHandler(mux, "OPTIONS", "/api/users/me/tour-completed/", userProfileFlagsOptionsHandler("Update User Tour Completed Endpoint"))
		for _, mount := range userProfileFlagsServer.Mounts {
			logHTTPRouteMount(ctx, "user_profile_flags", mount.Method, mount.Verb, mount.Pattern)
		}
	}
}
