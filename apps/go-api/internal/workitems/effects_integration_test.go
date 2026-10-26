//go:build integration

package workitems

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
)

func TestTransactionalEffectsCommitTogetherWithoutEarlyAdmission(t *testing.T) {
	ctx := context.Background()
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pair.Close() }()
	conn, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	for _, commit := range []bool{false, true} {
		tx, beginErr := conn.Begin(ctx)
		if beginErr != nil {
			t.Fatal(beginErr)
		}
		legacy := &versionTaskRecorder{}
		native := &nativeVersionRecorder{}
		repo := &Repository{db: tx, publisher: legacy, issueDescriptionVersions: native, transactionalEffects: true}
		if err = repo.publishTask(ctx, "plane.bgtasks.issue_activities_task.issue_activity", []any{}, map[string]any{"type": "issue.activity.updated", "requested_data": "private"}); err != nil {
			t.Fatal(err)
		}
		if err = repo.publishTask(ctx, "plane.bgtasks.webhook_task.model_activity", []any{}, map[string]any{"model_name": "issue"}); err != nil {
			t.Fatal(err)
		}
		if err = repo.admitIssueDescriptionVersion(ctx, map[string]any{"name": "Private title"}, "issue", "actor", false); err != nil {
			t.Fatal(err)
		}
		if err = repo.softDelete(ctx, "issue", "issue"); err != nil {
			t.Fatal(err)
		}
		if len(legacy.names) != 0 || native.calls != 0 {
			t.Fatal("transaction admitted uncommitted effects")
		}
		var raw []byte
		if err = tx.QueryRow(ctx, `SELECT jsonb_object_agg(kind,payload) FROM work_item_effects`).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var payloads map[string]map[string]any
		if err = json.Unmarshal(raw, &payloads); err != nil {
			t.Fatal(err)
		}
		if len(payloads) != 4 || payloads["webhook"]["name"] != "plane.bgtasks.webhook_task.model_activity" || payloads["description_version"]["snapshot"] != `{"name":"Private title"}` || payloads["soft_delete"]["model"] != "issue" || payloads["activity"]["requested_data"] != "private" {
			t.Fatalf("effect contract lost: %#v", payloads)
		}
		if commit {
			err = tx.Commit(ctx)
		} else {
			err = tx.Rollback(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		var count int
		if err = conn.QueryRow(ctx, `SELECT count(*) FROM work_item_effects`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		want := 0
		if commit {
			want = 4
		}
		if count != want {
			t.Fatalf("commit=%v persisted %d effects, want %d", commit, count, want)
		}
	}
}
