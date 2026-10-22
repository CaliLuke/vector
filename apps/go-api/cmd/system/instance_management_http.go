package main

import (
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
)

func instanceManagementPlainRedirectOptionsHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		setDjangoProtocolCompatibilityHeaders(response.Header(), "POST, OPTIONS")
		response.WriteHeader(http.StatusOK)
	})
}

func newInstanceManagementProtocol(pattern, allow, name string, head bool) readServiceProtocol {
	protocol := newSessionServiceProtocol(pattern, allow, name, head)
	protocol.permissionBeforeTimezone = true
	switch pattern {
	case "/api/instances/":
		protocol.permissionBeforeTimezoneExceptMethods = []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE", "TRACE", "CONNECT"}
	case "/api/instances/admins/sign-up-screen-visited/":
		protocol.permissionBeforeTimezone = false
	case "/api/instances/admins/sign-in/", "/api/instances/admins/sign-up/", "/api/instances/admins/sign-out/":
		// These are plain Django Views, not DRF permission-bearing APIViews.
		// They accept anonymous form posts and must reach the handler before
		// any admin-session authorization or timezone validation.
		protocol.permissionBeforeTimezone = false
		protocol.authenticateMethod = false
	case "/api/instances/admins/{pk}/":
		// Django advertises POST on this shared view, but its signature cannot
		// accept pk. The corrected companion rejects it after admin authorization.
		protocol.protocolOnlyMethods = []string{http.MethodPost}
	}
	return protocol
}

func isInstanceManagementProtocol(pattern string) bool {
	switch pattern {
	case "/api/instances/", "/api/instances/admins/", "/api/instances/admins/{pk}/",
		"/api/instances/admins/sign-up-screen-visited/", "/api/instances/admins/sign-in/",
		"/api/instances/admins/sign-up/", "/api/instances/admins/sign-out/", "/api/instances/configurations/",
		"/api/instances/configurations/disable-email-feature/", "/api/instances/email-credentials-check/",
		"/api/instances/workspaces/":
		return true
	}
	return false
}

func isInstanceManagementPlainRedirect(pattern string) bool {
	switch pattern {
	case "/api/instances/admins/sign-in/", "/api/instances/admins/sign-up/", "/api/instances/admins/sign-out/":
		return true
	default:
		return false
	}
}

func instanceManagementRequest(request *http.Request) bool {
	protocol, ok := resolveReadServiceProtocol(strings.TrimSuffix(request.URL.Path, "/") + "/")
	return ok && isInstanceManagementProtocol(protocol.pattern)
}

func instanceManagementHandler(instance, other http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if instanceManagementRequest(request) {
			instance.ServeHTTP(response, request)
			return
		}
		other.ServeHTTP(response, request)
	})
}

// instanceManagementAnonymousThrottleMiddleware applies DRF's shared anon
// budget to the two AllowAny API views in this family. It runs after the
// shared admin-session middleware, so it does not inspect or rotate cookies a
// second time and authenticated sessions bypass the anonymous bucket.
func instanceManagementAnonymousThrottleMiddleware(throttle instanceManagementAnonymousThrottle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if throttle == nil || !instanceManagementAnonymousThrottleRequest(request) {
				next.ServeHTTP(response, request)
				return
			}
			if userID, _ := systemapi.ReadServiceSessionUserID(request.Context()); userID != "" {
				next.ServeHTTP(response, request)
				return
			}
			allowed, wait, err := throttle.AllowAnonymousRequest(request.Context(), request)
			if err != nil {
				writeReadServiceJSON(response, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"})
				return
			}
			if !allowed {
				if wait > 0 {
					response.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
				}
				writeReadServiceJSON(response, http.StatusTooManyRequests, map[string]any{
					"error_code": 5900, "error_message": "RATE_LIMIT_EXCEEDED",
				})
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

func instanceManagementAnonymousThrottleRequest(request *http.Request) bool {
	path := strings.TrimSuffix(request.URL.Path, "/") + "/"
	switch path {
	case "/api/instances/":
		return request.Method != http.MethodPatch
	case "/api/instances/admins/sign-up-screen-visited/":
		return true
	default:
		return false
	}
}

func instanceAdminLiteralSegment(segment string) bool {
	return slices.Contains([]string{"me", "session", "sign-in", "sign-out", "sign-up", "sign-up-screen-visited"}, segment)
}
