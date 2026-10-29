// Package djangosession validates Plane's database-backed Django sessions.
//
// The implementation matches Django 5.2.15. Validation is read-only, while
// Refresh supports Django's optional SESSION_SAVE_EVERY_REQUEST behavior.
package djangosession

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/google/uuid"
)

const (
	DefaultCookieName  = "session-id"
	SessionSigningSalt = "django.contrib.sessions.SessionStore"
	AuthHashSalt       = "django.contrib.auth.models.AbstractBaseUser.get_session_auth_hash"
)

var (
	ErrNoCookie            = errors.New("Django session cookie is missing")
	ErrInvalidKey          = errors.New("Django session key is invalid")
	ErrSessionNotFound     = errors.New("Django session is missing or expired")
	ErrInvalidData         = errors.New("Django session data is invalid")
	ErrInvalidSessionShape = errors.New("Django session data is not an object")
	ErrBadSignature        = errors.New("Django session signature does not match")
	ErrUserMismatch        = errors.New("Django session user does not match its row")
	ErrBackend             = errors.New("Django authentication backend is not configured")
	ErrInvalidUserID       = errors.New("Django session user identifier is invalid")
	ErrUserNotFound        = errors.New("Django session user does not exist")
	ErrInactiveUser        = errors.New("Django session user is inactive")
	ErrInvalidAuthHash     = errors.New("Django session authentication hash is invalid")
	ErrInvalidTimezone     = errors.New("Django user timezone key is invalid")
	ErrTimezoneNotFound    = errors.New("Django user timezone does not exist")
	ErrStore               = errors.New("Django session store read failed")
	ErrSessionStore        = errors.New("Django session table access failed")
	ErrUserStore           = errors.New("Django user table access failed")
	ErrKeyCollision        = errors.New("Django session key already exists")
	ErrSessionUpdate       = errors.New("Django session update failed")
)

type Session struct {
	Data string
}

type User struct {
	Password string
	Active   bool
	Timezone string
}

// Reader exposes the reads needed by session validation and response-time
// refresh. ReadSession must use the application clock when it checks expiry;
// ReadSessionForRefresh must retain an already-loaded row that expires while
// the request is running.
type Reader interface {
	ReadSession(context.Context, string) (Session, error)
	ReadSessionForRefresh(context.Context, string) (Session, error)
	ReadUser(context.Context, string) (User, error)
}

type Config struct {
	CookieName           string
	CookieNameConfigured bool
	SecretKey            string
	FallbackKeys         []string
	AllowedBackends      []string
}

type Principal struct {
	UserID               string
	SessionKey           string
	UsedFallback         bool
	AuthHashUsedFallback bool
	UserTimezone         string
	values               map[string]any
	serialized           []byte
	currentAuthHash      string
}

type SessionRefresh struct {
	SessionKey      string
	ExpiresAt       time.Time
	CookieExpiresAt time.Time
	MaxAgeSeconds   float64
	BrowserClose    bool
	principal       *Principal
}

type sessionRefreshWriter interface {
	UpdateSession(context.Context, string, string, time.Time, any, any) error
}

type sessionCreateWriter interface {
	CreateSession(context.Context, string, string, time.Time, any, any) error
}

type sessionCycleWriter interface {
	CycleSession(context.Context, string, string, string, time.Time, any, any) error
}

type sessionDeleteWriter interface {
	DeleteSession(context.Context, string) error
}

type Validator struct {
	reader   Reader
	cookie   string
	secrets  []string
	backends map[string]struct{}
}

func New(reader Reader, config Config) (*Validator, error) {
	if reader == nil {
		return nil, errors.New("Django session reader is required")
	}
	if config.SecretKey == "" {
		return nil, errors.New("Django SECRET_KEY is required")
	}
	if len(config.AllowedBackends) == 0 {
		return nil, errors.New("at least one Django authentication backend is required")
	}
	cookie := config.CookieName
	if !config.CookieNameConfigured && cookie == "" {
		cookie = DefaultCookieName
	}
	secrets := make([]string, 0, 1+len(config.FallbackKeys))
	secrets = append(secrets, config.SecretKey)
	for _, key := range config.FallbackKeys {
		if key == "" {
			return nil, errors.New("Django fallback SECRET_KEY must not be empty")
		}
		secrets = append(secrets, key)
	}
	backends := make(map[string]struct{}, len(config.AllowedBackends))
	for _, backend := range config.AllowedBackends {
		if backend == "" {
			return nil, errors.New("Django authentication backend must not be empty")
		}
		backends[backend] = struct{}{}
	}
	return &Validator{reader: reader, cookie: cookie, secrets: secrets, backends: backends}, nil
}

func (validator *Validator) ValidateRequest(ctx context.Context, request *http.Request) (Principal, error) {
	cookie, err := request.Cookie(validator.cookie)
	if err != nil || cookie.Value == "" {
		return Principal{}, ErrNoCookie
	}
	return validator.Validate(ctx, cookie.Value)
}

func (validator *Validator) Validate(ctx context.Context, sessionKey string) (Principal, error) {
	principal, _, err := validator.Inspect(ctx, sessionKey)
	return principal, err
}

// Inspect authenticates a session and retains the request-local session data
// even when the data represents an anonymous user. AuthenticationMiddleware
// can leave that data unchanged while SessionMiddleware saves it later.
func (validator *Validator) Inspect(ctx context.Context, sessionKey string) (Principal, SessionRefresh, error) {
	principal, values, serialized, err := validator.validate(ctx, sessionKey)
	if values == nil {
		return principal, SessionRefresh{}, err
	}
	prepared := validator.prepareSessionValues(sessionKey, values, serialized)
	if err == nil {
		prepared = principal.PrepareRefresh()
	}
	return principal, prepared, err
}

func (validator *Validator) validate(ctx context.Context, sessionKey string) (Principal, map[string]any, []byte, error) {
	// Django rejects false or shorter-than-eight keys before querying its store.
	if utf8.RuneCountInString(sessionKey) < 8 {
		return Principal{}, nil, nil, ErrInvalidKey
	}
	session, err := validator.reader.ReadSession(ctx, sessionKey)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Principal{}, nil, nil, ErrSessionNotFound
		}
		return Principal{}, nil, nil, fmt.Errorf("%w: %w: %w", ErrStore, ErrSessionStore, err)
	}
	values, serialized, sessionSecret, err := decodeSessionData(session.Data, validator.secrets)
	if err != nil {
		if errors.Is(err, ErrInvalidSessionShape) {
			return Principal{}, nil, nil, err
		}
		logSuspiciousSession(ctx, err)
		return Principal{}, map[string]any{}, []byte("{}"), err
	}
	principal, _, err := validator.validateValues(ctx, sessionKey, values, sessionSecret)
	principal.serialized = append([]byte(nil), serialized...)
	return principal, values, serialized, err
}

func (validator *Validator) validateValues(
	ctx context.Context,
	sessionKey string,
	values map[string]any,
	sessionSecret int,
) (Principal, string, error) {
	rawUserID, found := values["_auth_user_id"]
	if !found {
		return Principal{}, "", ErrUserMismatch
	}
	var userID string
	if rawUserID != nil {
		var err error
		userID, err = djangoUUIDValue(rawUserID)
		if err != nil {
			return Principal{}, "", ErrInvalidUserID
		}
	}
	backend, ok := stringValue(values, "_auth_user_backend")
	if !ok {
		return Principal{}, "", ErrBackend
	}
	if _, supported := validator.backends[backend]; !supported {
		return Principal{}, "", ErrBackend
	}
	// UUIDField.get_prep_value() retains None. ModelBackend then performs a
	// primary-key NULL lookup and treats the session as anonymous.
	if rawUserID == nil {
		return Principal{}, "", ErrUserNotFound
	}
	user, err := validator.reader.ReadUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrInvalidUserID) {
			return Principal{}, "", ErrInvalidUserID
		}
		if errors.Is(err, ErrUserNotFound) {
			return Principal{}, "", ErrUserNotFound
		}
		return Principal{}, "", fmt.Errorf("%w: %w: %w", ErrStore, ErrUserStore, err)
	}
	if !user.Active {
		return Principal{}, "", ErrInactiveUser
	}
	authHash, ok := stringValue(values, "_auth_user_hash")
	if !ok || authHash == "" {
		return Principal{}, "", ErrInvalidAuthHash
	}
	authSecret := matchingAuthSecret(authHash, user.Password, validator.secrets)
	if authSecret < 0 {
		return Principal{}, "", ErrInvalidAuthHash
	}
	currentAuthHash := hex.EncodeToString(djangoHMAC(AuthHashSalt, user.Password, validator.secrets[0]))
	return Principal{
		UserID:               userID,
		SessionKey:           sessionKey,
		UsedFallback:         sessionSecret > 0 || authSecret > 0,
		AuthHashUsedFallback: authSecret > 0,
		UserTimezone:         user.Timezone,
		values:               cloneSessionValues(values),
		currentAuthHash:      currentAuthHash,
	}, currentAuthHash, nil
}

// Refresh revalidates and saves a session with a new signing timestamp and
// database expiry, matching Django's database SessionStore.save() path.
func (validator *Validator) Refresh(ctx context.Context, sessionKey string, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if utf8.RuneCountInString(sessionKey) < 8 {
		return SessionRefresh{}, ErrInvalidKey
	}
	// Django saves the session data loaded during request authentication. It
	// does not reapply the expiry predicate when SESSION_SAVE_EVERY_REQUEST
	// persists that data at response time.
	session, err := validator.reader.ReadSessionForRefresh(ctx, sessionKey)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return SessionRefresh{}, ErrSessionNotFound
		}
		return SessionRefresh{}, fmt.Errorf("%w: %w: %w", ErrStore, ErrSessionStore, err)
	}
	values, serialized, sessionSecret, decodeErr := decodeSessionData(session.Data, validator.secrets)
	if decodeErr != nil {
		if errors.Is(decodeErr, ErrInvalidSessionShape) {
			return SessionRefresh{}, fmt.Errorf("%w: %w", ErrStore, decodeErr)
		}
		logSuspiciousSession(ctx, decodeErr)
		// Django's signed-session decoder converts corrupt stored payloads to
		// an empty session before SESSION_SAVE_EVERY_REQUEST persists them.
		values = map[string]any{}
		serialized = []byte("{}")
	}
	rotateKey := false
	if decodeErr == nil {
		principal, currentAuthHash, validationErr := validator.validateValues(ctx, sessionKey, values, sessionSecret)
		if validationErr == nil && principal.AuthHashUsedFallback {
			values["_auth_user_hash"] = currentAuthHash
			serialized, decodeErr = replaceTopLevelJSONString(serialized, "_auth_user_hash", currentAuthHash)
			if decodeErr != nil {
				return SessionRefresh{}, fmt.Errorf("%w: update Django session authentication hash: %w", ErrStore, decodeErr)
			}
			rotateKey = true
		}
	}
	return validator.refreshValues(ctx, sessionKey, values, serialized, rotateKey, false, defaultAgeSeconds, now)
}

// RefreshPrincipal persists the session snapshot loaded by Validate. Django
// keeps this request-local data through cycle_key(), so concurrent rotations
// can each create a replacement even after the original row is deleted.
func (validator *Validator) RefreshPrincipal(ctx context.Context, principal Principal, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if utf8.RuneCountInString(principal.SessionKey) < 8 || principal.values == nil {
		return SessionRefresh{}, ErrInvalidKey
	}
	values := cloneSessionValues(principal.values)
	serialized := append([]byte(nil), principal.serialized...)
	if principal.AuthHashUsedFallback {
		values["_auth_user_hash"] = principal.currentAuthHash
		var err error
		serialized, err = replaceTopLevelJSONString(serialized, "_auth_user_hash", principal.currentAuthHash)
		if err != nil {
			return SessionRefresh{}, fmt.Errorf("%w: update Django session authentication hash: %w", ErrStore, err)
		}
	}
	return validator.refreshValues(
		ctx, principal.SessionKey, values, serialized, principal.AuthHashUsedFallback, false, defaultAgeSeconds, now,
	)
}

// PrepareRefresh retains the request-local session snapshot without writing
// it. SessionMiddleware saves this snapshot after the response is produced.
func (principal Principal) PrepareRefresh() SessionRefresh {
	prepared := principal
	prepared.values = cloneSessionValues(principal.values)
	prepared.serialized = append([]byte(nil), principal.serialized...)
	return SessionRefresh{SessionKey: principal.SessionKey, principal: &prepared}
}

func (validator *Validator) prepareSessionValues(sessionKey string, values map[string]any, serialized []byte) SessionRefresh {
	principal := Principal{
		SessionKey:      sessionKey,
		values:          cloneSessionValues(values),
		serialized:      append([]byte(nil), serialized...),
		currentAuthHash: stringValueOrEmpty(values, "_auth_user_hash"),
	}
	principal.UserID = stringValueOrEmpty(values, "_auth_user_id")
	return SessionRefresh{SessionKey: sessionKey, principal: &principal}
}

// RefreshPrepared saves a request-local snapshot when one is available and
// falls back to loading by key for anonymous or corrupt sessions.
func (validator *Validator) RefreshPrepared(ctx context.Context, prepared SessionRefresh, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if prepared.principal != nil {
		return validator.RefreshPrincipal(ctx, *prepared.principal, defaultAgeSeconds, now)
	}
	return validator.Refresh(ctx, prepared.SessionKey, defaultAgeSeconds, now)
}

// RefreshUnaccessed applies SESSION_SAVE_EVERY_REQUEST to a session that no
// inner middleware loaded. Django saves the decoded session without running
// authentication. A missing or expired load clears Django's in-memory key;
// get_expire_at_browser_close() triggers that load before save(), so save()
// creates a replacement row under a new key.
func (validator *Validator) RefreshUnaccessed(ctx context.Context, sessionKey string, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if utf8.RuneCountInString(sessionKey) < 8 {
		return SessionRefresh{}, ErrInvalidKey
	}
	session, err := validator.reader.ReadSession(ctx, sessionKey)
	if errors.Is(err, ErrSessionNotFound) {
		return validator.refreshValues(ctx, sessionKey, map[string]any{}, []byte("{}"), false, true, defaultAgeSeconds, now)
	}
	if err != nil {
		return SessionRefresh{}, fmt.Errorf("%w: %w: %w", ErrStore, ErrSessionStore, err)
	}
	values, serialized, _, decodeErr := decodeSessionData(session.Data, validator.secrets)
	if decodeErr != nil {
		if errors.Is(decodeErr, ErrInvalidSessionShape) {
			return SessionRefresh{}, fmt.Errorf("%w: %w", ErrStore, decodeErr)
		}
		logSuspiciousSession(ctx, decodeErr)
		values = map[string]any{}
		serialized = []byte("{}")
	}
	return validator.refreshValues(ctx, sessionKey, values, serialized, false, false, defaultAgeSeconds, now)
}

// RotateFallbackPrepared matches Django's two-stage fallback-hash rotation.
// cycle_key() first writes the loaded snapshot under a new key. The request's
// in-memory hash is updated only afterwards, and SessionMiddleware persists
// that second snapshot only when the response is eligible for a session save.
func (validator *Validator) RotateFallbackPrepared(ctx context.Context, prepared SessionRefresh, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if prepared.principal == nil || !prepared.principal.AuthHashUsedFallback {
		return SessionRefresh{}, fmt.Errorf("%w: Django fallback session snapshot is invalid", ErrInvalidKey)
	}
	principal := *prepared.principal
	rotated, err := validator.refreshValues(
		ctx,
		principal.SessionKey,
		cloneSessionValues(principal.values),
		append([]byte(nil), principal.serialized...),
		true,
		false,
		defaultAgeSeconds,
		now,
	)
	if err != nil {
		return SessionRefresh{}, err
	}
	values := cloneSessionValues(principal.values)
	values["_auth_user_hash"] = principal.currentAuthHash
	serialized, err := replaceTopLevelJSONString(principal.serialized, "_auth_user_hash", principal.currentAuthHash)
	if err != nil {
		return SessionRefresh{}, fmt.Errorf("%w: update Django session authentication hash: %w", ErrStore, err)
	}
	requestPrincipal := principal
	requestPrincipal.SessionKey = rotated.SessionKey
	requestPrincipal.UsedFallback = false
	requestPrincipal.AuthHashUsedFallback = false
	requestPrincipal.values = values
	requestPrincipal.serialized = serialized
	rotated.principal = &requestPrincipal
	return rotated, nil
}

func (validator *Validator) refreshValues(
	ctx context.Context,
	sessionKey string,
	values map[string]any,
	serialized []byte,
	rotateKey bool,
	createKey bool,
	defaultAgeSeconds int64,
	now time.Time,
) (SessionRefresh, error) {
	writer, ok := validator.reader.(sessionRefreshWriter)
	if !ok {
		return SessionRefresh{}, fmt.Errorf("%w: Django session store does not support refresh", ErrStore)
	}
	maxAgeSeconds, expiresAt, browserClose, expiryErr := sessionExpiry(values, defaultAgeSeconds, now)
	if expiryErr != nil {
		return SessionRefresh{}, fmt.Errorf("%w: calculate Django session age: %w", ErrStore, expiryErr)
	}
	data, err := encodeSerialized(serialized, validator.secrets[0], now)
	if err != nil {
		return SessionRefresh{}, fmt.Errorf("%w: encode Django session: %w", ErrStore, err)
	}
	userID := djangoSessionUserIDForCharField(values["_auth_user_id"])
	var deviceInfo any
	if value, ok := djangoJSONToMap(values["device_info"]); ok {
		deviceInfo = value
	}
	refreshedKey := sessionKey
	if createKey {
		creator, ok := validator.reader.(sessionCreateWriter)
		if !ok {
			return SessionRefresh{}, fmt.Errorf("%w: Django session store does not support creation", ErrStore)
		}
		for {
			refreshedKey, err = randomSessionKey()
			if err != nil {
				return SessionRefresh{}, fmt.Errorf("%w: generate Django session key: %w", ErrStore, err)
			}
			err = creator.CreateSession(ctx, refreshedKey, data, expiresAt, userID, deviceInfo)
			if errors.Is(err, ErrKeyCollision) {
				continue
			}
			if err != nil {
				return SessionRefresh{}, fmt.Errorf("%w: %w", ErrStore, err)
			}
			break
		}
	} else if rotateKey {
		cycler, ok := validator.reader.(sessionCycleWriter)
		if !ok {
			return SessionRefresh{}, fmt.Errorf("%w: Django session store does not support key rotation", ErrStore)
		}
		for {
			refreshedKey, err = randomSessionKey()
			if err != nil {
				return SessionRefresh{}, fmt.Errorf("%w: generate Django session key: %w", ErrStore, err)
			}
			err = cycler.CycleSession(ctx, sessionKey, refreshedKey, data, expiresAt, userID, deviceInfo)
			if errors.Is(err, ErrKeyCollision) {
				continue
			}
			if err != nil {
				return SessionRefresh{}, fmt.Errorf("%w: %w", ErrStore, err)
			}
			break
		}
	} else if err := writer.UpdateSession(ctx, sessionKey, data, expiresAt, userID, deviceInfo); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return SessionRefresh{}, fmt.Errorf("%w: %w", ErrStore, err)
		}
		return SessionRefresh{}, fmt.Errorf("%w: %w: %w", ErrStore, ErrSessionUpdate, err)
	}
	preparedPrincipal := &Principal{
		SessionKey:      refreshedKey,
		values:          cloneSessionValues(values),
		serialized:      append([]byte(nil), serialized...),
		currentAuthHash: stringValueOrEmpty(values, "_auth_user_hash"),
	}
	preparedPrincipal.UserID = stringValueOrEmpty(values, "_auth_user_id")
	return SessionRefresh{
		SessionKey: refreshedKey, ExpiresAt: expiresAt, CookieExpiresAt: cookieExpiry(maxAgeSeconds, now),
		MaxAgeSeconds: maxAgeSeconds, BrowserClose: browserClose,
		principal: preparedPrincipal,
	}, nil
}

func cookieExpiry(maxAgeSeconds float64, now time.Time) time.Time {
	whole, fraction := math.Modf(maxAgeSeconds)
	return time.Unix(now.Unix()+int64(whole), int64(now.Nanosecond())+int64(fraction*float64(time.Second))).In(now.Location())
}

func addSeconds(value time.Time, seconds float64) (time.Time, error) {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds > math.MaxInt64 || seconds < math.MinInt64 {
		return time.Time{}, errors.New("Django session expiry is outside the supported time range")
	}
	whole, fraction := math.Modf(seconds)
	// datetime.timedelta rounds fractional arguments to the nearest
	// microsecond using round-half-to-even before datetime arithmetic.
	microseconds := math.RoundToEven(fraction * 1_000_000)
	if microseconds >= 1_000_000 {
		whole++
		microseconds -= 1_000_000
	} else if microseconds <= -1_000_000 {
		whole--
		microseconds += 1_000_000
	}
	unix := value.Unix()
	wholeSeconds := int64(whole)
	if (wholeSeconds > 0 && unix > math.MaxInt64-wholeSeconds) || (wholeSeconds < 0 && unix < math.MinInt64-wholeSeconds) {
		return time.Time{}, errors.New("Django session expiry is outside the supported time range")
	}
	result := time.Unix(unix+wholeSeconds, int64(value.Nanosecond())+int64(microseconds)*1_000).In(value.Location())
	if result.Year() < 1 || result.Year() > 9999 {
		return time.Time{}, errors.New("Django session expiry is outside the datetime range")
	}
	return result, nil
}

func cloneSessionValues(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func stringValueOrEmpty(values map[string]any, key string) string {
	value, _ := stringValue(values, key)
	return value
}

func randomSessionKey() (string, error) {
	const characters = "abcdefghijklmnopqrstuvwxyz0123456789"
	key := make([]byte, 128)
	for index := range key {
		choice, err := rand.Int(rand.Reader, big.NewInt(int64(len(characters))))
		if err != nil {
			return "", err
		}
		key[index] = characters[choice.Int64()]
	}
	return string(key), nil
}

func sessionExpiry(values map[string]any, defaultAgeSeconds int64, now time.Time) (float64, time.Time, bool, error) {
	expiry, found := values["_session_expiry"]
	if !found {
		expiresAt, err := addSeconds(now, float64(defaultAgeSeconds))
		return float64(defaultAgeSeconds), expiresAt, false, err
	}
	browserClose := djangoJSONNumericZero(expiry)
	if !djangoJSONTruthy(expiry) {
		expiresAt, err := addSeconds(now, float64(defaultAgeSeconds))
		return float64(defaultAgeSeconds), expiresAt, browserClose, err
	}
	var age float64
	switch value := expiry.(type) {
	case float64:
		age = value
	case json.Number:
		seconds, err := value.Float64()
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return 0, time.Time{}, browserClose, err
		}
		age = seconds
	case bool:
		if value {
			age = 1
			break
		}
		age = float64(defaultAgeSeconds)
	case string:
		parsed, err := djangoDateTimeFromISOFormat(value)
		if err != nil {
			return 0, time.Time{}, browserClose, err
		}
		seconds := parsed.Unix() - now.Unix()
		if parsed.Nanosecond() < now.Nanosecond() {
			seconds--
		}
		return float64(seconds), parsed, false, nil
	case djangoSurrogateString:
		// Python permits any single Unicode character, including a lone
		// surrogate, as datetime.fromisoformat()'s date/time separator.
		parsed, err := djangoDateTimeFromISOFormat(djangoRunesToString(value.text))
		if err != nil {
			return 0, time.Time{}, browserClose, err
		}
		seconds := parsed.Unix() - now.Unix()
		if parsed.Nanosecond() < now.Nanosecond() {
			seconds--
		}
		return float64(seconds), parsed, false, nil
	default:
		return 0, time.Time{}, browserClose, fmt.Errorf("unsupported Django session expiry type %T", expiry)
	}
	expiresAt, err := addSeconds(now, age)
	return age, expiresAt, browserClose, err
}

func djangoJSONNumericZero(value any) bool {
	switch typed := value.(type) {
	case bool:
		return !typed
	case float64:
		return typed == 0
	case json.Number:
		parsed, err := typed.Float64()
		return err == nil && parsed == 0
	default:
		return false
	}
}

func djangoJSONTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case djangoSurrogateString:
		return len(typed.text) > 0
	case float64:
		return typed != 0
	case json.Number:
		return !djangoJSONNumericZero(typed)
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	case djangoOrderedObject:
		return len(typed.entries) > 0
	default:
		return true
	}
}

func encodeSerialized(data []byte, secret string, now time.Time) (string, error) {
	if len(data) == 0 {
		return "", ErrInvalidData
	}
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(data); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	useCompressed := compressed.Len() < len(data)-1
	if useCompressed {
		data = compressed.Bytes()
	}
	payload := base64.RawURLEncoding.EncodeToString(data)
	if useCompressed {
		payload = "." + payload
	}
	value := payload + ":" + base62(now.Unix())
	signature := base64.RawURLEncoding.EncodeToString(djangoHMAC(SessionSigningSalt+"signer", value, secret))
	return value + ":" + signature, nil
}

func djangoASCIIJSON(data []byte) []byte {
	output := make([]byte, 0, len(data))
	for len(data) > 0 {
		character, size := utf8.DecodeRune(data)
		if character < utf8.RuneSelf {
			output = append(output, byte(character))
		} else if character <= 0xffff {
			output = fmt.Appendf(output, `\u%04x`, character)
		} else {
			character -= 0x10000
			output = fmt.Appendf(output, `\u%04x\u%04x`, 0xd800+(character>>10), 0xdc00+(character&0x3ff))
		}
		data = data[size:]
	}
	return output
}

func base62(value int64) string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	if value == 0 {
		return "0"
	}
	var encoded [11]byte
	index := len(encoded)
	for value > 0 {
		index--
		encoded[index] = alphabet[value%62]
		value /= 62
	}
	return string(encoded[index:])
}

func decode(sessionData string, secrets []string) (map[string]any, int, error) {
	values, _, secret, err := decodeSessionData(sessionData, secrets)
	return values, secret, err
}

func logSuspiciousSession(ctx context.Context, err error) {
	if errors.Is(err, ErrBadSignature) {
		log.Warn(
			ctx,
			log.KV{K: "msg", V: "Session data corrupted"},
			log.KV{K: "logger", V: "django.security.SuspiciousSession"},
		)
	}
}

func decodeSessionData(sessionData string, secrets []string) (map[string]any, []byte, int, error) {
	// The payload can contain a leading dot but no colon. Cut the final fields
	// explicitly so malformed TimestampSigner values fail closed.
	last := strings.LastIndex(sessionData, ":")
	if last <= 0 {
		return nil, nil, -1, fmt.Errorf("%w: %w", ErrInvalidData, ErrBadSignature)
	}
	signedValue, signature := sessionData[:last], sessionData[last+1:]
	secretIndex := matchingSignature(signedValue, signature, secrets)
	if secretIndex < 0 {
		return nil, nil, -1, fmt.Errorf("%w: %w", ErrInvalidData, ErrBadSignature)
	}
	payload, timestamp, ok := strings.Cut(signedValue, ":")
	if !ok || strings.Contains(timestamp, ":") || !validBase62(timestamp) {
		return nil, nil, -1, ErrInvalidData
	}
	compressed := strings.HasPrefix(payload, ".")
	if compressed {
		payload = payload[1:]
	}
	encoded, err := djangoBase64Decode(payload)
	if err != nil {
		return nil, nil, -1, ErrInvalidData
	}
	if compressed {
		reader, newReaderErr := zlib.NewReader(bytes.NewReader(encoded))
		if newReaderErr != nil {
			return nil, nil, -1, ErrInvalidData
		}
		decoded, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			return nil, nil, -1, ErrInvalidData
		}
		encoded = decoded
	}
	values, serialized, err := decodeDjangoJSON(encoded)
	if err != nil {
		if errors.Is(err, ErrInvalidSessionShape) {
			return nil, nil, -1, err
		}
		return nil, nil, -1, ErrInvalidData
	}
	return values, serialized, secretIndex, nil
}

func djangoBase64Decode(value string) ([]byte, error) {
	padded := value + strings.Repeat("=", (4-len(value)%4)%4)
	alphabet := make([]byte, 0, len(padded))
	pendingSecondPadding := false
	terminated := false
	for index := 0; index < len(padded); index++ {
		octet := padded[index]
		if (octet >= 'a' && octet <= 'z') || (octet >= 'A' && octet <= 'Z') ||
			(octet >= '0' && octet <= '9') || octet == '-' || octet == '_' || octet == '+' || octet == '/' {
			switch octet {
			case '+':
				octet = '-'
			case '/':
				octet = '_'
			}
			alphabet = append(alphabet, octet)
			pendingSecondPadding = false
			continue
		}
		if octet != '=' {
			continue
		}
		switch len(alphabet) % 4 {
		case 3:
			terminated = true
		case 2:
			if pendingSecondPadding {
				terminated = true
			} else {
				pendingSecondPadding = true
			}
		default:
			pendingSecondPadding = false
		}
		if terminated {
			break
		}
	}
	if !terminated && len(alphabet)%4 != 0 {
		return nil, ErrInvalidData
	}
	return base64.RawURLEncoding.DecodeString(string(alphabet))
}

// ValidateUserTimezone preserves TimezoneMixin's distinction between invalid
// relative keys (ValueError) and normalized keys that do not exist (KeyError).
func ValidateUserTimezone(value string) error {
	if value == "" || strings.HasPrefix(value, "/") || value == "." || value == ".." || strings.HasPrefix(value, "../") || path.Clean(value) != value || strings.ContainsRune(value, '\x00') {
		return ErrInvalidTimezone
	}
	if value == "Local" {
		return ErrTimezoneNotFound
	}
	if _, err := time.LoadLocation(value); err != nil {
		return ErrTimezoneNotFound
	}
	return nil
}

func replaceTopLevelJSONString(serialized []byte, key string, replacement string) ([]byte, error) {
	depth := 0
	for index := 0; index < len(serialized); {
		switch serialized[index] {
		case '{', '[':
			depth++
			index++
		case '}', ']':
			depth--
			index++
		case '"':
			end, ok := jsonStringEnd(serialized, index)
			if !ok {
				return nil, ErrInvalidData
			}
			if depth != 1 {
				index = end
				continue
			}
			colon := end
			for colon < len(serialized) && (serialized[colon] == ' ' || serialized[colon] == '\t' || serialized[colon] == '\r' || serialized[colon] == '\n') {
				colon++
			}
			if colon >= len(serialized) || serialized[colon] != ':' {
				index = end
				continue
			}
			var decodedKey string
			if err := json.Unmarshal(serialized[index:end], &decodedKey); err != nil || decodedKey != key {
				index = end
				continue
			}
			valueStart := colon + 1
			for valueStart < len(serialized) && (serialized[valueStart] == ' ' || serialized[valueStart] == '\t' || serialized[valueStart] == '\r' || serialized[valueStart] == '\n') {
				valueStart++
			}
			if valueStart >= len(serialized) || serialized[valueStart] != '"' {
				return nil, ErrInvalidData
			}
			valueEnd, ok := jsonStringEnd(serialized, valueStart)
			if !ok {
				return nil, ErrInvalidData
			}
			encodedReplacement, _ := json.Marshal(replacement)
			result := make([]byte, 0, len(serialized)-valueEnd+valueStart+len(encodedReplacement))
			result = append(result, serialized[:valueStart]...)
			result = append(result, encodedReplacement...)
			result = append(result, serialized[valueEnd:]...)
			return result, nil
		default:
			index++
		}
	}
	return nil, ErrInvalidData
}

func jsonStringEnd(value []byte, start int) (int, bool) {
	escaped := false
	for index := start + 1; index < len(value); index++ {
		if escaped {
			escaped = false
			continue
		}
		if value[index] == '\\' {
			escaped = true
			continue
		}
		if value[index] == '"' {
			return index + 1, true
		}
	}
	return 0, false
}

func matchingSignature(value string, signature string, secrets []string) int {
	match := -1
	for index, secret := range secrets {
		expected := djangoHMAC(SessionSigningSalt+"signer", value, secret)
		if constantTimeEqual(signature, base64.RawURLEncoding.EncodeToString(expected)) && match < 0 {
			match = index
		}
	}
	return match
}

func matchingAuthSecret(sessionHash string, password string, secrets []string) int {
	match := -1
	for index, secret := range secrets {
		expected := hex.EncodeToString(djangoHMAC(AuthHashSalt, password, secret))
		if constantTimeEqual(sessionHash, expected) && match < 0 {
			match = index
		}
	}
	return match
}

func djangoHMAC(salt string, value string, secret string) []byte {
	derived := sha256.Sum256([]byte(salt + secret))
	digest := hmac.New(sha256.New, derived[:])
	_, _ = digest.Write([]byte(value))
	return digest.Sum(nil)
}

func constantTimeEqual(left string, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func stringValue(values map[string]any, key string) (string, bool) {
	value, ok := values[key].(string)
	return value, ok
}

func djangoUUIDValue(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		parsed, err := uuid.Parse(typed)
		if err != nil {
			return "", err
		}
		return parsed.String(), nil
	case bool:
		if typed {
			return "00000000-0000-0000-0000-000000000001", nil
		}
		return "00000000-0000-0000-0000-000000000000", nil
	case json.Number:
		integer, ok := new(big.Int).SetString(string(typed), 10)
		if !ok || integer.Sign() < 0 || integer.BitLen() > 128 {
			return "", ErrInvalidUserID
		}
		bytes := make([]byte, 16)
		integer.FillBytes(bytes)
		parsed, err := uuid.FromBytes(bytes)
		if err != nil {
			return "", err
		}
		return parsed.String(), nil
	default:
		return "", ErrInvalidUserID
	}
}

func djangoSessionUserIDForCharField(value any) any {
	if value == nil {
		return nil
	}
	if _, invalid := value.(djangoSurrogateString); invalid {
		// psycopg rejects Python strings containing lone UTF-16 surrogates when
		// it encodes the varchar parameter as UTF-8. Use an unsupported pgx
		// value so PostgreSQL persistence fails at the same point.
		return func() {}
	}
	if text, ok := value.(string); ok {
		return text
	}
	return djangoPythonRepr(value)
}

func djangoJSONToMap(value any) (map[string]any, bool) {
	object, ok := value.(djangoOrderedObject)
	if !ok {
		plain, plainOK := value.(map[string]any)
		return plain, plainOK
	}
	result := make(map[string]any, len(object.entries))
	for _, entry := range object.entries {
		if entry.keyHasSurrogate {
			result[entry.key] = func() {}
			continue
		}
		if _, invalid := entry.value.(djangoSurrogateString); invalid {
			// PostgreSQL JSONB rejects lone UTF-16 surrogates. An unsupported
			// value makes pgx's JSON encoder fail instead of silently replacing
			// the code unit with U+FFFD.
			result[entry.key] = func() {}
			continue
		}
		if nested, nestedOK := djangoJSONToMap(entry.value); nestedOK {
			result[entry.key] = nested
			continue
		}
		if array, arrayOK := entry.value.([]any); arrayOK {
			result[entry.key] = djangoJSONPlainArray(array)
			continue
		}
		result[entry.key] = entry.value
	}
	return result, true
}

func djangoJSONPlainArray(value []any) []any {
	result := make([]any, 0, len(value))
	for _, item := range value {
		if object, ok := djangoJSONToMap(item); ok {
			result = append(result, object)
		} else if array, ok := item.([]any); ok {
			result = append(result, djangoJSONPlainArray(array))
		} else if _, invalid := item.(djangoSurrogateString); invalid {
			result = append(result, func() {})
		} else {
			result = append(result, item)
		}
	}
	return result
}

func djangoPythonRepr(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case string:
		return djangoPythonStringRepr(typed)
	case djangoSurrogateString:
		return djangoPythonRunesRepr(typed.text)
	case json.Number:
		return typed.String()
	case float64:
		switch {
		case math.IsNaN(typed):
			return "nan"
		case math.IsInf(typed, 1):
			return "inf"
		case math.IsInf(typed, -1):
			return "-inf"
		default:
			return pythonFloatString(typed)
		}
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, djangoPythonRepr(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case djangoOrderedObject:
		parts := make([]string, 0, len(typed.entries))
		for _, entry := range typed.entries {
			parts = append(parts, djangoPythonStringRepr(entry.key)+": "+djangoPythonRepr(entry.value))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return fmt.Sprint(value)
	}
}

func djangoPythonStringRepr(value string) string {
	return djangoPythonRunesRepr([]rune(value))
}

func djangoPythonRunesRepr(value []rune) string {
	quote := rune('\'')
	containsSingle := false
	containsDouble := false
	for _, character := range value {
		containsSingle = containsSingle || character == '\''
		containsDouble = containsDouble || character == '"'
	}
	if containsSingle && !containsDouble {
		quote = '"'
	}
	var result strings.Builder
	result.WriteRune(quote)
	for _, character := range value {
		switch character {
		case '\\':
			result.WriteString(`\\`)
		case '\t':
			result.WriteString(`\t`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		default:
			switch {
			case character == quote:
				result.WriteRune('\\')
				result.WriteRune(character)
			case unicode.IsPrint(character):
				result.WriteRune(character)
			case character <= 0xff:
				_, _ = fmt.Fprintf(&result, `\x%02x`, character)
			case character <= 0xffff:
				_, _ = fmt.Fprintf(&result, `\u%04x`, character)
			default:
				_, _ = fmt.Fprintf(&result, `\U%08x`, character)
			}
		}
	}
	result.WriteRune(quote)
	return result.String()
}

func validBase62(value string) bool {
	if value == "" {
		return false
	}
	if value[0] == '-' {
		value = value[1:]
		if value == "" {
			return false
		}
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') {
			return false
		}
	}
	return true
}
