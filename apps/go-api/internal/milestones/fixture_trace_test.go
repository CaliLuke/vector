//go:build integration

package milestones_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/stretchr/testify/require"
)

func milestoneTraceEvent(ctx context.Context, message string, data map[string]any) string {
	statement, _ := data["sql"].(string)
	return fmt.Sprintf("milestone database %s elapsed=%v sql_sha256=%x backend_pid=%v error_type=%T context_error=%v",
		message, data["time"], sha256.Sum256([]byte(statement)), data["pid"], data["err"], ctx.Err())
}

func milestoneFixtureTrace(t *testing.T) *tracelog.TraceLog {
	t.Helper()
	var mu sync.Mutex
	var recent []string
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for _, event := range recent {
			t.Log(event)
		}
	})
	return &tracelog.TraceLog{LogLevel: tracelog.LogLevelInfo, Logger: tracelog.LoggerFunc(func(ctx context.Context, _ tracelog.LogLevel, message string, data map[string]any) {
		// Keep statement fingerprints and timings, never SQL text or arguments.
		event := milestoneTraceEvent(ctx, message, data)
		mu.Lock()
		defer mu.Unlock()
		if len(recent) == 16 {
			recent = recent[1:]
		}
		recent = append(recent, event)
	})}
}

type milestoneWait struct {
	PID                        int32
	State, WaitType, WaitEvent string
	BlockingPIDs               []int32
	StatementSHA256            string
}

func milestoneDatabaseWaits(ctx context.Context, db *pgx.Conn) ([]milestoneWait, error) {
	rows, err := db.Query(ctx, `SELECT pid,coalesce(state,''),coalesce(wait_event_type,''),coalesce(wait_event,''),pg_blocking_pids(pid),
 encode(sha256(convert_to(coalesce(query,''),'UTF8')),'hex') FROM pg_stat_activity
 WHERE datname=current_database() AND pid<>pg_backend_pid()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var waits []milestoneWait
	for rows.Next() {
		var wait milestoneWait
		if err := rows.Scan(&wait.PID, &wait.State, &wait.WaitType, &wait.WaitEvent, &wait.BlockingPIDs, &wait.StatementSHA256); err != nil {
			return nil, err
		}
		waits = append(waits, wait)
	}
	return waits, rows.Err()
}

func TestMilestoneFixtureDiagnosticsExcludeStatementLiteralsArgumentsAndErrorMessages(t *testing.T) {
	const marker = "sensitive-fixture-marker"
	entry := milestoneTraceEvent(t.Context(), "Query", map[string]any{
		"sql": "SELECT '" + marker + "'", "args": []any{marker}, "err": errors.New(marker), "time": time.Second,
	})
	require.NotContains(t, entry, marker)
	require.Contains(t, entry, "elapsed=1s")
	require.Contains(t, entry, "sql_sha256=")
	require.Contains(t, entry, "error_type=")
}

func TestMilestoneFixtureDiagnosticsIdentifyBlockedDatabaseOperations(t *testing.T) {
	f := newWriteFixture(t)
	observer, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(f.ctx), 5*time.Second)
		defer cancel()
		testsupport.CloseContext(t, observer, cleanup)
	})
	tx, err := f.db.Begin(f.ctx)
	require.NoError(t, err)
	t.Cleanup(func() { testsupport.Rollback(t, tx, f.ctx) })
	_, err = tx.Exec(f.ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, f.opts.ProjectID)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		_, createErr := f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: textField("Blocked diagnostic")})
		done <- createErr
	}()
	require.Eventually(t, func() bool {
		waits, queryErr := milestoneDatabaseWaits(f.ctx, observer)
		if queryErr != nil {
			return false
		}
		for _, wait := range waits {
			if wait.WaitType == "Lock" && len(wait.BlockingPIDs) > 0 && len(wait.StatementSHA256) == 64 {
				return true
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, tx.Rollback(f.ctx))
	require.NoError(t, <-done)
}
