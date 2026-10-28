//go:build integration

package apikey

import (
	"context"
	"errors"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestValidatorEnforcesDjangoTokenStateAndWritesOnlyLastUsed(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, func(ctx context.Context, databaseURL string) error {
		connection, err := pgx.Connect(ctx, databaseURL)
		if err != nil {
			return err
		}
		defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
		_, err = connection.Exec(ctx, `
CREATE TABLE users (id uuid PRIMARY KEY, user_timezone text NOT NULL, is_active boolean NOT NULL);
CREATE TABLE api_tokens (
 token text PRIMARY KEY, user_id uuid NOT NULL, deleted_at timestamptz,
 is_active boolean NOT NULL, expired_at timestamptz, last_used timestamptz
);
INSERT INTO users VALUES
 ('00000000-0000-4000-8000-000000000001','Asia/Kolkata',true),
 ('00000000-0000-4000-8000-000000000002','UTC',false);
INSERT INTO api_tokens VALUES
 ('valid','00000000-0000-4000-8000-000000000001',NULL,true,NULL,NULL),
 ('expired','00000000-0000-4000-8000-000000000001',NULL,true,'2026-01-01',NULL),
 ('inactive-owner','00000000-0000-4000-8000-000000000002',NULL,true,NULL,NULL);
`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, pair)
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.CloseContext(t, connection, context.WithoutCancel(ctx))
	validator, err := New(connection)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	validator.now = func() time.Time { return now }
	principal, err := validator.Validate(ctx, "valid")
	if err != nil || principal.Timezone != "Asia/Kolkata" {
		t.Fatalf("principal=%+v error=%v", principal, err)
	}
	var lastUsed time.Time
	if err := connection.QueryRow(ctx, "SELECT last_used FROM api_tokens WHERE token='valid'").Scan(&lastUsed); err != nil || !lastUsed.Equal(now) {
		t.Fatalf("last_used=%s error=%v", lastUsed, err)
	}
	for _, token := range []string{"expired", "inactive-owner", "missing"} {
		if _, err := validator.Validate(ctx, token); !errors.Is(err, ErrInvalid) {
			t.Errorf("token %q error=%v, want ErrInvalid", token, err)
		}
	}
}
