package djangosession

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// AuthenticatedSession contains the Django authentication fields needed when
// login() writes the session. User timestamps and request device metadata are
// owned by the caller; DeviceInfo is copied into the Django session row.
type AuthenticatedSession struct {
	UserID       string
	PasswordHash string
	Backend      string
	DeviceInfo   any
}

// Get reads a value from the request-local Django session snapshot.
func (refresh SessionRefresh) Get(key string) (any, bool) {
	if refresh.principal == nil || refresh.principal.values == nil {
		return nil, false
	}
	value, found := refresh.principal.values[key]
	return value, found
}

// Set changes the request-local snapshot. Persist the change with
// Validator.SaveValues before expecting it to survive the response.
func (refresh *SessionRefresh) Set(key string, value any) {
	if refresh == nil || refresh.principal == nil {
		return
	}
	if refresh.principal.values == nil {
		refresh.principal.values = make(map[string]any)
	}
	refresh.principal.values[key] = value
	refresh.principal.serialized = nil
}

// Delete removes a value from the request-local snapshot. Persist the change
// with Validator.SaveValues before expecting it to survive the response.
func (refresh *SessionRefresh) Delete(key string) {
	if refresh == nil || refresh.principal == nil {
		return
	}
	delete(refresh.principal.values, key)
	refresh.principal.serialized = nil
}

// SaveValues merges values into the current request snapshot and persists the
// result using Django's database session format. With no current session it
// creates a new anonymous session row.
func (validator *Validator) SaveValues(
	ctx context.Context,
	current *SessionRefresh,
	values map[string]any,
	defaultAgeSeconds int64,
	now time.Time,
) (SessionRefresh, error) {
	base := requestSessionSnapshot(current)
	for key, value := range values {
		base[key] = value
	}
	return validator.saveValues(ctx, current, base, false, defaultAgeSeconds, now)
}

// Login writes Django's authenticated-session fields, preserves unrelated
// request session values, cycles an existing session key, and returns the
// cookie attributes required by the HTTP response.
func (validator *Validator) Login(
	ctx context.Context,
	current *SessionRefresh,
	identity AuthenticatedSession,
	defaultAgeSeconds int64,
	now time.Time,
) (SessionRefresh, error) {
	if identity.UserID == "" || identity.PasswordHash == "" || identity.Backend == "" {
		return SessionRefresh{}, errors.New("Django login requires user id, password hash, and backend")
	}
	if _, ok := validator.backends[identity.Backend]; !ok {
		return SessionRefresh{}, ErrBackend
	}
	values := requestSessionSnapshot(current)
	userHash := fmt.Sprintf("%x", djangoHMAC(AuthHashSalt, identity.PasswordHash, validator.secrets[0]))
	flush := false
	flushCurrent := current
	_, hasAuthenticatedUser := values["_auth_user_id"]
	if hasAuthenticatedUser {
		sessionUserID, _ := stringValue(values, "_auth_user_id")
		existingHash, _ := stringValue(values, "_auth_user_hash")
		flush = sessionUserID != identity.UserID || subtle.ConstantTimeCompare([]byte(existingHash), []byte(userHash)) != 1
	}
	if flush {
		if current != nil && current.SessionKey != "" {
			if err := validator.deleteSession(ctx, current.SessionKey); err != nil {
				return SessionRefresh{}, err
			}
		}
		values = map[string]any{}
		flushCurrent = nil
	}
	values["_auth_user_id"] = identity.UserID
	values["_auth_user_backend"] = identity.Backend
	values["_auth_user_hash"] = userHash
	if identity.DeviceInfo != nil {
		values["device_info"] = identity.DeviceInfo
	}
	rotateKey := !hasAuthenticatedUser && current != nil && current.SessionKey != ""
	return validator.saveValues(ctx, flushCurrent, values, rotateKey, defaultAgeSeconds, now)
}

// Flush removes the current database row. Django's SessionMiddleware expires
// the cookie for the now-empty session instead of creating an empty row.
func (validator *Validator) Flush(
	ctx context.Context,
	current *SessionRefresh,
) error {
	if current != nil && current.SessionKey != "" {
		if err := validator.deleteSession(ctx, current.SessionKey); err != nil {
			return err
		}
	}
	return nil
}

func (validator *Validator) deleteSession(ctx context.Context, sessionKey string) error {
	deleter, ok := validator.reader.(sessionDeleteWriter)
	if !ok {
		return fmt.Errorf("%w: Django session store does not support deletion", ErrStore)
	}
	if err := deleter.DeleteSession(ctx, sessionKey); err != nil {
		return fmt.Errorf("%w: delete Django session: %w", ErrStore, err)
	}
	return nil
}

func (validator *Validator) saveValues(
	ctx context.Context,
	current *SessionRefresh,
	values map[string]any,
	rotateKey bool,
	defaultAgeSeconds int64,
	now time.Time,
) (SessionRefresh, error) {
	serialized, err := json.Marshal(values)
	if err != nil {
		return SessionRefresh{}, fmt.Errorf("%w: serialize Django session: %w", ErrStore, err)
	}
	serialized = djangoASCIIJSON(serialized)
	if current == nil || current.SessionKey == "" {
		return validator.refreshValues(ctx, "", values, serialized, false, true, defaultAgeSeconds, now)
	}
	if rotateKey {
		return validator.refreshValues(ctx, current.SessionKey, values, serialized, true, false, defaultAgeSeconds, now)
	}
	return validator.refreshValues(ctx, current.SessionKey, values, serialized, false, false, defaultAgeSeconds, now)
}

func requestSessionSnapshot(current *SessionRefresh) map[string]any {
	if current == nil || current.principal == nil || current.principal.values == nil {
		return map[string]any{}
	}
	return cloneSessionValues(current.principal.values)
}

type responseActionKey struct{}
type currentSessionKey struct{}

// ResponseSessionAction is request-scoped output from an auth/session handler.
// Middleware uses it to replace the automatic refresh or expire the cookie.
type ResponseSessionAction struct {
	mu      sync.Mutex
	refresh *SessionRefresh
	expire  bool
}

// WithResponseSessionAction installs the mutable action slot consumed by the
// system response middleware.
func WithResponseSessionAction(ctx context.Context) (context.Context, *ResponseSessionAction) {
	action := &ResponseSessionAction{}
	return context.WithValue(ctx, responseActionKey{}, action), action
}

// WithCurrentSession makes the middleware's prepared request snapshot
// available to auth handlers that need to preserve arbitrary Django session
// values across login or OAuth state changes.
func WithCurrentSession(ctx context.Context, refresh SessionRefresh) context.Context {
	copy := cloneSessionRefresh(refresh)
	return context.WithValue(ctx, currentSessionKey{}, &copy)
}

// CurrentSessionFromContext returns the middleware-prepared session snapshot.
func CurrentSessionFromContext(ctx context.Context) (*SessionRefresh, bool) {
	refresh, ok := ctx.Value(currentSessionKey{}).(*SessionRefresh)
	if !ok || refresh == nil {
		return nil, false
	}
	copy := cloneSessionRefresh(*refresh)
	return &copy, true
}

// SetResponseSession asks the system middleware to emit the returned Django
// session cookie instead of refreshing the original request snapshot.
func SetResponseSession(ctx context.Context, refresh SessionRefresh) bool {
	action, ok := ctx.Value(responseActionKey{}).(*ResponseSessionAction)
	if !ok {
		return false
	}
	action.mu.Lock()
	defer action.mu.Unlock()
	copy := cloneSessionRefresh(refresh)
	action.refresh = &copy
	action.expire = false
	return true
}

// ExpireResponseSession asks the system middleware to expire the Django
// session cookie and skip the automatic refresh.
func ExpireResponseSession(ctx context.Context) bool {
	action, ok := ctx.Value(responseActionKey{}).(*ResponseSessionAction)
	if !ok {
		return false
	}
	action.mu.Lock()
	defer action.mu.Unlock()
	action.refresh = nil
	action.expire = true
	return true
}

// ResponseSessionActionFromContext returns the response override, if one was
// installed by a handler.
func ResponseSessionActionFromContext(ctx context.Context) (refresh *SessionRefresh, expire bool, found bool) {
	action, ok := ctx.Value(responseActionKey{}).(*ResponseSessionAction)
	if !ok {
		return nil, false, false
	}
	action.mu.Lock()
	defer action.mu.Unlock()
	if action.refresh != nil {
		copy := cloneSessionRefresh(*action.refresh)
		return &copy, false, true
	}
	if action.expire {
		return nil, true, true
	}
	return nil, false, false
}

func cloneSessionRefresh(refresh SessionRefresh) SessionRefresh {
	if refresh.principal == nil {
		return refresh
	}
	principal := *refresh.principal
	principal.values = cloneSessionValues(refresh.principal.values)
	principal.serialized = append([]byte(nil), refresh.principal.serialized...)
	refresh.principal = &principal
	return refresh
}
