//go:build integration

package main

import (
	"context"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestBootstrapReplaysOnlyTheSelectedQuarantinedEffect(t *testing.T) {
	ctx := t.Context()
	databaseURL := postgresfixture.EmptyDatabase(t)
	t.Setenv("DATABASE_URL", databaseURL)
	require.NoError(t, run(ctx, []string{"schema"}))
	conn, err := pgx.Connect(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { testsupport.CloseContext(t, conn, context.Background()) })

	replayID, otherID := uuid.New(), uuid.New()
	_, err = conn.Exec(ctx, `INSERT INTO work_item_effects(id,kind,payload,attempts,quarantined_at,next_attempt_at)
		VALUES($1,'activity','{"type":"issue.activity.created"}',3,now(),'infinity'),($2,'activity','{}',1,now(),'infinity')`,
		replayID, otherID)
	require.NoError(t, err)
	require.ErrorContains(t, run(ctx, []string{"replay-work-item-effect", "invalid"}), "effect ID must be a UUID")
	require.ErrorContains(t, run(ctx, []string{"replay-work-item-effect", uuid.NewString()}), "no quarantined work item effect")
	require.NoError(t, run(ctx, []string{"replay-work-item-effect", replayID.String()}))
	require.ErrorContains(t, run(ctx, []string{"replay-work-item-effect", replayID.String()}), "no quarantined work item effect")

	var ready, untouched bool
	require.NoError(t, conn.QueryRow(ctx, `SELECT quarantined_at IS NULL AND admitted_at IS NULL
		AND next_attempt_at<=clock_timestamp() AND attempts=3 AND payload='{"type":"issue.activity.created"}'
		FROM work_item_effects WHERE id=$1`, replayID).Scan(&ready))
	require.NoError(t, conn.QueryRow(ctx, `SELECT quarantined_at IS NOT NULL AND next_attempt_at='infinity'
		FROM work_item_effects WHERE id=$1`, otherID).Scan(&untouched))
	require.True(t, ready)
	require.True(t, untouched)
}
