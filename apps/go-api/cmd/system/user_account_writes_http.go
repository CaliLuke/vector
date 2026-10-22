package main

import (
	_ "embed"
	"encoding/json"
	"net/http"

	loomhttp "github.com/CaliLuke/loom/http"
)

//go:embed user_email_code_options.json
var userEmailCodeOptionsMetadata []byte

func userAccountEmailCodeProtocol() readServiceProtocol {
	protocol := newSessionServiceProtocol("/api/users/me/email/generate-code/", "POST, OPTIONS", "User Endpoint", false)
	protocol.throttleBeforeTimezone = true
	protocol.optionsMetadata = json.RawMessage(userEmailCodeOptionsMetadata)
	return protocol
}

func userAccountMeWriteProtocol() readServiceProtocol {
	protocol := newSessionServiceProtocol("/api/users/me/", "GET, PATCH, DELETE, HEAD, OPTIONS", "User Endpoint", false)
	// The retired identity service owns the existing read and metadata mounts.
	protocol.passthroughMethods = []string{http.MethodGet, http.MethodHead}
	protocol.methodAllow = map[string]string{
		http.MethodPatch:  "PATCH, DELETE, OPTIONS",
		http.MethodDelete: "PATCH, DELETE, OPTIONS",
	}
	return protocol
}

func userAccountMeSlashlessCompatibilityMiddleware(next http.Handler, debugMode bool, corsPolicy loomhttp.RuntimeCORSPolicy) http.Handler {
	redirect := corsPolicy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		mergeVary(response.Header(), "Cookie", "origin")
		response.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		response.Header().Set("Referrer-Policy", "same-origin")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		if debugMode && (request.Method == http.MethodPost || request.Method == http.MethodPut ||
			request.Method == http.MethodPatch || request.Method == http.MethodDelete) {
			body, contentType := djangoDebugAppendSlashResponse(request)
			response.Header().Set("Content-Type", contentType)
			response.WriteHeader(http.StatusInternalServerError)
			_, _ = response.Write(body)
			return
		}
		location := request.URL.Path + "/"
		if request.URL.RawQuery != "" {
			location += "?" + request.URL.RawQuery
		}
		response.Header().Set("Location", location)
		response.WriteHeader(http.StatusMovedPermanently)
	}))
	preflight := djangoProtocolCORSOptionsHandler(corsPolicy, []string{http.MethodGet}, redirect)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/users/me" {
			preflight.ServeHTTP(response, request)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func isUserAccountWriteProtocol(pattern string) bool {
	switch pattern {
	case "/api/users/me/", "/api/users/me/email/", "/api/users/me/email/generate-code/":
		return true
	default:
		return false
	}
}
