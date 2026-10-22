package main

import (
	"bytes"
	"context"
	_ "embed"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	systemapi "github.com/CaliLuke/go-plane"
	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	loomhttp "github.com/CaliLuke/loom/http"
)

//go:embed csrf_failure.html
var authRedirectCSRFFailureTemplate []byte

type authSessionLifecycle interface {
	readServicesSessionLifecycle
	InspectReadServiceSessionWithoutTimezone(context.Context, *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error)
}

func authRedirectSessionPath(request *http.Request) bool {
	path := request.URL.Path
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return authRedirectPlainViewPath(path)
}

func authRedirectPlainViewPath(path string) bool {
	if !strings.HasSuffix(path, "/") {
		return false
	}
	switch path {
	case "/api/instances/admins/sign-in/", "/api/instances/admins/sign-up/", "/api/instances/admins/sign-out/":
		return true
	case "/auth/sign-in/", "/auth/sign-up/", "/auth/sign-out/",
		"/auth/magic-sign-in/", "/auth/magic-sign-up/",
		"/auth/spaces/sign-in/", "/auth/spaces/sign-up/", "/auth/spaces/sign-out/",
		"/auth/spaces/magic-sign-in/", "/auth/spaces/magic-sign-up/",
		"/auth/google/", "/auth/google/callback/", "/auth/spaces/google/", "/auth/spaces/google/callback/",
		"/auth/github/", "/auth/github/callback/", "/auth/spaces/github/", "/auth/spaces/github/callback/",
		"/auth/gitlab/", "/auth/gitlab/callback/", "/auth/spaces/gitlab/", "/auth/spaces/gitlab/callback/",
		"/auth/gitea/", "/auth/gitea/callback/", "/auth/spaces/gitea/", "/auth/spaces/gitea/callback/":
		return true
	default:
		return authRedirectResetPasswordPath(path)
	}
}

func authRedirectResetPasswordPath(path string) bool {
	for _, prefix := range []string{"/auth/reset-password/", "/auth/spaces/reset-password/"} {
		if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, "/") {
			continue
		}
		parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/"), "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return true
		}
	}
	return false
}

// djangoPlainViewCSRFMiddleware reproduces CsrfViewMiddleware's ordering for
// Django plain Views: it checks every unsafe request before method dispatch,
// including anonymous requests and methods the view does not implement.
func djangoPlainViewCSRFMiddleware(trustedOrigins []string, bodyLimit int64) func(http.Handler) http.Handler {
	if bodyLimit <= 0 {
		bodyLimit = defaultFileSizeLimit
	}
	return func(next http.Handler) http.Handler {
		csrf := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if !authRedirectPlainViewPath(request.URL.Path) || !accountAuthenticationUnsafeMethod(request.Method) {
				next.ServeHTTP(response, request)
				return
			}
			reason := accountAuthenticationCSRFReason(request, trustedOrigins)
			if reason == "" {
				next.ServeHTTP(response, request)
				return
			}
			setAuthRedirectSecurityHeaders(response.Header())
			writeAuthRedirectCSRFFailure(response)
		})
		bounded := activityReadsRequestSizeMiddleware(bodyLimit)(csrf)
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if !authRedirectPlainViewPath(request.URL.Path) || !accountAuthenticationUnsafeMethod(request.Method) {
				next.ServeHTTP(response, request)
				return
			}
			mediaType, _, _ := mime.ParseMediaType(request.Header.Get("Content-Type"))
			if request.Method != http.MethodPost || (mediaType != "application/x-www-form-urlencoded" && mediaType != "multipart/form-data") {
				csrf.ServeHTTP(response, request)
				return
			}
			bounded.ServeHTTP(response, request)
		})
	}
}

func writeAuthRedirectCSRFFailure(response http.ResponseWriter) {
	rootURL := os.Getenv("WEB_URL")
	if rootURL == "" {
		rootURL = os.Getenv("APP_BASE_URL")
	}
	body := bytes.ReplaceAll(authRedirectCSRFFailureTemplate, []byte("{{ root_url }}"), []byte(html.EscapeString(rootURL)))
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Content-Length", strconv.Itoa(len(body)))
	response.WriteHeader(http.StatusForbidden)
	_, _ = response.Write(body)
}

func accountAuthenticationSessionPath(request *http.Request) bool {
	path := strings.TrimSuffix(request.URL.Path, "/") + "/"
	return isAccountAuthenticationProtocol(path)
}

func authResponseSessionPath(request *http.Request) bool {
	path := strings.TrimSuffix(request.URL.Path, "/") + "/"
	return isAccountAuthenticationProtocol(path) || authRedirectSessionPath(request)
}

func accountAuthenticationProtocolMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setDjangoProtocolCompatibilityHeaders(response.Header(), "POST, OPTIONS")
		next.ServeHTTP(response, request)
	})
}

func accountAuthenticationCSRFMiddleware(trustedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if !accountAuthenticationSessionPath(request) || !accountAuthenticationUnsafeMethod(request.Method) {
				next.ServeHTTP(response, request)
				return
			}
			if !accountAuthenticationHasSessionPrincipal(request) {
				next.ServeHTTP(response, request)
				return
			}
			reason := accountAuthenticationCSRFReason(request, trustedOrigins)
			if reason == "" {
				next.ServeHTTP(response, request)
				return
			}
			setDjangoProtocolCompatibilityHeaders(response.Header(), "POST, OPTIONS")
			writeReadServiceJSON(response, http.StatusForbidden, map[string]string{"detail": "CSRF Failed: " + reason})
		})
	}
}

func accountAuthenticationHasSessionPrincipal(request *http.Request) bool {
	userID, authenticated := systemapi.ReadServiceSessionUserID(request.Context())
	return authenticated && userID != ""
}

func accountAuthenticationUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

func accountAuthenticationCSRFReason(request *http.Request, trustedOrigins []string) string {
	scheme := request.URL.Scheme
	if metadata, ok := loomhttp.RequestMetadataFromContext(request.Context()); ok && metadata.Scheme != "" {
		scheme = metadata.Scheme
	} else if request.TLS != nil {
		scheme = "https"
	}
	origin := request.Header.Get("Origin")
	if len(request.Header.Values("Origin")) > 0 {
		if !accountAuthenticationOriginMatches(origin, scheme, request.Host, trustedOrigins) {
			return "Origin checking failed - " + origin + " does not match any trusted origins."
		}
	} else if scheme == "https" {
		referer := request.Header.Get("Referer")
		if referer == "" {
			return "Referer checking failed - no Referer."
		}
		if !accountAuthenticationRefererMatches(referer, request.Host, trustedOrigins) {
			return "Referer checking failed - " + referer + " does not match any trusted origins."
		}
	}
	cookie, err := request.Cookie(defaultCSRFCookieName)
	if err != nil {
		return "CSRF cookie not set."
	}
	if reason := systemapi.CSRFTokenFormatReason(cookie.Value); reason != "" {
		return "CSRF cookie " + reason + "."
	}
	token, tokenSource := accountAuthenticationCSRFSubmittedToken(request)
	if tokenSource == "" {
		return "CSRF token missing."
	}
	if reason := systemapi.CSRFTokenFormatReason(token); reason != "" {
		return "CSRF token from " + tokenSource + " " + reason + "."
	}
	if !systemapi.ValidCSRFToken(cookie.Value, token) {
		return "CSRF token from " + tokenSource + " incorrect."
	}
	return ""
}

func accountAuthenticationCSRFSubmittedToken(request *http.Request) (string, string) {
	contentType := request.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if request.Method == http.MethodPost && request.Body != nil && (mediaType == "application/x-www-form-urlencoded" || mediaType == "multipart/form-data") {
		// Reuse the application's form parser and restore the bytes for the
		// generated handler. The outer request-size boundary remains in force.
		var raw bytes.Buffer
		input, err := djangoinput.Parse(io.TeeReader(request.Body, &raw), contentType)
		_ = request.Body.Close()
		request.Body = io.NopCloser(bytes.NewReader(raw.Bytes()))
		if err == nil {
			if token, ok := input.Fields["csrfmiddlewaretoken"].(string); ok && token != "" {
				return token, "POST"
			}
		}
	}
	if len(request.Header.Values("X-CSRFToken")) > 0 {
		return request.Header.Get("X-CSRFToken"), "the 'X-Csrftoken' HTTP header"
	}
	return "", ""
}

func accountAuthenticationOriginMatches(value, scheme, requestHost string, trustedOrigins []string) bool {
	parsed, ok := accountAuthenticationURLOrigin(value)
	if !ok {
		return false
	}
	if strings.EqualFold(parsed.Scheme, scheme) && strings.EqualFold(parsed.Host, requestHost) {
		return true
	}
	for _, trusted := range trustedOrigins {
		trustedOrigin, ok := accountAuthenticationURLOrigin(trusted)
		if ok && strings.EqualFold(parsed.Scheme, trustedOrigin.Scheme) && strings.EqualFold(parsed.Host, trustedOrigin.Host) {
			return true
		}
	}
	return false
}

func accountAuthenticationRefererMatches(value, requestHost string, trustedOrigins []string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	if strings.EqualFold(parsed.Host, requestHost) {
		return true
	}
	for _, trusted := range trustedOrigins {
		trustedOrigin, ok := accountAuthenticationURLOrigin(trusted)
		if ok && strings.EqualFold(parsed.Scheme, trustedOrigin.Scheme) && strings.EqualFold(parsed.Host, trustedOrigin.Host) {
			return true
		}
	}
	return false
}

func accountAuthenticationURLOrigin(value string) (*url.URL, bool) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, false
	}
	return parsed, true
}

func accountAuthenticationGzipMiddleware(next http.Handler) http.Handler {
	return readServicesGzipMiddlewareWithAuthBypass(next, false)
}

func authRedirectSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		setAuthRedirectSecurityHeaders(response.Header())
		next.ServeHTTP(response, request)
	})
}

func setAuthRedirectSecurityHeaders(headers http.Header) {
	headers.Set("Cross-Origin-Opener-Policy", "same-origin")
	headers.Set("Referrer-Policy", "same-origin")
	mergeVary(headers, "Accept-Encoding", "Cookie", "origin")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "DENY")
}

// passwordLoginCSRFCookieMiddleware mirrors Django's CSRF token rotation when
// the password endpoints call user_login. The session action is the domain
// signal that a password write completed and replaced the authenticated
// session; ordinary validation failures leave both cookies untouched.
func passwordLoginCSRFCookieMiddleware(config djangoCSRFCookieConfig) func(http.Handler) http.Handler {
	policy := djangoCSRFResponseCookiePolicy(config)
	return func(next http.Handler) http.Handler {
		return policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/auth/change-password/" && request.URL.Path != "/auth/set-password/" {
				next.ServeHTTP(response, request)
				return
			}
			writer := &passwordLoginCSRFCookieWriter{ResponseWriter: response, request: request}
			next.ServeHTTP(writer, request)
		}))
	}
}

type passwordLoginCSRFCookieWriter struct {
	http.ResponseWriter
	request *http.Request
	wrote   bool
}

func (writer *passwordLoginCSRFCookieWriter) WriteHeader(status int) {
	if writer.wrote {
		return
	}
	writer.wrote = true
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		if refresh, expire, found := djangosession.ResponseSessionActionFromContext(writer.request.Context()); found && !expire && refresh != nil {
			result, err := systemapi.NewCSRFToken().CsrfToken(writer.request.Context(), &csrftoken.CsrfTokenPayload{})
			if err == nil {
				cookie := &http.Cookie{
					Name: defaultCSRFCookieName, Value: result.CsrfCookie, MaxAge: 31449600,
					Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
				}
				_ = loomhttp.SetResponseCookie(writer.request.Context(), writer.ResponseWriter, cookie)
			}
		}
	}
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *passwordLoginCSRFCookieWriter) Write(body []byte) (int, error) {
	if !writer.wrote {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

type authSessionWithoutTimezone struct{ authSessionLifecycle }

func (sessions authSessionWithoutTimezone) InspectReadServiceSession(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	return sessions.InspectReadServiceSessionWithoutTimezone(ctx, request)
}

// Plain Django auth Views resolve sessions without DRF's timezone validation.
func djangoAuthSessionMiddleware(sessions authSessionLifecycle, cookie djangoSessionCookieConfig, matches func(*http.Request) bool) func(http.Handler) http.Handler {
	if sessions == nil {
		return djangoSessionLifecycleMiddleware(nil, cookie, matches)
	}
	return djangoSessionLifecycleMiddleware(authSessionWithoutTimezone{sessions}, cookie, matches)
}
