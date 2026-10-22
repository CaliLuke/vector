package apikey

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrInvalid = errors.New("API token is not valid")

type Principal struct {
	UserID   uuid.UUID
	Timezone string
}

type Reader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Writer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type database interface {
	Reader
	Writer
}

type Validator struct {
	reader Reader
	writer Writer
	now    func() time.Time
}

func New(database database) (*Validator, error) {
	return NewWithReaderWriter(database, database)
}

func NewWithReaderWriter(reader Reader, writer Writer) (*Validator, error) {
	if reader == nil {
		return nil, fmt.Errorf("API key reader is required")
	}
	if writer == nil {
		return nil, fmt.Errorf("API key writer is required")
	}
	return &Validator{reader: reader, writer: writer, now: time.Now}, nil
}

// Validate reproduces APIKeyAuthentication: the token and owner must be live,
// active, and unexpired. The optional token workspace is intentionally not a
// scope boundary. A successful authentication updates only last_used.
func (validator *Validator) Validate(ctx context.Context, token string) (Principal, error) {
	if token == "" {
		return Principal{}, ErrInvalid
	}
	var principal Principal
	err := validator.reader.QueryRow(ctx, `
SELECT t.user_id, COALESCE(u.user_timezone, 'UTC')
FROM api_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.token = $1
  AND t.deleted_at IS NULL
  AND t.is_active = true
  AND (t.expired_at IS NULL OR t.expired_at > $2)
  AND u.is_active = true
LIMIT 1
`, token, validator.now()).Scan(&principal.UserID, &principal.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrInvalid
	}
	if err != nil {
		return Principal{}, fmt.Errorf("read API token: %w", err)
	}
	if _, err := validator.writer.Exec(ctx, `UPDATE api_tokens SET last_used = $2 WHERE token = $1`, token, validator.now()); err != nil {
		return Principal{}, fmt.Errorf("update API token last_used: %w", err)
	}
	return principal, nil
}
