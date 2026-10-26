//go:build integration

package workitemtypes_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/jackc/pgx/v5/pgxpool"
)

type effectAdmissionRecorder struct {
	mu       sync.Mutex
	calls    map[string]int
	failNext bool
}

type fileAssetDeleteAdmissionRecorder struct{ recorder *effectAdmissionRecorder }

func (r fileAssetDeleteAdmissionRecorder) Admit(_ context.Context, _, _, eventID string) error {
	return r.recorder.record(eventID)
}

type fileAssetMetadataAdmissionRecorder struct{ recorder *effectAdmissionRecorder }

func (r fileAssetMetadataAdmissionRecorder) Admit(_ context.Context, _, eventID string) (string, error) {
	return eventID, r.recorder.record(eventID)
}

func (r *effectAdmissionRecorder) record(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls[id]++
	if r.failNext {
		r.failNext = false
		return errors.New("admission unavailable")
	}
	return nil
}
func (r *effectAdmissionRecorder) AdmitIssueActivityTask(_ context.Context, id string, _ map[string]any) error {
	return r.record(id)
}
func (r *effectAdmissionRecorder) AdmitIssueDescriptionVersion(_ context.Context, id, _, _, _ string, _ bool) error {
	return r.record(id)
}
func (r *effectAdmissionRecorder) AdmitWebhookTask(_ context.Context, id, _ string, _ []any, _ map[string]any) error {
	return r.record(id)
}
func (r *effectAdmissionRecorder) Admit(_ context.Context, input backgroundtasks.DeletionInput) (string, error) {
	return input.EventID, r.record(input.EventID)
}

func effectStoreForTest(t *testing.T, f typeFixture, recorder *effectAdmissionRecorder) *backgroundtasks.PostgresWorkItemEffectStore {
	t.Helper()
	pool, err := pgxpool.New(f.ctx, f.db.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store, err := backgroundtasks.NewPostgresWorkItemEffectStore(pool, recorder, recorder, recorder, recorder, fileAssetDeleteAdmissionRecorder{recorder}, fileAssetMetadataAdmissionRecorder{recorder})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestWorkItemEffectsRetrySameIdentityWithoutStarvingOtherRows(t *testing.T) {
	f := newTypeFixture(t)
	first, second, third, fourth, fifth, sixth := uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO work_item_effects(id,kind,payload,next_attempt_at) VALUES($1,'activity','{"type":"issue.activity.created"}',now()-interval '3 minutes'),($2,'description_version','{"snapshot":"{}","issue_id":"test"}',now()-interval '2 minutes'),($3,'soft_delete','{"app":"db","model":"draftissue","id":"test"}',now()-interval '1 minute')`, first, second, third)
	f.exec(`INSERT INTO work_item_effects(id,kind,payload) VALUES($1,'webhook','{"name":"plane.bgtasks.webhook_task.model_activity","args":[],"kwargs":{"model_name":"issue"}}')`, fourth)
	f.exec(`INSERT INTO work_item_effects(id,kind,payload) VALUES($1,'asset_storage_delete','{"asset_id":"00000000-0000-4000-8000-000000000011","object_key":"workspace/file.png"}'),($2,'asset_storage_metadata','{"asset_id":"00000000-0000-4000-8000-000000000012"}')`, fifth, sixth)
	recorder := &effectAdmissionRecorder{calls: map[string]int{}, failNext: true}
	store := effectStoreForTest(t, f, recorder)
	count, err := store.Drain(f.ctx)
	if err != nil || count != 6 || recorder.calls[first] != 1 || recorder.calls[second] != 1 || recorder.calls[third] != 1 || recorder.calls[fourth] != 1 || recorder.calls[fifth] != 1 || recorder.calls[sixth] != 1 {
		t.Fatalf("failed row starved later effects: %d %v %#v", count, err, recorder.calls)
	}
	var admitted, retained, retryDelayed bool
	var attempts int
	if err = f.db.QueryRow(f.ctx, `SELECT admitted_at IS NOT NULL,payload<>'{}',next_attempt_at>clock_timestamp(),attempts FROM work_item_effects WHERE id=$1`, first).Scan(&admitted, &retained, &retryDelayed, &attempts); err != nil || admitted || !retained || !retryDelayed || attempts != 1 {
		t.Fatalf("failed effect lost: admitted=%v retained=%v delayed=%v attempts=%d %v", admitted, retained, retryDelayed, attempts, err)
	}
	f.exec(`UPDATE work_item_effects SET next_attempt_at=now() WHERE id=$1`, first)
	// A new consumer after restart sees the same pending effect identity.
	count, err = effectStoreForTest(t, f, recorder).Drain(f.ctx)
	if err != nil || count != 1 || recorder.calls[first] != 2 || len(recorder.calls) != 6 {
		t.Fatalf("retry identity changed: %d %v %#v", count, err, recorder.calls)
	}
	var pending, payloads int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FILTER (WHERE admitted_at IS NULL),count(*) FILTER (WHERE payload<>'{}') FROM work_item_effects`).Scan(&pending, &payloads); err != nil || pending != 0 || payloads != 0 {
		t.Fatalf("admitted private payloads retained: pending=%d payloads=%d %v", pending, payloads, err)
	}
}

func TestWorkItemEffectAcknowledgementFailureReusesAdmissionIdentity(t *testing.T) {
	f := newTypeFixture(t)
	id := uuid.New().String()
	f.exec(`INSERT INTO work_item_effects(id,kind,payload) VALUES($1,'activity','{"type":"issue.activity.created"}')`, id)
	f.exec(`CREATE FUNCTION fail_effect_ack() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected acknowledgement failure'; END $$; CREATE TRIGGER fail_effect_ack BEFORE UPDATE ON work_item_effects FOR EACH STATEMENT EXECUTE FUNCTION fail_effect_ack()`)
	recorder := &effectAdmissionRecorder{calls: map[string]int{}}
	store := effectStoreForTest(t, f, recorder)
	if _, err := store.Drain(f.ctx); err == nil || recorder.calls[id] != 1 {
		t.Fatalf("acknowledgement failure not exercised: %v %#v", err, recorder.calls)
	}
	f.exec(`DROP TRIGGER fail_effect_ack ON work_item_effects; DROP FUNCTION fail_effect_ack()`)
	if count, err := store.Drain(f.ctx); err != nil || count != 1 || recorder.calls[id] != 2 || len(recorder.calls) != 1 {
		t.Fatalf("uncertain admission acquired another identity: %d %v %#v", count, err, recorder.calls)
	}
}
