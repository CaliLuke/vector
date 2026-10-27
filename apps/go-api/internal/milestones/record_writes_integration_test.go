//go:build integration

package milestones_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMilestoneCreateNormalizesFieldsAndReplaysOneCommittedEffect(t *testing.T) {
	f := newWriteFixture(t)
	requestID := uuid.New()
	f.opts.RequestID = &requestID
	fields := milestones.Fields{Title: textField(" \tRelease\r\n"), TargetDate: textField("2026-12-31"), ExternalSource: textField(" source ")}
	created, err := f.repo.Create(f.ctx, f.opts, fields)
	require.NoError(t, err)
	require.Equal(t, "Release", created.Result.Title)
	require.Equal(t, "2026-12-31", *created.Result.TargetDate)
	require.Equal(t, " source ", *created.Result.ExternalSource)
	require.Nil(t, created.Result.ExternalID)
	require.Equal(t, "milestone:"+created.MilestoneID+":1", created.Revision)
	require.False(t, *created.Replayed)
	require.NotNil(t, created.ReplayExpiresAt)
	fields.Title = textField("Release")
	replayed, err := f.repo.Create(f.ctx, f.opts, fields)
	require.NoError(t, err)
	require.True(t, *replayed.Replayed)
	require.Equal(t, created.Result, replayed.Result)
	require.Equal(t, created.MilestoneID, replayed.MilestoneID)
	f.counts(t, 1, 1, 1)
	fields.ExternalID = milestones.StringField{Present: true}
	_, err = f.repo.Create(f.ctx, f.opts, fields)
	require.ErrorIs(t, err, mutationreceipts.ErrConflict, "omission and explicit null remain different receipt inputs")
	f.opts.RequestID = nil
	_, err = f.repo.Create(f.ctx, f.opts, fields)
	require.ErrorIs(t, err, milestones.ErrConflict, "normalized live title is unique")
	f.counts(t, 1, 1, 1)
	var payload []byte
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT payload FROM work_item_effects`).Scan(&payload))
	var effect map[string]any
	require.NoError(t, json.Unmarshal(payload, &effect))
	require.Equal(t, "plane.bgtasks.webhook_task.model_activity", effect["name"])
	kwargs := effect["kwargs"].(map[string]any)
	require.Equal(t, "milestone", kwargs["model_name"])
	require.Equal(t, f.opts.WorkspaceID.String(), kwargs["workspace_id"])
	require.Nil(t, kwargs["current_instance"])
}

func TestMilestoneUpdateNoopClearRevisionAndReceiptOrdering(t *testing.T) {
	f := newWriteFixture(t)
	created, err := f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: textField("Release"), TargetDate: textField("2026-12-31"), ExternalSource: textField(" source ")})
	require.NoError(t, err)
	id := uuid.MustParse(created.MilestoneID)
	f.opts.ExpectedRevision = &created.Revision
	unchanged, err := f.repo.Update(f.ctx, f.opts, id, milestones.Fields{Title: textField(" Release ")})
	require.NoError(t, err)
	require.Equal(t, created, unchanged)
	f.counts(t, 1, 1, 0)
	requestID := uuid.New()
	f.opts.RequestID = &requestID
	patch := milestones.Fields{Title: textField("Next"), TargetDate: milestones.StringField{Present: true}, ExternalID: textField("")}
	updated, err := f.repo.Update(f.ctx, f.opts, id, patch)
	require.NoError(t, err)
	require.Equal(t, "milestone:"+id.String()+":2", updated.Revision)
	require.Nil(t, updated.Result.TargetDate)
	require.Equal(t, "", *updated.Result.ExternalID)
	require.Equal(t, " source ", *updated.Result.ExternalSource)
	replayed, err := f.repo.Update(f.ctx, f.opts, id, patch)
	require.NoError(t, err)
	require.True(t, *replayed.Replayed, "receipt recovery precedes now-stale CAS")
	require.Equal(t, updated.Result, replayed.Result)
	f.counts(t, 1, 2, 1)
	f.opts.RequestID = nil
	_, err = f.repo.Update(f.ctx, f.opts, id, milestones.Fields{Title: textField("Stale")})
	require.ErrorIs(t, err, milestones.ErrRevisionConflict)
	f.counts(t, 1, 2, 1)
	var effect []byte
	require.NoError(t, f.db.QueryRow(f.ctx, `SELECT payload->'kwargs'->'requested_data' FROM work_item_effects ORDER BY created_at DESC,id DESC LIMIT 1`).Scan(&effect))
	require.JSONEq(t, `{"title":"Next","target_date":null,"external_id":""}`, string(effect))
	f.opts.RequestID = &requestID
	_, err = f.db.Exec(f.ctx, `UPDATE workspace_members SET is_active=false`)
	require.NoError(t, err)
	_, err = f.repo.Update(f.ctx, f.opts, id, patch)
	require.ErrorIs(t, err, milestones.ErrForbidden, "replay still requires current authorization")
	f.counts(t, 1, 2, 1)
}

func TestMilestoneWriteValidationPreservesUnicodeAndNullableDistinctions(t *testing.T) {
	f := newWriteFixture(t)
	for _, fields := range []milestones.Fields{
		{}, {Title: milestones.StringField{Present: true}}, {Title: textField(" \t\r\n\v\f")},
		{Title: textField(strings.Repeat("雪", 256))}, {Title: textField("bad\x00title")},
		{Title: textField("Valid"), TargetDate: textField("")}, {Title: textField("Valid"), TargetDate: textField("2026-02-30")},
		{Title: textField("Valid"), TargetDate: textField("0000-01-01")}, {Title: textField("Valid"), TargetDate: textField("2026-1-01")},
		{Title: textField("Valid"), ExternalID: textField(strings.Repeat("雪", 256))},
	} {
		_, err := f.repo.Create(f.ctx, f.opts, fields)
		require.ErrorIs(t, err, milestones.ErrInvalidInput)
	}
	f.counts(t, 0, 0, 0)
	created, err := f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: textField(strings.Repeat("雪", 255)), TargetDate: textField("0001-01-01")})
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("雪", 255), created.Result.Title)
	_, err = f.repo.Update(f.ctx, f.opts, uuid.MustParse(created.MilestoneID), milestones.Fields{Title: milestones.StringField{Present: true}})
	require.ErrorIs(t, err, milestones.ErrInvalidInput)
	f.opts.ExpectedRevision = &created.Revision
	_, err = f.repo.Create(f.ctx, f.opts, milestones.Fields{Title: textField("Another")})
	require.ErrorIs(t, err, milestones.ErrInvalidInput)
	f.counts(t, 1, 1, 0)
}

func TestMilestoneOutboxFailureRollsBackResourceAndReceipt(t *testing.T) {
	f := newWriteFixture(t)
	_, err := f.db.Exec(f.ctx, `CREATE FUNCTION reject_milestone_effect() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'effect unavailable'; END; $$;
 CREATE TRIGGER reject_milestone_effect BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_milestone_effect();`)
	require.NoError(t, err)
	requestID := uuid.New()
	f.opts.RequestID = &requestID
	fields := milestones.Fields{Title: textField("Release")}
	_, err = f.repo.Create(f.ctx, f.opts, fields)
	require.Error(t, err)
	f.counts(t, 0, 0, 0)
	_, err = f.db.Exec(f.ctx, `DROP TRIGGER reject_milestone_effect ON work_item_effects`)
	require.NoError(t, err)
	_, err = f.repo.Create(f.ctx, f.opts, fields)
	require.NoError(t, err)
	f.counts(t, 1, 1, 1)
}
