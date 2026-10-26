//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestReviewedPropertyRemovalPreservesArchivedItemsAndPrivateDraftsOnce(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Risk <level>", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	repo, draft, _ := seedPublicationDraft(t, f)
	if err := editValueDraft(f, repo, draft, `{"custom_properties":{"`+id+`":0}}`); err != nil {
		t.Fatal(err)
	}
	result, err := writeValueItem(f, repo, "POST", "", `{"name":"Archived","description_html":"<p>Keep</p>","custom_properties":{"`+id+`":7}}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, result)
	f.exec(`UPDATE issues SET archived_at=now() WHERE id=$1`, item)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Action: "delete", ID: id, Revision: catalog.Revision}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	if review.Items != 2 || review.Types != 1 || review.Projects != 1 {
		t.Fatalf("impact %+v", review)
	}
	raw, _ := json.Marshal(review)
	if strings.Contains(string(raw), draft) || strings.Contains(string(raw), "Archived") || strings.Contains(string(raw), "Risk") {
		t.Fatalf("review exposed item content: %s", raw)
	}
	command.ReviewID = review.ID
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatalf("retry: %v", err)
	}
	var description string
	var activity int
	if err = f.db.QueryRow(f.ctx, `SELECT description_html,(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='custom_properties') FROM issues WHERE id=$1`, item).Scan(&description, &activity); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(description, "<p>Keep</p>") || strings.Count(description, "<p>7</p>") != 1 || activity != 1 {
		t.Fatalf("preservation duplicated/lost: %q activity=%d", description, activity)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT description_html FROM draft_issues WHERE id=$1`, draft).Scan(&description); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(description, "<p>0</p>") {
		t.Fatalf("lost falsy draft value: %s", description)
	}
	if len(draftValues(t, f, "issues", item)) != 0 || len(draftValues(t, f, "draft_issues", draft)) != 0 {
		t.Fatal("removed values remain attached")
	}
}

func TestPropertyRemovalReviewRejectsChangedItemsAndOtherActors(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Notes", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}})
	repo, draft, _ := seedPublicationDraft(t, f)
	if err := editValueDraft(f, repo, draft, `{"custom_properties":{"`+id+`":"before"}}`); err != nil {
		t.Fatal(err)
	}
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Action: "delete", ID: id, Revision: catalog.Revision}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	command.ReviewID = review.ID
	_, err = f.repo.MutateProperty(f.ctx, f.member, "types", command)
	requireTypeStatus(t, err, 403)
	if err = editValueDraft(f, repo, draft, `{"custom_properties":{"`+id+`":"after"}}`); err != nil {
		t.Fatal(err)
	}
	_, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 409)
	if draftValues(t, f, "draft_issues", draft)[id] != "after" {
		t.Fatal("stale review altered current values")
	}
	var count int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_properties WHERE id=$1`, id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("stale removal changed schema: %d %v", count, err)
	}
}

func TestPropertyRemovalRejectsOversizedPreservationWithoutChangingValues(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Notes", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Paragraph"}})
	repo, draft, _ := seedPublicationDraft(t, f)
	if err := editValueDraft(f, repo, draft, `{"custom_properties":{"`+id+`":"Retain me"}}`); err != nil {
		t.Fatal(err)
	}
	description := "<p>" + strings.Repeat("x", content.MaxContentBytes-8) + "</p>"
	f.exec(`UPDATE draft_issues SET description_html=$2 WHERE id=$1`, draft, description)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Action: "delete", ID: id, Revision: catalog.Revision}
	_, err = f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 422)
	if draftValues(t, f, "draft_issues", draft)[id] != "Retain me" {
		t.Fatal("rejected preview changed value")
	}
	var actual string
	if err = f.db.QueryRow(f.ctx, `SELECT description_html FROM draft_issues WHERE id=$1`, draft).Scan(&actual); err != nil {
		t.Fatal(err)
	}
	if actual != description {
		t.Fatal("rejected preview changed existing document")
	}
}

func TestPropertyRemovalWithoutPreservationRetainsOnlyExistingHistory(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}})
	repo, _, _ := seedPublicationDraft(t, f)
	result, err := writeValueItem(f, repo, "POST", "", `{"name":"History","description_html":"<p>Keep</p>","custom_properties":{"`+id+`":"<p>Discard current value</p>"}}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, result)
	f.exec(`INSERT INTO work_item_property_versions(id,workspace_id,issue_id,property_id,value,search_text,actor_id) VALUES(gen_random_uuid(),$1,$2,$3,'"<p>Earlier version</p>"','Earlier version',$4)`, f.workspace, item, id, f.member)
	var before int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_property_versions WHERE issue_id=$1 AND property_id=$2`, item, id).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before == 0 {
		t.Fatal("fixture has no history")
	}
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Action: "delete", ID: id, Revision: catalog.Revision, Permanent: true}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	command.ReviewID = review.ID
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatal(err)
	}
	var after int
	var description, oldValue, newValue string
	if err = f.db.QueryRow(f.ctx, `SELECT description_html,(SELECT count(*) FROM work_item_property_versions WHERE issue_id=$1 AND property_id=$2) FROM issues WHERE id=$1`, item, id).Scan(&description, &after); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT old_value,new_value FROM issue_activities WHERE issue_id=$1 AND field='custom_properties' ORDER BY created_at DESC LIMIT 1`, item).Scan(&oldValue, &newValue); err != nil {
		t.Fatal(err)
	}
	if description != "<p>Keep</p>" || after != before || oldValue != "" || !strings.Contains(newValue, "Existing history retained") || len(draftValues(t, f, "issues", item)) != 0 {
		t.Fatalf("retention contract violated: %q versions=%d/%d activity=%q,%q", description, before, after, oldValue, newValue)
	}
}

func TestPropertyRemovalRollsBackAllItemsAndSchemaOnFailure(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Risk", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	repo, draft, _ := seedPublicationDraft(t, f)
	if err := editValueDraft(f, repo, draft, `{"custom_properties":{"`+id+`":3}}`); err != nil {
		t.Fatal(err)
	}
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Atomic removal","description_html":"<p>Keep</p>","custom_properties":{"`+id+`":7}}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, created)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Action: "delete", ID: id, Revision: catalog.Revision}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	command.ReviewID = review.ID
	f.exec(`CREATE FUNCTION reject_preservation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'reject preservation'; END $$; CREATE TRIGGER reject_preservation BEFORE UPDATE ON draft_issues FOR EACH ROW EXECUTE FUNCTION reject_preservation()`)
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err == nil {
		t.Fatal("forced failure succeeded")
	}
	var description string
	var activity, definitions int
	var committed bool
	if err = f.db.QueryRow(f.ctx, `SELECT description_html,(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='custom_properties'),(SELECT count(*) FROM work_item_properties WHERE id=$2),(SELECT committed_at IS NOT NULL FROM work_item_schema_reviews WHERE id=$3) FROM issues WHERE id=$1`, item, id, review.ID).Scan(&description, &activity, &definitions, &committed); err != nil {
		t.Fatal(err)
	}
	if description != "<p>Keep</p>" || activity != 0 || definitions != 1 || committed {
		t.Fatalf("partial removal survived: %q activities=%d definitions=%d committed=%v", description, activity, definitions, committed)
	}
	if draftValues(t, f, "issues", item)[id] != float64(7) || draftValues(t, f, "draft_issues", draft)[id] != float64(3) {
		t.Fatal("failed removal changed values")
	}
	f.exec(`DROP TRIGGER reject_preservation ON draft_issues`)
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatalf("same review could not retry after rollback: %v", err)
	}
}
