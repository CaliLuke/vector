package workspaceslugcheck

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("workspace slug repository database is required")
	}
	return &Repository{database: database}, nil
}

// Exists applies Django's default Workspace manager semantics: deleted rows
// are invisible, while the equality predicate remains case-sensitive.
func (repository *Repository) Exists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := repository.database.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM workspaces
  WHERE slug = $1 AND deleted_at IS NULL
)
`, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check workspace slug: %w", err)
	}
	return exists, nil
}
