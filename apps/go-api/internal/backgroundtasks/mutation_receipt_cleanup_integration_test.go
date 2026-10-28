//go:build integration

package backgroundtasks

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestMutationReceiptCleanupRetainsLiveAndRenewedReceipts(t *testing.T) {
	executionURL, pool := openCatalogueActivityDatabase(t)
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	store, err := NewPostgresCleanupStore(pool)
	require.NoError(t, err)

	t.Run("bounded cleanup skips renewal locks and rejects future expiry", func(t *testing.T) {
		seedCleanupReceipts(t, pool, CleanupBatchSize+2, "-1 hour")
		seedCleanupReceipts(t, pool, 1, "48 hours")
		renewal, beginErr := pool.Begin(ctx)
		require.NoError(t, beginErr)
		defer func() { _ = renewal.Rollback(context.WithoutCancel(ctx)) }()
		var request uuid.UUID
		require.NoError(t, renewal.QueryRow(ctx, `SELECT request_id FROM mutation_receipts WHERE replay_expires_at < statement_timestamp() LIMIT 1 FOR UPDATE`).Scan(&request))
		input := CleanupBatchInput{Task: CleanupMutationReceipts, Cutoff: time.Now().Add(7 * 24 * time.Hour)}
		deleted, deleteErr := store.DeleteCleanupBatch(ctx, input)
		require.NoError(t, deleteErr)
		require.EqualValues(t, CleanupBatchSize, deleted)
		deleted, deleteErr = store.DeleteCleanupBatch(ctx, input)
		require.NoError(t, deleteErr)
		require.EqualValues(t, 1, deleted)
		deleted, deleteErr = store.DeleteCleanupBatch(ctx, input)
		require.NoError(t, deleteErr)
		require.Zero(t, deleted)

		_, renewErr := renewal.Exec(ctx, `UPDATE mutation_receipts SET created_at=statement_timestamp(),replay_expires_at=statement_timestamp()+INTERVAL '48 hours' WHERE request_id=$1`, request)
		require.NoError(t, renewErr)
		require.NoError(t, renewal.Commit(ctx))
		deleted, deleteErr = store.DeleteCleanupBatch(ctx, input)
		require.NoError(t, deleteErr)
		require.Zero(t, deleted)
		var remaining int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM mutation_receipts`).Scan(&remaining))
		require.Equal(t, 2, remaining)
	})

	t.Run("durable admission drains batches and registers its daily timer", func(t *testing.T) {
		seedCleanupReceipts(t, pool, CleanupBatchSize+1, "-1 hour")
		app := selvedge.NewApp(selvedge.WithDatabase(executionURL), selvedge.WithServiceName("mutation-receipt-cleanup-proof"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond))
		dispatcher, registerErr := RegisterCleanupWorkflows(app, store, nil)
		require.NoError(t, registerErr)
		require.NoError(t, app.Start(ctx))
		t.Cleanup(func() {
			shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			require.NoError(t, app.Shutdown(shutdown))
		})
		scheduled := time.Now().UTC()
		id, admitErr := dispatcher.Admit(ctx, CleanupMutationReceipts, scheduled)
		require.NoError(t, admitErr)
		result, waitErr := selvedge.WaitWorkflow[CleanupRunOutput](ctx, app, id)
		require.NoError(t, waitErr)
		require.Equal(t, "completed", result.Status)
		require.EqualValues(t, CleanupBatchSize+1, result.Output.Deleted)
		require.EqualValues(t, 2, result.Output.Batches)
		repeated, repeatErr := dispatcher.Admit(ctx, CleanupMutationReceipts, scheduled)
		require.NoError(t, repeatErr)
		require.Equal(t, id, repeated)
		require.NoError(t, dispatcher.StartBeatSchedules(ctx))
		waitForBeatTimer(t, ctx, pool, "plane-beat:delete-expired-mutation-receipts")
		var remaining int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM mutation_receipts`).Scan(&remaining))
		require.Equal(t, 2, remaining)
	})
}

func seedCleanupReceipts(t *testing.T, pool *pgxpool.Pool, count int, expiry string) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `INSERT INTO mutation_receipts(actor_id,workspace_id,tool,action,request_id,input_hash,result,created_at,replay_expires_at)
 SELECT gen_random_uuid(),gen_random_uuid(),'cleanup-test','update',gen_random_uuid(),decode(repeat('00',32),'hex'),'{}',statement_timestamp()-INTERVAL '72 hours',statement_timestamp()+$2::interval
 FROM generate_series(1,$1::int)`, count, expiry)
	require.NoError(t, err)
}
