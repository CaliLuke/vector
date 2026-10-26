//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func writeHistoryValue(t *testing.T, f projectTypeFixture, repo *workitems.Repository, method, item, field string, value any) workitems.Result {
	t.Helper()
	body, err := json.Marshal(map[string]any{"name": "History writer", "custom_properties": map[string]any{field: value}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := writeValueItem(f, repo, method, item, string(body))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertHistoryCounts(t *testing.T, f projectTypeFixture, item string, versions, activity, types int) {
	t.Helper()
	var gotVersions, gotActivity, gotTypes int
	err := f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM work_item_property_versions WHERE issue_id=$1),(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='custom_property'),(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='type')`, item).Scan(&gotVersions, &gotActivity, &gotTypes)
	if err != nil || gotVersions != versions || gotActivity != activity || gotTypes != types {
		t.Fatalf("history versions/activity/type=%d/%d/%d want %d/%d/%d: %v", gotVersions, gotActivity, gotTypes, versions, activity, types, err)
	}
}

func TestPublishedWriterRecordsCreateEditNoopAndClear(t *testing.T) {
	f := newProjectTypeFixture(t)
	field := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: field, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}})
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	result := writeHistoryValue(t, f, repo, "POST", "", field, "<p>First</p>")
	item := resultIssueID(t, result)
	if result.Body.(map[string]any)["type_id"] != f.task {
		t.Fatal("omitted type did not resolve to Task")
	}
	assertHistoryCounts(t, f, item, 1, 1, 0)
	writeHistoryValue(t, f, repo, "PATCH", item, field, "<p>Second</p>")
	assertHistoryCounts(t, f, item, 2, 2, 0)
	writeHistoryValue(t, f, repo, "PATCH", item, field, "<p>Second</p>")
	assertHistoryCounts(t, f, item, 2, 2, 0)
	writeHistoryValue(t, f, repo, "PATCH", item, field, nil)
	assertHistoryCounts(t, f, item, 3, 3, 0)
	var value string
	var cleared bool
	err = f.db.QueryRow(f.ctx, `SELECT value::text,source='change' AND actor_id=$2 AND property_title='Notes' FROM work_item_property_versions WHERE issue_id=$1 ORDER BY created_at DESC,id DESC LIMIT 1`, item, f.member).Scan(&value, &cleared)
	if err != nil || value != "null" || !cleared {
		t.Fatalf("clear=%s metadata=%v error=%v", value, cleared, err)
	}
}

func TestReviewedTypeChangeRecordsSubmittedFormerValueOnce(t *testing.T) {
	f, repo, q, input, field := reviewedEditFixture(t)
	input.Changes["custom_properties"] = map[string]any{field: "<p>Edited former value</p>"}
	preview, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, input.ItemID, 0, 0, 0)
	input.ReviewID = preview.Body.(workitems.ItemEditReview).ID
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, input.ItemID, 2, 1, 1)
	var before, preserved string
	err = f.db.QueryRow(f.ctx, `SELECT (SELECT value#>>'{}' FROM work_item_property_versions WHERE issue_id=$1 AND source='baseline' AND actor_id IS NULL),(SELECT value#>>'{}' FROM work_item_property_versions WHERE issue_id=$1 AND source='preservation' AND property_title='Old details')`, input.ItemID).Scan(&before, &preserved)
	if err != nil || before != "<p>Preserve this value</p>" || preserved != "<p>Edited former value</p>" {
		t.Fatalf("before/preserved=%q/%q: %v", before, preserved, err)
	}
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, input.ItemID, 2, 1, 1)
}

func TestReviewedTypeChangeRecordsExplicitFormerClear(t *testing.T) {
	f, repo, q, input, field := reviewedEditFixture(t)
	input.Changes["custom_properties"] = map[string]any{field: nil}
	preview, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	review := preview.Body.(workitems.ItemEditReview)
	if review.PreservedProperties != 0 {
		t.Fatal("explicitly cleared value was preserved")
	}
	input.ReviewID = review.ID
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, input.ItemID, 2, 1, 1)
	var clear bool
	err = f.db.QueryRow(f.ctx, `SELECT count(*)=1 FROM work_item_property_versions WHERE issue_id=$1 AND source='change' AND value='null'::jsonb`, input.ItemID).Scan(&clear)
	if err != nil || !clear {
		t.Fatalf("clear missing: %v %v", clear, err)
	}
}

func TestPublishedHistoryRollsBackAfterLateEffectFailure(t *testing.T) {
	f, repo, q, input, _ := reviewedEditFixture(t)
	preview, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = preview.Body.(workitems.ItemEditReview).ID
	f.exec(`CREATE FUNCTION reject_history_effect() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'late effect failure'; END $$; CREATE TRIGGER reject_history_effect BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_history_effect()`)
	if _, err = repo.CommitItemChange(f.ctx, q, input); err == nil {
		t.Fatal("expected rejected effect")
	}
	assertHistoryCounts(t, f, input.ItemID, 0, 0, 0)
	f.exec(`DROP TRIGGER reject_history_effect ON work_item_effects`)
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, input.ItemID, 1, 1, 1)
}

func TestDraftPublicationStartsHistoryOnlyAtPublication(t *testing.T) {
	f := newProjectTypeFixture(t)
	field := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: field, Title: "Draft notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}})
	repo, draft, _ := seedPublicationDraft(t, f)
	f.exec(`UPDATE draft_issues SET custom_properties=jsonb_build_object($2::text,'<p>Publish me</p>') WHERE id=$1`, draft, field)
	assertHistoryCounts(t, f, draft, 0, 0, 0)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, result)
	assertHistoryCounts(t, f, item, 1, 1, 0)
	retry, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil || resultIssueID(t, retry) != item {
		t.Fatalf("retry=%+v error=%v", retry, err)
	}
	assertHistoryCounts(t, f, item, 1, 1, 0)
}

func TestReviewedPublicationCapturesFormerDraftField(t *testing.T) {
	f := newProjectTypeFixture(t)
	field := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: field, Title: "Former draft notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}})
	f.mutate(t, "import", f.bug)
	repo, draft, _ := seedPublicationDraft(t, f)
	f.exec(`UPDATE draft_issues SET custom_properties=jsonb_build_object($2::text,'<p>Former draft content</p>') WHERE id=$1`, draft, field)
	input := workitems.ItemEditInput{Mode: "publish", ItemID: draft, ProjectID: f.project, Revision: f.catalog(t).Revision, Changes: map[string]any{"type_id": f.bug}}
	q := workitems.Request{ActorID: f.member, Slug: "types", Timezone: "UTC"}
	setEditSourceVersion(t, f, repo, q, &input)
	preview, err := repo.PreviewItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, draft, 0, 0, 0)
	input.ReviewID = preview.Body.(workitems.ItemEditReview).ID
	result, err := repo.CommitItemChange(f.ctx, q, input)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, result)
	assertHistoryCounts(t, f, item, 1, 1, 0)
	var retained bool
	err = f.db.QueryRow(f.ctx, `SELECT count(*)=1 FROM work_item_property_versions WHERE issue_id=$1 AND value='"<p>Former draft content</p>"'::jsonb AND source='preservation' AND property_title='Former draft notes'`, item).Scan(&retained)
	if err != nil || !retained {
		t.Fatalf("former field lost: %v %v", retained, err)
	}
	if _, err = repo.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	assertHistoryCounts(t, f, item, 1, 1, 0)
}
