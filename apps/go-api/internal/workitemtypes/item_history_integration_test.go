//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/go-plane/internal/workitems"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

func newHistoryRecorderFixture(t *testing.T) (projectTypeFixture, types.ItemHistoryState, string) {
	t.Helper()
	f := newProjectTypeFixture(t)
	field := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: field, Title: "Acceptance notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}})
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	result, err := writeValueItem(f, repo, "POST", "", `{"name":"History"}`)
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, result)
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	state, err := types.LoadItemHistoryState(f.ctx, tx, types.ItemHistoryState{ItemID: id, ProjectID: f.project, WorkspaceID: f.workspace, TypeID: f.task, Values: map[string]any{field: "<p>Before</p>"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,'<p>Before</p>') WHERE id=$1`, id, field)
	return f, state, field
}

func recordHistoryChange(t *testing.T, f projectTypeFixture, tx pgx.Tx, before types.ItemHistoryState, field string, value any) types.ItemHistoryState {
	t.Helper()
	after := before
	after.Values = map[string]any{field: value}
	raw, err := json.Marshal(after.Values)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(f.ctx, `UPDATE issues SET custom_properties=$2,updated_by_id=$3 WHERE id=$1`, before.ItemID, raw, f.member); err != nil {
		t.Fatal(err)
	}
	if err = types.RecordItemPropertyChanges(f.ctx, tx, f.member, &before, after); err != nil {
		t.Fatal(err)
	}
	return after
}

func TestPropertyHistoryRecordsCanonicalEditAndClear(t *testing.T) {
	f, before, field := newHistoryRecorderFixture(t)
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	after := recordHistoryChange(t, f, tx, before, field, "<p>After</p>")
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	var versions int
	var baselineActor *string
	var title, oldValue, newValue string
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_property_versions WHERE issue_id=$1`, before.ItemID).Scan(&versions); err != nil || versions != 2 {
		t.Fatalf("versions=%d, error=%v", versions, err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT actor_id::text FROM work_item_property_versions WHERE issue_id=$1 AND source='baseline'`, before.ItemID).Scan(&baselineActor); err != nil || baselineActor != nil {
		t.Fatalf("baseline author=%v, error=%v", baselineActor, err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT comment,old_value,new_value FROM issue_activities WHERE issue_id=$1 AND field='custom_property'`, before.ItemID).Scan(&title, &oldValue, &newValue); err != nil || title != "Acceptance notes" || oldValue != "Before" || newValue != "After" {
		t.Fatalf("activity=%q/%q/%q, error=%v", title, oldValue, newValue, err)
	}
	tx, err = f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	recordHistoryChange(t, f, tx, after, field, nil)
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	var cleared bool
	if err = f.db.QueryRow(f.ctx, `SELECT count(*)=3 AND count(*) FILTER (WHERE value='null'::jsonb AND actor_id=$2 AND source='change')=1 FROM work_item_property_versions WHERE issue_id=$1`, before.ItemID, f.member).Scan(&cleared); err != nil || !cleared {
		t.Fatalf("clear not recorded: %v %v", cleared, err)
	}
}

func TestPropertyHistoryRollsBackWithRejectedItemTransaction(t *testing.T) {
	f, before, field := newHistoryRecorderFixture(t)
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	recordHistoryChange(t, f, tx, before, field, "<p>Rejected</p>")
	if _, err = tx.Exec(f.ctx, `SELECT 1/0`); err == nil {
		t.Fatal("expected late transaction failure")
	}
	if err = tx.Rollback(f.ctx); err != nil {
		t.Fatal(err)
	}
	var versions, activities int
	var value string
	if err = f.db.QueryRow(f.ctx, `SELECT custom_properties->>$2,(SELECT count(*) FROM work_item_property_versions WHERE issue_id=$1),(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='custom_property') FROM issues WHERE id=$1`, before.ItemID, field).Scan(&value, &versions, &activities); err != nil || value != "<p>Before</p>" || versions != 0 || activities != 0 {
		t.Fatalf("rollback value=%q versions=%d activities=%d error=%v", value, versions, activities, err)
	}
}

func TestPropertyHistoryCapturesUnrecordedValueAfterPreservationOnlyHistory(t *testing.T) {
	f, before, field := newHistoryRecorderFixture(t)
	f.exec(`INSERT INTO work_item_property_versions(id,workspace_id,issue_id,property_id,value,search_text,actor_id,source) VALUES(gen_random_uuid(),$1,$2,$3,'"<p>Ancient</p>"','Ancient',$4,'preservation')`, f.workspace, before.ItemID, field, f.owner)
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	recordHistoryChange(t, f, tx, before, field, "<p>After</p>")
	if err = tx.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	var captured bool
	if err = f.db.QueryRow(f.ctx, `SELECT count(*)=3 AND count(*) FILTER (WHERE source='baseline' AND value='"<p>Before</p>"'::jsonb AND actor_id IS NULL)=1 FROM work_item_property_versions WHERE issue_id=$1`, before.ItemID).Scan(&captured); err != nil || !captured {
		t.Fatalf("legacy value omitted: %v %v", captured, err)
	}
}

func TestPropertyHistoryBaselineUsesPreviousDefinitionTitle(t *testing.T) {
	f, before, field := newHistoryRecorderFixture(t)
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	after := before
	after.Definitions = append([]property.Definition(nil), before.Definitions...)
	after.Definitions[0].Title = "Renamed notes"
	after.Values = map[string]any{field: "<p>After</p>"}
	if err = types.RecordItemPropertyChanges(f.ctx, tx, f.member, &before, after); err != nil {
		t.Fatal(err)
	}
	var oldTitle, newTitle string
	if err = tx.QueryRow(f.ctx, `SELECT (SELECT property_title FROM work_item_property_versions WHERE issue_id=$1 AND source='baseline'),(SELECT property_title FROM work_item_property_versions WHERE issue_id=$1 AND source='change')`, before.ItemID).Scan(&oldTitle, &newTitle); err != nil || oldTitle != "Acceptance notes" || newTitle != "Renamed notes" {
		t.Fatalf("titles=%q/%q, error=%v", oldTitle, newTitle, err)
	}
}
