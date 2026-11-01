package browsersession

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNativeSessionLoginDoesNotRequireDjangoBackend(t *testing.T) {
	reader := &memoryReader{
		sessions: map[string]Session{},
		users:    map[string]User{fixtureUserID: {Password: fixturePassword, Active: true, Timezone: "UTC"}},
	}
	validator, err := New(reader, Config{SecretKey: fixtureCurrentSecret})
	if err != nil {
		t.Fatalf("configure native sessions without a Django backend: %v", err)
	}
	loggedIn, err := validator.Login(t.Context(), nil, AuthenticatedSession{
		UserID: fixtureUserID, PasswordHash: fixturePassword,
	}, 3600, time.Now())
	if err != nil {
		t.Fatalf("native login: %v", err)
	}
	if !strings.HasPrefix(reader.sessions[loggedIn.SessionKey].Data, "v1.") {
		t.Fatal("login did not persist the versioned native session format")
	}
	principal, err := validator.Validate(t.Context(), loggedIn.SessionKey)
	if err != nil || principal.UserID != fixtureUserID {
		t.Fatalf("validate newly issued native session: principal=%+v error=%v", principal, err)
	}

}

func TestNativeSessionRejectsDjangoPayload(t *testing.T) {
	reader := &memoryReader{
		sessions: map[string]Session{"native-rejection-session": {Data: ".eJxtjMsOwiAURP-FtZLyKheX7v0GwuMiVQNJaVfGf7dNMNHEWcxmzpknsW5dsl0bznaK5ERYz_FPfUIO35p34Y5ld-PNlWuloZZlnjzdEdrXRi814uPc2Z-D7FrebJOGAb1CxjgfUfE0MilBaKmi8FsjOAEJMLkQwUttOEpkEZgOyUAQ5PUGIa0-Xg:1AbC:xXWGwnCfPtoweEMR_QW1DR1egdVd0BEMax3RSgk9N8k"}},
		users:    map[string]User{fixtureUserID: {Password: fixturePassword, Active: true, Timezone: "UTC"}},
	}
	validator, err := New(reader, Config{SecretKey: fixtureCurrentSecret})
	if err != nil {
		t.Fatalf("configure native sessions: %v", err)
	}
	if _, err := validator.Validate(t.Context(), "native-rejection-session"); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("legacy session format must be rejected: %v", err)
	}
}

func nativeSessionFixture(t *testing.T) (*Validator, *memoryReader, SessionRefresh) {
	t.Helper()
	reader := &memoryReader{sessions: map[string]Session{}, users: map[string]User{
		fixtureUserID: {Password: fixturePassword, Active: true, Timezone: "UTC"},
	}}
	validator, err := New(reader, Config{SecretKey: fixtureCurrentSecret})
	if err != nil {
		t.Fatal(err)
	}
	loggedIn, err := validator.Login(t.Context(), nil, AuthenticatedSession{
		UserID: fixtureUserID, PasswordHash: fixturePassword,
	}, 3600, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return validator, reader, loggedIn
}

func TestNativeSessionPasswordChangeInvalidatesAuthentication(t *testing.T) {
	validator, reader, loggedIn := nativeSessionFixture(t)
	reader.users[fixtureUserID] = User{Password: "changed-password-hash", Active: true, Timezone: "UTC"}
	if _, err := validator.Validate(t.Context(), loggedIn.SessionKey); !errors.Is(err, ErrInvalidAuthHash) {
		t.Fatalf("changed password retained authenticated session: %v", err)
	}
}

func TestNativeSessionInactiveUserLosesAuthentication(t *testing.T) {
	validator, reader, loggedIn := nativeSessionFixture(t)
	reader.users[fixtureUserID] = User{Password: fixturePassword, Active: false, Timezone: "UTC"}
	if _, err := validator.Validate(t.Context(), loggedIn.SessionKey); !errors.Is(err, ErrInactiveUser) {
		t.Fatalf("inactive account retained authenticated session: %v", err)
	}
}

func TestNativeSessionLogoutInvalidatesAuthentication(t *testing.T) {
	validator, _, loggedIn := nativeSessionFixture(t)
	if err := validator.Flush(t.Context(), &loggedIn); err != nil {
		t.Fatal(err)
	}
	if _, err := validator.Validate(t.Context(), loggedIn.SessionKey); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("logout retained authenticated session: %v", err)
	}
}

func TestNativeSessionLoginRotatesAnonymousKeyAndPreservesOAuthState(t *testing.T) {
	validator, reader, _ := nativeSessionFixture(t)
	anonymous, err := validator.SaveValues(t.Context(), nil, map[string]any{"state": "native-oauth-state", "next_path": "/settings/"}, 600, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	loggedIn, err := validator.Login(t.Context(), &anonymous, AuthenticatedSession{
		UserID: fixtureUserID, PasswordHash: fixturePassword,
	}, 3600, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if loggedIn.SessionKey == anonymous.SessionKey {
		t.Fatal("login reused the anonymous session key")
	}
	if _, exists := reader.sessions[anonymous.SessionKey]; exists {
		t.Fatal("login retained the old anonymous key")
	}
	if state, ok := loggedIn.Get("state"); !ok || state != "native-oauth-state" {
		t.Fatalf("OAuth state lost: %v", state)
	}
}

func TestNativeSessionRotatesFallbackSigningKey(t *testing.T) {
	_, reader, original := nativeSessionFixture(t)
	validator, err := New(reader, Config{SecretKey: "new-native-secret", FallbackKeys: []string{fixtureCurrentSecret}})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := validator.Validate(t.Context(), original.SessionKey)
	if err != nil || !principal.UsedFallback {
		t.Fatalf("fallback validation: principal=%+v error=%v", principal, err)
	}
	rotated, err := validator.RefreshPrincipal(t.Context(), principal, 3600, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rotated.SessionKey == original.SessionKey {
		t.Fatal("fallback rotation reused the original key")
	}
	currentOnly, err := New(reader, Config{SecretKey: "new-native-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := currentOnly.Validate(t.Context(), rotated.SessionKey); err != nil {
		t.Fatalf("rotated session still needs the fallback key: %v", err)
	}
	if _, err := currentOnly.Validate(t.Context(), original.SessionKey); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("rotation retained old session: %v", err)
	}
}

func TestNativeSessionTamperingFailsClosed(t *testing.T) {
	validator, reader, loggedIn := nativeSessionFixture(t)
	row := reader.sessions[loggedIn.SessionKey]
	parts := strings.Split(row.Data, ".")
	row.Data = parts[0] + ".e30." + parts[2]
	reader.sessions[loggedIn.SessionKey] = row
	if _, err := validator.Validate(t.Context(), loggedIn.SessionKey); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("tampered payload retained authentication: %v", err)
	}
}

func TestNativeSessionStoreFailureFailsClosed(t *testing.T) {
	validator, reader, loggedIn := nativeSessionFixture(t)
	reader.userErr = errors.New("database unavailable")
	if _, err := validator.Validate(t.Context(), loggedIn.SessionKey); !errors.Is(err, ErrUserStore) {
		t.Fatalf("user store failure retained authentication: %v", err)
	}
}

func TestNativeSessionLoginPersistsTypedDeviceMetadata(t *testing.T) {
	validator, reader, _ := nativeSessionFixture(t)
	_, err := validator.Login(t.Context(), nil, AuthenticatedSession{
		UserID: fixtureUserID, PasswordHash: fixturePassword,
		DeviceInfo: map[string]string{"user_agent": "native-client", "ip_address": "192.0.2.1"},
	}, 3600, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	device, ok := reader.updatedDeviceInfo.(map[string]any)
	if !ok || device["user_agent"] != "native-client" || device["ip_address"] != "192.0.2.1" {
		t.Fatalf("persisted device metadata=%#v", reader.updatedDeviceInfo)
	}
}
