package workspacewebhooks

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type failingWriteDatabase struct {
	cause   error
	queries int
}

func (db *failingWriteDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	db.queries++
	return db
}
func (db *failingWriteDatabase) Scan(...any) error { return db.cause }
func (*failingWriteDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected exec")
}

func TestWebhookIntegrityFailurePreservesPostgresCause(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			cause := &pgconn.PgError{Code: "23505", ConstraintName: "other_unique_constraint", Message: "private write diagnostic"}
			db := &failingWriteDatabase{cause: fmt.Errorf("write: %w", cause)}
			repository, err := NewRepository(db)
			require.NoError(t, err)
			if operation == "create" {
				_, err = repository.Create(t.Context(), CreateParams{})
			} else {
				_, err = repository.Update(t.Context(), PatchParams{})
			}
			require.ErrorIs(t, err, ErrIntegrity)
			require.ErrorIs(t, err, cause)
			var typed *pgconn.PgError
			require.ErrorAs(t, err, &typed)
			require.Same(t, cause, typed)
			require.Equal(t, 1, db.queries)
		})
	}
}

func TestWebhookWriteErrorClassificationRemainsOperationSpecific(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cause      error
		createWant error
		updateWant error
	}{
		{"duplicate constraint", &pgconn.PgError{Code: "23505", ConstraintName: "webhook_url_unique_url_when_deleted_at_null"}, ErrDuplicateURL, ErrIntegrity},
		{"duplicate message", &pgconn.PgError{Code: "23505", Message: "URL already exists"}, ErrDuplicateURL, ErrIntegrity},
		{"missing row", pgx.ErrNoRows, pgx.ErrNoRows, ErrNotFound},
		{"connection failure", &pgconn.PgError{Code: "08006"}, nil, nil},
		{"cancelled", context.Canceled, context.Canceled, context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := &failingWriteDatabase{cause: fmt.Errorf("write: %w", tc.cause)}
			repository, err := NewRepository(db)
			require.NoError(t, err)
			_, createErr := repository.Create(t.Context(), CreateParams{})
			_, updateErr := repository.Update(t.Context(), PatchParams{})
			createWant, updateWant := tc.createWant, tc.updateWant
			if createWant == nil {
				createWant = tc.cause
			}
			if updateWant == nil {
				updateWant = tc.cause
			}
			require.ErrorIs(t, createErr, createWant)
			require.ErrorIs(t, updateErr, updateWant)
			require.Equal(t, 2, db.queries)
		})
	}
}
