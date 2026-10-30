//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	labels "github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type labelMutationOutcome struct {
	value labels.LabelMutationResult
	err   error
}

func labelMutationCommand(action string) labels.LabelMutation {
	return labels.LabelMutation{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(testsupport.LabelWorkspace),
		ProjectID: uuid.MustParse(testsupport.LabelProject), WorkspaceSlug: "workspace-one", Action: action}
}

func competingLabelMutations(t *testing.T, f milestoneFixture, commands [2]labels.LabelMutation) [2]labelMutationOutcome {
	t.Helper()
	require.Equal(t, commands[0].ProjectID, commands[1].ProjectID)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	repository, err := labels.NewRepository(f.pool)
	require.NoError(t, err)
	blocker, err := f.pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, dbutil.RollbackTransaction(ctx, blocker.Rollback, nil)) }()
	_, err = blocker.Exec(ctx, `SELECT id FROM projects WHERE id=$1 FOR UPDATE`, commands[0].ProjectID)
	require.NoError(t, err)
	done := make(chan labelMutationOutcome, 2)
	for _, command := range commands {
		go func() {
			value, mutationErr := repository.MutateLabel(ctx, command)
			done <- labelMutationOutcome{value, mutationErr}
		}()
	}
	require.Eventually(t, func() bool {
		var waiting int
		queryErr := f.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&waiting)
		return queryErr == nil && waiting == 2
	}, 10*time.Second, 10*time.Millisecond, "both label requests must wait before the competing commits")
	require.NoError(t, blocker.Commit(ctx))
	var results [2]labelMutationOutcome
	for i := range results {
		select {
		case results[i] = <-done:
		case <-ctx.Done():
			t.Fatal("label requests did not finish after releasing the project lock")
		}
	}
	return results
}

func TestLabelConcurrentIdenticalRequestCommitsOnce(t *testing.T) {
	f := newMilestoneFixture(t)
	key, name := uuid.New(), uuid.NewString()
	command := labelMutationCommand("create")
	command.RequestID, command.Changes.Name = &key, &name
	results := competingLabelMutations(t, f, [2]labels.LabelMutation{command, command})
	require.NoError(t, results[0].err)
	require.NoError(t, results[1].err)
	first, second := results[0].value, results[1].value
	require.NotNil(t, first.Replayed)
	require.NotNil(t, second.Replayed)
	require.NotEqual(t, *first.Replayed, *second.Replayed)
	require.Equal(t, first.LabelID, second.LabelID)
	require.Equal(t, first.Revision, second.Revision)
	require.JSONEq(t, string(first.Result), string(second.Result))
	var records, receipts, revision int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM labels WHERE project_id=$1 AND name=$2`, testsupport.LabelProject, name).Scan(&records))
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool='label' AND request_id=$1`, key).Scan(&receipts))
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM label_edit_revisions WHERE label_id=$1`, first.LabelID).Scan(&revision))
	require.Equal(t, 1, records)
	require.Equal(t, 1, receipts)
	require.Equal(t, 1, revision)
}

func TestLabelConcurrentChangedRequestCommitsOneInput(t *testing.T) {
	f := newMilestoneFixture(t)
	key, firstName, secondName := uuid.New(), uuid.NewString(), uuid.NewString()
	first := labelMutationCommand("create")
	first.RequestID, first.Changes.Name = &key, &firstName
	second := first
	second.Changes.Name = &secondName
	results := competingLabelMutations(t, f, [2]labels.LabelMutation{first, second})
	if results[0].err != nil {
		results[0], results[1] = results[1], results[0]
	}
	require.NoError(t, results[0].err)
	require.ErrorIs(t, results[1].err, mutationreceipts.ErrConflict)
	require.NotNil(t, results[0].value.Replayed)
	require.False(t, *results[0].value.Replayed)
	var records, receipts int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM labels WHERE project_id=$1 AND name IN ($2,$3)`, testsupport.LabelProject, firstName, secondName).Scan(&records))
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool='label' AND request_id=$1`, key).Scan(&receipts))
	require.Equal(t, 1, records)
	require.Equal(t, 1, receipts)
	var saved, committed struct{ Name string }
	require.NoError(t, json.Unmarshal(results[0].value.Result, &committed))
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT name FROM labels WHERE id=$1`, results[0].value.LabelID).Scan(&saved.Name))
	require.Equal(t, committed.Name, saved.Name)
}

func TestLabelConcurrentRevisionAllowsOneUpdate(t *testing.T) {
	f := newMilestoneFixture(t)
	repository, err := labels.NewRepository(f.pool)
	require.NoError(t, err)
	initialName, initialColor := uuid.NewString(), "#123456"
	create := labelMutationCommand("create")
	create.Changes = labels.LabelChanges{Name: &initialName, Color: &initialColor}
	created, err := repository.MutateLabel(t.Context(), create)
	require.NoError(t, err)
	firstKey, secondKey, firstName, secondColor := uuid.New(), uuid.New(), uuid.NewString(), "#abcdef"
	first := labelMutationCommand("update")
	first.LabelID, first.ExpectedRevision, first.RequestID = uuid.MustParse(created.LabelID), &created.Revision, &firstKey
	first.Changes.Name = &firstName
	second := first
	second.RequestID, second.Changes = &secondKey, labels.LabelChanges{Color: &secondColor}
	results := competingLabelMutations(t, f, [2]labels.LabelMutation{first, second})
	if results[0].err != nil {
		results[0], results[1] = results[1], results[0]
	}
	require.NoError(t, results[0].err)
	require.ErrorIs(t, results[1].err, labels.ErrLabelRevision)
	var saved, committed struct{ Name, Color string }
	require.NoError(t, json.Unmarshal(results[0].value.Result, &committed))
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT name,color FROM labels WHERE id=$1`, created.LabelID).Scan(&saved.Name, &saved.Color))
	require.Equal(t, committed, saved)
	require.NotEqual(t, saved.Name == firstName, saved.Color == secondColor)
	var revision, receipts int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT revision FROM label_edit_revisions WHERE label_id=$1`, created.LabelID).Scan(&revision))
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT count(*) FROM mutation_receipts WHERE tool='label' AND request_id IN ($1,$2)`, firstKey, secondKey).Scan(&receipts))
	require.Equal(t, 2, revision)
	require.Equal(t, 1, receipts)
}
