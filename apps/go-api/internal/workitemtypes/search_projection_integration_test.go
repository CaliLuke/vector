//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemsearch"
	"github.com/CaliLuke/go-plane/internal/workspacesearch"
)

type richSearchResult struct {
	ID    string `json:"id"`
	Match *struct {
		PropertyID string `json:"property_id"`
		Title      string `json:"property_title"`
		Text       string `json:"text"`
	} `json:"custom_property_match"`
}

func searchRichFields(t *testing.T, f projectTypeFixture, term string) [][]richSearchResult {
	t.Helper()
	reader, err := workspacesearch.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	var all [][]richSearchResult
	for _, scope := range []string{"", f.project} {
		workspace := "true"
		if scope != "" {
			workspace = "false"
		}
		raw, queryErr := reader.Global(f.ctx, uuid.MustParse(f.member), "types", term, "issue", scope, workspace)
		if queryErr != nil {
			t.Fatal(queryErr)
		}
		var global struct {
			Results struct {
				Issues []richSearchResult `json:"issue"`
			} `json:"results"`
		}
		if err = json.Unmarshal(raw, &global); err != nil {
			t.Fatal(err)
		}
		all = append(all, global.Results.Issues)
		raw, err = reader.Entity(f.ctx, uuid.MustParse(f.member), "types", term, "issue", scope, 100)
		if err != nil {
			t.Fatal(err)
		}
		var entity struct {
			Issues []richSearchResult `json:"issue"`
		}
		if err = json.Unmarshal(raw, &entity); err != nil {
			t.Fatal(err)
		}
		all = append(all, entity.Issues)
	}
	writer, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	result, err := writer.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_search_issues", ActorID: f.member, Slug: "types", ProjectID: f.project, Params: map[string]string{"search": term}}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	var picker []richSearchResult
	if err = json.Unmarshal(raw, &picker); err != nil {
		t.Fatal(err)
	}
	return append(all, picker)
}
func expectRichSearch(t *testing.T, f projectTypeFixture, term, item, field string) {
	t.Helper()
	for n, rows := range searchRichFields(t, f, term) {
		if item == "" {
			if len(rows) != 0 {
				t.Fatalf("search %d %q leaked results: %+v", n, term, rows)
			}
			continue
		}
		if len(rows) != 1 || rows[0].ID != item || rows[0].Match == nil || rows[0].Match.PropertyID != field || len([]rune(rows[0].Match.Text)) > 240 {
			t.Fatalf("search %d %q mismatch: %+v", n, term, rows)
		}
	}
}
func TestRichTextSearchCurrentValueAndEligibility(t *testing.T) {
	f, state, field := newHistoryRecorderFixture(t)
	writer, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	setValue := func(value any) {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"custom_properties": map[string]any{field: value}})
		if _, err = writeValueItem(f, writer, "PATCH", state.ItemID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	setValue(`<p>Álpha &amp; Omega 50% under_score path\name</p><p>Second block</p><script>secret_script</script><a href="https://example.test/secret_attribute">Link</a>`)
	for _, term := range []string{"álpha & omega", "50%", "under_score", `path\name`, "Second block"} {
		expectRichSearch(t, f, term, state.ItemID, field)
	}
	for _, term := range []string{"secret_script", "secret_attribute", "Rich text", field, "underXscore", "50Z"} {
		expectRichSearch(t, f, term, "", "")
	}
	setValue("<p>Replacement phrase</p>")
	expectRichSearch(t, f, "álpha & omega", "", "")
	expectRichSearch(t, f, "Replacement phrase", state.ItemID, field)
	setValue(nil)
	expectRichSearch(t, f, "Replacement phrase", "", "")
	setValue("<p>Current needle</p>")
	// Stale projections never reveal a formerly current value.
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,'<p>Unindexed legacy</p>') WHERE id=$1`, state.ItemID, field)
	expectRichSearch(t, f, "Current needle", "", "")
	if err = workitemsearch.Backfill(f.ctx, f.db); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "Unindexed legacy", state.ItemID, field)
	for _, change := range []struct{ disable, restore string }{
		{`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, `UPDATE work_item_type_settings SET enabled=true WHERE workspace_id=$1`},
		{`UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,active}','false') WHERE workspace_id=$1`, `UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,active}','true') WHERE workspace_id=$1`},
	} {
		f.exec(change.disable, f.workspace)
		expectRichSearch(t, f, "Unindexed legacy", "", "")
		f.exec(change.restore, f.workspace)
		expectRichSearch(t, f, "Unindexed legacy", state.ItemID, field)
	}
	contexts, _ := json.Marshal([]map[string]any{{"id": uuid.New().String(), "name": "Hidden here", "project_ids": []string{f.project}, "type_ids": []string{f.task}, "override_keys": []string{"active"}, "configuration": map[string]any{"active": false}}})
	f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{contexts}',$2::jsonb) WHERE id=$1`, field, contexts)
	expectRichSearch(t, f, "Unindexed legacy", "", "")
	f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{contexts}','[]'::jsonb) WHERE id=$1`, field)
	f.exec(`UPDATE issue_types SET is_active=false WHERE id=$1`, f.task)
	expectRichSearch(t, f, "Unindexed legacy", state.ItemID, field)
	f.exec(`DELETE FROM work_item_type_properties WHERE type_id=$1 AND property_id=$2`, f.task, field)
	expectRichSearch(t, f, "Unindexed legacy", "", "")
}

func TestRichTextSearchProjectionBackfillDoesNotEditItems(t *testing.T) {
	f, state, field := newHistoryRecorderFixture(t)
	var before, after []byte
	if err := f.db.QueryRow(f.ctx, `SELECT to_jsonb(i) FROM issues i WHERE id=$1`, state.ItemID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	if err := workitemsearch.Backfill(f.ctx, f.db); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(f.ctx, `SELECT to_jsonb(i) FROM issues i WHERE id=$1`, state.ItemID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("backfill modified canonical item/source fingerprint")
	}
	assertHistoryCounts(t, f, state.ItemID, 0, 0, 0)
	f.exec(`UPDATE work_item_type_settings SET enabled=true WHERE workspace_id=$1`, f.workspace)
	expectRichSearch(t, f, "Before", state.ItemID, field)
	var completed string
	if err := f.db.QueryRow(f.ctx, `SELECT completed_at::text FROM work_item_search_backfills WHERE workspace_id=$1`, f.workspace).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if err := workitemsearch.Backfill(f.ctx, f.db); err != nil {
		t.Fatal(err)
	}
	var unchanged bool
	if err := f.db.QueryRow(f.ctx, `SELECT completed_at::text=$2 FROM work_item_search_backfills WHERE workspace_id=$1`, f.workspace, completed).Scan(&unchanged); err != nil || !unchanged {
		t.Fatalf("backfill replayed completed workspace: %v", err)
	}
	// Search snippets are plain text and bounded even for long Unicode content.
	writer, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"custom_properties": map[string]any{field: "<p>" + strings.Repeat("ø", 400) + "needle &lt;img onerror=bad&gt;" + strings.Repeat("界", 400) + "</p>"}})
	if _, err = writeValueItem(f, writer, "PATCH", state.ItemID, string(raw)); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "needle", state.ItemID, field)
}

func TestRichTextSearchRestoreAndRollback(t *testing.T) {
	f, writer, q, input, field := propertyRestoreFixture(t)
	// This fixture seeds legacy values directly; bootstrap must index those first.
	if err := workitemsearch.Backfill(f.ctx, f.db); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "Preserve this value", input.ItemID, field)
	expectRichSearch(t, f, "Earlier value", "", "")
	reviewPropertyRestore(t, f, writer, q, &input)
	if _, err := writer.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "Earlier value", input.ItemID, field)
	expectRichSearch(t, f, "Preserve this value", "", "")
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(f.ctx, `SELECT work_item_schema_lock($1::uuid)`, f.workspace); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(f.ctx, `UPDATE issues SET custom_properties=jsonb_build_object($2::text,'<p>Rolled back</p>') WHERE id=$1`, input.ItemID, field); err != nil {
		t.Fatal(err)
	}
	if err = workitemsearch.Refresh(f.ctx, tx, f.workspace, input.ItemID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Rollback(f.ctx); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "Earlier value", input.ItemID, field)
	expectRichSearch(t, f, "Rolled back", "", "")
}
