package userapitokens

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("api token not found")
var ErrValidation = errors.New("api token validation failed")

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type Repository struct{ database database }

func NewRepository(db database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("api token database is required")
	}
	return &Repository{database: db}, nil
}

const tokenJSON = `json_build_object('id', id, 'created_at', created_at, 'updated_at', updated_at, 'label', label, 'description', description, 'is_active', (expired_at IS NULL OR expired_at > NOW()), 'last_used', last_used, 'created_by', created_by_id, 'updated_by', updated_by_id, 'deleted_at', deleted_at, 'user', user_id, 'user_type', user_type, 'workspace', workspace_id, 'expired_at', expired_at, 'is_service', is_service, 'allowed_rate_limit', allowed_rate_limit)`
const tokenFullJSON = `json_build_object('id', id, 'created_at', created_at, 'updated_at', updated_at, 'label', label, 'description', description, 'is_active', (expired_at IS NULL OR expired_at > NOW()), 'last_used', last_used, 'token', token, 'created_by', created_by_id, 'updated_by', updated_by_id, 'deleted_at', deleted_at, 'user', user_id, 'user_type', user_type, 'workspace', workspace_id, 'expired_at', expired_at, 'is_service', is_service, 'allowed_rate_limit', allowed_rate_limit)`

func (r *Repository) List(ctx context.Context, user uuid.UUID) (json.RawMessage, error) {
	var b []byte
	err := r.database.QueryRow(ctx, `SELECT COALESCE(json_agg(x ORDER BY created_at DESC),'[]'::json) FROM (SELECT `+tokenJSON+` AS x, created_at FROM api_tokens WHERE user_id=$1 AND is_service=false AND deleted_at IS NULL) x`, user).Scan(&b)
	return b, err
}
func (r *Repository) Get(ctx context.Context, user, id uuid.UUID) (json.RawMessage, error) {
	var b []byte
	err := r.database.QueryRow(ctx, `SELECT `+tokenJSON+` FROM api_tokens WHERE id=$1 AND user_id=$2 AND is_service=false AND deleted_at IS NULL`, id, user).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
func (r *Repository) Create(ctx context.Context, user uuid.UUID, label, description, token string, expiredAt *string) (json.RawMessage, error) {
	var b []byte
	err := r.database.QueryRow(ctx, `INSERT INTO api_tokens (label,description,user_id,token,user_type,created_by_id,expired_at) VALUES ($1,$2,$3,$4,(SELECT CASE WHEN is_bot THEN 1 ELSE 0 END FROM users WHERE id=$3),$3,$5) RETURNING `+tokenFullJSON, label, description, user, token, expiredAt).Scan(&b)
	return b, err
}
func (r *Repository) Patch(ctx context.Context, user, id uuid.UUID, label, description *string) (json.RawMessage, error) {
	var b []byte
	err := r.database.QueryRow(ctx, `UPDATE api_tokens SET label=COALESCE($3,label), description=COALESCE($4,description), updated_by_id=$2, updated_at=NOW() WHERE id=$1 AND user_id=$2 AND is_service=false AND deleted_at IS NULL RETURNING `+tokenFullJSON, id, user, label, description).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return b, err
}
func (r *Repository) Delete(ctx context.Context, user, id uuid.UUID) error {
	tag, err := r.database.Exec(ctx, `UPDATE api_tokens SET deleted_at=NOW(), updated_by_id=$2, updated_at=NOW() WHERE id=$1 AND user_id=$2 AND is_service=false AND deleted_at IS NULL`, id, user)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
