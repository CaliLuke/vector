package browsersession

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type QueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type sessionUpdater interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// PostgreSQLReader loads browser sessions and current account credentials.
// The connection must use database.ConfigurePool for standard-library UUIDs.
type PostgreSQLReader struct {
	database QueryRower
}

func NewPostgreSQLReader(database QueryRower) *PostgreSQLReader {
	return &PostgreSQLReader{database: database}
}

func (reader *PostgreSQLReader) ReadSession(ctx context.Context, sessionKey string) (Session, error) {
	var session Session
	err := reader.database.QueryRow(ctx, `
SELECT session_data
FROM sessions
WHERE session_key = $1
  AND expire_date > $2
`, sessionKey, time.Now()).Scan(&session.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("read browser session: %w", err)
	}
	return session, err
}

func (reader *PostgreSQLReader) ReadUser(ctx context.Context, userID string) (User, error) {
	identifier, err := uuid.Parse(userID)
	if err != nil {
		return User{}, ErrInvalidUserID
	}
	var user User
	err = reader.database.QueryRow(ctx, `
SELECT password, is_active, user_timezone
FROM users
WHERE id = $1
`, identifier).Scan(&user.Password, &user.Active, &user.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("read account: %w", err)
	}
	return user, err
}

func (reader *PostgreSQLReader) UpdateSession(ctx context.Context, sessionKey string, sessionData string, expireAt time.Time, userID any, deviceInfo any) error {
	database, ok := reader.database.(sessionUpdater)
	if !ok {
		return errors.New("session database does not support updates")
	}
	tag, err := database.Exec(ctx, `
UPDATE sessions
SET session_data = $2, expire_date = $3, user_id = $4, device_info = $5
WHERE session_key = $1
`, sessionKey, sessionData, expireAt, userID, deviceInfo)
	if err != nil {
		return fmt.Errorf("update browser session: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrSessionNotFound
	}
	return nil
}

func (reader *PostgreSQLReader) CreateSession(ctx context.Context, sessionKey string, sessionData string, expireAt time.Time, userID any, deviceInfo any) error {
	database, ok := reader.database.(sessionUpdater)
	if !ok {
		return errors.New("session database does not support creation")
	}
	_, err := database.Exec(ctx, `
INSERT INTO sessions (session_key, session_data, expire_date, user_id, device_info)
VALUES ($1, $2, $3, $4, $5)
`, sessionKey, sessionData, expireAt, userID, deviceInfo)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return ErrKeyCollision
		}
		return fmt.Errorf("create browser session: %w", err)
	}
	return nil
}

func (reader *PostgreSQLReader) CycleSession(
	ctx context.Context,
	oldKey string,
	newKey string,
	sessionData string,
	expireAt time.Time,
	userID any,
	deviceInfo any,
) error {
	database, ok := reader.database.(sessionUpdater)
	if !ok {
		return errors.New("session database does not support key rotation")
	}
	if oldKey == newKey {
		return ErrKeyCollision
	}
	tag, err := database.Exec(ctx, `
WITH previous AS (
    DELETE FROM sessions WHERE session_key = $1 RETURNING session_key
)
INSERT INTO sessions (session_key, session_data, expire_date, user_id, device_info)
SELECT $2, $3, $4, $5, $6 FROM previous
`, oldKey, newKey, sessionData, expireAt, userID, deviceInfo)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return ErrKeyCollision
		}
		return fmt.Errorf("cycle browser session key: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrSessionNotFound
	}
	return nil
}

func (reader *PostgreSQLReader) DeleteSession(ctx context.Context, sessionKey string) error {
	database, ok := reader.database.(sessionUpdater)
	if !ok {
		return errors.New("session database does not support deletion")
	}
	if _, err := database.Exec(ctx, "DELETE FROM sessions WHERE session_key = $1", sessionKey); err != nil {
		return fmt.Errorf("delete browser session: %w", err)
	}
	return nil
}
