//go:build integration

package milestones_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (f writeFixture) item(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := f.db.Exec(f.ctx, `INSERT INTO issues(id,workspace_id,project_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at,created_by_id)
 VALUES($1,$2,$3,'Work','{}','','none',0,1,false,now(),now(),$4)`, id, f.opts.WorkspaceID, f.opts.ProjectID, f.opts.ActorID)
	require.NoError(t, err)
	return id
}

func (f writeFixture) milestone(t *testing.T, title string) milestones.MutationResult {
	t.Helper()
	result, err := f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: textField(title)})
	require.NoError(t, err)
	return result
}

func TestMilestoneAssociationsApplyNetChangeAndKeepReceiptInputsCanonical(t *testing.T) {
	f := newWriteFixture(t)
	created := f.milestone(t, "Release")
	id, item := uuid.MustParse(created.MilestoneID), f.item(t)
	overlap, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item, item}, []uuid.UUID{item})
	require.NoError(t, err)
	require.Equal(t, created.Revision, overlap.Revision)
	require.Nil(t, overlap.Result)
	f.counts(t, 1, 1, 0)
	added, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	require.NotEqual(t, created.Revision, added.Revision)
	page, err := f.repo.ListWorkItems(f.ctx, f.opts.Scope, id, djangoapi.Query{})
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	linkID := page.Results[0].ID
	unchanged, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item, item}, nil)
	require.NoError(t, err)
	require.Equal(t, added.Revision, unchanged.Revision)
	f.counts(t, 1, 3, 0)
	removed, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item}, []uuid.UUID{item, item})
	require.NoError(t, err)
	require.NotEqual(t, added.Revision, removed.Revision)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	page, err = f.repo.ListWorkItems(f.ctx, f.opts.Scope, id, djangoapi.Query{})
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	require.NotEqual(t, linkID, page.Results[0].ID, "re-adding allocates a new association identity")
	other := f.item(t)
	requestID := uuid.New()
	f.opts.RequestID = &requestID
	first, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{other, item, other}, nil)
	require.NoError(t, err)
	replayed, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item, other}, []uuid.UUID{})
	require.NoError(t, err)
	require.True(t, *replayed.Replayed)
	require.Equal(t, first.Revision, replayed.Revision)
	f.counts(t, 1, 9, 1)
}

func TestMilestoneAssociationValidationIsAtomicAndDoesNotMoveOtherAssignments(t *testing.T) {
	f := newWriteFixture(t)
	first, second := f.milestone(t, "First"), f.milestone(t, "Second")
	id, other := uuid.MustParse(first.MilestoneID), uuid.MustParse(second.MilestoneID)
	item, another := f.item(t), f.item(t)
	_, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{item}, []uuid.UUID{uuid.New()})
	require.ErrorIs(t, err, milestones.ErrNotFound)
	f.counts(t, 2, 2, 0)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, other, []uuid.UUID{item}, nil)
	require.NoError(t, err)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{another, item}, nil)
	require.ErrorIs(t, err, milestones.ErrConflict)
	unchanged, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, nil, []uuid.UUID{item})
	require.NoError(t, err)
	require.Equal(t, first.Revision, unchanged.Revision)
	page, err := f.repo.ListWorkItems(f.ctx, f.opts.Scope, other, djangoapi.Query{})
	require.NoError(t, err)
	require.Len(t, page.Results, 1)
	require.Equal(t, item.String(), page.Results[0].Issue)
	f.counts(t, 2, 4, 0)
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET archived_at=now() WHERE id=$1`, another)
	require.NoError(t, err)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{another}, nil)
	require.ErrorIs(t, err, milestones.ErrInvalidInput)
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET is_draft=true WHERE id=$1`, another)
	require.NoError(t, err)
	_, err = f.repo.ManageWorkItems(f.ctx, f.opts, id, nil, []uuid.UUID{another})
	require.ErrorIs(t, err, milestones.ErrNotFound)
	f.counts(t, 2, 4, 0)
}

func TestMilestoneDeleteCleansHiddenLinksPreservesItemsAndReplays(t *testing.T) {
	f := newWriteFixture(t)
	created := f.milestone(t, "Release")
	id := uuid.MustParse(created.MilestoneID)
	visible, draft, deleted := f.item(t), f.item(t), f.item(t)
	linked, err := f.repo.ManageWorkItems(f.ctx, f.opts, id, []uuid.UUID{visible, draft, deleted}, nil)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET is_draft=true WHERE id=$1`, draft)
	require.NoError(t, err)
	_, err = f.db.Exec(f.ctx, `UPDATE issues SET deleted_at=now() WHERE id=$1`, deleted)
	require.NoError(t, err)
	requestID := uuid.New()
	f.opts.RequestID = &requestID
	f.opts.ExpectedRevision = &linked.Revision
	result, err := f.repo.Delete(f.ctx, f.opts, id)
	require.NoError(t, err)
	require.Nil(t, result.Result)
	require.NotEqual(t, linked.Revision, result.Revision)
	replayed, err := f.repo.Delete(f.ctx, f.opts, id)
	require.NoError(t, err)
	require.True(t, *replayed.Replayed)
	require.Equal(t, result.Revision, replayed.Revision)
	f.counts(t, 1, 10, 1)
	var links, items int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM milestone_issues WHERE deleted_at IS NULL),(SELECT count(*) FROM issues)`).Scan(&links, &items))
	require.Zero(t, links)
	require.Equal(t, 3, items)
	var payload []byte
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT payload FROM work_item_effects WHERE payload->'kwargs'->>'event'='milestone'`).Scan(&payload))
	var effect map[string]any
	require.NoError(t, json.Unmarshal(payload, &effect))
	require.Equal(t, "plane.bgtasks.webhook_task.webhook_activity", effect["name"])
	kwargs := effect["kwargs"].(map[string]any)
	require.Equal(t, "deleted", kwargs["verb"])
	require.Equal(t, id.String(), kwargs["event_id"])
	var removals int
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects WHERE kind='activity' AND payload->'requested_data'->'milestone'='null'::jsonb`).Scan(&removals))
	require.Equal(t, 1, removals, "hidden remnants do not produce item history or notifications")
	f.opts.RequestID = nil
	_, err = f.repo.Delete(f.ctx, f.opts, id)
	require.ErrorIs(t, err, milestones.ErrNotFound)
	_, err = f.repo.Retrieve(f.ctx, f.opts.Scope, id, false)
	require.ErrorIs(t, err, milestones.ErrNotFound)
}
