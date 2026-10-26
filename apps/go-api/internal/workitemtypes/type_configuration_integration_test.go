//go:build integration

package workitemtypes_test

import (
	"reflect"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestUnusedTypeDeletionPrunesContextsWithoutBroadeningScope(t *testing.T) {
	f := newProjectTypeFixture(t)
	only, shared, wildcard := uuid.New().String(), uuid.New().String(), uuid.New().String()
	onlyScope, sharedScope, wildcardScope := uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: only, Title: "Only Bug", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(1)}, Contexts: []property.Context{{ID: onlyScope, Name: "Bug default", Types: []string{f.bug}, Overrides: []string{"default_value"}, Configuration: property.Configuration{Default: float64(99)}}}},
		property.Definition{ID: shared, Title: "Shared", Kind: property.Number, Configuration: property.Configuration{Active: true}, Contexts: []property.Context{{ID: sharedScope, Name: "Shared default", Types: []string{f.bug, f.task}, Overrides: []string{"default_value"}, Configuration: property.Configuration{Default: float64(7)}}}},
		property.Definition{ID: wildcard, Title: "Project default", Kind: property.Number, Configuration: property.Configuration{Active: true}, Contexts: []property.Context{{ID: wildcardScope, Name: "Project default", Projects: []string{f.project}, Overrides: []string{"default_value"}, Configuration: property.Configuration{Default: float64(4)}}}},
	)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Action: "attach", Revision: catalog.Revision, TypeID: f.bug, PropertyIDs: []string{only, shared, wildcard}})
	if err != nil {
		t.Fatal(err)
	}
	f.mutate(t, "import", f.bug)
	before := f.mutate(t, "set_default", f.bug)
	// Existing Task values and their definition records remain usable.
	_, draft, _ := seedPublicationDraft(t, f)
	f.exec(`UPDATE draft_issues SET custom_properties=jsonb_build_object($2::text,11,$3::text,12,$4::text,13) WHERE id=$1`, draft, only, shared, wildcard)
	values := draftValues(t, f, "draft_issues", draft)
	_, err = f.repo.Mutate(f.ctx, f.owner, "types", "delete", f.bug, types.Input{Revision: before.Revision})
	if err != nil {
		t.Fatal(err)
	}
	after, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Properties) != 3 || len(after.Attachments) != 1 || after.Attachments[0].TypeID != f.task || len(after.Attachments[0].PropertyIDs) != 3 {
		t.Fatalf("shared definitions/attachments lost: %+v", after)
	}
	definitions := map[string]property.Definition{}
	for _, record := range after.Properties {
		definitions[record.Definition.ID] = record.Definition
	}
	if len(definitions[only].Contexts) != 0 {
		t.Fatal("deleted type's sole context survived or became wildcard")
	}
	if len(definitions[shared].Contexts) != 1 || !reflect.DeepEqual(definitions[shared].Contexts[0].Types, []string{f.task}) || definitions[shared].Contexts[0].ID != sharedScope {
		t.Fatal("shared context lost its remaining scope/identity")
	}
	if len(definitions[wildcard].Contexts) != 1 || definitions[wildcard].Contexts[0].ID != wildcardScope || len(definitions[wildcard].Contexts[0].Types) != 0 {
		t.Fatal("project-wide context changed")
	}
	effective, _, err := property.Effective(definitions[only], f.project, f.task)
	if err != nil || effective.Default != float64(1) {
		t.Fatalf("deleted context leaked into remaining type: %+v %v", effective, err)
	}
	if !reflect.DeepEqual(values, draftValues(t, f, "draft_issues", draft)) {
		t.Fatal("deleting unused type changed other type's stored values")
	}
	var attachments, imports, preferences int
	if err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM work_item_type_properties WHERE type_id=$1),(SELECT count(*) FROM project_issue_types WHERE issue_type_id=$1 AND deleted_at IS NULL),(SELECT count(*) FROM work_item_project_defaults WHERE type_id=$1)`, f.bug).Scan(&attachments, &imports, &preferences); err != nil {
		t.Fatal(err)
	}
	if attachments != 0 || imports != 0 || preferences != 0 {
		t.Fatal("deleted type retained live configuration")
	}
	// The retained property can still be edited through the normal validated API.
	renamed := definitions[shared]
	renamed.Title = "Shared effort"
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Action: "update", Revision: after.Revision, Definition: renamed}); err != nil {
		t.Fatalf("deleted type prevents future property edits: %v", err)
	}
}
