package instanceworkspaceslugcheck

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("instance workspace slug repository database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) IsInstanceAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var authorized bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM instance_admins AS admins
  JOIN (
    SELECT id
    FROM instances
    WHERE deleted_at IS NULL
    ORDER BY created_at DESC
    LIMIT 1
  ) AS instance ON instance.id = admins.instance_id
  WHERE admins.user_id = $1
    AND admins.role >= 15
    AND admins.deleted_at IS NULL
)
`, userID).Scan(&authorized)
	if err != nil {
		return false, fmt.Errorf("check instance administrator: %w", err)
	}
	return authorized, nil
}

// Exists applies Django's case-insensitive workspace lookup and default
// manager visibility semantics.
func (repository *Repository) Exists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM workspaces
  WHERE UPPER(slug::text) = UPPER($1) AND deleted_at IS NULL
)
`, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check workspace slug: %w", err)
	}
	return exists, nil
}
