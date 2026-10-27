//go:build integration

package workitemtypes_test

import (
	"errors"
	"slices"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func assertTypeError(t *testing.T, err error, status int) {
	t.Helper()
	var failure *types.Error
	if !errors.As(err, &failure) || failure.Status != status {
		t.Fatalf("want status %d, got %v", status, err)
	}
}

func TestUnusedTextConversionPersistsThroughPropertyMutation(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Score", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}})
	before, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	changed := before.Properties[0].Definition
	changed.Kind = property.Number
	changed.Configuration = property.Configuration{Active: true, Default: float64(0)}
	_, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: before.Revision, Action: "update", Definition: changed})
	if err != nil {
		t.Fatal(err)
	}
	after, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Properties) != 1 || after.Revision != before.Revision+1 {
		t.Fatalf("unexpected catalog after conversion: %+v", after)
	}
	saved := after.Properties[0].Definition
	if saved.ID != id || saved.Name != "score" || saved.Kind != property.Number || saved.Configuration.Default != float64(0) {
		t.Fatalf("conversion did not retain identity and target configuration: %+v", saved)
	}
	attachment := slices.IndexFunc(after.Attachments, func(value types.PropertyAttachment) bool { return value.TypeID == f.task })
	if attachment < 0 || len(after.Attachments[attachment].PropertyIDs) != 1 || after.Attachments[attachment].PropertyIDs[0] != id {
		t.Fatal("conversion lost the type attachment")
	}
}

func TestPropertyPersistenceAndUsage(t *testing.T) {
	f := newTypeFixture(t)
	repo := types.NewRepository(f.db)
	_, err := repo.PropertyCatalog(f.ctx, f.outsider, "types")
	assertTypeError(t, err, 403)
	typesCatalog, err := repo.Mutate(f.ctx, f.owner, "types", "enable", "", types.Input{})
	if err != nil {
		t.Fatal(err)
	}
	task := typesCatalog.Types[0].ID
	catalog, err := repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Properties) != 0 || len(catalog.Attachments) != 1 || len(catalog.Projects) != 0 || len(catalog.Members) != 2 {
		t.Fatalf("unexpected empty catalog or private project leaked: %+v", catalog)
	}
	number := property.Definition{ID: uuid.New().String(), Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(0)}}
	command := types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: number}
	_, err = repo.MutateProperty(f.ctx, f.member, "types", command)
	assertTypeError(t, err, 403)
	catalog, err = repo.MutateProperty(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	number = catalog.Properties[0].Definition
	if number.Name != "effort" || number.Configuration.Default != float64(0) {
		t.Fatalf("default or internal name lost: %+v", number)
	}
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", command)
	assertTypeError(t, err, 409)
	command = types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: task, PropertyIDs: []string{number.ID}}
	catalog, err = repo.MutateProperty(f.ctx, f.owner, "types", command)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Attachments[0].PropertyIDs) != 1 || catalog.Attachments[0].PropertyIDs[0] != number.ID {
		t.Fatal("attachment not persisted")
	}
	formulaDef := property.Definition{ID: uuid.New().String(), Title: "Double effort", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} * 2", Result: formula.Number}}
	catalog, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: formulaDef})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: task, PropertyIDs: []string{formulaDef.ID, number.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Attachments[0].PropertyIDs[0] != formulaDef.ID {
		t.Fatal("attachment order lost")
	}
	// A display title rename does not invalidate the formula's immutable reference.
	number.Title = "Complexity"
	catalog, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "update", Definition: number})
	if err != nil {
		t.Fatal(err)
	}
	invalid := number
	invalid.Name = "different"
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "update", Definition: invalid})
	assertTypeError(t, err, 422)
	// Invalid scope IDs and unavailable formula references roll back completely.
	invalid = number
	invalid.Contexts = []property.Context{{ID: uuid.New().String(), Name: "Foreign", Projects: []string{uuid.New().String()}, Overrides: []string{"required"}, Configuration: property.Configuration{Required: true}}}
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "update", Definition: invalid})
	assertTypeError(t, err, 422)
	invalid = number
	invalid.Kind = property.Text
	invalid.Configuration = property.Configuration{Active: true, Format: "Single line"}
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "update", Definition: invalid})
	assertTypeError(t, err, 422) // It would change the attached formula's operand type.
	draft := uuid.New().String()
	f.exec(`INSERT INTO draft_issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,created_at,updated_at,custom_properties,created_by_id) VALUES($1,$2,$3,$4,'Private draft','{}','','none',0,now(),now(),jsonb_build_object($5::text,0),$6)`, draft, f.workspace, f.project, task, number.ID, f.member)
	f.exec(`UPDATE draft_issues SET custom_properties='{}' WHERE id=$1`, draft)
	f.exec(`DELETE FROM draft_issues WHERE id=$1`, draft)
	catalog, err = repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Properties[0].Used {
		t.Fatal("clearing/deleting private draft erased ever-used lock")
	}
	// Catalog revisions are shared with type edits, not a parallel counter.
	changed, err := repo.Mutate(f.ctx, f.owner, "types", "create", "", types.Input{Revision: catalog.Revision, Name: "Bug", Icon: "bug", Color: "#123456"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "update", Definition: number})
	assertTypeError(t, err, 409)
	reloaded, err := repo.PropertyCatalog(f.ctx, f.member, "types")
	if err != nil || reloaded.Revision != changed.Revision || reloaded.CanManage || len(reloaded.Properties) != 2 {
		t.Fatalf("reload/scope mismatch: %+v %v", reloaded, err)
	}
}

func TestPropertyAttachmentIsolationAndCollision(t *testing.T) {
	f := newTypeFixture(t)
	repo := types.NewRepository(f.db)
	typesCatalog, err := repo.Mutate(f.ctx, f.owner, "types", "enable", "", types.Input{})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, name := range []string{"alpha", "beta"} {
		def := property.Definition{ID: uuid.New().String(), Name: name, Title: "Impact", Kind: property.Number, Configuration: property.Configuration{Active: true}}
		catalog, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: def})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, def.ID)
	}
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: typesCatalog.Types[0].ID, PropertyIDs: ids})
	assertTypeError(t, err, 409)
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: uuid.New().String(), PropertyIDs: ids[:1]})
	assertTypeError(t, err, 404)
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: typesCatalog.Types[0].ID, PropertyIDs: []string{uuid.New().String()}})
	assertTypeError(t, err, 422)
	after, err := repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || after.Revision != catalog.Revision || len(after.Attachments[0].PropertyIDs) != 0 {
		t.Fatalf("failed attach changed state: %+v %v", after, err)
	}
}

func TestUnattachedFormulaDependenciesStayValid(t *testing.T) {
	f := newTypeFixture(t)
	repo := types.NewRepository(f.db)
	catalog, err := repo.Mutate(f.ctx, f.owner, "types", "enable", "", types.Input{})
	if err != nil {
		t.Fatal(err)
	}
	def := property.Definition{ID: uuid.New().String(), Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}}
	props, err := repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: def})
	if err != nil {
		t.Fatal(err)
	}
	def = props.Properties[0].Definition
	calculation := property.Definition{ID: uuid.New().String(), Title: "Calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} * 2", Result: formula.Number}}
	props, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "create", Definition: calculation})
	if err != nil {
		t.Fatal(err)
	}
	def.Kind = property.Boolean
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "update", Definition: def})
	assertTypeError(t, err, 422)
	after, err := repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || after.Revision != props.Revision || after.Properties[0].Definition.Kind != property.Number {
		t.Fatalf("rejected dependency edit mutated catalog: %+v %v", after, err)
	}
}

func TestUnusedPropertyRemovalAndFormulaProtection(t *testing.T) {
	f := newTypeFixture(t)
	repo := types.NewRepository(f.db)
	catalog, err := repo.Mutate(f.ctx, f.owner, "types", "enable", "", types.Input{})
	if err != nil {
		t.Fatal(err)
	}
	def := property.Definition{ID: uuid.New().String(), Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}}
	props, err := repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: def})
	if err != nil {
		t.Fatal(err)
	}
	task := catalog.Types[0].ID
	props, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "attach", TypeID: task, PropertyIDs: []string{def.ID}})
	if err != nil {
		t.Fatal(err)
	}
	calculation := property.Definition{ID: uuid.New().String(), Title: "Calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} * 2", Result: formula.Number}}
	props, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "create", Definition: calculation})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "delete", ID: def.ID})
	assertTypeError(t, err, 422)
	props, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "delete", ID: calculation.ID})
	if err != nil {
		t.Fatal(err)
	}
	props, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "detach", ID: def.ID, TypeID: task})
	if err != nil {
		t.Fatal(err)
	}
	if len(props.Attachments[0].PropertyIDs) != 0 || len(props.Properties) != 1 {
		t.Fatal("detach deleted the definition or kept the attachment")
	}
	props, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "delete", ID: def.ID})
	if err != nil || len(props.Properties) != 0 {
		t.Fatalf("unused deletion failed: %+v %v", props, err)
	}
}

func TestPropertyContextsDefaultsAndDestructiveCascadeEdits(t *testing.T) {
	f := newTypeFixture(t)
	repo := types.NewRepository(f.db)
	catalog, err := repo.Mutate(f.ctx, f.owner, "types", "enable", "", types.Input{})
	if err != nil {
		t.Fatal(err)
	}
	task := catalog.Types[0].ID
	member := property.Definition{ID: uuid.New().String(), Title: "Reviewer", Kind: property.Member, Configuration: property.Configuration{Active: true, Default: f.owner}, Contexts: []property.Context{{ID: uuid.New().String(), Name: "Task scope", Projects: []string{f.project}, Types: []string{task}, Overrides: []string{"default_value"}, Configuration: property.Configuration{Default: f.member}}}}
	props, err := repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: member})
	if err != nil {
		t.Fatal(err)
	}
	member = props.Properties[0].Definition
	if member.Contexts[0].Configuration.Default != f.member {
		t.Fatal("context default was not durable")
	}
	member.Configuration.Default = f.outsider
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "update", Definition: member})
	assertTypeError(t, err, 422)
	cascade := property.Definition{ID: uuid.New().String(), Title: "Area", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: uuid.New().String(), Name: "Product"}, {ID: uuid.New().String(), Name: "Component"}}}}
	root, leaf := uuid.New().String(), uuid.New().String()
	cascade.Configuration.Options = []property.Option{{ID: root, Name: "Platform"}, {ID: leaf, Name: "API", Parent: root, Level: 1}}
	props, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "create", Definition: cascade})
	if err != nil {
		t.Fatal(err)
	}
	cascade = props.Properties[1].Definition
	props, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "attach", TypeID: task, PropertyIDs: []string{cascade.ID}})
	if err != nil {
		t.Fatal(err)
	}
	issue := uuid.New().String()
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at,custom_properties) VALUES($1,$2,$3,$4,'Work','{}','','none',0,1,false,now(),now(),jsonb_build_object($5::text,jsonb_build_array($6::text,$7::text)))`, issue, f.workspace, f.project, task, cascade.ID, root, leaf)
	cascade.Configuration.Options = cascade.Configuration.Options[:1]
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "update", Definition: cascade})
	assertTypeError(t, err, 409)
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "detach", TypeID: task, ID: cascade.ID})
	assertTypeError(t, err, 409)
	_, err = repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: props.Revision, Action: "delete", ID: cascade.ID})
	assertTypeError(t, err, 409)
	after, err := repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || after.Revision != props.Revision || len(after.Properties[1].Definition.Configuration.Options) != 2 {
		t.Fatalf("rejected destructive edit lost values: %+v %v", after, err)
	}
}
