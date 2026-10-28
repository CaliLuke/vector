//go:build integration

package djangosession

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/database"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestPostgreSQLReaderUsesDatabaseExpiryAndDoesNotMutate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.Start(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE users (
    id uuid PRIMARY KEY,
    password varchar(128) NOT NULL,
    is_active boolean NOT NULL,
    user_timezone varchar(255) NOT NULL DEFAULT 'UTC'
);
CREATE TABLE sessions (
    session_key varchar(128) PRIMARY KEY,
    session_data text NOT NULL,
    expire_date timestamptz NOT NULL,
    user_id varchar(50),
    device_info jsonb
);
`)
		return err
	})
	if err != nil {
		t.Fatalf("start PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Errorf("close PostgreSQL: %v", closeErr)
		}
	})
	connection, err := pgx.Connect(ctx, pair.SourceURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
	if registerUUIDTypesErr := database.RegisterUUIDTypes(connection.TypeMap()); registerUUIDTypesErr != nil {
		t.Fatalf("register UUID types: %v", registerUUIDTypesErr)
	}
	if _, execErr := connection.Exec(ctx, "INSERT INTO users (id, password, is_active) VALUES ($1, $2, true)", fixtureUserID, fixturePassword); execErr != nil {
		t.Fatalf("seed user: %v", execErr)
	}
	if _, execErr := connection.Exec(ctx, `
INSERT INTO sessions (session_key, session_data, expire_date, user_id)
VALUES ($1, $2, CURRENT_TIMESTAMP + interval '5 minutes', $3)
`, fixtureSessionKey, fixtureCurrentSession, fixtureUserID); execErr != nil {
		t.Fatalf("seed session: %v", execErr)
	}
	validator := configuredValidator(t, NewPostgreSQLReader(connection))
	if _, readUserErr := NewPostgreSQLReader(connection).ReadUser(ctx, "not-a-uuid"); !errors.Is(readUserErr, ErrInvalidUserID) {
		t.Fatalf("malformed ReadUser() error = %v, want ErrInvalidUserID", readUserErr)
	}
	if _, validationErr := validator.Validate(ctx, fixtureSessionKey); validationErr != nil {
		t.Fatalf("Validate() error = %v", validationErr)
	}
	if _, execErr := connection.Exec(ctx, "UPDATE sessions SET expire_date = $2 WHERE session_key = $1", fixtureSessionKey, time.Now().Add(-time.Minute)); execErr != nil {
		t.Fatalf("expire session between validation and refresh: %v", execErr)
	}
	// Refresh and expiry validation must use the same clock as PostgreSQL.
	var refreshNow time.Time
	if scanErr := connection.QueryRow(ctx, "SELECT CURRENT_TIMESTAMP").Scan(&refreshNow); scanErr != nil {
		t.Fatalf("read database time: %v", scanErr)
	}
	refresh, err := validator.Refresh(ctx, fixtureSessionKey, 86400, refreshNow)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	var refreshedData string
	var refreshedExpiry time.Time
	if scanErr := connection.QueryRow(ctx, "SELECT session_data, expire_date FROM sessions WHERE session_key = $1", fixtureSessionKey).Scan(&refreshedData, &refreshedExpiry); scanErr != nil {
		t.Fatalf("read refreshed session: %v", scanErr)
	}
	if refreshedData == fixtureCurrentSession || !refreshedExpiry.Equal(refresh.ExpiresAt) {
		t.Fatalf("refresh did not persist: data changed=%v expiry=%v want=%v", refreshedData != fixtureCurrentSession, refreshedExpiry, refresh.ExpiresAt)
	}
	if _, validationErr := validator.Validate(ctx, fixtureSessionKey); validationErr != nil {
		t.Fatalf("refreshed Validate() error = %v", validationErr)
	}
	fallbackKey := "fallback-session-key"
	if _, execErr := connection.Exec(ctx, `
INSERT INTO sessions (session_key, session_data, expire_date, user_id)
VALUES ($1, $2, CURRENT_TIMESTAMP + interval '5 minutes', $3)
`, fallbackKey, fixtureFallbackSession, fixtureUserID); execErr != nil {
		t.Fatalf("seed fallback session: %v", execErr)
	}
	rotated, err := validator.Refresh(ctx, fallbackKey, 86400, refreshNow)
	if err != nil {
		t.Fatalf("rotate fallback session: %v", err)
	}
	if rotated.SessionKey == fallbackKey || len(rotated.SessionKey) != 128 {
		t.Fatalf("rotated session=%#v", rotated)
	}
	var oldCount int
	if scanErr := connection.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE session_key = $1", fallbackKey).Scan(&oldCount); scanErr != nil || oldCount != 0 {
		t.Fatalf("old fallback session count=%d err=%v", oldCount, scanErr)
	}
	if principal, validationErr := validator.Validate(ctx, rotated.SessionKey); validationErr != nil || principal.UsedFallback {
		t.Fatalf("rotated fallback principal=%#v err=%v", principal, validationErr)
	}
	concurrentKey := "concurrent-cycle-new-key"
	if cycleSessionErr := NewPostgreSQLReader(connection).CycleSession(
		ctx, "already-deleted-old-key", concurrentKey, fixtureCurrentSession,
		time.Now().Add(time.Hour), fixtureUserID, nil,
	); cycleSessionErr != nil {
		t.Fatalf("cycle after concurrent old-key deletion: %v", cycleSessionErr)
	}
	if _, validationErr := validator.Validate(ctx, concurrentKey); validationErr != nil {
		t.Fatalf("new session missing after concurrent old-key deletion: %v", validationErr)
	}
	if _, execErr := connection.Exec(ctx, `
INSERT INTO sessions (session_key, session_data, expire_date, user_id)
VALUES ('null-user-session', $1, CURRENT_TIMESTAMP + interval '5 minutes', NULL)
`, fixtureCurrentSession); execErr != nil {
		t.Fatalf("seed NULL-user session: %v", execErr)
	}
	if _, validationErr := validator.Validate(ctx, "null-user-session"); validationErr != nil {
		t.Fatalf("NULL denormalized user must not affect Django validation: %v", validationErr)
	}
	for _, invalidDevice := range []struct {
		key  string
		data string
	}{
		{key: "invalid-device-nan", data: `{"device_info":{"value":NaN}}`},
		{key: "invalid-device-surrogate", data: `{"device_info":{"value":"\ud800"}}`},
		{key: "invalid-device-surrogate-key", data: `{"device_info":{"\ud800":"value"}}`},
		{key: "invalid-user-surrogate", data: `{"_auth_user_id":"\ud800"}`},
	} {
		invalidDeviceData := signPayloadForTest([]byte(invalidDevice.data), false, fixtureCurrentSecret)
		if _, execErr := connection.Exec(ctx, `
INSERT INTO sessions (session_key, session_data, expire_date)
VALUES ($1, $2, CURRENT_TIMESTAMP + interval '5 minutes')
`, invalidDevice.key, invalidDeviceData); execErr != nil {
			t.Fatalf("seed invalid device session: %v", execErr)
		}
		if _, refreshUnaccessedErr := validator.RefreshUnaccessed(ctx, invalidDevice.key, 3600, time.Now()); !errors.Is(refreshUnaccessedErr, ErrStore) || !errors.Is(refreshUnaccessedErr, ErrSessionUpdate) {
			t.Fatalf("invalid JSONField refresh for %s error=%v", invalidDevice.key, refreshUnaccessedErr)
		}
	}
	var sessionData string
	var rowCount int
	if scanErr := connection.QueryRow(ctx, "SELECT session_data, count(*) OVER () FROM sessions WHERE session_key = $1", fixtureSessionKey).Scan(&sessionData, &rowCount); scanErr != nil {
		t.Fatalf("read session after validation: %v", scanErr)
	}
	if sessionData != refreshedData || rowCount != 1 {
		t.Fatalf("validation mutated refreshed session: data match = %t, count = %d", sessionData == refreshedData, rowCount)
	}
	if _, execErr := connection.Exec(ctx, "UPDATE sessions SET expire_date = $2 WHERE session_key = $1", fixtureSessionKey, time.Now().Add(-time.Minute)); execErr != nil {
		t.Fatalf("expire session: %v", execErr)
	}
	if _, validationErr := validator.Validate(ctx, fixtureSessionKey); !errors.Is(validationErr, ErrSessionNotFound) {
		t.Fatalf("expired Validate() error = %v, want ErrSessionNotFound", validationErr)
	}
	replacement, err := validator.RefreshUnaccessed(ctx, fixtureSessionKey, 3600, time.Now())
	if err != nil || replacement.SessionKey == fixtureSessionKey || len(replacement.SessionKey) != 128 {
		t.Fatalf("refresh expired unaccessed session=%#v err=%v", replacement, err)
	}
	var expiredCount int
	if err := connection.QueryRow(ctx, "SELECT count(*) FROM sessions WHERE session_key = $1", fixtureSessionKey).Scan(&expiredCount); err != nil || expiredCount != 1 {
		t.Fatalf("expired source session count=%d err=%v", expiredCount, err)
	}
	if _, err := validator.Validate(ctx, replacement.SessionKey); !errors.Is(err, ErrUserMismatch) {
		t.Fatalf("replacement Validate() error=%v, want ErrUserMismatch", err)
	}
	if _, err := connection.Exec(ctx, "DELETE FROM sessions WHERE session_key = $1", fixtureSessionKey); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := validator.Validate(ctx, fixtureSessionKey); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("deleted Validate() error = %v, want ErrSessionNotFound", err)
	}
}
