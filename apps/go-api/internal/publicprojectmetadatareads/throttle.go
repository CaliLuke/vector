package publicprojectmetadatareads

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

const (
	anonThrottleLimit  = 30
	anonThrottleWindow = time.Minute
)

var ErrSessionInvalidation = errors.New("Django session invalidation failed")

type throttleCache interface {
	Get(context.Context, string) ([]byte, bool, error)
	Set(context.Context, string, []byte, time.Duration) error
}

type throttleSessionValidator interface {
	Inspect(context.Context, string) (djangosession.Principal, djangosession.SessionRefresh, error)
	RefreshPrepared(context.Context, djangosession.SessionRefresh, int64, time.Time) (djangosession.SessionRefresh, error)
	RefreshUnaccessed(context.Context, string, int64, time.Time) (djangosession.SessionRefresh, error)
	RotateFallbackPrepared(context.Context, djangosession.SessionRefresh, int64, time.Time) (djangosession.SessionRefresh, error)
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
	requests        *big.Int
	window          time.Duration
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
	requests, window, err := parseAnonymousThrottleRate(config.Rate)
	if err != nil {
		return nil, err
	}
	adminCookieName := "admin-session-id"
	adminCookieAge := int64(time.Hour / time.Second)
	if len(admin) > 1 {
		return nil, fmt.Errorf("at most one Django admin session configuration is allowed")
	}
	if len(admin) == 1 {
		adminCookieName = admin[0].CookieName
		adminCookieAge = admin[0].CookieAge
	}
	if adminCookieName == "" {
		return nil, fmt.Errorf("Django admin session cookie name is required")
	}
	return &AnonymousThrottle{
		cache: cache, sessions: sessions, invalidator: invalidator, cookieName: cookieName,
		cookieAge: cookieAge, adminCookieName: adminCookieName, adminCookieAge: adminCookieAge,
		saveEvery: saveEvery, scope: config.Scope, requests: requests, window: window, now: time.Now,
	}, nil
}

func (throttle *AnonymousThrottle) AllowRequest(ctx context.Context, request *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	return throttle.allowRequest(ctx, request, true)
}

// AllowRequestWithoutTimezone preserves anonymous throttling and Django session
// behavior for views that do not include TimezoneMixin.
func (throttle *AnonymousThrottle) AllowRequestWithoutTimezone(ctx context.Context, request *http.Request) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	return throttle.allowRequest(ctx, request, false)
}

func (throttle *AnonymousThrottle) allowRequest(ctx context.Context, request *http.Request, activateTimezone bool) (bool, time.Duration, bool, bool, *djangosession.SessionRefresh, string, error) {
	authenticated, expireSession, refreshSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, activateTimezone)
	userID := principal.UserID
	if err != nil {
		return false, 0, expireSession, refreshSession, rotatedSession, userID, err
	}
	if authenticated {
		return true, 0, expireSession, refreshSession, rotatedSession, userID, nil
	}
	allowed, wait, err := throttle.AllowAnonymousRequest(ctx, request)
	return allowed, wait, expireSession, refreshSession, rotatedSession, userID, err
}

// AllowAnonymousRequest applies Django's anonymous throttle bucket without
// inspecting or refreshing a session. Callers that already ran the shared
// session middleware can use this to avoid a second session lifecycle pass.
func (throttle *AnonymousThrottle) AllowAnonymousRequest(ctx context.Context, request *http.Request) (bool, time.Duration, error) {
	key := ":1:throttle_" + throttle.scope + "_" + anonymousThrottleIdentifier(request)
	data, found, err := throttle.cache.Get(ctx, key)
	if err != nil {
		return false, 0, fmt.Errorf("read Django anonymous throttle: %w", err)
	}
	var history []float64
	if found {
		history, err = decodeFloatListPickle(data)
		if err != nil {
			return false, 0, fmt.Errorf("decode Django anonymous throttle: %w", err)
		}
	}
	now := float64(throttle.now().UnixNano()) / float64(time.Second)
	cutoff := now - throttle.window.Seconds()
	for len(history) > 0 && history[len(history)-1] <= cutoff {
		history = history[:len(history)-1]
	}
	historyLength := int64(len(history))
	requestLimit := int64(0)
	limitReached := throttle.requests.Sign() < 0
	if throttle.requests.IsInt64() {
		requestLimit = throttle.requests.Int64()
		limitReached = historyLength >= requestLimit
	}
	if limitReached {
		if !throttle.requests.IsInt64() || historyLength != requestLimit {
			return false, 0, nil
		}
		remaining := throttle.window.Seconds()
		if len(history) > 0 {
			remaining -= now - history[len(history)-1]
		}
		return false, time.Duration(math.Ceil(remaining)) * time.Second, nil
	}
	history = append([]float64{now}, history...)
	if err := throttle.cache.Set(ctx, key, encodeFloatListPickle(history), throttle.window); err != nil {
		return false, 0, fmt.Errorf("write Django anonymous throttle: %w", err)
	}
	return true, 0, nil
}

func parseAnonymousThrottleRate(rate string) (*big.Int, time.Duration, error) {
	parts := strings.Split(rate, "/")
	if len(parts) != 2 {
		return nil, 0, fmt.Errorf("invalid anonymous throttle rate %q", rate)
	}
	requests, err := pythoncompat.DecimalInteger(parts[0])
	if err != nil {
		return nil, 0, fmt.Errorf("invalid anonymous throttle rate %q", rate)
	}
	if parts[1] == "" {
		return nil, 0, fmt.Errorf("invalid anonymous throttle rate period %q", parts[1])
	}
	var window time.Duration
	switch parts[1][0] {
	case 's':
		window = time.Second
	case 'm':
		window = time.Minute
	case 'h':
		window = time.Hour
	case 'd':
		window = 24 * time.Hour
	default:
		return nil, 0, fmt.Errorf("invalid anonymous throttle rate period %q", parts[1])
	}
	return requests, window, nil
}

func (throttle *AnonymousThrottle) InspectSession(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error) {
	_, expireSession, refreshSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, false)
	return expireSession, refreshSession, rotatedSession, principal.UserID, err
}

// InspectSessionWithTimezone matches authentication inside BaseViewSet, whose
// TimezoneMixin activates and validates the authenticated user's timezone.
func (throttle *AnonymousThrottle) InspectSessionWithTimezone(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, string, error) {
	_, expireSession, refreshSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, true)
	return expireSession, refreshSession, rotatedSession, principal.UserID, err
}

func (throttle *AnonymousThrottle) InspectReadServiceSession(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	_, expireSession, refreshSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, true)
	return expireSession, refreshSession, rotatedSession, principal, err
}

// InspectReadServiceSessionWithoutTimezone leaves timezone validation to views
// whose permission classes must run before TimezoneMixin.
func (throttle *AnonymousThrottle) InspectReadServiceSessionWithoutTimezone(ctx context.Context, request *http.Request) (bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	_, expireSession, refreshSession, rotatedSession, principal, err := throttle.inspectSession(ctx, request, false)
	return expireSession, refreshSession, rotatedSession, principal, err
}

func (throttle *AnonymousThrottle) inspectSession(ctx context.Context, request *http.Request, activateTimezone bool) (bool, bool, bool, *djangosession.SessionRefresh, djangosession.Principal, error) {
	expireSession := false
	refreshSession := false
	cookieName, _ := throttle.sessionConfig(request)
	if cookie := lastCookie(request, cookieName); cookie != nil {
		if principal, prepared, validationErr := throttle.sessions.Inspect(ctx, cookie.Value); validationErr == nil {
			// Django cycles the session key during authentication only when the
			// authentication hash used a fallback SECRET_KEY.
			var rotatedSession *djangosession.SessionRefresh
			if principal.AuthHashUsedFallback {
				rotated, refreshErr := throttle.sessions.RotateFallbackPrepared(ctx, prepared, throttle.cookieAge, throttle.now())
				if refreshErr != nil {
					return false, false, false, nil, djangosession.Principal{}, fmt.Errorf("refresh Django session: %w", refreshErr)
				}
				rotatedSession = &rotated
			}
			if activateTimezone {
				if timezoneErr := djangosession.ValidateUserTimezone(principal.UserTimezone); timezoneErr != nil {
					if rotatedSession == nil && throttle.saveEvery {
						rotatedSession = &prepared
					}
					return false, false, false, rotatedSession, principal, fmt.Errorf("activate Django user timezone: %w", timezoneErr)
				}
			}
			if rotatedSession != nil {
				return true, false, false, rotatedSession, principal, nil
			}
			if throttle.saveEvery {
				return true, false, false, &prepared, principal, nil
			}
			return true, false, false, nil, principal, nil
		} else if errors.Is(validationErr, djangosession.ErrStore) ||
			errors.Is(validationErr, djangosession.ErrSessionStore) ||
			errors.Is(validationErr, djangosession.ErrUserStore) ||
			errors.Is(validationErr, djangosession.ErrInvalidSessionShape) ||
			errors.Is(validationErr, djangosession.ErrInvalidUserID) {
			// BaseAPIView can map the first lazy-authentication exception, but
			// RequestLogger evaluates the still-unresolved SimpleLazyObject again
			// outside the view and Django ultimately returns a technical 500.
			return false, false, false, nil, djangosession.Principal{}, fmt.Errorf("validate Django session for anonymous throttle: %w", validationErr)
		} else if djangoSessionCookieMustExpire(validationErr) {
			expireSession = true
			if errors.Is(validationErr, djangosession.ErrInvalidAuthHash) {
				if err := throttle.invalidator.InvalidateSession(ctx, cookie.Value); err != nil {
					return false, false, false, nil, djangosession.Principal{}, fmt.Errorf("%w: invalidate Django session after authentication hash mismatch: %w", ErrSessionInvalidation, err)
				}
			}
		} else if throttle.saveEvery {
			// Database SessionStore retains a supplied key when a valid or corrupt
			// payload decodes to {}. Django therefore considers it non-empty and
			// re-signs it under SESSION_SAVE_EVERY_REQUEST.
			return false, false, false, &prepared, djangosession.Principal{}, nil
		}
	}
	return false, expireSession, refreshSession, nil, djangosession.Principal{}, nil
}

func lastCookie(request *http.Request, name string) *http.Cookie {
	values := make(map[string]string)
	header := latin1String([]byte(strings.Join(request.Header.Values("Cookie"), ",")))
	for _, chunk := range strings.Split(header, ";") {
		key, value, found := strings.Cut(chunk, "=")
		if !found {
			key, value = "", key
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" && value == "" {
			continue
		}
		values[key] = djangoCookieUnquote(value)
	}
	value, found := values[name]
	if !found {
		return nil
	}
	return &http.Cookie{Name: name, Value: value}
}

func djangoCookieUnquote(value string) string {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return value
	}
	characters := []rune(value[1 : len(value)-1])
	var decoded strings.Builder
	decoded.Grow(len(value))
	for index := 0; index < len(characters); index++ {
		if characters[index] != '\\' || index+1 >= len(characters) {
			decoded.WriteRune(characters[index])
			continue
		}
		if index+3 < len(characters) && characters[index+1] >= '0' && characters[index+1] <= '3' &&
			characters[index+2] >= '0' && characters[index+2] <= '7' && characters[index+3] >= '0' && characters[index+3] <= '7' {
			decoded.WriteRune((characters[index+1]-'0')*64 + (characters[index+2]-'0')*8 + characters[index+3] - '0')
			index += 3
			continue
		}
		decoded.WriteRune(characters[index+1])
		index++
	}
	return decoded.String()
}

func latin1String(value []byte) string {
	characters := make([]rune, len(value))
	for index, octet := range value {
		characters[index] = rune(octet)
	}
	return string(characters)
}

func (throttle *AnonymousThrottle) RefreshSession(ctx context.Context, request *http.Request, prepared djangosession.SessionRefresh) (djangosession.SessionRefresh, error) {
	_, adminSession := throttle.sessionConfig(request)
	now := throttle.now()
	refresh, err := throttle.sessions.RefreshPrepared(ctx, prepared, throttle.cookieAge, now)
	if err != nil {
		return djangosession.SessionRefresh{}, fmt.Errorf("refresh Django session: %w", err)
	}
	if adminSession && !refresh.BrowserClose {
		expiresAt, expiryErr := addCookieAge(now, throttle.adminCookieAge)
		if expiryErr != nil {
			return djangosession.SessionRefresh{}, fmt.Errorf("refresh Django admin session cookie: %w", expiryErr)
		}
		refresh.MaxAgeSeconds = float64(throttle.adminCookieAge)
		refresh.CookieExpiresAt = expiresAt
	}
	return refresh, nil
}

// FinalizeUnaccessedSession applies SessionMiddleware's response behavior
// when the body-size middleware returns before authentication and throttling.
func (throttle *AnonymousThrottle) FinalizeUnaccessedSession(ctx context.Context, request *http.Request, status int) (bool, *djangosession.SessionRefresh, error) {
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
		return false, nil, fmt.Errorf("refresh unaccessed Django session: %w", err)
	}
	if adminSession && !refresh.BrowserClose {
		expiresAt, expiryErr := addCookieAge(now, throttle.adminCookieAge)
		if expiryErr != nil {
			return false, nil, fmt.Errorf("refresh unaccessed Django admin session cookie: %w", expiryErr)
		}
		refresh.MaxAgeSeconds = float64(throttle.adminCookieAge)
		refresh.CookieExpiresAt = expiresAt
	}
	return false, &refresh, nil
}

func addCookieAge(now time.Time, seconds int64) (time.Time, error) {
	unix := now.Unix()
	if (seconds > 0 && unix > math.MaxInt64-seconds) || (seconds < 0 && unix < math.MinInt64-seconds) {
		return time.Time{}, errors.New("Django admin session cookie expiry is outside the supported time range")
	}
	result := time.Unix(unix+seconds, int64(now.Nanosecond())).In(now.Location())
	if result.Year() < 1 || result.Year() > 9999 {
		return time.Time{}, errors.New("Django admin session cookie expiry is outside the datetime range")
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

func djangoSessionCookieMustExpire(err error) bool {
	return errors.Is(err, djangosession.ErrInvalidKey) ||
		errors.Is(err, djangosession.ErrSessionNotFound) ||
		errors.Is(err, djangosession.ErrInvalidAuthHash)
}

type sessionInvalidationDatabase interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type PostgreSQLSessionInvalidator struct{ database sessionInvalidationDatabase }

func NewPostgreSQLSessionInvalidator(database sessionInvalidationDatabase) (*PostgreSQLSessionInvalidator, error) {
	if database == nil {
		return nil, fmt.Errorf("Django session invalidation database is required")
	}
	return &PostgreSQLSessionInvalidator{database: database}, nil
}

func (invalidator *PostgreSQLSessionInvalidator) InvalidateSession(ctx context.Context, sessionKey string) error {
	if _, err := invalidator.database.Exec(ctx, "DELETE FROM sessions WHERE session_key = $1", sessionKey); err != nil {
		return fmt.Errorf("delete Django session: %w", err)
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

func encodeFloatListPickle(values []float64) []byte {
	payload := []byte{']', 0x94}
	if len(values) == 1 {
		payload = append(payload, 'G')
		payload = binary.BigEndian.AppendUint64(payload, math.Float64bits(values[0]))
		payload = append(payload, 'a')
	} else if len(values) > 1 {
		payload = append(payload, '(')
		for _, value := range values {
			payload = append(payload, 'G')
			payload = binary.BigEndian.AppendUint64(payload, math.Float64bits(value))
		}
		payload = append(payload, 'e')
	}
	payload = append(payload, '.')
	encoded := []byte{0x80, 0x04, 0x95}
	encoded = binary.LittleEndian.AppendUint64(encoded, uint64(len(payload)))
	return append(encoded, payload...)
}

func decodeFloatListPickle(data []byte) ([]float64, error) {
	values := make([]float64, 0, anonThrottleLimit)
	for index := 0; index < len(data); {
		opcode := data[index]
		index++
		switch opcode {
		case 0x80:
			if index >= len(data) || data[index] > 5 {
				return nil, fmt.Errorf("unsupported pickle protocol")
			}
			index++
		case 0x95:
			if index+8 > len(data) {
				return nil, fmt.Errorf("truncated pickle frame")
			}
			index += 8
		case 'G':
			if index+8 > len(data) {
				return nil, fmt.Errorf("truncated pickle float")
			}
			values = append(values, math.Float64frombits(binary.BigEndian.Uint64(data[index:index+8])))
			index += 8
		case ']', 0x94, '(', 'a', 'e':
		case '.':
			return values, nil
		default:
			return nil, fmt.Errorf("unsupported pickle opcode 0x%02x", opcode)
		}
	}
	return nil, fmt.Errorf("pickle has no stop opcode")
}

type RedisThrottleCache struct {
	redisURL string
	once     sync.Once
	client   *redis.Client
	err      error
}

func NewRedisThrottleCache(redisURL string) (*RedisThrottleCache, error) {
	return &RedisThrottleCache{redisURL: redisURL}, nil
}

func redisThrottleOptions(redisURL string) (*redis.Options, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis URL: %w", err)
	}
	// Django's cache URL sets ssl_cert_reqs=False for rediss:// connections.
	// Preserve that compatibility policy while this service shares the cache.
	if options.TLSConfig != nil {
		options.TLSConfig.InsecureSkipVerify = true //nolint:gosec // Matches the Django deployment configuration.
	}
	return options, nil
}

func (cache *RedisThrottleCache) Ping(ctx context.Context) error {
	client, err := cache.clientForUse()
	if err != nil {
		return err
	}
	return client.Ping(ctx).Err()
}

func (cache *RedisThrottleCache) Close() error {
	if cache.client == nil {
		return nil
	}
	return cache.client.Close()
}

func (cache *RedisThrottleCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	client, err := cache.clientForUse()
	if err != nil {
		return nil, false, err
	}
	data, err := client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	return data, err == nil, err
}

func (cache *RedisThrottleCache) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	client, err := cache.clientForUse()
	if err != nil {
		return err
	}
	return client.Set(ctx, key, value, expiration).Err()
}

func (cache *RedisThrottleCache) clientForUse() (*redis.Client, error) {
	cache.once.Do(func() {
		options, err := redisThrottleOptions(cache.redisURL)
		if err != nil {
			cache.err = err
			return
		}
		cache.client = redis.NewClient(options)
	})
	return cache.client, cache.err
}
