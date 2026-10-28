package userprofileflags

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
)

type databaseStub struct {
	query string
	args  []any
	tag   pgconn.CommandTag
	err   error
}

func (stub *databaseStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	stub.query, stub.args = query, args
	return stub.tag, stub.err
}

func TestRepositoryUpdatesFlags(t *testing.T) {
	stub := &databaseStub{tag: pgconn.NewCommandTag("UPDATE 1")}
	repository, err := NewRepository(stub)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if err := repository.UpdateOnboarded(context.Background(), userID, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stub.query, "is_onboarded = $2") || stub.args[0] != userID || stub.args[1] != true {
		t.Fatalf("query = %q, args = %#v", stub.query, stub.args)
	}
	if err := repository.UpdateTourCompleted(context.Background(), userID, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stub.query, "is_tour_completed = $2") || stub.args[1] != false {
		t.Fatalf("query = %q, args = %#v", stub.query, stub.args)
	}
}

func TestRepositoryErrors(t *testing.T) {
	repository, err := NewRepository(&databaseStub{err: errors.New("boom")})
	if err != nil {
		t.Fatal(err)
	}
	if updateOnboardedErr := repository.UpdateOnboarded(context.Background(), uuid.Nil(), true); updateOnboardedErr == nil {
		t.Fatal("update returned nil error")
	}
	zeroRows := &databaseStub{tag: pgconn.NewCommandTag("UPDATE 0")}
	repository, err = NewRepository(zeroRows)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.UpdateTourCompleted(context.Background(), uuid.Nil(), true); err == nil {
		t.Fatal("missing profile returned nil error")
	} else if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("missing profile error = %v, want ErrProfileNotFound", err)
	}
}
