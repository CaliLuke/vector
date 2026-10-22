package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	loomhttp "github.com/CaliLuke/loom/http"
)

type sessionResourceProtocolAuthorizerWithHeaders interface {
	AuthorizeSessionResourceProtocolWithHeaders(context.Context, string, systemapi.SessionProtocolResource) (int, map[string]any, http.Header)
}

type sessionProtocolOptionsMetadataAuthorizer interface {
	SessionProtocolOptionsMetadata(context.Context, string, systemapi.SessionProtocolResource, json.RawMessage, json.RawMessage) (int, any, bool)
}

//go:embed notification_metadata.json
var notificationActionMetadata []byte

func notificationProtocolMetadata(protocol readServiceProtocol) (int, any) {
	if protocol.optionsStatus != 0 {
		return protocol.optionsStatus, protocol.optionsMetadata
	}
	if len(protocol.optionsMetadata) > 0 {
		return 200, protocol.optionsMetadata
	}
	if strings.HasSuffix(protocol.pattern, "/read/") || strings.HasSuffix(protocol.pattern, "/archive/") {
		return 200, json.RawMessage(notificationActionMetadata)
	}

	return 200, map[string]any{"name": protocol.name, "description": "", "renders": []string{"application/json"}, "parses": []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}}
}
func newSessionServiceProtocol(pattern, allow, name string, head bool) readServiceProtocol {
	result := newReadServiceProtocol(pattern, allow, name, "", head)
	result.authenticateMethod = true
	return result
}

// Session CRUD services use the same Django middleware as activity reads, but
// keep their string-valued queries intact for service-level parsing.
func sessionServiceMiddlewareStack(options apiV1CompatibilityOptions, sessions readServicesSessionLifecycle, cookie djangoSessionCookieConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		next = djangoRendererFailureMiddleware(next)
		next = readServicesQueryPresenceMiddleware(next)
		next = activityReadsCompatibilityMiddleware(options)(next)
		next = activityReadsSessionMiddleware(sessionServicePrincipalLifecycle(sessions), cookie)(next)
		next = readServicesContentNegotiationMiddleware(next)
		next = activityReadsAuditFallbackMiddleware(options)(next)
		next = activityReadsRequestSizeMiddleware(options.bodySizeLimit)(next)
		return readServicesGzipMiddleware(next)
	}
}

// Django renderer failures occur outside DRF's JSON exception handling. Reuse
// the common HTML response boundary only when the endpoint marks that failure;
// ordinary application failures retain their designed JSON response.
func djangoRendererFailureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		request = request.WithContext(systemapi.WithReadServiceDjangoHTML500(request.Context()))
		next.ServeHTTP(&djangoReadServiceHTML500ResponseWriter{
			ResponseWriter: response, request: request, omitAllow: true,
		}, request)
	})
}

// newSessionServiceProtocolMiddleware handles DRF's authenticated metadata and
// unsupported-method responses through the same stack as generated methods.
func newSessionServiceProtocolMiddleware(metadata func(readServiceProtocol) (int, any), authorizer systemapi.SessionProtocolAuthorizer, policy loomhttp.RuntimeCORSPolicy, stack func(http.Handler) http.Handler, cookie djangoSessionCookieConfig, debugMode bool, options apiV1CompatibilityOptions, csrfTrustedOrigins ...[]string) func(http.Handler) http.Handler {
	var trustedOrigins []string
	if len(csrfTrustedOrigins) > 0 {
		trustedOrigins = csrfTrustedOrigins[0]
	}
	ordinary := policy.Handler(loomhttp.AsHandlerFunc(stack(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		protocol, _ := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
		if request.Method == http.MethodOptions || !readServiceMethodAllowed(protocol.allow, request.Method) {
			response.Header().Set("Allow", protocol.allow)
		}
		session := ""
		if value, err := request.Cookie(sessionCookieName(cookie)); err == nil {
			session = value.Value
		}
		var status int
		var body map[string]any
		if scoped, ok := authorizer.(systemapi.SessionResourceProtocolAuthorizer); ok {
			request = withReadServicePathValues(request, protocol)
			resource := sessionProtocolResource(request, protocol)
			if withHeaders, ok := authorizer.(sessionResourceProtocolAuthorizerWithHeaders); ok {
				var headers http.Header
				status, body, headers = withHeaders.AuthorizeSessionResourceProtocolWithHeaders(request.Context(), session, resource)
				for key, values := range headers {
					response.Header()[key] = append([]string(nil), values...)
				}
			} else {
				status, body = scoped.AuthorizeSessionResourceProtocol(request.Context(), session, resource)
			}
		} else {
			status, body = authorizer.AuthorizeSessionProtocol(request.Context(), session)
		}
		if status == 0 {
			if protocol.csrfForAuthenticatedSession && accountAuthenticationUnsafeMethod(request.Method) && accountAuthenticationHasSessionPrincipal(request) {
				if reason := accountAuthenticationCSRFReason(request, trustedOrigins); reason != "" {
					setDjangoProtocolCompatibilityHeaders(response.Header(), protocol.allow)
					writeReadServiceJSON(response, http.StatusForbidden, map[string]string{"detail": "CSRF Failed: " + reason})
					return
				}
			}
			status, body = assetsProtocolTimezone(request, protocol)
		}
		if status != 0 {
			writeReadServiceJSON(response, status, body)
			return
		}
		if request.Method == http.MethodOptions {
			if len(protocol.optionsMetadataMissing) > 0 {
				if lookup, ok := authorizer.(interface {
					SessionProtocolResourceExists(context.Context, systemapi.SessionProtocolResource) (bool, error)
				}); ok {
					exists, err := lookup.SessionProtocolResourceExists(request.Context(), sessionProtocolResource(request, protocol))
					if err != nil {
						writeReadServiceJSON(response, http.StatusInternalServerError, map[string]string{"error": "Something went wrong please try again later"})
						return
					}
					if !exists {
						protocol.optionsMetadata = protocol.optionsMetadataMissing
					}
				}
			}
			if metadataAuthorizer, ok := authorizer.(sessionProtocolOptionsMetadataAuthorizer); ok {
				status, body, handled := metadataAuthorizer.SessionProtocolOptionsMetadata(request.Context(), session, sessionProtocolResource(request, protocol), protocol.optionsMetadata, protocol.optionsMetadataGuest)
				if handled {
					writeReadServiceJSON(response, status, body)
					return
				}
			}
			status, body := metadata(protocol)
			writeReadServiceJSON(response, status, body)
			return
		}
		writeReadServiceJSON(response, http.StatusMethodNotAllowed, map[string]string{"detail": fmt.Sprintf("Method %q not allowed.", request.Method)})
	}))))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			protocol, matched := resolveReadServiceProtocolForMethod(request.URL.Path, request.Method)
			slashless := false
			if !matched && !strings.HasSuffix(request.URL.Path, "/") {
				protocol, matched = resolveReadServiceProtocolForMethod(request.URL.Path+"/", request.Method)
				slashless = matched
				if matched && readServiceProtocolHasMalformedUUID(request.URL.Path+"/", protocol) {
					matched = false
				}
			}
			if !matched || !protocol.authenticateMethod || (!slashless && slices.Contains(protocol.passthroughMethods, request.Method)) {
				next.ServeHTTP(response, request)
				return
			}
			if request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0 {
				methods := strings.Split(strings.ReplaceAll(protocol.allow, " ", ""), ",")
				djangoProtocolCORSOptionsHandler(policy, methods, ordinary).ServeHTTP(response, request)
				return
			}
			if slashless {
				redirect := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
					response.Header().Set("Content-Type", "text/html; charset=utf-8")
					mergeVary(response.Header(), "Cookie", "origin")
					response.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
					response.Header().Set("Referrer-Policy", "same-origin")
					response.Header().Set("X-Content-Type-Options", "nosniff")
					// Django CommonMiddleware may return its slash redirect before
					// XFrameOptionsMiddleware runs. Match that ordering for auth
					// APIViews while preserving the normal protocol headers elsewhere.
					if !isAccountAuthenticationProtocol(protocol.pattern) {
						response.Header().Set("X-Frame-Options", "DENY")
					}
					if debugMode && (request.Method == "POST" || request.Method == "PUT" || request.Method == "PATCH" || request.Method == "DELETE") {
						body, kind := djangoDebugAppendSlashResponse(request)
						response.Header().Set("Content-Type", kind)
						response.WriteHeader(500)
						_, _ = response.Write(body)
						return
					}
					location := request.URL.Path + "/"
					if request.URL.RawQuery != "" {
						location += "?" + request.URL.RawQuery
					}
					response.Header().Set("Location", location)
					response.WriteHeader(http.StatusMovedPermanently)
					return
				})
				unresolved := func(next http.Handler) http.Handler {
					return activityReadsRequestSizeMiddleware(options.bodySizeLimit)(activityReadsCompatibilityMiddleware(options)(next))
				}
				activityReadsSlashlessCompatibilityHandler(unresolved, redirect).ServeHTTP(response, request)
				return
			}

			if owner, ok := authorizer.(interface{ OwnsSessionProtocolMethod(string, string) bool }); ok && owner.OwnsSessionProtocolMethod(protocol.pattern, request.Method) {
				next.ServeHTTP(response, request)
				return
			}
			if (request.Method == http.MethodOptions && !protocol.authoredOptions) || !readServiceMethodAllowed(protocol.allow, request.Method) || slices.Contains(protocol.protocolOnlyMethods, request.Method) {
				ordinary.ServeHTTP(response, request)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}
