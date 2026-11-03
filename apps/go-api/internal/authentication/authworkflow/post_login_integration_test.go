//go:build integration

package authworkflow

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type forbiddenPublisher struct{ t *testing.T }

func (p forbiddenPublisher) Task(context.Context, string, []any, map[string]any) error {
	p.t.Fatal("post-login effects must be admitted transactionally")
	return nil
}

func TestPostLoginUsesCurrentAccountAndDurableJoinBoundary(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	pool, err := pgxpool.New(ctx, pair.SeedURL())
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	const actor = "00000000-0000-4000-8000-000000000003"
	_, err = pool.Exec(ctx, `DELETE FROM project_members; DELETE FROM workspace_members;`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO workspace_member_invites(id,workspace_id,email,role,token,accepted,created_at,updated_at) SELECT gen_random_uuid(),'00000000-0000-4000-8000-000000000001',email,15,'token',true,now(),now() FROM users WHERE id=$1`, actor)
	require.NoError(t, err)
	repository, err := projectmembershipreads.NewInvitationsRepositoryWithReaderWriter(pool, pool, forbiddenPublisher{t}, "", "")
	require.NoError(t, err)
	workflow, err := NewPostUserAuthWorkflow(repository)
	require.NoError(t, err)
	require.NoError(t, workflow.Run(ctx, actor, "stale-email@example.test"))
	var members, events int
	require.NoError(t, pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM workspace_members WHERE member_id=$1 AND is_active),(SELECT count(*) FROM work_item_effects WHERE kind='event_tracking' AND payload->>'user_id'=$1::text)`, actor).Scan(&members, &events))
	require.Equal(t, 1, members)
	require.Equal(t, 1, events)
	_, err = pool.Exec(ctx, `UPDATE users SET is_active=false WHERE id=$1`, actor)
	require.NoError(t, err)
	require.Error(t, workflow.Run(ctx, actor, "stale-email@example.test"))
}
