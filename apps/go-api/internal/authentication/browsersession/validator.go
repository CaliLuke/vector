package browsersession

import (
	"bytes"
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
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/google/uuid"
)

const (
	DefaultCookieName  = "session-id"
	SessionSigningSalt = "vector.session.v1"
	AuthHashSalt       = "vector.auth.password.v1"
)

var (
	ErrNoCookie            = errors.New("session cookie is missing")
	ErrInvalidKey          = errors.New("session key is invalid")
	ErrSessionNotFound     = errors.New("session is missing or expired")
	ErrInvalidData         = errors.New("session data is invalid")
	ErrInvalidSessionShape = errors.New("session data is not an object")
	ErrBadSignature        = errors.New("session signature does not match")
	ErrUserMismatch        = errors.New("session user does not match its row")
	ErrInvalidUserID       = errors.New("session user identifier is invalid")
	ErrUserNotFound        = errors.New("session user does not exist")
	ErrInactiveUser        = errors.New("session user is inactive")
	ErrInvalidAuthHash     = errors.New("session authentication hash is invalid")
	ErrInvalidTimezone     = errors.New("session user timezone key is invalid")
	ErrTimezoneNotFound    = errors.New("session user timezone does not exist")
	ErrStore               = errors.New("session store read failed")
	ErrSessionStore        = errors.New("session table access failed")
	ErrUserStore           = errors.New("session user table access failed")
	ErrKeyCollision        = errors.New("session key already exists")
	ErrSessionUpdate       = errors.New("session update failed")
)

type Session struct {
	Data string
}

type User struct {
	Password string
	Active   bool
	Timezone string
}

// Reader loads unexpired sessions and current account credentials.
type Reader interface {
	ReadSession(context.Context, string) (Session, error)
	ReadUser(context.Context, string) (User, error)
}

type Config struct {
	CookieName           string
	CookieNameConfigured bool
	SecretKey            string
	FallbackKeys         []string
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
	reader  Reader
	cookie  string
	secrets []string
}

func New(reader Reader, config Config) (*Validator, error) {
	if reader == nil {
		return nil, errors.New("session reader is required")
	}
	if config.SecretKey == "" {
		return nil, errors.New("session SECRET_KEY is required")
	}
	cookie := config.CookieName
	if !config.CookieNameConfigured && cookie == "" {
		cookie = DefaultCookieName
	}
	if err := (&http.Cookie{Name: cookie, Value: "valid"}).Valid(); err != nil {
		return nil, errors.New("session cookie name is invalid")
	}
	secrets := make([]string, 0, 1+len(config.FallbackKeys))
	secrets = append(secrets, config.SecretKey)
	for _, key := range config.FallbackKeys {
		if key == "" {
			return nil, errors.New("session fallback SECRET_KEY must not be empty")
		}
		secrets = append(secrets, key)
	}
	return &Validator{reader: reader, cookie: cookie, secrets: secrets}, nil
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
	// Reject implausible keys before querying the session store.
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
		logSuspiciousSession(ctx, err)
		return Principal{}, nil, nil, err
	}
	principal, _, err := validator.validateValues(ctx, sessionKey, values, sessionSecret)
	principal.serialized = append([]byte(nil), serialized...)
	return principal, values, serialized, err
}

func (validator *Validator) validateValues(ctx context.Context, sessionKey string, values map[string]any, sessionSecret int) (Principal, string, error) {
	rawUserID, found := values["user_id"]
	if !found {
		return Principal{}, "", ErrUserMismatch
	}
	userID, ok := rawUserID.(string)
	if !ok {
		return Principal{}, "", ErrInvalidUserID
	}
	identifier, err := uuid.Parse(userID)
	if err != nil {
		return Principal{}, "", ErrInvalidUserID
	}
	userID = identifier.String()
	user, err := validator.reader.ReadUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return Principal{}, "", ErrUserNotFound
		}
		return Principal{}, "", fmt.Errorf("%w: %w: %w", ErrStore, ErrUserStore, err)
	}
	if !user.Active {
		return Principal{}, "", ErrInactiveUser
	}
	authHash, ok := stringValue(values, "auth_hash")
	if !ok || authHash == "" {
		return Principal{}, "", ErrInvalidAuthHash
	}
	authSecret := matchingAuthSecret(authHash, user.Password, validator.secrets)
	if authSecret < 0 {
		return Principal{}, "", ErrInvalidAuthHash
	}
	currentAuthHash := hex.EncodeToString(sessionHMAC(AuthHashSalt, user.Password, validator.secrets[0]))
	return Principal{UserID: userID, SessionKey: sessionKey, UsedFallback: sessionSecret > 0 || authSecret > 0,
		AuthHashUsedFallback: authSecret > 0, UserTimezone: user.Timezone,
		values: cloneSessionValues(values), currentAuthHash: currentAuthHash}, currentAuthHash, nil
}

func (validator *Validator) Refresh(ctx context.Context, sessionKey string, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if utf8.RuneCountInString(sessionKey) < 8 {
		return SessionRefresh{}, ErrInvalidKey
	}
	session, err := validator.reader.ReadSession(ctx, sessionKey)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return SessionRefresh{}, ErrSessionNotFound
		}
		return SessionRefresh{}, fmt.Errorf("%w: %w: %w", ErrStore, ErrSessionStore, err)
	}
	values, serialized, sessionSecret, decodeErr := decodeSessionData(session.Data, validator.secrets)
	if decodeErr != nil {
		return SessionRefresh{}, decodeErr
	}

	rotateKey := false
	principal, currentAuthHash, validationErr := validator.validateValues(ctx, sessionKey, values, sessionSecret)
	if validationErr == nil && principal.AuthHashUsedFallback {
		values["auth_hash"] = currentAuthHash
		serialized, decodeErr = replaceTopLevelJSONString(serialized, "auth_hash", currentAuthHash)
		if decodeErr != nil {
			return SessionRefresh{}, fmt.Errorf("%w: update session authentication hash: %w", ErrStore, decodeErr)
		}
		rotateKey = true
	}
	return validator.refreshValues(ctx, sessionKey, values, serialized, rotateKey, false, defaultAgeSeconds, now)
}

// RefreshPrincipal persists the request-local snapshot loaded by Validate.
func (validator *Validator) RefreshPrincipal(ctx context.Context, principal Principal, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if utf8.RuneCountInString(principal.SessionKey) < 8 || principal.values == nil {
		return SessionRefresh{}, ErrInvalidKey
	}
	values := cloneSessionValues(principal.values)
	serialized := append([]byte(nil), principal.serialized...)
	if principal.AuthHashUsedFallback {
		values["auth_hash"] = principal.currentAuthHash
		var err error
		serialized, err = replaceTopLevelJSONString(serialized, "auth_hash", principal.currentAuthHash)
		if err != nil {
			return SessionRefresh{}, fmt.Errorf("%w: update session authentication hash: %w", ErrStore, err)
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
		currentAuthHash: stringValueOrEmpty(values, "auth_hash"),
	}
	principal.UserID = stringValueOrEmpty(values, "user_id")
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

// RefreshUnaccessed renews a valid stored session without authenticating a user.
// Missing or invalid sessions remain rejected.
func (validator *Validator) RefreshUnaccessed(ctx context.Context, sessionKey string, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if utf8.RuneCountInString(sessionKey) < 8 {
		return SessionRefresh{}, ErrInvalidKey
	}
	session, err := validator.reader.ReadSession(ctx, sessionKey)
	if errors.Is(err, ErrSessionNotFound) {
		return SessionRefresh{}, ErrSessionNotFound
	}
	if err != nil {
		return SessionRefresh{}, fmt.Errorf("%w: %w: %w", ErrStore, ErrSessionStore, err)
	}
	values, serialized, _, decodeErr := decodeSessionData(session.Data, validator.secrets)
	if decodeErr != nil {
		return SessionRefresh{}, decodeErr
	}
	return validator.refreshValues(ctx, sessionKey, values, serialized, false, false, defaultAgeSeconds, now)
}

// RotateFallbackPrepared upgrades signing and password-version digests together.
func (validator *Validator) RotateFallbackPrepared(ctx context.Context, prepared SessionRefresh, defaultAgeSeconds int64, now time.Time) (SessionRefresh, error) {
	if prepared.principal == nil || !prepared.principal.AuthHashUsedFallback {
		return SessionRefresh{}, ErrInvalidKey
	}
	return validator.RefreshPrincipal(ctx, *prepared.principal, defaultAgeSeconds, now)
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
		return SessionRefresh{}, fmt.Errorf("%w: session store does not support refresh", ErrStore)
	}
	maxAgeSeconds, expiresAt, expiryErr := sessionExpiry(defaultAgeSeconds, now)
	if expiryErr != nil {
		return SessionRefresh{}, fmt.Errorf("%w: calculate session age: %w", ErrStore, expiryErr)
	}
	data, err := encodeSerialized(serialized, validator.secrets[0])
	if err != nil {
		return SessionRefresh{}, fmt.Errorf("%w: encode session: %w", ErrStore, err)
	}
	var userID any
	if id, ok := stringValue(values, "user_id"); ok {
		userID = id
	}
	var deviceInfo any
	if value, ok := values["device_info"].(map[string]any); ok {
		deviceInfo = value
	}
	refreshedKey := sessionKey
	if createKey {
		creator, ok := validator.reader.(sessionCreateWriter)
		if !ok {
			return SessionRefresh{}, fmt.Errorf("%w: session store does not support creation", ErrStore)
		}
		for {
			refreshedKey, err = randomSessionKey()
			if err != nil {
				return SessionRefresh{}, fmt.Errorf("%w: generate session key: %w", ErrStore, err)
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
			return SessionRefresh{}, fmt.Errorf("%w: session store does not support key rotation", ErrStore)
		}
		for {
			refreshedKey, err = randomSessionKey()
			if err != nil {
				return SessionRefresh{}, fmt.Errorf("%w: generate session key: %w", ErrStore, err)
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
		currentAuthHash: stringValueOrEmpty(values, "auth_hash"),
	}
	preparedPrincipal.UserID = stringValueOrEmpty(values, "user_id")
	return SessionRefresh{
		SessionKey: refreshedKey, ExpiresAt: expiresAt, CookieExpiresAt: cookieExpiry(maxAgeSeconds, now),
		MaxAgeSeconds: maxAgeSeconds,
		principal:     preparedPrincipal,
	}, nil
}

func cookieExpiry(maxAgeSeconds float64, now time.Time) time.Time {
	whole, fraction := math.Modf(maxAgeSeconds)
	return time.Unix(now.Unix()+int64(whole), int64(now.Nanosecond())+int64(fraction*float64(time.Second))).In(now.Location())
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

func sessionExpiry(age int64, now time.Time) (float64, time.Time, error) {
	if age <= 0 {
		return 0, time.Time{}, errors.New("session age must be positive")
	}
	if now.Unix() > math.MaxInt64-age {
		return 0, time.Time{}, errors.New("session expiry is outside the supported time range")
	}
	expiresAt := time.Unix(now.Unix()+age, int64(now.Nanosecond())).In(now.Location())
	if expiresAt.Year() < 1 || expiresAt.Year() > 9999 {
		return 0, time.Time{}, errors.New("session expiry is outside the calendar range")
	}
	return float64(age), expiresAt, nil
}

func encodeSerialized(data []byte, secret string) (string, error) {
	_, canonical, err := decodeNativeJSON(data)
	if err != nil {
		return "", err
	}
	value := "v1." + base64.RawURLEncoding.EncodeToString(canonical)
	signature := base64.RawURLEncoding.EncodeToString(sessionHMAC(SessionSigningSalt+"signer", value, secret))
	return value + "." + signature, nil
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
			log.KV{K: "logger", V: "vector.security.session"},
		)
	}
}

func decodeSessionData(sessionData string, secrets []string) (map[string]any, []byte, int, error) {
	parts := strings.Split(sessionData, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return nil, nil, -1, ErrInvalidData
	}
	secretIndex := matchingSignature(parts[0]+"."+parts[1], parts[2], secrets)
	if secretIndex < 0 {
		return nil, nil, -1, fmt.Errorf("%w: %w", ErrInvalidData, ErrBadSignature)
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil {
		return nil, nil, -1, ErrInvalidData
	}
	values, canonical, err := decodeNativeJSON(data)
	if err != nil {
		return nil, nil, -1, err
	}
	return values, canonical, secretIndex, nil
}

func decodeNativeJSON(data []byte) (map[string]any, []byte, error) {
	if !utf8.Valid(data) {
		return nil, nil, ErrInvalidData
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, nil, ErrInvalidData
	}
	values, ok := decoded.(map[string]any)
	if !ok {
		return nil, nil, ErrInvalidSessionShape
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, nil, ErrInvalidData
	}
	canonical, err := json.Marshal(values)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrInvalidData, err)
	}
	return values, canonical, nil
}

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
	values, _, err := decodeNativeJSON(serialized)
	if err != nil {
		return nil, err
	}
	values[key] = replacement
	return json.Marshal(values)
}

func matchingSignature(value string, signature string, secrets []string) int {
	match := -1
	for index, secret := range secrets {
		expected := sessionHMAC(SessionSigningSalt+"signer", value, secret)
		if constantTimeEqual(signature, base64.RawURLEncoding.EncodeToString(expected)) && match < 0 {
			match = index
		}
	}
	return match
}

func matchingAuthSecret(sessionHash string, password string, secrets []string) int {
	match := -1
	for index, secret := range secrets {
		expected := hex.EncodeToString(sessionHMAC(AuthHashSalt, password, secret))
		if constantTimeEqual(sessionHash, expected) && match < 0 {
			match = index
		}
	}
	return match
}

func sessionHMAC(salt string, value string, secret string) []byte {
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
