//go:build integration

package mutationreceipts_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	receipts "github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var errDenied = errors.New("test domain permission denied")

type fixture struct {
	db      *pgxpool.Pool
	url     string
	command receipts.Command
}

func receiptDatabase(t *testing.T) fixture {
	t.Helper()
	schema, err := os.ReadFile("../bootstrap/schema_migrations/000016_mutation_receipts.up.sql")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	pair, err := postgres.StartUncloned(ctx, func(ctx context.Context, connection string) error {
		db, connectErr := pgx.Connect(ctx, connection)
		if connectErr != nil {
			return connectErr
		}
		defer func() { _ = db.Close(context.WithoutCancel(ctx)) }()
		_, seedErr := db.Exec(ctx, string(schema)+`
CREATE TABLE receipt_test_permission (allowed boolean NOT NULL);
INSERT INTO receipt_test_permission VALUES(true);
CREATE TABLE receipt_test_changes (id bigserial PRIMARY KEY, value text NOT NULL);
CREATE TABLE receipt_test_outbox (entity_id bigint NOT NULL, event text NOT NULL);`)
		return seedErr
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	db, err := pgxpool.New(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(db.Close)
	request := uuid.New()
	return fixture{db: db, url: pair.SeedURL(), command: receipts.Command{
		ActorID: uuid.New(), WorkspaceID: uuid.New(), Tool: "project", Action: "create", RequestID: &request,
		NormalizedInput: json.RawMessage(`{"name":"First","nested":{"b":2,"a":1}}`),
	}}
}

func authorize(ctx context.Context, tx pgx.Tx) error {
	var allowed bool
	if err := tx.QueryRow(ctx, `SELECT allowed FROM receipt_test_permission FOR SHARE`).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return errDenied
	}
	return nil
}

func operation() receipts.Operation {
	return receipts.Operation{Authorize: authorize, Apply: func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		var id int64
		if err := tx.QueryRow(ctx, `INSERT INTO receipt_test_changes(value) VALUES('created') RETURNING id`).Scan(&id); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO receipt_test_outbox VALUES($1,'created')`, id); err != nil {
			return nil, err
		}
		return json.RawMessage(fmt.Sprintf(`{"id":%d,"revision":1}`, id)), nil
	}}
}

func assertCounts(t *testing.T, f fixture, changes, effects, saved int) {
	t.Helper()
	var actualChanges, actualEffects, actualReceipts int
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM receipt_test_changes),(SELECT count(*) FROM receipt_test_outbox),(SELECT count(*) FROM mutation_receipts)`).Scan(&actualChanges, &actualEffects, &actualReceipts))
	require.Equal(t, changes, actualChanges, "domain changes")
	require.Equal(t, effects, actualEffects, "durable effects")
	require.Equal(t, saved, actualReceipts, "receipts")
}

func TestReceiptReplaysCanonicalInputBeforeRevisionValidation(t *testing.T) {
	f := receiptDatabase(t)
	first, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	require.False(t, first.Replayed)
	require.NotNil(t, first.ReplayExpiresAt)
	require.Greater(t, time.Until(*first.ReplayExpiresAt), 24*time.Hour)
	f.command.NormalizedInput = json.RawMessage(" {\"nested\": {\"a\":1, \"b\":2}, \"name\": \"First\"} ")
	op := operation()
	op.Apply = func(context.Context, pgx.Tx) (json.RawMessage, error) {
		return nil, errors.New("stale revision must not be checked on replay")
	}
	replay, err := receipts.Execute(t.Context(), f.db, f.command, op)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, first.Value, replay.Value)
	require.Equal(t, first.ReplayExpiresAt, replay.ReplayExpiresAt)
	assertCounts(t, f, 1, 1, 1)
}

func TestReceiptInputConflictDoesNotRoundLargeIntegers(t *testing.T) {
	f := receiptDatabase(t)
	f.command.NormalizedInput = json.RawMessage(`{"value":9007199254740992}`)
	_, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	f.command.NormalizedInput = json.RawMessage(`{"value":9007199254740993}`)
	result, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.ErrorIs(t, err, receipts.ErrConflict)
	require.Empty(t, result)
	assertCounts(t, f, 1, 1, 1)
}

func TestReceiptIdentityContainsEveryScopeDimension(t *testing.T) {
	f := receiptDatabase(t)
	cases := []struct {
		name   string
		change func(*receipts.Command)
	}{
		{"original", func(*receipts.Command) {}},
		{"actor", func(c *receipts.Command) { c.ActorID = uuid.New() }},
		{"workspace", func(c *receipts.Command) { c.WorkspaceID = uuid.New() }},
		{"project", func(c *receipts.Command) { id := uuid.New(); c.ProjectID = &id }},
		{"tool", func(c *receipts.Command) { c.Tool = "module" }},
		{"action", func(c *receipts.Command) { c.Action = "update" }},
		{"request", func(c *receipts.Command) { id := uuid.New(); c.RequestID = &id }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := f.command
			tc.change(&cmd)
			result, err := receipts.Execute(t.Context(), f.db, cmd, operation())
			require.NoError(t, err)
			require.False(t, result.Replayed)
			replay, err := receipts.Execute(t.Context(), f.db, cmd, operation())
			require.NoError(t, err)
			require.True(t, replay.Replayed)
			require.Equal(t, result.Value, replay.Value)
		})
	}
	assertCounts(t, f, 7, 7, 7)
}

func TestUnkeyedMutationsRemainOrdinaryWrites(t *testing.T) {
	f := receiptDatabase(t)
	f.command.RequestID = nil
	first, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	second, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	require.NotEqual(t, first.Value, second.Value)
	require.Nil(t, first.ReplayExpiresAt)
	require.Nil(t, second.ReplayExpiresAt)
	require.False(t, first.Replayed)
	require.False(t, second.Replayed)
	assertCounts(t, f, 2, 2, 0)
}

func TestReceiptReplayReauthorizesBeforeDisclosingConflictOrDeletedResult(t *testing.T) {
	f := receiptDatabase(t)
	first, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	_, err = f.db.Exec(t.Context(), `DELETE FROM receipt_test_changes; UPDATE receipt_test_permission SET allowed=false`)
	require.NoError(t, err)
	for _, input := range []string{string(f.command.NormalizedInput), `{"changed":true}`} {
		cmd := f.command
		cmd.NormalizedInput = json.RawMessage(input)
		result, callErr := receipts.Execute(t.Context(), f.db, cmd, operation())
		require.ErrorIs(t, callErr, errDenied)
		require.Empty(t, result)
	}
	_, err = f.db.Exec(t.Context(), `UPDATE receipt_test_permission SET allowed=true`)
	require.NoError(t, err)
	replay, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, first.Value, replay.Value)
	assertCounts(t, f, 0, 1, 1)
}

func TestMutationFailureAndCancellationRollBackReceiptAndEffects(t *testing.T) {
	for _, failure := range []string{"domain", "invalid result", "cancellation", "panic"} {
		t.Run(failure, func(t *testing.T) {
			f := receiptDatabase(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			op := operation()
			apply := op.Apply
			op.Apply = func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
				value, err := apply(ctx, tx)
				if err != nil {
					return nil, err
				}
				switch failure {
				case "domain":
					return nil, errDenied
				case "invalid result":
					return json.RawMessage(`{`), nil
				case "cancellation":
					cancel()
				case "panic":
					panic("domain failure")
				}
				return value, nil
			}
			if failure == "panic" {
				require.Panics(t, func() { _, _ = receipts.Execute(ctx, f.db, f.command, op) })
			} else {
				result, err := receipts.Execute(ctx, f.db, f.command, op)
				require.Error(t, err)
				require.Empty(t, result)
			}
			assertCounts(t, f, 0, 0, 0)
			result, err := receipts.Execute(t.Context(), f.db, f.command, operation())
			require.NoError(t, err)
			require.False(t, result.Replayed)
			assertCounts(t, f, 1, 1, 1)
		})
	}
}

func TestExpiredReceiptAllowsNewMutationAndAdvertisesNewExpiry(t *testing.T) {
	f := receiptDatabase(t)
	_, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	_, err = f.db.Exec(t.Context(), `UPDATE mutation_receipts SET created_at=clock_timestamp()-INTERVAL '49 hours',replay_expires_at=clock_timestamp()-INTERVAL '1 second'`)
	require.NoError(t, err)
	f.command.NormalizedInput = json.RawMessage(`{"new":"input"}`)
	result, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	require.False(t, result.Replayed)
	require.Greater(t, time.Until(*result.ReplayExpiresAt), 24*time.Hour)
	replay, err := receipts.Execute(t.Context(), f.db, f.command, operation())
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, result.Value, replay.Value)
	assertCounts(t, f, 2, 2, 1)
}

func TestInvalidReceiptCommandsFailBeforeDomainAccess(t *testing.T) {
	f := receiptDatabase(t)
	cases := []struct {
		name   string
		change func(*receipts.Command, *receipts.Operation)
	}{
		{"actor", func(c *receipts.Command, _ *receipts.Operation) { c.ActorID = uuid.Nil }},
		{"workspace", func(c *receipts.Command, _ *receipts.Operation) { c.WorkspaceID = uuid.Nil }},
		{"project", func(c *receipts.Command, _ *receipts.Operation) { id := uuid.Nil; c.ProjectID = &id }},
		{"request", func(c *receipts.Command, _ *receipts.Operation) { id := uuid.Nil; c.RequestID = &id }},
		{"tool", func(c *receipts.Command, _ *receipts.Operation) { c.Tool = "" }},
		{"action", func(c *receipts.Command, _ *receipts.Operation) { c.Action = "" }},
		{"null input", func(c *receipts.Command, _ *receipts.Operation) { c.NormalizedInput = json.RawMessage(`null`) }},
		{"trailing input", func(c *receipts.Command, _ *receipts.Operation) { c.NormalizedInput = json.RawMessage(`{} {}`) }},
		{"malformed input", func(c *receipts.Command, _ *receipts.Operation) { c.NormalizedInput = json.RawMessage(`{`) }},
		{"authorization", func(_ *receipts.Command, o *receipts.Operation) { o.Authorize = nil }},
		{"mutation", func(_ *receipts.Command, o *receipts.Operation) { o.Apply = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, op := f.command, operation()
			tc.change(&cmd, &op)
			result, err := receipts.Execute(t.Context(), f.db, cmd, op)
			require.ErrorIs(t, err, receipts.ErrInvalidRequest)
			require.Empty(t, result)
		})
	}
	_, err := receipts.Execute(t.Context(), nil, f.command, operation())
	require.ErrorIs(t, err, receipts.ErrInvalidRequest)
	assertCounts(t, f, 0, 0, 0)
}
