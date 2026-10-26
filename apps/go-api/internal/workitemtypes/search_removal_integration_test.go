//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemsearch"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestRichTextSearchReviewedTypeSwitchAndFieldRemoval(t *testing.T) {
	f, writer, q, input, field := reviewedEditFixture(t)
	if err := workitemsearch.Backfill(f.ctx, f.db); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "Preserve this value", input.ItemID, field)
	reviewPropertyRestore(t, f, writer, q, &input)
	if _, err := writer.CommitItemChange(f.ctx, q, input); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "Preserve this value", "", "")
	assertSearchBackfillCounts(t, f, 0, 1)
	// An independent field has its current projection removed by the reviewed
	// definition-deletion transaction, while its history remains separate.
	raw, _ := json.Marshal(map[string]any{"name": "Current field item", "type_id": f.task, "custom_properties": map[string]any{field: "<p>Delete this field value</p>"}})
	result, err := writeValueItem(f, writer, "POST", "", string(raw))
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, result)
	expectRichSearch(t, f, "Delete this field value", id, field)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Action: "delete", ID: field, Revision: catalog.Revision}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	command.ReviewID = review.ID
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "Delete this field value", "", "")
	assertSearchBackfillCounts(t, f, 0, 1)
}
func TestRichTextSearchTypeReplacementKeepsSharedFieldOnly(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	shared, removed := uuid.New().String(), uuid.New().String()
	source := f
	source.task = f.bug
	attachValueProperties(t, source,
		property.Definition{ID: shared, Title: "Shared", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: removed, Title: "Removed", Kind: property.RichText, Configuration: property.Configuration{Active: true}})
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Action: "attach", TypeID: f.task, PropertyIDs: []string{shared}, Revision: catalog.Revision}); err != nil {
		t.Fatal(err)
	}
	writer, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"name": "Replacement item", "type_id": f.bug, "custom_properties": map[string]any{shared: "<p>Retained needle</p>", removed: "<p>Preserved needle</p>"}})
	result, err := writeValueItem(f, writer, "POST", "", string(raw))
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, result)
	expectRichSearch(t, f, "Retained needle", id, shared)
	expectRichSearch(t, f, "Preserved needle", id, removed)
	command := replacementCommand(t, f)
	review, err := f.repo.PreviewTypeRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	command.ReviewID = review.ID
	if _, err = f.repo.CommitTypeRemoval(f.ctx, f.owner, "types", command); err != nil {
		t.Fatal(err)
	}
	expectRichSearch(t, f, "Retained needle", id, shared)
	expectRichSearch(t, f, "Preserved needle", "", "")
	assertSearchBackfillCounts(t, f, 1, 0)
}
