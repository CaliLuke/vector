//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func editableCascade() property.Definition {
	root := uuid.New().String()
	return property.Definition{ID: uuid.New().String(), Title: "Location", Kind: property.Cascading, Configuration: property.Configuration{
		Active:  true,
		Levels:  []property.Level{{ID: uuid.New().String(), Name: "Country"}, {ID: uuid.New().String(), Name: "City"}},
		Options: []property.Option{{ID: root, Name: "Canada"}, {ID: uuid.New().String(), Name: "Toronto", Parent: root, Level: 1}, {ID: uuid.New().String(), Name: "Ottawa", Parent: root, Level: 1}},
	}}
}

func TestReviewedCascadeEditPreservesOnlyAffectedSelectionsAndReplaysOnce(t *testing.T) {
	f := newProjectTypeFixture(t)
	def := editableCascade()
	attachValueProperties(t, f, def)
	repo, _, _ := seedPublicationDraft(t, f)
	root, removed, retained := def.Configuration.Options[0].ID, def.Configuration.Options[1].ID, def.Configuration.Options[2].ID
	changed, err := writeValueItem(f, repo, "POST", "", `{"name":"Changed","description_html":"<p>Keep</p>","custom_properties":{"`+def.ID+`":["`+root+`","`+removed+`"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	untouched, err := writeValueItem(f, repo, "POST", "", `{"name":"Unchanged","custom_properties":{"`+def.ID+`":["`+root+`","`+retained+`"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	changedID, retainedID := resultIssueID(t, changed), resultIssueID(t, untouched)
	f.exec(`UPDATE issues SET archived_at=now() WHERE id=$1`, changedID)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	proposed := catalog.Properties[0].Definition
	proposed.Title = "Region"
	proposed.Configuration.Options = []property.Option{def.Configuration.Options[0], def.Configuration.Options[2]}
	command := types.PropertyInput{Action: "update", Revision: catalog.Revision, Definition: proposed}
	_, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 409)

	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	if review.Items != 1 || review.Types != 1 || review.Projects != 1 {
		t.Fatalf("impact: %+v", review)
	}
	beforeCommit, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || beforeCommit.Revision != catalog.Revision || beforeCommit.Properties[0].Definition.Title != "Location" {
		t.Fatalf("preview changed definition: %+v %v", beforeCommit, err)
	}
	command.ReviewID = review.ID
	updated, err := f.repo.MutateProperty(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatal(err)
	}
	if len(updated.Properties) != 1 || updated.Properties[0].Definition.Title != "Region" || len(updated.Properties[0].Definition.Configuration.Options) != 2 {
		t.Fatalf("definition not updated: %+v", updated.Properties)
	}
	var description string
	var activities int
	if err = f.db.QueryRow(f.ctx, `SELECT description_html,(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='custom_properties') FROM issues WHERE id=$1`, changedID).Scan(&description, &activities); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(description, "<p>Keep</p>") || strings.Count(description, "Canada → Toronto") != 1 || !strings.Contains(description, "Location configuration changed") || activities != 1 {
		t.Fatalf("preservation: %q activities=%d", description, activities)
	}
	if _, exists := draftValues(t, f, "issues", changedID)[def.ID]; exists {
		t.Fatal("removed selection remains attached")
	}
	if !reflect.DeepEqual(draftValues(t, f, "issues", retainedID)[def.ID], []any{root, retained}) {
		t.Fatal("unaffected selection changed")
	}
}

func TestCascadeEditReviewBindsTheDefinitionAndCurrentValues(t *testing.T) {
	f := newProjectTypeFixture(t)
	def := editableCascade()
	attachValueProperties(t, f, def)
	repo, _, _ := seedPublicationDraft(t, f)
	root, leaf := def.Configuration.Options[0].ID, def.Configuration.Options[1].ID
	result, err := writeValueItem(f, repo, "POST", "", `{"name":"Review race","custom_properties":{"`+def.ID+`":["`+root+`","`+leaf+`"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, result)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	proposed := catalog.Properties[0].Definition
	proposed.Configuration.Options = proposed.Configuration.Options[:1]
	command := types.PropertyInput{Action: "update", Revision: catalog.Revision, Definition: proposed}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	command.ReviewID = review.ID
	forged := command
	forged.Definition.Title = "Changed after review"
	_, err = f.repo.MutateProperty(f.ctx, f.owner, "types", forged)
	requireTypeStatus(t, err, 409)
	f.exec(`UPDATE issues SET description_html='<p>New description</p>',updated_at=clock_timestamp() WHERE id=$1`, item)
	_, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 409)
	actual, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || actual.Revision != catalog.Revision || len(actual.Properties[0].Definition.Configuration.Options) != 3 {
		t.Fatalf("stale update altered schema: %+v %v", actual, err)
	}
	if !reflect.DeepEqual(draftValues(t, f, "issues", item)[def.ID], []any{root, leaf}) {
		t.Fatal("rejected review lost selected path")
	}
}

func TestCascadeEditRejectsDiscardAndUsedStructuralChanges(t *testing.T) {
	f := newProjectTypeFixture(t)
	def := editableCascade()
	attachValueProperties(t, f, def)
	repo, _, _ := seedPublicationDraft(t, f)
	value, _ := json.Marshal([]string{def.Configuration.Options[0].ID})
	if _, err := writeValueItem(f, repo, "POST", "", `{"name":"Used","custom_properties":{"`+def.ID+`":`+string(value)+`}}`); err != nil {
		t.Fatal(err)
	}
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Action: "update", Revision: catalog.Revision, Definition: catalog.Properties[0].Definition, Permanent: true}
	_, err = f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 422)
	command.Permanent = false
	command.Definition.Kind = property.Number
	_, err = f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 422)
}

func TestContextFallbackReviewRollsBackDefinitionAndPreservationThenRetries(t *testing.T) {
	f := newProjectTypeFixture(t)
	def := editableCascade()
	root, leaf := def.Configuration.Options[0].ID, def.Configuration.Options[1].ID
	def.Contexts = []property.Context{{ID: uuid.New().String(), Name: "Local catalog", Projects: []string{f.project}, Overrides: []string{"options"}, Configuration: property.Configuration{Options: []property.Option{
		{ID: root, Name: "Former country"}, {ID: leaf, Name: "Former city", Parent: root, Level: 1},
	}}}}
	attachValueProperties(t, f, def)
	repo, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Context fallback","description_html":"<p>Original</p>","custom_properties":{"`+def.ID+`":["`+root+`","`+leaf+`"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, created)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	proposed := catalog.Properties[0].Definition
	proposed.Contexts = nil
	command := types.PropertyInput{Action: "update", Revision: catalog.Revision, Definition: proposed}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	if review.Items != 1 {
		t.Fatalf("context fallback failed to detect old catalog: %+v", review)
	}
	command.ReviewID = review.ID
	f.exec(`CREATE FUNCTION reject_schema_activity() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'reject schema activity'; END $$; CREATE TRIGGER reject_schema_activity BEFORE INSERT ON issue_activities FOR EACH ROW WHEN (NEW.field='custom_properties') EXECUTE FUNCTION reject_schema_activity()`)
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err == nil {
		t.Fatal("forced activity failure succeeded")
	}
	after, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || after.Revision != catalog.Revision || len(after.Properties[0].Definition.Contexts) != 1 {
		t.Fatalf("definition did not roll back: %+v %v", after, err)
	}
	var description string
	var committed bool
	var activities int
	if err = f.db.QueryRow(f.ctx, `SELECT description_html,(SELECT committed_at IS NOT NULL FROM work_item_schema_reviews WHERE id=$2),(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='custom_properties') FROM issues WHERE id=$1`, item, review.ID).Scan(&description, &committed, &activities); err != nil {
		t.Fatal(err)
	}
	if description != "<p>Original</p>" || committed || activities != 0 || !reflect.DeepEqual(draftValues(t, f, "issues", item)[def.ID], []any{root, leaf}) {
		t.Fatalf("partial migration survived: %q committed=%v activities=%d", description, committed, activities)
	}
	f.exec(`DROP TRIGGER reject_schema_activity ON issue_activities`)
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT description_html FROM issues WHERE id=$1`, item).Scan(&description); err != nil {
		t.Fatal(err)
	}
	if strings.Count(description, "Former country → Former city") != 1 || strings.Contains(description, "Canada") {
		t.Fatalf("preservation used new catalog or duplicated: %s", description)
	}
}
