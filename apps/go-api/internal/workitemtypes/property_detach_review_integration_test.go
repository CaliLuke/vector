//go:build integration

package workitemtypes_test

import (
	"reflect"
	"strings"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestReviewedDetachPreservesOnlyItsTypeAndLeavesPreviewCatalogIntact(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Notes", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}})
	project := f.catalog(t)
	if _, err := f.repo.MutateProject(f.ctx, f.owner, "types", f.project, types.ProjectInput{Revision: project.Revision, Action: "import", TypeIDs: []string{f.bug}}); err != nil {
		t.Fatal(err)
	}
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: f.bug, PropertyIDs: []string{id}})
	if err != nil {
		t.Fatal(err)
	}
	repo, _, _ := seedPublicationDraft(t, f)
	changed, err := writeValueItem(f, repo, "POST", "", `{"name":"Task notes","description_html":"<p>Keep</p>","custom_properties":{"`+id+`":"Task value"}}`)
	if err != nil {
		t.Fatal(err)
	}
	untouched, err := writeValueItem(f, repo, "POST", "", `{"name":"Bug notes","type_id":"`+f.bug+`","custom_properties":{"`+id+`":"Bug value"}}`)
	if err != nil {
		t.Fatal(err)
	}
	command := types.PropertyInput{Revision: catalog.Revision, Action: "detach", ID: id, TypeID: f.task}
	review, err := f.repo.PreviewPropertyRemoval(f.ctx, f.owner, "types", command)
	if err != nil || review.Items != 1 || review.Types != 1 || review.Projects != 1 {
		t.Fatalf("scoped review: %+v %v", review, err)
	}
	afterPreview, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || !reflect.DeepEqual(catalog.Attachments, afterPreview.Attachments) || afterPreview.Revision != catalog.Revision {
		t.Fatalf("preview changed attachments: %+v %v", afterPreview, err)
	}
	changedID, untouchedID := resultIssueID(t, changed), resultIssueID(t, untouched)
	if draftValues(t, f, "issues", changedID)[id] != "Task value" {
		t.Fatal("preview removed a value")
	}
	command.ReviewID = review.ID
	for range 2 {
		if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", command); err != nil {
			t.Fatal(err)
		}
	}
	var description string
	var remainingAttachments, definitions, activities int
	if err = f.db.QueryRow(f.ctx, `SELECT description_html,(SELECT count(*) FROM work_item_type_properties WHERE property_id=$2),(SELECT count(*) FROM work_item_properties WHERE id=$2),(SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='custom_properties') FROM issues WHERE id=$1`, changedID, id).Scan(&description, &remainingAttachments, &definitions, &activities); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(description, "<p>Keep</p>") || strings.Count(description, "Task value") != 1 || activities != 1 || remainingAttachments != 1 || definitions != 1 {
		t.Fatalf("detach/retry altered the shared definition or duplicated preservation: %q %d/%d/%d", description, activities, remainingAttachments, definitions)
	}
	if _, exists := draftValues(t, f, "issues", changedID)[id]; exists {
		t.Fatal("detached current value remains")
	}
	if draftValues(t, f, "issues", untouchedID)[id] != "Bug value" {
		t.Fatal("other type's value changed")
	}
}
