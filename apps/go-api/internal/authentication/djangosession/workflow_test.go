package djangosession

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSaveValuesPreservesOtherDjangoSessionValues(t *testing.T) {
	validator, key := newFixtureValidator(t, fixtureCurrentSession, fixturePassword, true)
	principal, prepared, err := validator.Inspect(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	prepared = principal.PrepareRefresh()

	updated, err := validator.SaveValues(context.Background(), &prepared, map[string]any{"state": "oauth-state", "host": "app.example.test"}, 3600, time.Now())
	if err != nil {
		t.Fatalf("SaveValues() error = %v", err)
	}
	if updated.SessionKey != key {
		t.Fatalf("SaveValues() changed session key: got %q, want %q", updated.SessionKey, key)
	}
	if got, ok := updated.Get("_auth_user_id"); !ok || got != fixtureUserID {
		t.Fatalf("SaveValues() lost authenticated user: value=%#v found=%v", got, ok)
	}
	if got, ok := updated.Get("state"); !ok || got != "oauth-state" {
		t.Fatalf("SaveValues() state = %#v, found=%v", got, ok)
	}
}

func TestLoginCyclesAnonymousSessionAndPreservesOAuthState(t *testing.T) {
	validator, _ := newFixtureValidator(t, fixtureCurrentSession, fixturePassword, true)
	ctx := context.Background()
	initial, err := validator.SaveValues(ctx, nil, map[string]any{
		"host":      "app.example.test",
		"next_path": "/settings/",
		"state":     "oauth-state",
	}, 3600, time.Now())
	if err != nil {
		t.Fatalf("SaveValues() error = %v", err)
	}
	oldKey := initial.SessionKey
	reader := validator.reader.(*memoryReader)

	loggedIn, err := validator.Login(ctx, &initial, AuthenticatedSession{
		UserID:       fixtureUserID,
		PasswordHash: fixturePassword,
		Backend:      fixtureBackend,
		DeviceInfo:   map[string]any{"user_agent": "test-agent"},
	}, 3600, time.Now())
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if loggedIn.SessionKey == oldKey {
		t.Fatalf("Login() did not cycle the session key")
	}
	if _, exists := reader.sessions[oldKey]; exists {
		t.Fatalf("Login() retained old session key %q", oldKey)
	}
	for key, want := range map[string]any{
		"host":               "app.example.test",
		"next_path":          "/settings/",
		"state":              "oauth-state",
		"_auth_user_id":      fixtureUserID,
		"_auth_user_backend": fixtureBackend,
	} {
		if got, ok := loggedIn.Get(key); !ok || got != want {
			t.Errorf("loggedIn.Get(%q) = %#v, found=%v; want %#v", key, got, ok, want)
		}
	}
	if got, ok := loggedIn.Get("device_info"); !ok || got == nil {
		t.Fatalf("Login() did not retain device_info: %#v, found=%v", got, ok)
	}
	if got, err := validator.Validate(ctx, loggedIn.SessionKey); err != nil || got.UserID != fixtureUserID {
		t.Fatalf("Validate(logged in session) = %#v, %v", got, err)
	}
}

func TestLoginMatchesDjangoExistingAuthenticationDecision(t *testing.T) {
	ctx := context.Background()
	identity := AuthenticatedSession{
		UserID: fixtureUserID, PasswordHash: fixturePassword, Backend: fixtureBackend,
		DeviceInfo: map[string]any{"user_agent": "test-agent"},
	}
	matchingHash := ""
	for _, secret := range []string{fixtureCurrentSecret} {
		matchingHash = fmt.Sprintf("%x", djangoHMAC(AuthHashSalt, fixturePassword, secret))
	}
	tests := []struct {
		name       string
		values     map[string]any
		wantRotate bool
		wantState  bool
	}{
		{name: "anonymous session cycles and preserves state", values: map[string]any{"state": "keep"}, wantRotate: true, wantState: true},
		{name: "same user with matching auth hash retains key", values: map[string]any{"_auth_user_id": fixtureUserID, "_auth_user_hash": matchingHash, "state": "keep"}, wantState: true},
		{name: "same user with missing hash flushes", values: map[string]any{"_auth_user_id": fixtureUserID, "state": "discard"}, wantRotate: true},
		{name: "different user flushes", values: map[string]any{"_auth_user_id": "22222222-2222-2222-2222-222222222222", "_auth_user_hash": matchingHash, "state": "discard"}, wantRotate: true},
		{name: "mismatched auth hash flushes", values: map[string]any{"_auth_user_id": fixtureUserID, "_auth_user_hash": "wrong", "state": "discard"}, wantRotate: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator, _ := newFixtureValidator(t, fixtureCurrentSession, fixturePassword, true)
			current, err := validator.SaveValues(ctx, nil, test.values, 3600, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			oldKey := current.SessionKey
			reader := validator.reader.(*memoryReader)

			loggedIn, err := validator.Login(ctx, &current, identity, 3600, time.Now())
			if err != nil {
				t.Fatalf("Login() error = %v", err)
			}
			rotated := loggedIn.SessionKey != oldKey
			if rotated != test.wantRotate {
				t.Fatalf("Login() rotated key = %v, want %v", rotated, test.wantRotate)
			}
			if _, oldExists := reader.sessions[oldKey]; oldExists == test.wantRotate {
				t.Fatalf("Login() old-key existence = %v, want %v", oldExists, !test.wantRotate)
			}
			_, hasState := loggedIn.Get("state")
			if hasState != test.wantState {
				t.Fatalf("Login() preserved arbitrary state = %v, want %v", hasState, test.wantState)
			}
		})
	}
}

func TestFlushDeletesOldRowWithoutCreatingReplacement(t *testing.T) {
	validator, _ := newFixtureValidator(t, fixtureCurrentSession, fixturePassword, true)
	ctx := context.Background()
	initial, err := validator.SaveValues(ctx, nil, map[string]any{"state": "temporary"}, 3600, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	reader := validator.reader.(*memoryReader)
	if err := validator.Flush(ctx, &initial); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if _, exists := reader.sessions[initial.SessionKey]; exists {
		t.Fatalf("Flush() retained old session %q", initial.SessionKey)
	}
	if got := len(reader.sessions); got != 1 {
		t.Fatalf("Flush() left %d session rows, want only unrelated fixture row", got)
	}
	if _, exists := reader.sessions[fixtureSessionKey]; !exists {
		t.Fatal("Flush() removed an unrelated session row")
	}
}

func TestResponseSessionActionOverridesAndExpires(t *testing.T) {
	ctx, _ := WithResponseSessionAction(context.Background())
	refresh := SessionRefresh{SessionKey: "rotated-session-key"}
	if !SetResponseSession(ctx, refresh) {
		t.Fatal("SetResponseSession() found no response action")
	}
	got, expire, found := ResponseSessionActionFromContext(ctx)
	if !found || expire || got == nil || got.SessionKey != refresh.SessionKey {
		t.Fatalf("ResponseSessionActionFromContext() = %#v, %v, %v", got, expire, found)
	}
	if !ExpireResponseSession(ctx) {
		t.Fatal("ExpireResponseSession() found no response action")
	}
	got, expire, found = ResponseSessionActionFromContext(ctx)
	if !found || !expire || got != nil {
		t.Fatalf("expired action = %#v, %v, %v", got, expire, found)
	}
}

func TestCurrentSessionContextReturnsCopy(t *testing.T) {
	prepared := SessionRefresh{SessionKey: "request-session-key"}
	ctx := WithCurrentSession(context.Background(), prepared)
	got, found := CurrentSessionFromContext(ctx)
	if !found || got == nil || got.SessionKey != prepared.SessionKey {
		t.Fatalf("CurrentSessionFromContext() = %#v, %v", got, found)
	}
	got.SessionKey = "changed"
	second, _ := CurrentSessionFromContext(ctx)
	if second.SessionKey != prepared.SessionKey {
		t.Fatalf("context snapshot mutated through returned copy: %q", second.SessionKey)
	}
}
