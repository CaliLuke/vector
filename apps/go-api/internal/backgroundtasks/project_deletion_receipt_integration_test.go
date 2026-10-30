//go:build integration

package backgroundtasks

import (
	"context"
	"testing"
	"time"
	stduuid "uuid"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workspaceprojectreads"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type projectDeletionReceiptFixture struct {
	pool    *pgxpool.Pool
	repo    *workspaceprojectreads.LifecycleRepository
	store   *PostgresDeletionStore
	cmd     projectmutations.Command
	deleted projectmutations.Result
}

func newProjectDeletionReceiptFixture(t *testing.T, keyed bool) projectDeletionReceiptFixture {
	t.Helper()
	_, _, pool := openBackgroundTaskDatabase(t)
	t.Cleanup(pool.Close)
	_, err := pool.Exec(t.Context(), testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	repo, err := workspaceprojectreads.NewLifecycleRepository(pool, projectmutations.Effects{}, "https://vector.example", func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) { return false, nil })
	require.NoError(t, err)
	cmd := projectmutations.Command{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace), WorkspaceSlug: "milestones", Action: "create", Fields: map[string]any{"name": "Receipt recovery", "identifier": "RECOVERY"}}
	created, err := repo.MutateResource(t.Context(), cmd)
	require.NoError(t, err)
	cmd.ProjectID, cmd.ResourceID, cmd.Action, cmd.Fields = uuid.MustParse(created.ID), uuid.MustParse(created.ID), "delete", nil
	if keyed {
		requestID := uuid.New()
		cmd.RequestID = &requestID
	}
	deleted, err := repo.MutateResource(t.Context(), cmd)
	require.NoError(t, err)
	if keyed {
		require.False(t, *deleted.Replayed)
	}
	store, err := NewPostgresDeletionStore(pool)
	require.NoError(t, err)
	return projectDeletionReceiptFixture{pool: pool, repo: repo, store: store, cmd: cmd, deleted: deleted}
}

func (f projectDeletionReceiptFixture) cascadeMembers(t *testing.T) {
	t.Helper()
	model, err := f.store.Policy().Lookup("db", "project")
	require.NoError(t, err)
	index := -1
	for i, r := range deletionRelations(f.store.Policy(), model) {
		if r.model.Table == "project_members" && r.relation.Column == "project_id" {
			index = i
		}
	}
	require.NotEqual(t, -1, index)
	plan, err := f.store.Plan(t.Context(), deletionPlanInput{Root: DeletionInput{EventID: uuid.NewString(), App: "db", Model: "project", ID: f.deleted.ID}, Relation: index})
	require.NoError(t, err)
	require.Len(t, plan.IDs, 1)
	_, err = f.store.Apply(t.Context(), plan)
	require.NoError(t, err)
}

func (f projectDeletionReceiptFixture) assertReplay(t *testing.T) {
	t.Helper()
	replay, err := f.repo.MutateResource(t.Context(), f.cmd)
	require.NoError(t, err, "worker cleanup must preserve receipt recovery")
	require.True(t, *replay.Replayed)
	require.Equal(t, f.deleted.Revision, replay.Revision)
	require.Equal(t, f.deleted.ReplayExpiresAt, replay.ReplayExpiresAt)
}

func (f projectDeletionReceiptFixture) expire(t *testing.T) {
	t.Helper()
	_, err := f.pool.Exec(t.Context(), `UPDATE mutation_receipts SET created_at=now()-interval '3 days',replay_expires_at=now()-interval '1 day' WHERE request_id=$1`, f.cmd.RequestID)
	require.NoError(t, err)
}

func TestProjectDeletionCascadeRetainsReceiptAuthorization(t *testing.T) {
	f := newProjectDeletionReceiptFixture(t, true)
	f.assertReplay(t)
	f.cascadeMembers(t)
	f.assertReplay(t)
	reader, err := workspaceprojectreads.NewRepository(f.pool)
	require.NoError(t, err)
	_, err = reader.APIProject(t.Context(), stduuid.MustParse(f.cmd.ActorID.String()), f.cmd.WorkspaceSlug, stduuid.MustParse(f.cmd.ProjectID.String()), workspaceprojectreads.APIProjectDetailOptions{})
	require.ErrorIs(t, err, workspaceprojectreads.ErrNotFound, "retained membership must not expose the deleted project")
	for _, change := range []struct{ name, update, restore string }{
		{"project membership inactive", `UPDATE project_members SET is_active=false WHERE project_id=$1`, `UPDATE project_members SET is_active=true WHERE project_id=$1`},
		{"project role removed", `UPDATE project_members SET role=15 WHERE project_id=$1`, `UPDATE project_members SET role=20 WHERE project_id=$1`},
		{"project membership deleted", `UPDATE project_members SET deleted_at=now() WHERE project_id=$1`, `UPDATE project_members SET deleted_at=NULL WHERE project_id=$1`},
		{"workspace membership inactive", `UPDATE workspace_members SET is_active=false WHERE workspace_id=(SELECT workspace_id FROM projects WHERE id=$1)`, `UPDATE workspace_members SET is_active=true WHERE workspace_id=(SELECT workspace_id FROM projects WHERE id=$1)`},
		{"account inactive", `UPDATE users SET is_active=false WHERE id=(SELECT member_id FROM project_members WHERE project_id=$1)`, `UPDATE users SET is_active=true WHERE id=(SELECT member_id FROM project_members WHERE project_id=$1)`},
	} {
		t.Run(change.name, func(t *testing.T) {
			_, err := f.pool.Exec(t.Context(), change.update, f.cmd.ProjectID)
			require.NoError(t, err)
			_, err = f.repo.MutateResource(t.Context(), f.cmd)
			require.ErrorIs(t, err, workspaceprojectreads.ErrForbidden)
			_, err = f.pool.Exec(t.Context(), change.restore, f.cmd.ProjectID)
			require.NoError(t, err)
			f.assertReplay(t)
		})
	}
	f.expire(t)
	f.cascadeMembers(t)
	var deleted bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL FROM project_members WHERE project_id=$1`, f.cmd.ProjectID).Scan(&deleted))
	require.True(t, deleted, "expired receipt must not retain membership")
}

func TestProjectDeletionWithoutReceiptCascadesMembership(t *testing.T) {
	f := newProjectDeletionReceiptFixture(t, false)
	f.cascadeMembers(t)
	var deleted bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL FROM project_members WHERE project_id=$1`, f.cmd.ProjectID).Scan(&deleted))
	require.True(t, deleted)
}

func TestProjectPurgeRetainsReceiptAuthorizationUntilExpiry(t *testing.T) {
	f := newProjectDeletionReceiptFixture(t, true)
	_, err := f.pool.Exec(t.Context(), `UPDATE projects SET deleted_at=now()-interval '3 days' WHERE id=$1`, f.cmd.ProjectID)
	require.NoError(t, err)
	index := -1
	for i, m := range f.store.Policy().Models {
		if m.App == "db" && m.Name == "project" {
			index = i
		}
	}
	require.NotEqual(t, -1, index)
	input := hardDeletionInput{Cutoff: time.Now().Add(-24 * time.Hour), Model: index}
	purged, err := f.store.HardDelete(t.Context(), input)
	require.NoError(t, err)
	require.Zero(t, purged, "live receipt must retain its authorization target")
	f.assertReplay(t)
	f.expire(t)
	purged, err = f.store.HardDelete(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, 1, purged)
	var members int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM project_members WHERE project_id=$1`, f.cmd.ProjectID).Scan(&members))
	require.Zero(t, members, "physical project purge must remove retained memberships")
}

func TestProjectDeletionReceiptRetainsOnlyItsActorMembership(t *testing.T) {
	f := newProjectDeletionReceiptFixture(t, true)
	other := uuid.New()
	_, err := f.pool.Exec(t.Context(), `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(u)||jsonb_build_object('id',$1::text,'username',$1::text,'email',$1::text||'@example.test'))).* FROM users u WHERE id=$2`, other, f.cmd.ActorID)
	require.NoError(t, err)
	var otherMember string
	require.NoError(t, f.pool.QueryRow(t.Context(), `INSERT INTO project_members SELECT (jsonb_populate_record(NULL::project_members,to_jsonb(m)||jsonb_build_object('id',gen_random_uuid()::text,'member_id',$1::text))).* FROM project_members m WHERE project_id=$2 AND member_id=$3 RETURNING id`, other, f.cmd.ProjectID, f.cmd.ActorID).Scan(&otherMember))
	model, err := f.store.Policy().Lookup("db", "project")
	require.NoError(t, err)
	index := -1
	for i, r := range deletionRelations(f.store.Policy(), model) {
		if r.model.Table == "project_members" && r.relation.Column == "project_id" {
			index = i
		}
	}
	require.NotEqual(t, -1, index)
	plan, err := f.store.Plan(t.Context(), deletionPlanInput{Root: DeletionInput{EventID: uuid.NewString(), App: "db", Model: "project", ID: f.deleted.ID}, Relation: index})
	require.NoError(t, err)
	require.Len(t, plan.IDs, 2)
	children, err := f.store.Apply(t.Context(), plan)
	require.NoError(t, err)
	require.Len(t, children, 1)
	require.Equal(t, otherMember, children[0].ID)
	var deleted bool
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT deleted_at IS NOT NULL FROM project_members WHERE id=$1`, otherMember).Scan(&deleted))
	require.True(t, deleted)
	f.assertReplay(t)
	retried, err := f.store.Apply(t.Context(), plan)
	require.NoError(t, err)
	require.Equal(t, children, retried, "a committed cascade retry must retain child workflow admission")
	f.assertReplay(t)
}
