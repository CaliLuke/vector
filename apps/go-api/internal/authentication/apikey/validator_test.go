package apikey

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type rowFunc func(...any) error

func (row rowFunc) Scan(destinations ...any) error { return row(destinations...) }

type databaseStub struct {
	row      pgx.Row
	execErr  error
	execArgs []any
}

func (database *databaseStub) QueryRow(context.Context, string, ...any) pgx.Row { return database.row }
func (database *databaseStub) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	database.execArgs = args
	return pgconn.NewCommandTag("UPDATE 1"), database.execErr
}

func TestNewRequiresDatabase(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("New(nil) succeeded")
	}
}

func TestValidateEnforcesLookupAndUpdatesLastUsed(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*uuid.UUID) = userID
		*destinations[1].(*string) = "America/New_York"
		return nil
	})}
	validator, err := New(database)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	validator.now = func() time.Time { return now }
	principal, err := validator.Validate(context.Background(), "plane_api_token")
	if err != nil {
		t.Fatal(err)
	}
	if principal.UserID != userID || principal.Timezone != "America/New_York" {
		t.Fatalf("principal=%+v", principal)
	}
	if len(database.execArgs) != 2 || database.execArgs[0] != "plane_api_token" || database.execArgs[1] != now {
		t.Fatalf("update args=%v", database.execArgs)
	}
}

func TestValidateMapsMissingTokenToInvalid(t *testing.T) {
	validator, err := New(&databaseStub{row: rowFunc(func(...any) error { return pgx.ErrNoRows })})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validator.Validate(context.Background(), "missing"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("error=%v, want ErrInvalid", err)
	}
}

func TestValidateLeavesTimezoneActivationToAPIViewBoundary(t *testing.T) {
	database := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*uuid.UUID) = uuid.Nil
		*destinations[1].(*string) = "Not/A_Zone"
		return nil
	})}
	validator, err := New(database)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := validator.Validate(context.Background(), "token")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if principal.Timezone != "Not/A_Zone" {
		t.Fatalf("timezone=%q", principal.Timezone)
	}
	if len(database.execArgs) != 2 || database.execArgs[0] != "token" {
		t.Fatalf("last_used update args=%v", database.execArgs)
	}
}
func TestValidateCanReadFromReplicaAndWriteToPrimary(t *testing.T) {
	userID := uuid.New()
	reader := &databaseStub{row: rowFunc(func(destinations ...any) error {
		*destinations[0].(*uuid.UUID) = userID
		*destinations[1].(*string) = "UTC"
		return nil
	})}
	writer := &databaseStub{}
	validator, err := NewWithReaderWriter(reader, writer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validator.Validate(context.Background(), "replica-token"); err != nil {
		t.Fatal(err)
	}
	if reader.execArgs != nil {
		t.Fatalf("replica received last_used write: %v", reader.execArgs)
	}
	if len(writer.execArgs) != 2 || writer.execArgs[0] != "replica-token" {
		t.Fatalf("primary last_used write args = %v", writer.execArgs)
	}
}
