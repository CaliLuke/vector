//go:build integration

package workitemtypes_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func planTestItemTransition(t *testing.T, f projectTypeFixture, in types.ItemValueInput, description string) (types.ItemTransition, error) {
	t.Helper()
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(f.ctx)) }()
	return types.PlanItemTransition(f.ctx, tx, in, description)
}
func transitionTestInput(f projectTypeFixture) types.ItemValueInput {
	return types.ItemValueInput{WorkspaceID: f.workspace, ProjectID: f.project, PreviousProjectID: f.project, TypeID: f.bug, PreviousTypeID: f.task, CreatorID: f.member, ActorID: f.member, Existing: true, TypeSpecified: true}
}
func attachTransitionDestination(t *testing.T, f projectTypeFixture, ids ...string) {
	t.Helper()
	f.mutate(t, "import", f.bug)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Action: "attach", Revision: catalog.Revision, TypeID: f.bug, PropertyIDs: ids}); err != nil {
		t.Fatal(err)
	}
}

func TestItemTransitionPreservesOldValuesAndProposedDescription(t *testing.T) {
	f := newProjectTypeFixture(t)
	number, boolean, rich, shared := uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: boolean, Title: "Confirmed", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: rich, Title: "Details", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: shared, Title: "Shared", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}},
	)
	attachTransitionDestination(t, f, shared)
	in := transitionTestInput(f)
	in.Stored = map[string]any{number: float64(0), boolean: false, rich: "<p>Original details</p>", shared: "Keep me"}
	in.Submitted = map[string]any{shared: "Edited shared value"}
	// Ordinary writes continue to reject the destructive transition.
	_, err := validateTestItemValues(t, f, in)
	requireTypeStatus(t, err, 409)
	plan, err := planTestItemTransition(t, f, in, "<p>New proposed description</p>")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plan.Description, "<p>New proposed description</p>") || !strings.Contains(plan.Description, "<p>0</p>") || !strings.Contains(plan.Description, "<p>false</p>") || strings.Count(plan.Description, "Original details") != 1 {
		t.Fatalf("preservation lost proposed or former content: %s", plan.Description)
	}
	if len(plan.Preserved) != 3 || len(plan.Values) != 1 || plan.Values[shared] != "Edited shared value" || plan.TypeID != f.bug {
		t.Fatalf("wrong transition plan: %+v", plan)
	}
	if len(in.Stored) != 4 || in.Stored[shared] != "Keep me" || len(in.Submitted) != 1 {
		t.Fatal("planning mutated caller's baseline or proposed edit")
	}
}

func TestItemTransitionStillRequiresDestinationValues(t *testing.T) {
	f := newProjectTypeFixture(t)
	old, required := uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: old, Title: "Old notes", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}},
		property.Definition{ID: required, Title: "Required effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Required: true}},
	)
	attachTransitionDestination(t, f, required)
	in := transitionTestInput(f)
	in.Stored = map[string]any{old: "Preserve me"}
	_, err := planTestItemTransition(t, f, in, "<p>Edited</p>")
	var invalid *types.ItemValueError
	if !errors.As(err, &invalid) || invalid.Fields[required] == "" {
		t.Fatalf("review bypassed required destination field: %v", err)
	}
	in.Submitted = map[string]any{required: float64(0)}
	plan, err := planTestItemTransition(t, f, in, "<p>Edited</p>")
	if err != nil || plan.Values[required] != float64(0) || len(plan.Preserved) != 1 {
		t.Fatalf("valid replacement rejected: %+v %v", plan, err)
	}
}

func TestItemTransitionPreservesFormerCascadeCatalogBeforeReplacement(t *testing.T) {
	f := newProjectTypeFixture(t)
	root, leaf := uuid.New().String(), uuid.New().String()
	def := property.Definition{ID: uuid.New().String(), Title: "Area", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: uuid.New().String(), Name: "Product", Required: true}, {ID: uuid.New().String(), Name: "Component"}}, Options: []property.Option{{ID: root, Name: "Platform"}, {ID: leaf, Name: "API", Parent: root, Level: 1}}}}
	def.Contexts = []property.Context{{ID: uuid.New().String(), Name: "Bug catalog", Types: []string{f.bug}, Overrides: []string{"options"}, Configuration: property.Configuration{Options: []property.Option{{ID: root, Name: "Mobile"}, {ID: leaf, Name: "iOS", Parent: root, Level: 1}}}}}
	attachValueProperties(t, f, def)
	attachTransitionDestination(t, f, def.ID)
	in := transitionTestInput(f)
	in.Stored = map[string]any{def.ID: []any{root, leaf}}
	in.Submitted = map[string]any{def.ID: []any{root}}
	plan, err := planTestItemTransition(t, f, in, "<p>Edited</p>")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Preserved) != 1 || !strings.Contains(plan.Description, "Platform → API") || strings.Contains(plan.Description, "Mobile") || !reflect.DeepEqual(plan.Values[def.ID], []string{root}) {
		t.Fatalf("catalog identity/value lost: %+v", plan)
	}
}

func TestItemTransitionClearsDraftTypeWithoutLosingValues(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	in := transitionTestInput(f)
	in.Draft, in.TypeID = true, ""
	in.Stored = map[string]any{id: float64(7)}
	plan, err := planTestItemTransition(t, f, in, "<p>Private draft edits</p>")
	if err != nil || plan.TypeID != "" || len(plan.Values) != 0 || !strings.Contains(plan.Description, "<p>7</p>") {
		t.Fatalf("draft clear lost values: %+v %v", plan, err)
	}
}

func TestItemTransitionRejectsOversizedPreservation(t *testing.T) {
	f := newProjectTypeFixture(t)
	id := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	f.mutate(t, "import", f.bug)
	in := transitionTestInput(f)
	in.Stored = map[string]any{id: float64(7)}
	_, err := planTestItemTransition(t, f, in, "<p>"+strings.Repeat("x", content.MaxContentBytes-7)+"</p>")
	requireTypeStatus(t, err, 422)
	if len(in.Stored) != 1 || in.Stored[id] != float64(7) {
		t.Fatal("rejected preservation mutated source values")
	}
}

func TestItemTransitionPreservesSelectionLabelsForCanonicalAndJSONValues(t *testing.T) {
	f := newProjectTypeFixture(t)
	member, release, releaseID := uuid.New().String(), uuid.New().String(), uuid.New().String()
	f.exec(`UPDATE users SET display_name='Release reviewer' WHERE id=$1`, f.member)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Autumn release')`, releaseID, f.workspace, f.project)
	attachValueProperties(t, f,
		property.Definition{ID: member, Title: "Reviewers", Kind: property.Member, Configuration: property.Configuration{Active: true, Multiple: true}},
		property.Definition{ID: release, Title: "Releases", Kind: property.Release, Configuration: property.Configuration{Active: true, Multiple: true}},
	)
	f.mutate(t, "import", f.bug)
	for _, test := range []struct {
		name            string
		member, release any
	}{
		{"canonical", []string{f.member}, []string{releaseID}},
		{"decoded JSON", []any{f.member}, []any{releaseID}},
	} {
		t.Run(test.name, func(t *testing.T) {
			in := transitionTestInput(f)
			in.Stored = map[string]any{member: test.member, release: test.release}
			plan, err := planTestItemTransition(t, f, in, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Preserved) != 2 || !strings.Contains(plan.Description, "Release reviewer") || !strings.Contains(plan.Description, "Autumn release") || strings.Contains(plan.Description, "Unavailable selection") {
				t.Fatalf("reference labels lost: %s", plan.Description)
			}
		})
	}
}

func TestItemTransitionSelectsTypeForUntypedPrivateDraft(t *testing.T) {
	f := newProjectTypeFixture(t)
	in := transitionTestInput(f)
	in.PreviousTypeID, in.TypeID, in.Draft = "", f.task, true
	plan, err := planTestItemTransition(t, f, in, "<p>Keep draft</p>")
	if err != nil || plan.TypeID != f.task || len(plan.Preserved) != 0 || plan.Description != "<p>Keep draft</p>" {
		t.Fatalf("untyped draft transition failed: %+v %v", plan, err)
	}
}

func TestItemTransitionPreservesUnsavedFormerValueAndRejectsInvalidFormerInput(t *testing.T) {
	f := newProjectTypeFixture(t)
	rich, number := uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: rich, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: number, Title: "Count", Kind: property.Number, Configuration: property.Configuration{Active: true}},
	)
	attachTransitionDestination(t, f)
	in := transitionTestInput(f)
	in.Stored = map[string]any{rich: "<p>A</p>", number: float64(10)}
	in.Submitted = map[string]any{rich: "<p>B</p>", number: float64(0)}
	plan, err := planTestItemTransition(t, f, in, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.Description, "<p>B</p>") || strings.Contains(plan.Description, "<p>A</p>") || !strings.Contains(plan.Description, "<p>0</p>") || len(plan.Values) != 0 {
		t.Fatalf("unsaved former values lost: %+v", plan)
	}
	if in.Stored[rich] != "<p>A</p>" || len(in.Submitted) != 2 {
		t.Fatal("planner mutated caller values")
	}
	in.Submitted[number] = "invalid number"
	_, err = planTestItemTransition(t, f, in, "")
	var invalid *types.ItemValueError
	if !errors.As(err, &invalid) || invalid.Fields[number] == "" {
		t.Fatalf("former value bypassed validation: %v", err)
	}
}

func TestItemTransitionCannotResolveSelectionsAfterSourceAccessIsRevoked(t *testing.T) {
	f := newProjectTypeFixture(t)
	id, release := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Private release name')`, release, f.workspace, f.project)
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Release", Kind: property.Release, Configuration: property.Configuration{Active: true}})
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	in := transitionTestInput(f)
	in.ProjectID, in.TypeID, in.Draft = "", "", true
	in.Stored = map[string]any{id: release}
	plan, err := planTestItemTransition(t, f, in, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plan.Description, "Unavailable selection") || strings.Contains(plan.Description, "Private release name") || strings.Contains(plan.Description, release) {
		t.Fatalf("inaccessible release disclosed: %s", plan.Description)
	}
	in.Stored = nil
	in.Submitted = map[string]any{id: release}
	_, err = planTestItemTransition(t, f, in, "")
	var invalid *types.ItemValueError
	if !errors.As(err, &invalid) || invalid.Fields[id] == "" {
		t.Fatalf("inaccessible former selection accepted: %v", err)
	}
}

func TestItemTransitionCanClearRequiredFormerCascadeWithoutAcceptingInvalidPaths(t *testing.T) {
	f := newProjectTypeFixture(t)
	id, root, leaf := uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: id, Title: "Area", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: uuid.New().String(), Name: "Product", Required: true}, {ID: uuid.New().String(), Name: "Component", Required: true}}, Options: []property.Option{{ID: root, Name: "Platform"}, {ID: leaf, Name: "API", Parent: root, Level: 1}}}})
	attachTransitionDestination(t, f)
	for _, path := range [][]any{{}, {root}, {leaf}} {
		in := transitionTestInput(f)
		in.Stored = map[string]any{id: []any{root, leaf}}
		in.Submitted = map[string]any{id: path}
		plan, err := planTestItemTransition(t, f, in, "")
		if len(path) > 0 && path[0] == leaf {
			var invalid *types.ItemValueError
			if !errors.As(err, &invalid) || invalid.Fields[id] == "" {
				t.Fatalf("invalid former cascade accepted: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.Values) != 0 || strings.Contains(plan.Description, "API") || len(path) == 0 && len(plan.Preserved) != 0 || len(path) > 0 && !strings.Contains(plan.Description, "Platform") {
			t.Fatalf("explicit former cascade edit lost: %+v", plan)
		}
	}
}
