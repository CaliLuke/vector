package publicprojectmetadatareads

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	"github.com/CaliLuke/go-plane/internal/ratelimit"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrSessionInvalidation = errors.New("browser session invalidation failed")

type throttleCache = ratelimit.Store

type throttleSessionValidator interface {
	Inspect(context.Context, string) (browsersession.Principal, browsersession.SessionRefresh, error)
	RefreshPrepared(context.Context, browsersession.SessionRefresh, int64, time.Time) (browsersession.SessionRefresh, error)
	RefreshUnaccessed(context.Context, string, int64, time.Time) (browsersession.SessionRefresh, error)
	RotateFallbackPrepared(context.Context, browsersession.SessionRefresh, int64, time.Time) (browsersession.SessionRefresh, error)
}

type throttleSessionInvalidator interface {
	InvalidateSession(context.Context, string) error
}

type AnonymousThrottle struct {
	cache           throttleCache
	sessions        throttleSessionValidator
	invalidator     throttleSessionInvalidator
	cookieName      string
	cookieAge       int64
	adminCookieName string
	adminCookieAge  int64
	saveEvery       bool
	scope           string
	limiter         *ratelimit.Limiter
	now             func() time.Time
}

type AnonymousThrottleConfig struct {
	Scope string
	Rate  string
}

type AdminSessionConfig struct {
	CookieName string
	CookieAge  int64
}

func NewAnonymousThrottle(cache throttleCache, sessions throttleSessionValidator, invalidator throttleSessionInvalidator, cookieName string, saveEvery bool, cookieAge int64, admin ...AdminSessionConfig) (*AnonymousThrottle, error) {
	return NewConfiguredAnonymousThrottle(cache, sessions, invalidator, cookieName, saveEvery, cookieAge, AnonymousThrottleConfig{
		Scope: "anon",
		Rate:  "30/minute",
	}, admin...)
}

func NewConfiguredAnonymousThrottle(cache throttleCache, sessions throttleSessionValidator, invalidator throttleSessionInvalidator, cookieName string, saveEvery bool, cookieAge int64, config AnonymousThrottleConfig, admin ...AdminSessionConfig) (*AnonymousThrottle, error) {
	if cache == nil {
		return nil, fmt.Errorf("anonymous throttle cache is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("anonymous throttle session validator is required")
	}
	if invalidator == nil {
		return nil, fmt.Errorf("anonymous throttle session invalidator is required")
	}
	if config.Scope == "" {
		return nil, fmt.Errorf("anonymous throttle scope is required")
	}
	if _, err := ratelimit.BucketKey(config.Scope, "anonymous"); err != nil {
		return nil, err
	}
	limiter, err := ratelimit.New(cache, config.Rate)
	if err != nil {
		return nil, err
	}
	adminCookieName := "admin-session-id"
	adminCookieAge := int64(time.Hour / time.Second)
	if len(admin) > 1 {
		return nil, fmt.Errorf("at most one admin browser session configuration is allowed")
	}
	if len(admin) == 1 {
		adminCookieName = admin[0].CookieName
		adminCookieAge = admin[0].CookieAge
	}
	if adminCookieName == "" {
		return nil, fmt.Errorf("admin browser session cookie name is required")
	}
	return &AnonymousThrottle{
		cache: cache, sessions: sessions, invalidator: invalidator, cookieName: cookieName,
		cookieAge: cookieAge, adminCookieName: adminCookieName, adminCookieAge: adminCookieAge,
		saveEvery: saveEvery, scope: config.Scope, limiter: limiter, now: time.Now,
	}, nil
}

func (throttle *AnonymousThrottle) AllowRequest(ctx context.Context, request *http.Request) (bool, time.Duration, bool, bool, *browsersession.SessionRefresh, string, error) {
	return throttle.allowRequest(ctx, request, true)
}

// AllowRequestWithoutTimezone preserves anonymous throttling and browser session
// behavior for views that do not include TimezoneMixin.
func (throttle *AnonymousThrottle) AllowRequestWithoutTimezone(ctx context.Context, request *http.Request) (bool, time.Duration, bool, bool, *browsersession.SessionRefresh, string, error) {
	return throttle.allowRequest(ctx, request, false)
}

func (throttle *AnonymousThrottle) allowRequest(ctx context.Context, request *http.Request, activateTimezone bool) (bool, time.Duration, bool, bool, *browsersession.SessionRefresh, string, error) {
	authenticated, expireSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, activateTimezone)
	userID := principal.UserID
	if err != nil {
		return false, 0, expireSession, false, rotatedSession, userID, err
	}
	if authenticated {
		return true, 0, expireSession, false, rotatedSession, userID, nil
	}
	allowed, wait, err := throttle.AllowAnonymousRequest(ctx, request)
	return allowed, wait, expireSession, false, rotatedSession, userID, err
}

// AllowAnonymousRequest limits a client without repeating session inspection.
func (throttle *AnonymousThrottle) AllowAnonymousRequest(ctx context.Context, request *http.Request) (bool, time.Duration, error) {
	result, err := throttle.limiter.Allow(ctx, throttle.scope, anonymousThrottleIdentifier(request))
	return result.Allowed, result.Wait, err
}

func (throttle *AnonymousThrottle) InspectSession(ctx context.Context, request *http.Request) (bool, bool, *browsersession.SessionRefresh, string, error) {
	_, expireSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, false)
	return expireSession, false, rotatedSession, principal.UserID, err
}

// InspectSessionWithTimezone matches authentication inside BaseViewSet, whose
// TimezoneMixin activates and validates the authenticated user's timezone.
func (throttle *AnonymousThrottle) InspectSessionWithTimezone(ctx context.Context, request *http.Request) (bool, bool, *browsersession.SessionRefresh, string, error) {
	_, expireSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, true)
	return expireSession, false, rotatedSession, principal.UserID, err
}

func (throttle *AnonymousThrottle) InspectReadServiceSession(ctx context.Context, request *http.Request) (bool, bool, *browsersession.SessionRefresh, browsersession.Principal, error) {
	_, expireSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, true)
	return expireSession, false, rotatedSession, principal, err
}

// InspectReadServiceSessionWithoutTimezone leaves timezone validation to views
// whose permission classes must run before TimezoneMixin.
func (throttle *AnonymousThrottle) InspectReadServiceSessionWithoutTimezone(ctx context.Context, request *http.Request) (bool, bool, *browsersession.SessionRefresh, browsersession.Principal, error) {
	_, expireSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, false)
	return expireSession, false, rotatedSession, principal, err
}

func (throttle *AnonymousThrottle) inspectSession(ctx context.Context, request *http.Request, activateTimezone bool) (bool, bool, *browsersession.SessionRefresh, browsersession.Principal, error) {
	expireSession := false
	cookieName, _ := throttle.sessionConfig(request)
	if cookie := lastCookie(request, cookieName); cookie != nil {
		if principal, prepared, validationErr := throttle.sessions.Inspect(ctx, cookie.Value); validationErr == nil {
			// Rotate the session key when authentication used a fallback signing secret.
			var rotatedSession *browsersession.SessionRefresh
			if principal.AuthHashUsedFallback {
				rotated, refreshErr := throttle.sessions.RotateFallbackPrepared(ctx, prepared, throttle.cookieAge, throttle.now())
				if refreshErr != nil {
					if errors.Is(refreshErr, browsersession.ErrSessionNotFound) {
						return false, true, nil, browsersession.Principal{}, nil
					}
					return false, false, nil, browsersession.Principal{}, fmt.Errorf("refresh browser session: %w", refreshErr)
				}
				rotatedSession = &rotated
			}
			if activateTimezone {
				if timezoneErr := browsersession.ValidateUserTimezone(principal.UserTimezone); timezoneErr != nil {
					if rotatedSession == nil && throttle.saveEvery {
						rotatedSession = &prepared
					}
					return false, false, rotatedSession, principal, fmt.Errorf("activate browser user timezone: %w", timezoneErr)
				}
			}
			if rotatedSession != nil {
				return true, false, rotatedSession, principal, nil
			}
			if throttle.saveEvery {
				return true, false, &prepared, principal, nil
			}
			return true, false, nil, principal, nil
		} else if browsersession.IsReadFailure(validationErr) {
			// A failed session read cannot authorize an anonymous fallback.
			return false, false, nil, browsersession.Principal{}, fmt.Errorf("validate browser session for anonymous throttle: %w", validationErr)
		} else if sessionCookieMustExpire(validationErr) {
			expireSession = true
			if errors.Is(validationErr, browsersession.ErrInvalidAuthHash) {
				if err := throttle.invalidator.InvalidateSession(ctx, cookie.Value); err != nil {
					return false, false, nil, browsersession.Principal{}, fmt.Errorf("%w: invalidate browser session after authentication hash mismatch: %w", ErrSessionInvalidation, err)
				}
			}
		} else if throttle.saveEvery {
			// Persist valid anonymous snapshots, including pending OAuth state.
			return false, false, &prepared, browsersession.Principal{}, nil
		}
	}
	return false, expireSession, nil, browsersession.Principal{}, nil
}

func lastCookie(request *http.Request, name string) *http.Cookie {
	return browsersession.CookieFromHeaderValues(request.Header.Values("Cookie"), name)
}

func latin1String(value []byte) string {
	characters := make([]rune, len(value))
	for index, octet := range value {
		characters[index] = rune(octet)
	}
	return string(characters)
}

func (throttle *AnonymousThrottle) RefreshSession(ctx context.Context, request *http.Request, prepared browsersession.SessionRefresh) (browsersession.SessionRefresh, error) {
	_, adminSession := throttle.sessionConfig(request)
	now := throttle.now()
	refresh, err := throttle.sessions.RefreshPrepared(ctx, prepared, throttle.cookieAge, now)
	if err != nil {
		return browsersession.SessionRefresh{}, fmt.Errorf("refresh browser session: %w", err)
	}
	if adminSession && !refresh.BrowserClose {
		expiresAt, expiryErr := addCookieAge(now, throttle.adminCookieAge)
		if expiryErr != nil {
			return browsersession.SessionRefresh{}, fmt.Errorf("refresh admin browser session cookie: %w", expiryErr)
		}
		refresh.MaxAgeSeconds = float64(throttle.adminCookieAge)
		refresh.CookieExpiresAt = expiresAt
	}
	return refresh, nil
}

// FinalizeUnaccessedSession applies SessionMiddleware's response behavior
// when the body-size middleware returns before authentication and throttling.
func (throttle *AnonymousThrottle) FinalizeUnaccessedSession(ctx context.Context, request *http.Request, status int) (bool, *browsersession.SessionRefresh, error) {
	cookieName, adminSession := throttle.sessionConfig(request)
	cookie := lastCookie(request, cookieName)
	if cookie == nil {
		return false, nil, nil
	}
	if utf8.RuneCountInString(cookie.Value) < 8 {
		return true, nil, nil
	}
	if !throttle.saveEvery || status >= http.StatusInternalServerError {
		return false, nil, nil
	}
	now := throttle.now()
	refresh, err := throttle.sessions.RefreshUnaccessed(ctx, cookie.Value, throttle.cookieAge, now)
	if err != nil {
		if sessionCookieMustExpire(err) {
			return true, nil, nil
		}
		return false, nil, fmt.Errorf("refresh unaccessed session: %w", err)
	}
	if adminSession && !refresh.BrowserClose {
		expiresAt, expiryErr := addCookieAge(now, throttle.adminCookieAge)
		if expiryErr != nil {
			return false, nil, fmt.Errorf("refresh unaccessed admin browser session cookie: %w", expiryErr)
		}
		refresh.MaxAgeSeconds = float64(throttle.adminCookieAge)
		refresh.CookieExpiresAt = expiresAt
	}
	return false, &refresh, nil
}

func addCookieAge(now time.Time, seconds int64) (time.Time, error) {
	unix := now.Unix()
	if (seconds > 0 && unix > math.MaxInt64-seconds) || (seconds < 0 && unix < math.MinInt64-seconds) {
		return time.Time{}, errors.New("admin browser session cookie expiry is outside the supported time range")
	}
	result := time.Unix(unix+seconds, int64(now.Nanosecond())).In(now.Location())
	if result.Year() < 1 || result.Year() > 9999 {
		return time.Time{}, errors.New("admin browser session cookie expiry is outside the datetime range")
	}
	return result, nil
}

func (throttle *AnonymousThrottle) sessionConfig(request *http.Request) (string, bool) {
	// The instance API owns the admin session; user-controlled path segments do not.
	if request.URL.Path == "/api/instances" || strings.HasPrefix(request.URL.Path, "/api/instances/") {
		return throttle.adminCookieName, true
	}
	return throttle.cookieName, false
}

func sessionCookieMustExpire(err error) bool {
	return errors.Is(err, browsersession.ErrInvalidData) ||
		errors.Is(err, browsersession.ErrInvalidKey) ||
		errors.Is(err, browsersession.ErrSessionNotFound) ||
		errors.Is(err, browsersession.ErrInvalidAuthHash)
}

type sessionInvalidationDatabase interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type PostgreSQLSessionInvalidator struct{ database sessionInvalidationDatabase }

func NewPostgreSQLSessionInvalidator(database sessionInvalidationDatabase) (*PostgreSQLSessionInvalidator, error) {
	if database == nil {
		return nil, fmt.Errorf("browser session invalidation database is required")
	}
	return &PostgreSQLSessionInvalidator{database: database}, nil
}

func (invalidator *PostgreSQLSessionInvalidator) InvalidateSession(ctx context.Context, sessionKey string) error {
	if _, err := invalidator.database.Exec(ctx, "DELETE FROM sessions WHERE session_key = $1", sessionKey); err != nil {
		return fmt.Errorf("delete browser session: %w", err)
	}
	return nil
}

func anonymousThrottleIdentifier(request *http.Request) string {
	if forwarded := strings.Join(request.Header.Values("X-Forwarded-For"), ","); forwarded != "" {
		forwarded = latin1String([]byte(forwarded))
		return strings.Map(func(value rune) rune {
			if unicode.IsSpace(value) {
				return -1
			}
			return value
		}, forwarded)
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return request.RemoteAddr
}
