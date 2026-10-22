package projectpublication

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type intakeUUIDRow struct {
	exists bool
	err    error
}

func (row intakeUUIDRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	*dest[0].(*bool) = row.exists
	return nil
}

type intakeUUIDDB struct {
	queries int
	execs   int
	args    [][]any
}

func (db *intakeUUIDDB) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	db.queries++
	db.args = append(db.args, append([]any(nil), args...))
	if db.queries == 1 {
		return intakeUUIDRow{exists: true}
	}
	return intakeUUIDRow{err: pgx.ErrNoRows}
}

func (db *intakeUUIDDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	db.execs++
	return pgconn.CommandTag{}, nil
}

func TestCreateRejectsMalformedIntakeUUIDBeforeAnyDatabaseEffect(t *testing.T) {
	db := &intakeUUIDDB{}
	repository := &Repository{db: db}
	_, err := repository.Create(context.Background(), "user", "demo", "project", CreateInput{
		Intake: "x123e4567-e89b-12d3-a456-426614174000y",
		Views:  map[string]any{"list": true},
	})
	if !errors.Is(err, ErrInvalidIntake) {
		t.Fatalf("malformed intake UUID error = %v, want ErrInvalidIntake", err)
	}
	if db.queries != 0 || db.execs != 0 {
		t.Fatalf("malformed intake UUID reached database: queries=%d execs=%d", db.queries, db.execs)
	}
}

func TestIntakeIDAcceptsDjangoUUIDFormsAndCanonicalizes(t *testing.T) {
	for _, value := range []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"{123e4567-e89b-12d3-a456-426614174000}",
		"urn:uuid:123e4567-e89b-12d3-a456-426614174000",
	} {
		db := &intakeUUIDDB{}
		repository := &Repository{db: db}
		got, err := repository.intakeID(context.Background(), "demo", "project", value)
		if err != nil {
			t.Fatalf("Django UUID form %q rejected: %v", value, err)
		}
		if got != "123e4567-e89b-12d3-a456-426614174000" {
			t.Fatalf("Django UUID form %q normalized to %#v", value, got)
		}
		if db.queries != 1 || len(db.args) != 1 || db.args[0][0] != "123e4567-e89b-12d3-a456-426614174000" {
			t.Fatalf("Django UUID form %q did not use canonical scoped lookup: %#v", value, db)
		}
	}
}
