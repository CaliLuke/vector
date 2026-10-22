package userinstanceadmin

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
		return nil, fmt.Errorf("user instance admin repository database is required")
	}
	return &Repository{database: database}, nil
}

// IsInstanceAdmin applies UserEndpoint.retrieve_instance_admin's exact
// membership semantics: use the latest non-deleted instance and only test
// for a non-deleted membership row. Role and verification are not checked.
func (repository *Repository) IsInstanceAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isAdmin bool
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
    AND admins.deleted_at IS NULL
)
`, userID).Scan(&isAdmin)
	if err != nil {
		return false, fmt.Errorf("check instance administrator membership: %w", err)
	}
	return isAdmin, nil
}
