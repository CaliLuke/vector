package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
)

// djangoSessionLifecycleMiddleware also serves auth routes that are outside the
// API protocol registry. Their mount supplies the exact route matcher and cookie.
func djangoSessionLifecycleMiddleware(
	sessions readServicesSessionLifecycle,
	sessionCookie djangoSessionCookieConfig,
	matches func(*http.Request) bool,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			matched := matches(request)
			preflight := request.Method == http.MethodOptions && len(request.Header.Values("Access-Control-Request-Method")) > 0
			if !matched || preflight || sessions == nil {
				next.ServeHTTP(response, request)
				return
			}
			request = normalizeDjangoCookie(request, sessionCookieName(sessionCookie))
			expireSession, refreshSession, prepared, principal, err := sessions.InspectReadServiceSession(request.Context(), request)
			if expireSession {
				if expireDjangoErr := expireDjangoSessionCookie(response.Header(), sessionCookie); expireDjangoErr != nil {
					logPublicProjectMetadataCompatibilityError(request.Context(), expireDjangoErr)
					writeReadServicesSessionError(response, request)
					return
				}
			}
			if prepared == nil && refreshSession {
				if cookie := lastDjangoSessionCookie(request, sessionCookieName(sessionCookie)); cookie != nil {
					prepared = &djangosession.SessionRefresh{SessionKey: cookie.Value}
				}
			}
			if err != nil {
				logPublicProjectMetadataCompatibilityError(request.Context(), err)
				errorResponse := response
				if prepared != nil {
					errorResponse = &readServicesSessionResponseWriter{
						ResponseWriter: response,
						request:        request,
						sessions:       sessions,
						sessionCookie:  sessionCookie,
						refresh:        prepared,
					}
				}
				writeReadServicesSessionInspectionError(errorResponse, request, err)
				return
			}
			ctx, _ := djangosession.WithResponseSessionAction(request.Context())
			if prepared != nil {
				ctx = djangosession.WithCurrentSession(ctx, *prepared)
			} else if principal.SessionKey != "" {
				// Auth handlers need the loaded session even when automatic saving is disabled.
				ctx = djangosession.WithCurrentSession(ctx, principal.PrepareRefresh())
			}
			request = request.WithContext(systemapi.WithReadServiceSessionPrincipal(
				ctx, principal.UserID, principal.UserTimezone,
			))
			writer := &readServicesSessionResponseWriter{
				ResponseWriter: response,
				request:        request,
				sessions:       sessions,
				sessionCookie:  sessionCookie,
				refresh:        prepared,
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func finalizeUnaccessedDjangoSession(
	headers http.Header,
	request *http.Request,
	status int,
	sessionCookie djangoSessionCookieConfig,
	sessions publicProjectMetadataThrottle,
) error {
	expireSession, refreshSession, err := sessions.FinalizeUnaccessedSession(request.Context(), request, status)
	if expireSession {
		if cookieErr := expireDjangoSessionCookie(headers, sessionCookie); cookieErr != nil {
			return cookieErr
		}
	}
	if err != nil {
		return err
	}
	if refreshSession != nil && status < http.StatusInternalServerError {
		if cookieErr := refreshUnaccessedDjangoSessionCookie(headers, sessionCookie, *refreshSession); cookieErr != nil {
			return cookieErr
		}
	}
	return nil
}

type djangoSessionCookieConfig struct {
	Name       string
	Domain     string
	Secure     bool
	Configured bool
}

const defaultCSRFCookieName = "csrftoken"

func normalizeDjangoCookie(request *http.Request, name string) *http.Request {
	cookies := request.CookiesNamed(name)
	if len(cookies) < 2 {
		return request
	}
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	original := clone.Header.Values("Cookie")
	clone.Header.Del("Cookie")
	clone.Header.Add("Cookie", cookies[len(cookies)-1].String())
	for _, value := range original {
		clone.Header.Add("Cookie", value)
	}
	return clone
}

func removeResponseCookie(headers http.Header, name string) {
	values := headers.Values("Set-Cookie")
	headers.Del("Set-Cookie")
	for _, value := range values {
		cookie, err := http.ParseSetCookie(value)
		if err == nil && cookie.Name == name {
			continue
		}
		headers.Add("Set-Cookie", value)
	}
}

func lastDjangoSessionCookie(request *http.Request, name string) *http.Cookie {
	return djangosession.CookieFromHeaderValues(request.Header.Values("Cookie"), name)
}

func djangoLatin1String(value []byte) string {
	characters := make([]rune, len(value))
	for index, octet := range value {
		characters[index] = rune(octet)
	}
	return string(characters)
}

func expireUnaccessedDjangoSessionCookie(headers http.Header, request *http.Request, config djangoSessionCookieConfig) error {
	cookie := lastDjangoSessionCookie(request, sessionCookieName(config))
	// SessionBase retains an unaccessed, valid-shaped key. Its lazy cache is
	// empty only when the supplied key fails Django's eight-character minimum.
	if cookie != nil && utf8.RuneCountInString(cookie.Value) < 8 {
		return expireDjangoSessionCookie(headers, config)
	}
	return nil
}

func expireDjangoSessionCookie(headers http.Header, config djangoSessionCookieConfig) error {
	config.Name = sessionCookieName(config)
	cookie := &http.Cookie{
		Name:     config.Name,
		Value:    "",
		Path:     "/",
		Domain:   config.Domain,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	}
	if err := cookie.Valid(); err != nil {
		return fmt.Errorf("delete Django session cookie: %w", err)
	}
	parts := []string{config.Name + `=""`}
	if config.Domain != "" {
		parts = append(parts, "Domain="+config.Domain)
	}
	parts = append(parts, "expires=Thu, 01 Jan 1970 00:00:00 GMT", "Max-Age=0", "Path=/", "SameSite=Lax")
	if strings.HasPrefix(config.Name, "__Secure-") || strings.HasPrefix(config.Name, "__Host-") {
		parts = append(parts, "Secure")
	}
	headers.Add("Set-Cookie", strings.Join(parts, "; "))
	mergeVary(headers, "Cookie")
	return nil
}

func refreshDjangoSessionCookie(headers http.Header, config djangoSessionCookieConfig, refresh djangosession.SessionRefresh) error {
	return writeDjangoSessionCookie(headers, config, refresh, true)
}

func refreshUnaccessedDjangoSessionCookie(headers http.Header, config djangoSessionCookieConfig, refresh djangosession.SessionRefresh) error {
	return writeDjangoSessionCookie(headers, config, refresh, false)
}

func writeDjangoSessionCookie(headers http.Header, config djangoSessionCookieConfig, refresh djangosession.SessionRefresh, varyCookie bool) error {
	config.Name = sessionCookieName(config)
	cookie := &http.Cookie{Name: config.Name, Value: refresh.SessionKey, Path: "/", Domain: config.Domain}
	if err := cookie.Valid(); err != nil {
		return fmt.Errorf("set Django session cookie: %w", err)
	}
	parts := []string{config.Name + "=" + refresh.SessionKey}
	if config.Domain != "" {
		parts = append(parts, "Domain="+config.Domain)
	}
	if !refresh.BrowserClose {
		expiresAt := refresh.CookieExpiresAt
		if expiresAt.IsZero() {
			expiresAt = refresh.ExpiresAt
		}
		parts = append(parts, "expires="+expiresAt.UTC().Format(http.TimeFormat))
	}
	parts = append(parts, "HttpOnly")
	if !refresh.BrowserClose {
		parts = append(parts, "Max-Age="+strconv.FormatInt(int64(refresh.MaxAgeSeconds), 10))
	}
	parts = append(parts, "Path=/", "SameSite=Lax")
	if config.Secure {
		parts = append(parts, "Secure")
	}
	headers.Add("Set-Cookie", strings.Join(parts, "; "))
	if varyCookie {
		mergeVary(headers, "Cookie")
	}
	return nil
}

func sessionCookieName(config djangoSessionCookieConfig) string {
	if config.Configured || config.Name != "" {
		return config.Name
	}
	return djangosession.DefaultCookieName
}
