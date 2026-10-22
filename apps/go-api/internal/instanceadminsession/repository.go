package instanceadminsession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("user not found")

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("instance admin session database is required")
	}
	return &Repository{database: database}, nil
}

func (r *Repository) IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var result bool
	// Keep this aligned with InstanceAdminPermission: Django checks the
	// highest-priority (newest) live instance and requires an admin role.
	err := r.database.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM instance_admins AS admins
    WHERE admins.user_id = $1
      AND admins.role >= 15
      AND admins.deleted_at IS NULL
      AND admins.instance_id = (
          SELECT id
          FROM instances
          WHERE deleted_at IS NULL
          ORDER BY created_at DESC
          LIMIT 1
      )
)`, userID).Scan(&result)
	if err != nil {
		return false, fmt.Errorf("check instance admin: %w", err)
	}
	return result, nil
}

func (r *Repository) HasAdminMembership(ctx context.Context, userID uuid.UUID) (bool, error) {
	var result bool
	err := r.database.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM instance_admins
    WHERE user_id = $1
      AND deleted_at IS NULL
)`, userID).Scan(&result)
	if err != nil {
		return false, fmt.Errorf("check instance admin membership: %w", err)
	}
	return result, nil
}

func (r *Repository) User(ctx context.Context, userID uuid.UUID) (json.RawMessage, error) {
	var data []byte
	err := r.database.QueryRow(ctx, `
SELECT json_build_object(
 'id', id, 'avatar', avatar, 'avatar_url', CASE
    WHEN avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/' || avatar_asset_id || '/'
    ELSE NULLIF(avatar, '')
 END, 'cover_image', cover_image,
 'date_joined', date_joined, 'display_name', display_name, 'email', email,
 'first_name', first_name, 'last_name', last_name, 'is_active', is_active,
 'is_bot', is_bot, 'is_email_verified', is_email_verified, 'user_timezone', user_timezone,
 'username', username, 'is_password_autoset', is_password_autoset)
FROM users WHERE id=$1`, userID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get instance admin user: %w", err)
	}
	return json.RawMessage(data), nil
}
