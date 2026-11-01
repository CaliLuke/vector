package browsersession

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/loom/clue/log"
)

const (
	fixtureCurrentSecret     = "current-secret-for-go-session-fixture"
	fixtureFallbackSecret    = "fallback-secret-for-go-session-fixture"
	fixturePassword          = "pbkdf2_sha256$1000000$opaque$opaquehash"
	fixtureUserID            = "11111111-1111-1111-1111-111111111111"
	fixtureCurrentHash       = "14254b029477bbf9b6e7c90a133ecb722cb4c50f77199f73839bb2c05527d58a"
	fixtureFallbackHash      = "c36eb78278595dc39e296f158eec17c4c1e2d61e01fb5a0629420f04dd4d1df7"
	fixtureCurrentSession    = "v1.eyJ1c2VyX2lkIjoiMTExMTExMTEtMTExMS0xMTExLTExMTEtMTExMTExMTExMTExIiwiYXV0aF9oYXNoIjoiMTQyNTRiMDI5NDc3YmJmOWI2ZTdjOTBhMTMzZWNiNzIyY2I0YzUwZjc3MTk5ZjczODM5YmIyYzA1NTI3ZDU4YSJ9.ViWMqrXhdUPFA9McK1sDDAlUgGQh-p9hsu3RkP08VUE"
	fixtureFallbackSession   = "v1.eyJ1c2VyX2lkIjoiMTExMTExMTEtMTExMS0xMTExLTExMTEtMTExMTExMTExMTExIiwiYXV0aF9oYXNoIjoiYzM2ZWI3ODI3ODU5NWRjMzllMjk2ZjE1OGVlYzE3YzRjMWUyZDYxZTAxZmI1YTA2Mjk0MjBmMDRkZDRkMWRmNyJ9.Z4UzRM3ycYI0sII9CVpMNGUFdKrVRIJVaLaHjPR5BVM"
	fixtureCompressedSession = "v1.eyJmaWxsZXIiOiJyZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtcmVwZWF0LXJlcGVhdC1yZXBlYXQtIiwidXNlcl9pZCI6IjExMTExMTExLTExMTEtMTExMS0xMTExLTExMTExMTExMTExMSIsImF1dGhfaGFzaCI6IjE0MjU0YjAyOTQ3N2JiZjliNmU3YzkwYTEzM2VjYjcyMmNiNGM1MGY3NzE5OWY3MzgzOWJiMmMwNTUyN2Q1OGEifQ.Yy6_Uihm23GiE2mx-WdFGo4Jp6FVcV5FAbsm2dc4NK4"
)

type memoryReader struct {
	sessions          map[string]Session
	users             map[string]User
	expiry            time.Time
	updatedUserID     any
	updatedDeviceInfo any
	updateErr         error
	sessionErr        error
	userErr           error
}

func (reader *memoryReader) ReadSession(_ context.Context, key string) (Session, error) {
	if reader.sessionErr != nil {
		return Session{}, reader.sessionErr
	}
	session, ok := reader.sessions[key]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (reader *memoryReader) ReadUser(_ context.Context, id string) (User, error) {
	if reader.userErr != nil {
		return User{}, reader.userErr
	}
	user, ok := reader.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (reader *memoryReader) UpdateSession(_ context.Context, key string, data string, expireAt time.Time, userID any, deviceInfo any) error {
	if reader.updateErr != nil {
		return reader.updateErr
	}
	session, ok := reader.sessions[key]
	if !ok {
		return ErrSessionNotFound
	}
	session.Data = data
	reader.sessions[key] = session
	reader.expiry = expireAt
	reader.updatedUserID = userID
	reader.updatedDeviceInfo = deviceInfo
	return nil
}

func (reader *memoryReader) CreateSession(_ context.Context, key string, data string, expireAt time.Time, userID any, deviceInfo any) error {
	if _, found := reader.sessions[key]; found {
		return ErrKeyCollision
	}
	reader.sessions[key] = Session{Data: data}
	reader.expiry = expireAt
	reader.updatedUserID = userID
	reader.updatedDeviceInfo = deviceInfo
	return nil
}

func (reader *memoryReader) CycleSession(_ context.Context, oldKey string, newKey string, data string, expireAt time.Time, userID any, deviceInfo any) error {
	if _, found := reader.sessions[newKey]; found {
		return ErrKeyCollision
	}
	if _, found := reader.sessions[oldKey]; !found {
		return ErrSessionNotFound
	}
	reader.sessions[newKey] = Session{Data: data}
	delete(reader.sessions, oldKey)
	reader.expiry = expireAt
	reader.updatedUserID = userID
	reader.updatedDeviceInfo = deviceInfo
	return nil
}

func (reader *memoryReader) DeleteSession(_ context.Context, key string) error {
	delete(reader.sessions, key)
	return nil
}

func TestNativeFixturesValidateCurrentAndFallbackSecrets(t *testing.T) {
	tests := []struct {
		name         string
		data         string
		usedFallback bool
	}{
		{name: "current secret", data: fixtureCurrentSession},
		{name: "fallback secret", data: fixtureFallbackSession, usedFallback: true},
		{name: "compressed JSON", data: fixtureCompressedSession},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator, key := newFixtureValidator(t, test.data, fixturePassword)
			request := httptest.NewRequest("GET", "/api/workspaces/example/stickies/", nil)
			request.Header.Set("Cookie", DefaultCookieName+"="+key)
			principal, err := validator.ValidateRequest(request.Context(), request)
			if err != nil {
				t.Fatalf("ValidateRequest() error = %v", err)
			}
			if principal.UserID != fixtureUserID || principal.SessionKey != key || principal.UsedFallback != test.usedFallback {
				t.Fatalf("ValidateRequest() = %#v", principal)
			}
		})
	}
}

func TestValidateUserTimezoneMatchesZoneInfoErrorClasses(t *testing.T) {
	if err := ValidateUserTimezone("UTC"); err != nil {
		t.Fatalf("UTC rejected: %v", err)
	}
	if err := ValidateUserTimezone("Mars/Olympus"); !errors.Is(err, ErrTimezoneNotFound) {
		t.Fatalf("missing timezone error=%v", err)
	}
	if err := ValidateUserTimezone("Local"); !errors.Is(err, ErrTimezoneNotFound) {
		t.Fatalf("Go-special timezone error=%v", err)
	}
	for _, value := range []string{"", "/UTC", ".", "..", "../UTC", "Region/../UTC", "Region//City", "UTC\x00"} {
		if err := ValidateUserTimezone(value); !errors.Is(err, ErrInvalidTimezone) {
			t.Fatalf("invalid timezone %q error=%v", value, err)
		}
	}
}

func TestSessionKeyMinimumUsesUnicodeCharacters(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	reader.sessionErr = errors.New("database must not be queried")
	validator := configuredValidator(t, reader)
	if _, err := validator.Validate(context.Background(), "éééé"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Validate() error=%v", err)
	}
	if _, err := validator.Refresh(context.Background(), "éééé", 604800, time.Now()); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Refresh() error=%v", err)
	}
}

func TestSessionConfigurationRejectsInvalidCookieNames(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	for _, name := range []string{"", "session name", "session;id"} {
		t.Run(name, func(t *testing.T) {
			if _, err := New(reader, Config{SecretKey: fixtureCurrentSecret, CookieName: name, CookieNameConfigured: true}); err == nil {
				t.Fatal("invalid cookie name accepted")
			}
		})
	}
}

func TestBadSessionSignatureEmitsSuspiciousSessionWarning(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession[:len(fixtureCurrentSession)-1]+"A", fixturePassword, true)
	validator := configuredValidator(t, reader)
	var logs bytes.Buffer
	ctx := log.Context(
		context.Background(),
		log.WithOutputs(log.Output{Writer: &logs, Format: log.FormatJSON}),
		log.WithDisableBuffering(func(context.Context) bool { return true }),
	)
	if _, err := validator.Validate(ctx, fixtureSessionKey); !errors.Is(err, ErrBadSignature) || !errors.Is(err, ErrInvalidData) {
		t.Fatalf("Validate() error=%v", err)
	}
	if !strings.Contains(logs.String(), "vector.security.session") || !strings.Contains(logs.String(), "Session data corrupted") {
		t.Fatalf("warning log=%s", logs.String())
	}
}

func TestSignedNonObjectSessionFailsWithoutCorruptionRewrite(t *testing.T) {
	for _, serialized := range []string{`[]`, `null`, `"session"`, `1`} {
		t.Run(serialized, func(t *testing.T) {
			data := signPayloadForTest(t, []byte(serialized), false, fixtureCurrentSecret)
			reader := fixtureReader(data, fixturePassword, true)
			validator := configuredValidator(t, reader)
			var logs bytes.Buffer
			ctx := log.Context(
				context.Background(),
				log.WithOutputs(log.Output{Writer: &logs, Format: log.FormatJSON}),
				log.WithDisableBuffering(func(context.Context) bool { return true }),
			)
			if _, prepared, err := validator.Inspect(ctx, fixtureSessionKey); !errors.Is(err, ErrInvalidSessionShape) || errors.Is(err, ErrInvalidData) || prepared.SessionKey != "" {
				t.Fatalf("Inspect() prepared=%#v error=%v", prepared, err)
			}
			if strings.Contains(logs.String(), "vector.security.session") {
				t.Fatalf("valid signed payload logged as corrupt: %s", logs.String())
			}
			if _, err := validator.RefreshUnaccessed(ctx, fixtureSessionKey, 3600, time.Now()); !errors.Is(err, ErrInvalidSessionShape) {
				t.Fatalf("RefreshUnaccessed() error=%v", err)
			}
			if reader.sessions[fixtureSessionKey].Data != data {
				t.Fatal("valid signed non-object session was rewritten")
			}
		})
	}
}

func TestValidateClassifiesSessionAndUserTableFailures(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	reader.sessionErr = errors.New("sessions table unavailable")
	if _, err := validator.Validate(context.Background(), fixtureSessionKey); !errors.Is(err, ErrSessionStore) {
		t.Fatalf("session failure=%v, want ErrSessionStore", err)
	}
	reader.sessionErr = nil
	reader.userErr = errors.New("users table unavailable")
	if _, err := validator.Validate(context.Background(), fixtureSessionKey); !errors.Is(err, ErrUserStore) {
		t.Fatalf("user failure=%v, want ErrUserStore", err)
	}
}

func TestValidateRejectsMalformedUserIdentifier(t *testing.T) {
	values := claims("not-a-uuid", fixtureCurrentHash)
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, true)
	reader.userErr = ErrInvalidUserID
	validator := configuredValidator(t, reader)
	if _, err := validator.Validate(context.Background(), fixtureSessionKey); !errors.Is(err, ErrInvalidUserID) || errors.Is(err, ErrUserStore) {
		t.Fatalf("malformed user identifier error=%v, want only ErrInvalidUserID", err)
	}
}

func TestValidateConvertsUserIdentifierBeforeReadingBackendOrHash(t *testing.T) {
	values := map[string]any{"user_id": "not-a-uuid"}
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, true)
	validator := configuredValidator(t, reader)
	if _, err := validator.Validate(context.Background(), fixtureSessionKey); !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("validation error=%v, want ErrInvalidUserID", err)
	}
}

func TestValidateTreatsNullUserIdentifierAsMissingUser(t *testing.T) {
	values := map[string]any{
		"user_id": nil,

		"auth_hash": fixtureCurrentHash,
	}
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, true)
	reader.userErr = errors.New("null UUID must not query the user table")
	validator := configuredValidator(t, reader)
	_, prepared, err := validator.Inspect(context.Background(), fixtureSessionKey)
	if !errors.Is(err, ErrInvalidUserID) || prepared.SessionKey != fixtureSessionKey {
		t.Fatalf("Inspect() prepared=%#v error=%v", prepared, err)
	}
}

func TestValidateLooksUpUserBeforeCheckingAuthenticationHash(t *testing.T) {
	values := map[string]any{
		"user_id": fixtureUserID,
	}
	for _, testCase := range []struct {
		name        string
		active      bool
		missingUser bool
		want        error
	}{
		{name: "missing user", active: true, missingUser: true, want: ErrUserNotFound},
		{name: "inactive user", want: ErrInactiveUser},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, testCase.active)
			if testCase.missingUser {
				reader.users = map[string]User{}
			}
			validator := configuredValidator(t, reader)
			if _, err := validator.Validate(context.Background(), fixtureSessionKey); !errors.Is(err, testCase.want) {
				t.Fatalf("validation error=%v, want %v", err, testCase.want)
			}
		})
	}
}

func TestSessionSaveCanonicalizesDuplicateKeys(t *testing.T) {
	serialized := []byte(`{ "x" : 1, "x": 2 }`)
	reader := fixtureReader(signPayloadForTest(t, serialized, false, fixtureCurrentSecret), fixturePassword, true)
	validator := configuredValidator(t, reader)
	if _, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 3600, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, refreshed, _, err := decodeSessionData(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || string(refreshed) != `{"x":2}` {
		t.Fatalf("refreshed=%q err=%v", refreshed, err)
	}
}

func TestFallbackAuthHashIsAcceptedWithoutMutation(t *testing.T) {
	data := encodeForTest(t, fixtureCurrentSecret, fixtureFallbackHash, false)
	reader := fixtureReader(data, fixturePassword, true)
	validator := configuredValidator(t, reader)
	before := reader.sessions[fixtureSessionKey]
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !principal.UsedFallback {
		t.Fatal("Validate() did not report fallback auth hash")
	}
	if reader.sessions[fixtureSessionKey] != before {
		t.Fatal("Validate() mutated session state")
	}
}

func TestRefreshDoesNotCycleSignatureOnlyFallback(t *testing.T) {
	data := encodeForTest(t, fixtureFallbackSecret, fixtureCurrentHash, false)
	reader := fixtureReader(data, fixturePassword, true)
	validator := configuredValidator(t, reader)
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil || !principal.UsedFallback || principal.AuthHashUsedFallback {
		t.Fatalf("signature-only fallback principal=%#v err=%v", principal, err)
	}
	refresh, err := validator.Refresh(context.Background(), fixtureSessionKey, 3600, time.Now())
	if err != nil || refresh.SessionKey != fixtureSessionKey {
		t.Fatalf("signature-only refresh=%#v err=%v", refresh, err)
	}
}

func TestRefreshPrincipalRejectsFallbackAfterConcurrentLogout(t *testing.T) {
	reader := fixtureReader(fixtureFallbackSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil || !principal.AuthHashUsedFallback {
		t.Fatalf("fallback principal=%#v err=%v", principal, err)
	}
	delete(reader.sessions, fixtureSessionKey)
	refresh, err := validator.RefreshPrincipal(context.Background(), principal, 3600, time.Now())
	if !errors.Is(err, ErrSessionNotFound) || refresh.SessionKey != "" || len(reader.sessions) != 0 {
		t.Fatalf("revoked session refreshed=%#v err=%v remaining=%d", refresh, err, len(reader.sessions))
	}
}

func TestRefreshPrincipalPersistsRequestLocalSnapshotAfterConcurrentUpdate(t *testing.T) {
	original := claims(fixtureUserID, fixtureCurrentHash)
	original["note"] = "request snapshot"
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, original, false), fixturePassword, true)
	validator := configuredValidator(t, reader)
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil {
		t.Fatal(err)
	}
	concurrent := claims(fixtureUserID, fixtureCurrentHash)
	concurrent["note"] = "concurrent database update"
	reader.sessions[fixtureSessionKey] = Session{Data: encodeValuesForTest(t, fixtureCurrentSecret, concurrent, false)}
	if _, refreshPrincipalErr := validator.RefreshPrincipal(context.Background(), principal, 3600, time.Now()); refreshPrincipalErr != nil {
		t.Fatal(refreshPrincipalErr)
	}
	values, _, err := decode(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || values["note"] != "request snapshot" {
		t.Fatalf("refreshed values=%v err=%v", values, err)
	}
}

func TestInspectPreservesAnonymousRequestLocalSnapshot(t *testing.T) {
	original := map[string]any{"anonymous-note": "request snapshot"}
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, original, false), fixturePassword, true)
	validator := configuredValidator(t, reader)
	_, prepared, err := validator.Inspect(context.Background(), fixtureSessionKey)
	if !errors.Is(err, ErrUserMismatch) || prepared.SessionKey != fixtureSessionKey {
		t.Fatalf("Inspect() prepared=%#v err=%v", prepared, err)
	}
	reader.sessions[fixtureSessionKey] = Session{Data: encodeValuesForTest(
		t, fixtureCurrentSecret, map[string]any{"anonymous-note": "concurrent database update"}, false,
	)}
	if _, refreshPreparedErr := validator.RefreshPrepared(context.Background(), prepared, 3600, time.Now()); refreshPreparedErr != nil {
		t.Fatal(refreshPreparedErr)
	}
	values, _, err := decode(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || values["anonymous-note"] != "request snapshot" {
		t.Fatalf("refreshed values=%v err=%v", values, err)
	}
}

func TestRefreshReSignsAndExtendsNativeSession(t *testing.T) {
	values := claims(fixtureUserID, fixtureCurrentHash)
	values["note"] = "Plane ✈"
	values["device_info"] = map[string]any{"browser": "test"}
	serialized, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	reader := fixtureReader(signPayloadForTest(t, serialized, false, fixtureCurrentSecret), fixturePassword, true)
	validator := configuredValidator(t, reader)
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	refresh, err := validator.Refresh(context.Background(), fixtureSessionKey, 604800, now)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refresh.SessionKey != fixtureSessionKey || refresh.MaxAgeSeconds != 604800 || refresh.BrowserClose ||
		!refresh.ExpiresAt.Equal(now.Add(7*24*time.Hour)) || !refresh.CookieExpiresAt.Equal(now.Add(7*24*time.Hour)) {
		t.Fatalf("Refresh() = %#v", refresh)
	}
	if !reader.expiry.Equal(refresh.ExpiresAt) {
		t.Fatalf("database expiry = %v, want %v", reader.expiry, refresh.ExpiresAt)
	}
	if reader.updatedUserID != fixtureUserID || reader.updatedDeviceInfo == nil {
		t.Fatalf("updated denormalized fields: user=%#v device=%#v", reader.updatedUserID, reader.updatedDeviceInfo)
	}
	if _, validationErr := validator.Validate(context.Background(), fixtureSessionKey); validationErr != nil {
		t.Fatalf("refreshed session does not validate: %v", validationErr)
	}
	decoded, secretIndex, err := decode(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || secretIndex != 0 || decoded["note"] != "Plane ✈" {
		t.Fatalf("decoded refreshed session=%#v secret=%d err=%v", decoded, secretIndex, err)
	}
}

func TestRefreshUnaccessedSavesWithoutAuthentication(t *testing.T) {
	values := claims(fixtureUserID, "authentication-hash-not-validated")
	values["note"] = "request snapshot"
	reader := fixtureReader(encodeValuesForTest(t, fixtureFallbackSecret, values, false), fixturePassword, true)
	reader.userErr = errors.New("user lookup must not run")
	validator := configuredValidator(t, reader)
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	refresh, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 604800, now)
	if err != nil || refresh.SessionKey != fixtureSessionKey {
		t.Fatalf("refresh=%#v err=%v", refresh, err)
	}
	decoded, secretIndex, err := decode(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || secretIndex != 0 || decoded["note"] != "request snapshot" || decoded["auth_hash"] != "authentication-hash-not-validated" {
		t.Fatalf("decoded=%#v secret=%d err=%v", decoded, secretIndex, err)
	}
}

func TestRefreshPreservesLargeSessionAge(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	refresh, err := validator.Refresh(context.Background(), fixtureSessionKey, 10_000_000_000, now)
	if err != nil || refresh.MaxAgeSeconds != 10_000_000_000 {
		t.Fatalf("Refresh()=%#v err=%v", refresh, err)
	}
	want := time.Unix(now.Unix()+10_000_000_000, 0).UTC()
	if !refresh.ExpiresAt.Equal(want) || !reader.expiry.Equal(want) {
		t.Fatalf("response expiry=%v database expiry=%v want=%v", refresh.ExpiresAt, reader.expiry, want)
	}
}

func TestRefreshRejectsExpiryOutsideCalendarRange(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	if _, err := validator.Refresh(context.Background(), fixtureSessionKey, 1_000_000_000_000, time.Now()); !errors.Is(err, ErrStore) {
		t.Fatalf("Refresh() error=%v", err)
	}
}

func TestRefreshCyclesFallbackSessionAndReplacesAuthHash(t *testing.T) {
	reader := fixtureReader(fixtureFallbackSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	refresh, err := validator.Refresh(context.Background(), fixtureSessionKey, 604800, now)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refresh.SessionKey == fixtureSessionKey || len(refresh.SessionKey) != 128 {
		t.Fatalf("Refresh() did not cycle the 128-character session key: %#v", refresh)
	}
	if _, found := reader.sessions[fixtureSessionKey]; found {
		t.Fatal("fallback session retained its old key")
	}
	principal, err := validator.Validate(context.Background(), refresh.SessionKey)
	if err != nil || principal.UsedFallback {
		t.Fatalf("rotated session principal=%#v err=%v", principal, err)
	}
	values, secretIndex, err := decode(reader.sessions[refresh.SessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || secretIndex != 0 || values["auth_hash"] != fixtureCurrentHash {
		t.Fatalf("rotated values=%#v secret=%d err=%v", values, secretIndex, err)
	}
}

func TestRefreshClassifiesDatabaseUpdateFailures(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	reader.updateErr = errors.New("database unavailable")
	validator := configuredValidator(t, reader)
	_, err := validator.Refresh(context.Background(), fixtureSessionKey, 3600, time.Now())
	if !errors.Is(err, ErrStore) || !errors.Is(err, ErrSessionUpdate) {
		t.Fatalf("Refresh() error=%v", err)
	}
}

func TestPasswordChangesInvalidateExistingSessions(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "password change", password: "pbkdf2_sha256$new-password-hash"},
		{name: "password reset", password: "!reset-password-is-unusable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator, _ := newFixtureValidator(t, fixtureCurrentSession, test.password)
			_, err := validator.Validate(context.Background(), fixtureSessionKey)
			if !errors.Is(err, ErrInvalidAuthHash) {
				t.Fatalf("Validate() error = %v, want ErrInvalidAuthHash", err)
			}
		})
	}
}

func TestMalformedAndCorruptSessionsAreRejected(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "no fields", data: encodeValuesForTest(t, fixtureCurrentSecret, map[string]any{}, false)},
		{name: "malformed framing", data: "not-a-django-session"},
		{name: "unconfigured secret", data: encodeForTest(t, "unconfigured-secret", fixtureCurrentHash, false)},
		{name: "corrupt signature", data: fixtureCurrentSession[:len(fixtureCurrentSession)-1] + "A"},
		{name: "invalid timestamp", data: resignWithTimestamp(fixtureCurrentSession, "bad timestamp", fixtureCurrentSecret)},
		{name: "invalid base64", data: signForTest("*:1AbC", fixtureCurrentSecret)},
		{name: "invalid JSON", data: signPayloadForTest(t, []byte("{"), false, fixtureCurrentSecret)},
		{name: "corrupt zlib", data: signPayloadForTest(t, []byte("not-zlib"), true, fixtureCurrentSecret)},
		{name: "JSON array", data: signPayloadForTest(t, []byte("[]"), false, fixtureCurrentSecret)},
		{name: "wrong field type", data: encodeValuesForTest(t, fixtureCurrentSecret, map[string]any{
			"user_id": 7, "auth_hash": fixtureCurrentHash,
		}, false)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator, _ := newFixtureValidator(t, test.data, fixturePassword)
			_, err := validator.Validate(context.Background(), fixtureSessionKey)
			if err == nil {
				t.Fatal("Validate() accepted malformed session")
			}
		})
	}
}

func TestAuthenticationRejections(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*memoryReader) string
		want      error
	}{
		{name: "deleted or unknown key", configure: func(reader *memoryReader) string { return "unknown-session-key" }, want: ErrSessionNotFound},
		{name: "expired session omitted by reader", configure: func(reader *memoryReader) string {
			delete(reader.sessions, fixtureSessionKey)
			return fixtureSessionKey
		}, want: ErrSessionNotFound},
		{name: "deleted user", configure: func(reader *memoryReader) string { delete(reader.users, fixtureUserID); return fixtureSessionKey }, want: ErrUserNotFound},
		{name: "inactive user", configure: func(reader *memoryReader) string {
			user := reader.users[fixtureUserID]
			user.Active = false
			reader.users[fixtureUserID] = user
			return fixtureSessionKey
		}, want: ErrInactiveUser},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
			key := test.configure(reader)
			validator := configuredValidator(t, reader)
			_, err := validator.Validate(context.Background(), key)
			if !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestInvalidUserClaimsAreRejected(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
		want   error
	}{
		{name: "unknown auth user", values: claims("22222222-2222-2222-2222-222222222222", fixtureCurrentHash), want: ErrUserNotFound},
		{name: "missing auth hash", values: claims(fixtureUserID, ""), want: ErrInvalidAuthHash},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := encodeValuesForTest(t, fixtureCurrentSecret, test.values, false)
			validator, _ := newFixtureValidator(t, data, fixturePassword)
			_, err := validator.Validate(context.Background(), fixtureSessionKey)
			if !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestCookieAndConfigurationValidation(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	request := httptest.NewRequest("GET", "/", nil)
	if _, err := validator.ValidateRequest(request.Context(), request); !errors.Is(err, ErrNoCookie) {
		t.Fatalf("ValidateRequest() error = %v, want ErrNoCookie", err)
	}
	if _, err := validator.Validate(context.Background(), "short"); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("Validate() error = %v, want ErrInvalidKey", err)
	}
	if _, err := New(reader, Config{}); err == nil {
		t.Fatal("New() accepted missing SECRET_KEY")
	}
}

const fixtureSessionKey = "fixture-session-key"

func newFixtureValidator(t *testing.T, data string, password string) (*Validator, string) {
	t.Helper()
	return configuredValidator(t, fixtureReader(data, password, true)), fixtureSessionKey
}

func fixtureReader(data string, password string, active bool) *memoryReader {
	return &memoryReader{
		sessions: map[string]Session{fixtureSessionKey: {Data: data}},
		users:    map[string]User{fixtureUserID: {Password: password, Active: active, Timezone: "UTC"}},
	}
}

func configuredValidator(t *testing.T, reader Reader) *Validator {
	t.Helper()
	validator, err := New(reader, Config{
		SecretKey: fixtureCurrentSecret, FallbackKeys: []string{fixtureFallbackSecret},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return validator
}

func claims(userID string, authHash string) map[string]any {
	return map[string]any{"user_id": userID, "auth_hash": authHash}
}

func encodeForTest(t *testing.T, secret string, authHash string, compressed bool) string {
	t.Helper()
	return encodeValuesForTest(t, secret, claims(fixtureUserID, authHash), compressed)
}

func encodeValuesForTest(t *testing.T, secret string, values map[string]any, compressed bool) string {
	t.Helper()
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return signPayloadForTest(t, data, compressed, secret)
}

func signPayloadForTest(_ *testing.T, data []byte, _ bool, secret string) string {
	value := "v1." + base64.RawURLEncoding.EncodeToString(data)
	signature := base64.RawURLEncoding.EncodeToString(sessionHMAC(SessionSigningSalt+"signer", value, secret))
	return value + "." + signature
}

func signForTest(value string, secret string) string {
	signature := base64.RawURLEncoding.EncodeToString(sessionHMAC(SessionSigningSalt+"signer", value, secret))
	return value + ":" + signature
}

func resignWithTimestamp(value string, timestamp string, secret string) string {
	payload := strings.SplitN(value, ":", 2)[0]
	return signForTest(payload+":"+timestamp, secret)
}
