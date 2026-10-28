package djangosession

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	fixtureBackend           = "django.contrib.auth.backends.ModelBackend"
	fixtureCurrentHash       = "9f00eb5e11226e52f614483745d3b745e8a38f8efacd8b4792e4e1d817cf98c3"
	fixtureFallbackHash      = "0fbf54d64306e83d54602e7c14f72fccc5349abd612c1b37d41f90d61c1caa9d"
	fixtureCurrentSession    = ".eJxtjMsOwiAURP-FtZLyKheX7v0GwuMiVQNJaVfGf7dNMNHEWcxmzpknsW5dsl0bznaK5ERYz_FPfUIO35p34Y5ld-PNlWuloZZlnjzdEdrXRi814uPc2Z-D7FrebJOGAb1CxjgfUfE0MilBaKmi8FsjOAEJMLkQwUttOEpkEZgOyUAQ5PUGIa0-Xg:1AbC:xXWGwnCfPtoweEMR_QW1DR1egdVd0BEMax3RSgk9N8k"
	fixtureFallbackSession   = ".eJxtjM0KwjAQhN8lZw3Z5qfWY-8-Q9jsJrYqDTTtSXx3W4ig4BwGhplvnsLjugx-LXH2I4uzgKrjH_tIHL6xgHSP087yDadrlpSnZR6D3CeytkVeMsdHX7c_BwOWYaNVCskadkYrF0-arXGqiS2BSW2TiMhq02FgBw1B0C0bSJ3aIgEhdixebzHXPsM:1AbC:E6Ynd_kHlQMDD4kkn0oBayJpDU6jRwZR7jlyStciXlo"
	fixtureCompressedSession = ".eJztjE0OgjAUhO_StTT0jz5cuvcM5LV9FZSAKbAy3t2SYKKJFzBhFjOLmW8erMFlbptlotR0gR2Z2FT8sLfY4RNz6G80rGy44nAZuR-HOXWOrxO-tRM_j4H607b9OmhxajNdx7IkZ0gIKSsyMlZCa1BWm6BcdgJUEIEi-gBO21qSJhFAWB9r8Cqfxq7vKeWvRHfCudhjj78L9nwBuypN0Q:1AbC:sfcjhI0sJHafs9aaJ2VSnVxjzGDWPGpq9kujit6rKKQ"
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

func (reader *memoryReader) ReadSessionForRefresh(ctx context.Context, key string) (Session, error) {
	return reader.ReadSession(ctx, key)
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

func TestDjango5215FixturesValidateCurrentAndFallbackSecrets(t *testing.T) {
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

func TestValidateAcceptsDjangoPaddedBase64Payload(t *testing.T) {
	serialized, err := json.Marshal(claims(fixtureUserID, fixtureBackend, fixtureCurrentHash))
	if err != nil {
		t.Fatal(err)
	}
	serialized = djangoASCIIJSON(serialized)
	payload := base64.URLEncoding.EncodeToString(serialized)
	data := signForTest(payload+":1AbC", fixtureCurrentSecret)
	validator, _ := newFixtureValidator(t, data, fixturePassword)
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil || principal.UserID != fixtureUserID {
		t.Fatalf("Validate() principal=%#v error=%v", principal, err)
	}
}

func TestDjangoBase64DecodeMatchesPythonPaddingTolerance(t *testing.T) {
	for _, value := range []string{"e30", "e30=", "e30===", "e!3!0", "e30=garbage"} {
		decoded, err := djangoBase64Decode(value)
		if err != nil || string(decoded) != "{}" {
			t.Fatalf("djangoBase64Decode(%q)=%q error=%v", value, decoded, err)
		}
	}
	for _, value := range []string{"e=30", "e30\n"} {
		if _, err := djangoBase64Decode(value); err == nil {
			t.Fatalf("djangoBase64Decode(%q) unexpectedly succeeded", value)
		}
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

func TestExplicitEmptyCookieNameDoesNotUseDjangoDefault(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	validator, err := New(reader, Config{
		CookieNameConfigured: true,
		SecretKey:            fixtureCurrentSecret,
		AllowedBackends:      []string{fixtureBackend},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: fixtureSessionKey})
	if _, err := validator.ValidateRequest(request.Context(), request); !errors.Is(err, ErrNoCookie) {
		t.Fatalf("ValidateRequest() error=%v", err)
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
	if !strings.Contains(logs.String(), "django.security.SuspiciousSession") || !strings.Contains(logs.String(), "Session data corrupted") {
		t.Fatalf("warning log=%s", logs.String())
	}
}

func TestSignedNonObjectSessionFailsWithoutCorruptionRewrite(t *testing.T) {
	for _, serialized := range []string{`[]`, `null`, `"session"`, `1`} {
		t.Run(serialized, func(t *testing.T) {
			data := signPayloadForTest([]byte(serialized), false, fixtureCurrentSecret)
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
			if strings.Contains(logs.String(), "django.security.SuspiciousSession") {
				t.Fatalf("valid signed payload logged as corrupt: %s", logs.String())
			}
			if _, err := validator.RefreshUnaccessed(ctx, fixtureSessionKey, 3600, time.Now()); !errors.Is(err, ErrInvalidSessionShape) || !errors.Is(err, ErrStore) {
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
	values := claims("not-a-uuid", fixtureBackend, fixtureCurrentHash)
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, true)
	reader.userErr = ErrInvalidUserID
	validator := configuredValidator(t, reader)
	if _, err := validator.Validate(context.Background(), fixtureSessionKey); !errors.Is(err, ErrInvalidUserID) || errors.Is(err, ErrUserStore) {
		t.Fatalf("malformed user identifier error=%v, want only ErrInvalidUserID", err)
	}
}

func TestValidateConvertsUserIdentifierBeforeReadingBackendOrHash(t *testing.T) {
	values := map[string]any{"_auth_user_id": "not-a-uuid"}
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, true)
	validator := configuredValidator(t, reader)
	if _, err := validator.Validate(context.Background(), fixtureSessionKey); !errors.Is(err, ErrInvalidUserID) {
		t.Fatalf("validation error=%v, want ErrInvalidUserID", err)
	}
}

func TestValidateTreatsNullUserIdentifierAsMissingUser(t *testing.T) {
	values := map[string]any{
		"_auth_user_id":      nil,
		"_auth_user_backend": fixtureBackend,
		"_auth_user_hash":    fixtureCurrentHash,
	}
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, true)
	reader.userErr = errors.New("null UUID must not query the user table")
	validator := configuredValidator(t, reader)
	_, prepared, err := validator.Inspect(context.Background(), fixtureSessionKey)
	if !errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrInvalidUserID) || prepared.SessionKey != fixtureSessionKey {
		t.Fatalf("Inspect() prepared=%#v error=%v", prepared, err)
	}
}

func TestValidateLooksUpUserBeforeCheckingAuthenticationHash(t *testing.T) {
	values := map[string]any{
		"_auth_user_id":      fixtureUserID,
		"_auth_user_backend": fixtureBackend,
	}
	for _, testCase := range []struct {
		name   string
		active bool
		want   error
	}{
		{name: "missing user", active: true, want: ErrUserNotFound},
		{name: "inactive user", want: ErrInactiveUser},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, testCase.active)
			if testCase.want == ErrUserNotFound {
				reader.users = map[string]User{}
			}
			validator := configuredValidator(t, reader)
			if _, err := validator.Validate(context.Background(), fixtureSessionKey); !errors.Is(err, testCase.want) {
				t.Fatalf("validation error=%v, want %v", err, testCase.want)
			}
		})
	}
}

func TestDjangoJSONSerializerUsesLatin1(t *testing.T) {
	data := []byte(`{"_auth_user_id":"` + fixtureUserID + `","_auth_user_backend":"` + fixtureBackend + `","_auth_user_hash":"` + fixtureCurrentHash + `","note":"`)
	data = append(data, 0xe9)
	data = append(data, []byte(`"}`)...)
	values, _, err := decode(signPayloadForTest(data, false, fixtureCurrentSecret), []string{fixtureCurrentSecret})
	if err != nil {
		t.Fatalf("decode() error = %v", err)
	}
	if values["note"] != "é" {
		t.Fatalf("decode() note = %#v, want Latin-1 é", values["note"])
	}
}

func TestDjangoJSONSerializerPreservesPythonConstantsAndLoneSurrogates(t *testing.T) {
	serialized := []byte(`{"_auth_user_id":"` + fixtureUserID + `","_auth_user_backend":"` + fixtureBackend + `","_auth_user_hash":"` + fixtureCurrentHash + `","nan":NaN,"positive":Infinity,"negative":-Infinity,"surrogate":"\ud800"}`)
	reader := fixtureReader(signPayloadForTest(serialized, false, fixtureCurrentSecret), fixturePassword, true)
	validator := configuredValidator(t, reader)
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if _, refreshPrincipalErr := validator.RefreshPrincipal(context.Background(), principal, 3600, time.Now()); refreshPrincipalErr != nil {
		t.Fatalf("RefreshPrincipal() error = %v", refreshPrincipalErr)
	}
	_, refreshed, _, err := decodeSessionData(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil {
		t.Fatalf("decode refreshed session: %v", err)
	}
	for _, token := range [][]byte{[]byte(`"nan":NaN`), []byte(`"positive":Infinity`), []byte(`"negative":-Infinity`), []byte(`"surrogate":"\ud800"`)} {
		if !bytes.Contains(refreshed, token) {
			t.Fatalf("refreshed session %q does not preserve %q", refreshed, token)
		}
	}
}

func TestDjangoJSONFieldRejectsLoneSurrogatesInKeysAndValues(t *testing.T) {
	for _, serialized := range []string{
		`{"device_info":{"value":"\ud800"}}`,
		`{"device_info":{"\ud800":"value"}}`,
	} {
		values, _, err := decodeDjangoJSON([]byte(serialized))
		if err != nil {
			t.Fatal(err)
		}
		deviceInfo, ok := djangoJSONToMap(values["device_info"])
		if !ok {
			t.Fatalf("device_info=%#v", values["device_info"])
		}
		if _, err := json.Marshal(deviceInfo); err == nil {
			t.Fatalf("json.Marshal(%s) unexpectedly succeeded", serialized)
		}
	}
}

func TestSessionSaveCanonicalizesDuplicateKeys(t *testing.T) {
	serialized := []byte(`{ "x" : 1, "x": 2, "nan": NaN }`)
	reader := fixtureReader(signPayloadForTest(serialized, false, fixtureCurrentSecret), fixturePassword, true)
	validator := configuredValidator(t, reader)
	if _, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 3600, time.Now()); err != nil {
		t.Fatal(err)
	}
	_, refreshed, _, err := decodeSessionData(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || string(refreshed) != `{"x":2,"nan":NaN}` {
		t.Fatalf("refreshed=%q err=%v", refreshed, err)
	}
}

func TestSessionSaveCoercesUserIDThroughDjangoCharField(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		serialized string
		want       string
	}{
		{name: "float", serialized: `{"_auth_user_id":1.5}`, want: "1.5"},
		{name: "list", serialized: `{"_auth_user_id":[1,"x"]}`, want: `[1, 'x']`},
		{name: "object", serialized: `{"_auth_user_id":{"a":1}}`, want: `{'a': 1}`},
		{name: "contextual quote", serialized: `{"_auth_user_id":["a'b"]}`, want: `["a'b"]`},
		{name: "NUL escape", serialized: `{"_auth_user_id":["\u0000"]}`, want: `['\x00']`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			reader := fixtureReader(signPayloadForTest([]byte(testCase.serialized), false, fixtureCurrentSecret), fixturePassword, true)
			validator := configuredValidator(t, reader)
			if _, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 3600, time.Now()); err != nil {
				t.Fatal(err)
			}
			if reader.updatedUserID != testCase.want {
				t.Fatalf("user_id=%#v want=%q", reader.updatedUserID, testCase.want)
			}
		})
	}
}

func TestDjangoUUIDFieldAcceptsIntegerAndBooleanSessionIDs(t *testing.T) {
	const normalized = "00000000-0000-0000-0000-000000000001"
	for _, rawID := range []string{"1", "true"} {
		t.Run(rawID, func(t *testing.T) {
			serialized := []byte(`{"_auth_user_id":` + rawID + `,"_auth_user_backend":"` + fixtureBackend + `","_auth_user_hash":"` + fixtureCurrentHash + `"}`)
			reader := fixtureReader(signPayloadForTest(serialized, false, fixtureCurrentSecret), fixturePassword, true)
			reader.users = map[string]User{normalized: {Password: fixturePassword, Active: true, Timezone: "UTC"}}
			validator := configuredValidator(t, reader)
			principal, err := validator.Validate(context.Background(), fixtureSessionKey)
			if err != nil || principal.UserID != normalized {
				t.Fatalf("principal=%#v err=%v", principal, err)
			}
		})
	}
}

func TestFallbackHashRotationPreservesPythonJSONExtensions(t *testing.T) {
	serialized := []byte(`{"_auth_user_id":"` + fixtureUserID + `","_auth_user_backend":"` + fixtureBackend + `","_auth_user_hash":"` + fixtureFallbackHash + `","nan":NaN,"surrogate":"\ud800"}`)
	reader := fixtureReader(signPayloadForTest(serialized, false, fixtureCurrentSecret), fixturePassword, true)
	validator := configuredValidator(t, reader)
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil || !principal.AuthHashUsedFallback {
		t.Fatalf("fallback principal=%#v err=%v", principal, err)
	}
	refresh, err := validator.RefreshPrincipal(context.Background(), principal, 3600, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, refreshed, _, err := decodeSessionData(reader.sessions[refresh.SessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || !bytes.Contains(refreshed, []byte(`"nan":NaN`)) || !bytes.Contains(refreshed, []byte(`"surrogate":"\ud800"`)) || !bytes.Contains(refreshed, []byte(`"_auth_user_hash":"`+fixtureCurrentHash+`"`)) {
		t.Fatalf("refreshed session=%q err=%v", refreshed, err)
	}
}

func TestFallbackHashRotationWritesOldHashUntilResponseRefresh(t *testing.T) {
	reader := fixtureReader(fixtureFallbackSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	principal, prepared, err := validator.Inspect(context.Background(), fixtureSessionKey)
	if err != nil || !principal.AuthHashUsedFallback {
		t.Fatalf("fallback principal=%#v err=%v", principal, err)
	}
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	rotated, err := validator.RotateFallbackPrepared(context.Background(), prepared, 604800, now)
	if err != nil || rotated.SessionKey == fixtureSessionKey {
		t.Fatalf("rotated=%#v err=%v", rotated, err)
	}
	values, _, err := decode(reader.sessions[rotated.SessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || values["_auth_user_hash"] != fixtureFallbackHash {
		t.Fatalf("cycle_key values=%#v err=%v", values, err)
	}
	if _, refreshPreparedErr := validator.RefreshPrepared(context.Background(), rotated, 604800, now); refreshPreparedErr != nil {
		t.Fatal(refreshPreparedErr)
	}
	values, _, err = decode(reader.sessions[rotated.SessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || values["_auth_user_hash"] != fixtureCurrentHash {
		t.Fatalf("response refresh values=%#v err=%v", values, err)
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

func TestRefreshPrincipalCyclesFallbackAfterConcurrentOldKeyDeletion(t *testing.T) {
	reader := fixtureReader(fixtureFallbackSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil || !principal.AuthHashUsedFallback {
		t.Fatalf("fallback principal=%#v err=%v", principal, err)
	}
	delete(reader.sessions, fixtureSessionKey)
	refresh, err := validator.RefreshPrincipal(context.Background(), principal, 3600, time.Now())
	if err != nil || refresh.SessionKey == fixtureSessionKey {
		t.Fatalf("concurrent rotation refresh=%#v err=%v", refresh, err)
	}
	if refreshed, err := validator.Validate(context.Background(), refresh.SessionKey); err != nil || refreshed.UsedFallback {
		t.Fatalf("refreshed principal=%#v err=%v", refreshed, err)
	}
}

func TestRefreshPrincipalPersistsRequestLocalSnapshotAfterConcurrentUpdate(t *testing.T) {
	original := claims(fixtureUserID, fixtureBackend, fixtureCurrentHash)
	original["note"] = "request snapshot"
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, original, false), fixturePassword, true)
	validator := configuredValidator(t, reader)
	principal, err := validator.Validate(context.Background(), fixtureSessionKey)
	if err != nil {
		t.Fatal(err)
	}
	concurrent := claims(fixtureUserID, fixtureBackend, fixtureCurrentHash)
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

func TestRefreshReSignsAndExtendsDjangoSession(t *testing.T) {
	values := claims(fixtureUserID, fixtureBackend, fixtureCurrentHash)
	values["note"] = "Plane ✈"
	values["device_info"] = map[string]any{"browser": "test"}
	serialized, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	reader := fixtureReader(signPayloadForTest(djangoASCIIJSON(serialized), false, fixtureCurrentSecret), fixturePassword, true)
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
	values := claims(fixtureUserID, fixtureBackend, "authentication-hash-not-validated")
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
	if err != nil || secretIndex != 0 || decoded["note"] != "request snapshot" || decoded["_auth_user_hash"] != "authentication-hash-not-validated" {
		t.Fatalf("decoded=%#v secret=%d err=%v", decoded, secretIndex, err)
	}
}

func TestRefreshUnaccessedRewritesCorruptAndReplacesMissingSessions(t *testing.T) {
	reader := fixtureReader("corrupt", fixturePassword, true)
	validator := configuredValidator(t, reader)
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	corrupt, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 3600, now)
	if err != nil || corrupt.SessionKey != fixtureSessionKey {
		t.Fatalf("corrupt refresh=%#v err=%v", corrupt, err)
	}
	decoded, _, err := decode(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || len(decoded) != 0 {
		t.Fatalf("corrupt decoded=%#v err=%v", decoded, err)
	}

	delete(reader.sessions, fixtureSessionKey)
	replacement, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 3600, now)
	if err != nil || replacement.SessionKey == fixtureSessionKey || len(replacement.SessionKey) != 128 {
		t.Fatalf("missing refresh=%#v err=%v", replacement, err)
	}
	if len(reader.sessions) != 1 {
		t.Fatalf("missing refresh sessions=%#v", reader.sessions)
	}
	decoded, _, err = decode(reader.sessions[replacement.SessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || len(decoded) != 0 {
		t.Fatalf("replacement decoded=%#v err=%v", decoded, err)
	}
}

func TestRefreshAcceptsNonPositiveDefaultAges(t *testing.T) {
	for _, age := range []int64{0, -1} {
		t.Run(strconv.FormatInt(age, 10), func(t *testing.T) {
			reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
			validator := configuredValidator(t, reader)
			now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
			refresh, err := validator.Refresh(context.Background(), fixtureSessionKey, age, now)
			if err != nil || refresh.MaxAgeSeconds != float64(age) || refresh.BrowserClose || !refresh.ExpiresAt.Equal(now.Add(time.Duration(age)*time.Second)) {
				t.Fatalf("Refresh()=%#v err=%v", refresh, err)
			}
			if !reader.expiry.Equal(now.Add(time.Duration(age) * time.Second)) {
				t.Fatalf("database expiry=%v, want %v", reader.expiry, now.Add(time.Duration(age)*time.Second))
			}
		})
	}
}

func TestRefreshPreservesLargeDjangoSessionAge(t *testing.T) {
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

func TestRefreshRejectsAgeOutsidePythonDateTimeRange(t *testing.T) {
	reader := fixtureReader(fixtureCurrentSession, fixturePassword, true)
	validator := configuredValidator(t, reader)
	if _, err := validator.Refresh(context.Background(), fixtureSessionKey, 1_000_000_000_000, time.Now()); !errors.Is(err, ErrStore) {
		t.Fatalf("Refresh() error=%v", err)
	}
}

func TestSessionExpiryMatchesDjangoJSONCoercion(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 0, 0, 250_000_000, time.UTC)
	for _, testCase := range []struct {
		name       string
		expiry     any
		wantAge    float64
		wantExpiry time.Time
		wantCookie time.Time
		browser    bool
	}{
		{name: "true", expiry: true, wantAge: 1, wantExpiry: now.Add(time.Second), wantCookie: now.Add(time.Second)},
		{name: "false", expiry: false, wantAge: 3600, wantExpiry: now.Add(time.Hour), wantCookie: now.Add(time.Hour), browser: true},
		{name: "fraction", expiry: 1.5, wantAge: 1.5, wantExpiry: now.Add(1500 * time.Millisecond), wantCookie: now.Add(1500 * time.Millisecond)},
		{name: "fraction rounds into next second", expiry: 1.9999999, wantAge: 1.9999999, wantExpiry: now.Add(2 * time.Second), wantCookie: now.Add(1_999_999_900 * time.Nanosecond)},
		{name: "half microsecond rounds to even zero", expiry: 0.0000005, wantAge: 0.0000005, wantExpiry: now, wantCookie: now.Add(500 * time.Nanosecond)},
		{name: "one-and-half microseconds rounds to two", expiry: 0.0000015, wantAge: 0.0000015, wantExpiry: now.Add(2 * time.Microsecond), wantCookie: now.Add(1500 * time.Nanosecond)},
		{name: "negative fraction rounds into prior second", expiry: -1.9999999, wantAge: -1.9999999, wantExpiry: now.Add(-2 * time.Second), wantCookie: now.Add(-1_999_999_900 * time.Nanosecond)},
		{name: "ISO space separator", expiry: "2026-09-01 12:01:02+00:00", wantAge: 61, wantExpiry: time.Date(2026, time.September, 1, 12, 1, 2, 0, time.UTC), wantCookie: now.Add(61 * time.Second)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			values := map[string]any{"_session_expiry": testCase.expiry}
			reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, true)
			validator := configuredValidator(t, reader)
			refresh, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 3600, now)
			if err != nil || refresh.MaxAgeSeconds != testCase.wantAge || refresh.BrowserClose != testCase.browser ||
				!refresh.ExpiresAt.Equal(testCase.wantExpiry) || !refresh.CookieExpiresAt.Equal(testCase.wantCookie) {
				t.Fatalf("refresh=%#v err=%v", refresh, err)
			}
		})
	}

	for _, expiry := range []string{"NaN", "Infinity", "-Infinity"} {
		t.Run(expiry, func(t *testing.T) {
			serialized := []byte(`{"_session_expiry":` + expiry + `}`)
			reader := fixtureReader(signPayloadForTest(serialized, false, fixtureCurrentSecret), fixturePassword, true)
			validator := configuredValidator(t, reader)
			if _, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 3600, now); !errors.Is(err, ErrStore) {
				t.Fatalf("RefreshUnaccessed() error=%v", err)
			}
		})
	}

	surrogateSeparator := []byte(`{"_session_expiry":"2026-09-01\ud80012:01:02+00:00"}`)
	reader := fixtureReader(signPayloadForTest(surrogateSeparator, false, fixtureCurrentSecret), fixturePassword, true)
	validator := configuredValidator(t, reader)
	refresh, err := validator.RefreshUnaccessed(context.Background(), fixtureSessionKey, 3600, now)
	if err != nil || !refresh.ExpiresAt.Equal(time.Date(2026, time.September, 1, 12, 1, 2, 0, time.UTC)) ||
		!refresh.CookieExpiresAt.Equal(now.Add(61*time.Second)) {
		t.Fatalf("surrogate-separator refresh=%#v err=%v", refresh, err)
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
	if err != nil || secretIndex != 0 || values["_auth_user_hash"] != fixtureCurrentHash {
		t.Fatalf("rotated values=%#v secret=%d err=%v", values, secretIndex, err)
	}
}

func TestRefreshPreservesBrowserCloseExpiry(t *testing.T) {
	values := claims(fixtureUserID, fixtureBackend, fixtureCurrentHash)
	values["_session_expiry"] = 0
	reader := fixtureReader(encodeValuesForTest(t, fixtureCurrentSecret, values, false), fixturePassword, true)
	validator := configuredValidator(t, reader)
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	refresh, err := validator.Refresh(context.Background(), fixtureSessionKey, 86400, now)
	if err != nil || !refresh.BrowserClose || refresh.MaxAgeSeconds != 86400 || !reader.expiry.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("Refresh()=%#v database expiry=%v err=%v", refresh, reader.expiry, err)
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

func TestRefreshReplacesCorruptStoredDataWithEmptyDjangoSession(t *testing.T) {
	reader := fixtureReader("corrupt-session-data", fixturePassword, true)
	validator := configuredValidator(t, reader)
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	refresh, err := validator.Refresh(context.Background(), fixtureSessionKey, 3600, now)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	values, _, err := decode(reader.sessions[fixtureSessionKey].Data, []string{fixtureCurrentSecret})
	if err != nil || len(values) != 0 {
		t.Fatalf("refreshed corrupt values=%#v err=%v", values, err)
	}
	if reader.updatedUserID != nil || reader.updatedDeviceInfo != nil || !refresh.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("updated user=%#v device=%#v refresh=%#v", reader.updatedUserID, reader.updatedDeviceInfo, refresh)
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
		{name: "invalid JSON", data: signPayloadForTest([]byte("{"), false, fixtureCurrentSecret)},
		{name: "corrupt zlib", data: signPayloadForTest([]byte("not-zlib"), true, fixtureCurrentSecret)},
		{name: "JSON array", data: signPayloadForTest([]byte("[]"), false, fixtureCurrentSecret)},
		{name: "wrong field type", data: encodeValuesForTest(t, fixtureCurrentSecret, map[string]any{
			"_auth_user_id": 7, "_auth_user_backend": fixtureBackend, "_auth_user_hash": fixtureCurrentHash,
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

func TestWrongBackendAndUserClaimsAreRejected(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
		want   error
	}{
		{name: "wrong backend", values: claims(fixtureUserID, "other.Backend", fixtureCurrentHash), want: ErrBackend},
		{name: "unknown auth user", values: claims("22222222-2222-2222-2222-222222222222", fixtureBackend, fixtureCurrentHash), want: ErrUserNotFound},
		{name: "missing auth hash", values: claims(fixtureUserID, fixtureBackend, ""), want: ErrInvalidAuthHash},
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
		SecretKey: fixtureCurrentSecret, FallbackKeys: []string{fixtureFallbackSecret}, AllowedBackends: []string{fixtureBackend},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return validator
}

func claims(userID string, backend string, authHash string) map[string]any {
	return map[string]any{"_auth_user_id": userID, "_auth_user_backend": backend, "_auth_user_hash": authHash}
}

func encodeForTest(t *testing.T, secret string, authHash string, compressed bool) string {
	t.Helper()
	return encodeValuesForTest(t, secret, claims(fixtureUserID, fixtureBackend, authHash), compressed)
}

func encodeValuesForTest(t *testing.T, secret string, values map[string]any, compressed bool) string {
	t.Helper()
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return signPayloadForTest(data, compressed, secret)
}

func signPayloadForTest(data []byte, compressed bool, secret string) string {
	if compressed {
		var output bytes.Buffer
		writer := zlib.NewWriter(&output)
		_, _ = writer.Write(data)
		_ = writer.Close()
		data = output.Bytes()
	}
	payload := base64.RawURLEncoding.EncodeToString(data)
	if compressed {
		payload = "." + payload
	}
	return signForTest(payload+":1AbC", secret)
}

func signForTest(value string, secret string) string {
	signature := base64.RawURLEncoding.EncodeToString(djangoHMAC(SessionSigningSalt+"signer", value, secret))
	return value + ":" + signature
}

func resignWithTimestamp(value string, timestamp string, secret string) string {
	payload := strings.SplitN(value, ":", 2)[0]
	return signForTest(payload+":"+timestamp, secret)
}
