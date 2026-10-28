//go:build integration

package backgroundtasks

import (
	"context"
	"errors"
	"testing"
	"time"

	states "github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type lostStateDeletionAdmission struct {
	dispatcher *DeletionDispatcher
	instance   string
	lost       bool
}

func (a *lostStateDeletionAdmission) Admit(ctx context.Context, input DeletionInput) (string, error) {
	id, err := a.dispatcher.Admit(ctx, input)
	if err == nil {
		a.instance = id
		if !a.lost {
			a.lost = true
			return "", errors.New("lost admission acknowledgment")
		}
	}
	return id, err
}
func TestStateDeletionOutboxReconcilesAndRetainsHistoryUntilPurge(t *testing.T) {
	f := newMilestoneWebhookFixture(t)
	repo, err := states.NewWriteRepository(f.pool)
	require.NoError(t, err)
	cmd := states.MutationCommand{ActorID: f.opts.ActorID, WorkspaceID: f.opts.WorkspaceID, ProjectID: f.opts.ProjectID, WorkspaceSlug: "milestones", Action: "create", Changes: states.StateChanges{Name: new("History"), Color: new("#123456")}}
	created, err := repo.MutateState(f.ctx, cmd)
	require.NoError(t, err)
	id := created.StateID
	f.exec(t, `UPDATE projects SET default_state_id=$1 WHERE id=$2`, id, testsupport.LabelProject)
	for _, table := range []string{"issues", "draft_issues"} {
		extraColumns, extraValues := "", ""
		if table == "issues" {
			extraColumns = ",sequence_id,is_draft"
			extraValues = ",1,false"
		}
		f.exec(t, `INSERT INTO `+table+`(id,project_id,workspace_id,state_id,name,description_json,description_html,priority,sort_order,created_at,updated_at,deleted_at`+extraColumns+`)
 VALUES(gen_random_uuid(),$1,$2,$3,'History','{}','','none',0,now(),now(),now()`+extraValues+`)`, testsupport.LabelProject, testsupport.LabelWorkspace, id)
	}
	cmd.Action, cmd.StateID, cmd.Changes = "delete", uuid.MustParse(id), states.StateChanges{}
	cmd.RequestID = new(uuid.New())
	_, err = repo.MutateState(f.ctx, cmd)
	require.NoError(t, err)
	f.exec(t, `CREATE SCHEMA plane_execution`)
	executionURL, err := withExecutionSearchPath(f.pool.Config().ConnString())
	require.NoError(t, err)
	app := selvedge.NewApp(selvedge.WithDatabase(executionURL), selvedge.WithServiceName("state-deletion-proof"), selvedge.WithActivityPollInterval(10*time.Millisecond), selvedge.WithWorkflowResumptionInterval(10*time.Millisecond))
	store, err := NewPostgresDeletionStore(f.pool)
	require.NoError(t, err)
	dispatcher, err := RegisterDeletionWorkflows(app, store, 60, retry.Exponential(3, time.Millisecond, time.Millisecond, 1))
	require.NoError(t, err)
	require.NoError(t, app.Start(f.ctx))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, app.Shutdown(ctx))
	})
	admission := &lostStateDeletionAdmission{dispatcher: dispatcher}
	outbox := &PostgresWorkItemEffectStore{pool: f.pool, deletions: admission}
	count, err := outbox.Drain(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	var admitted bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT admitted_at IS NOT NULL FROM work_item_effects`).Scan(&admitted))
	require.False(t, admitted)
	firstInstance := admission.instance
	f.exec(t, `UPDATE work_item_effects SET next_attempt_at=now()`)
	count, err = outbox.Drain(f.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, firstInstance, admission.instance)
	waitVersionWorkflow(t, f.ctx, app, firstInstance)
	var cleared bool
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT default_state_id IS NULL FROM projects WHERE id=$1`, testsupport.LabelProject).Scan(&cleared))
	require.True(t, cleared)
	for _, table := range []string{"issues", "draft_issues"} {
		var history int
		require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM `+table+` WHERE state_id=$1 AND deleted_at IS NOT NULL`, id).Scan(&history))
		require.Equal(t, 1, history)
	}
	replay, err := repo.MutateState(f.ctx, cmd)
	require.NoError(t, err)
	require.True(t, *replay.Replayed)
	count, err = outbox.Drain(f.ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	f.exec(t, `UPDATE states SET deleted_at=now()-interval '61 days' WHERE id=$1`, id)
	purge, err := dispatcher.AdmitHardDelete(f.ctx, time.Now())
	require.NoError(t, err)
	waitVersionWorkflow(t, f.ctx, app, purge)
	for _, table := range []string{"states", "state_edit_revisions", "issues", "draft_issues"} {
		var remaining int
		require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM `+table).Scan(&remaining))
		require.Zero(t, remaining, table)
	}
	replay, err = repo.MutateState(f.ctx, cmd)
	require.NoError(t, err)
	require.True(t, *replay.Replayed)
}
