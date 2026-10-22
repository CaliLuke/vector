package useraccounts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("user account not found")

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("user accounts database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) List(ctx context.Context, userID uuid.UUID) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, `
SELECT COALESCE(json_agg(account ORDER BY created_at DESC), '[]'::json)
FROM (
  SELECT json_build_object(
    'id', id, 'created_at', created_at, 'updated_at', updated_at, 'user', user_id,
    'provider_account_id', provider_account_id, 'provider', provider,
    'access_token', access_token, 'access_token_expired_at', access_token_expired_at,
    'refresh_token', refresh_token, 'refresh_token_expired_at', refresh_token_expired_at,
    'last_connected_at', last_connected_at, 'id_token', id_token, 'metadata', metadata
  ) AS account, created_at
  FROM accounts
  WHERE user_id = $1
) AS ordered_accounts
`, userID).Scan(&data)
	if err != nil {
		return nil, fmt.Errorf("list user accounts: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Get(ctx context.Context, userID, accountID uuid.UUID) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, accountJSONQuery+` WHERE id = $1 AND user_id = $2`, accountID, userID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user account: %w", err)
	}
	return json.RawMessage(data), nil
}

func (repository *Repository) Delete(ctx context.Context, userID, accountID uuid.UUID) error {
	result, err := repository.database.Exec(ctx, `DELETE FROM accounts WHERE id = $1 AND user_id = $2`, accountID, userID)
	if err != nil {
		return fmt.Errorf("delete user account: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const accountJSONQuery = `
SELECT json_build_object(
  'id', id, 'created_at', created_at, 'updated_at', updated_at, 'user', user_id,
  'provider_account_id', provider_account_id, 'provider', provider,
  'access_token', access_token, 'access_token_expired_at', access_token_expired_at,
  'refresh_token', refresh_token, 'refresh_token_expired_at', refresh_token_expired_at,
  'last_connected_at', last_connected_at, 'id_token', id_token, 'metadata', metadata
)
FROM accounts`
