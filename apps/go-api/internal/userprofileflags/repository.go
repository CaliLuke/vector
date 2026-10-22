package userprofileflags

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrProfileNotFound = errors.New("profile not found")

type database interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ database database }

func NewRepository(database database) (*Repository, error) {
	if database == nil {
		return nil, fmt.Errorf("user profile flags repository database is required")
	}
	return &Repository{database: database}, nil
}

func (repository *Repository) UpdateOnboarded(ctx context.Context, userID uuid.UUID, value bool) error {
	return repository.update(ctx, userID, "is_onboarded", value)
}

func (repository *Repository) UpdateTourCompleted(ctx context.Context, userID uuid.UUID, value bool) error {
	return repository.update(ctx, userID, "is_tour_completed", value)
}

func (repository *Repository) update(ctx context.Context, userID uuid.UUID, field string, value bool) error {
	result, err := repository.database.Exec(ctx, fmt.Sprintf("UPDATE profiles SET %s = $2, updated_at = NOW() WHERE user_id = $1", field), userID, value)
	if err != nil {
		return fmt.Errorf("update profile %s: %w", field, err)
	}
	if result.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}
