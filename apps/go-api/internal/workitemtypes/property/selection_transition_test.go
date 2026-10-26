package property_test

import (
	"slices"
	"testing"

	p "github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func selectionDefinition() p.Definition {
	return p.Definition{ID: propertyID, Name: "location", Title: "Location", Kind: p.Cascading, Configuration: p.Configuration{
		Active:  true,
		Levels:  []p.Level{{ID: projectA, Name: "Country"}, {ID: projectB, Name: "City"}},
		Options: []p.Option{{ID: optionA, Name: "Canada"}, {ID: optionB, Name: "Toronto", Parent: optionA, Level: 1}},
	}}
}

func TestBehaviorContextAndUnselectedCatalogEditsRetainSelectedPath(t *testing.T) {
	before := selectionDefinition()
	after := selectionDefinition()
	after.Configuration.Options[0].Name = "Renamed country"
	after.Configuration.Options = append(after.Configuration.Options, p.Option{ID: optionC, Name: "Other"})
	slices.Reverse(after.Configuration.Options)
	after.Contexts = []p.Context{{ID: contextA, Name: "Project", Projects: []string{projectA}, Overrides: []string{"active"}, Configuration: p.Configuration{Active: false}}}
	scope := p.Scope{ProjectID: projectA, TypeID: typeA}
	removed, err := p.CascadingSelectionRequiresPreservation(before, after, scope, scope, []string{optionA, optionB})
	if err != nil || removed {
		t.Fatalf("valid base selection changed: removed=%v err=%v", removed, err)
	}
}

func TestIndependentCatalogOwnersPreserveEvenIdenticalOptions(t *testing.T) {
	def := selectionDefinition()
	def.Contexts = []p.Context{
		{ID: contextA, Name: "Project", Projects: []string{projectA}, Overrides: []string{"options"}, Configuration: def.Configuration},
		{ID: contextB, Name: "Specific", Projects: []string{projectB}, Overrides: []string{"options"}, Configuration: def.Configuration},
	}
	removed, err := p.CascadingSelectionRequiresPreservation(def, def, p.Scope{ProjectID: projectA}, p.Scope{ProjectID: projectB}, []string{optionA, optionB})
	if err != nil || !removed {
		t.Fatalf("independent catalogs reinterpreted selection: removed=%v err=%v", removed, err)
	}
}

func TestOptionsOverrideChangesCatalogOwnerWithoutChangingWinner(t *testing.T) {
	before := selectionDefinition()
	before.Contexts = []p.Context{{ID: contextA, Name: "Project", Projects: []string{projectA}, Overrides: []string{"active"}, Configuration: before.Configuration}}
	after := selectionDefinition()
	after.Contexts = []p.Context{{ID: contextA, Name: "Project", Projects: []string{projectA}, Overrides: []string{"active", "options"}, Configuration: after.Configuration}}
	scope := p.Scope{ProjectID: projectA}
	for _, pair := range [][2]p.Definition{{before, after}, {after, before}} {
		removed, err := p.CascadingSelectionRequiresPreservation(pair[0], pair[1], scope, scope, []string{optionA})
		if err != nil || !removed {
			t.Fatalf("override ownership ignored: removed=%v err=%v", removed, err)
		}
	}
}

func TestMoreSpecificBehaviorContextFallsBackToBaseCatalog(t *testing.T) {
	def := selectionDefinition()
	def.Contexts = []p.Context{
		{ID: contextA, Name: "Project", Projects: []string{projectA}, Overrides: []string{"options"}, Configuration: def.Configuration},
		{ID: contextB, Name: "Specific", Projects: []string{projectA}, Types: []string{typeA}, Overrides: []string{"active"}, Configuration: def.Configuration},
	}
	removed, err := p.CascadingSelectionRequiresPreservation(def, def, p.Scope{ProjectID: projectA, TypeID: typeB}, p.Scope{ProjectID: projectA, TypeID: typeA}, []string{optionA})
	if err != nil || !removed {
		t.Fatalf("inherited from lower-priority catalog: removed=%v err=%v", removed, err)
	}
}

func TestSelectionPreservationIgnoresNewRequirementsButDetectsRemovedPaths(t *testing.T) {
	before := selectionDefinition()
	for _, scenario := range []struct {
		name    string
		change  func(*p.Definition)
		removed bool
	}{
		{"required child", func(d *p.Definition) {
			d.Configuration.Levels[0].Required = true
			d.Configuration.Levels[1].Required = true
		}, false},
		{"new required level", func(d *p.Definition) {
			d.Configuration.Levels[0].Required = true
			d.Configuration.Levels[1].Required = true
			d.Configuration.Levels = append(d.Configuration.Levels, p.Level{ID: typeA, Name: "Street", Required: true})
		}, false},
		{"removed selected root while hidden", func(d *p.Definition) { d.Configuration.Active = false; d.Configuration.Options = nil }, true},
		{"removed unselected child", func(d *p.Definition) { d.Configuration.Options = d.Configuration.Options[:1] }, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			after := selectionDefinition()
			scenario.change(&after)
			if err := p.ValidateDefinition(after, &before, true); err != nil {
				t.Fatalf("invalid test definition: %v", err)
			}
			removed, err := p.CascadingSelectionRequiresPreservation(before, after, p.Scope{}, p.Scope{}, []string{optionA})
			if err != nil || removed != scenario.removed {
				t.Fatalf("removed=%v err=%v", removed, err)
			}
		})
	}
}

func TestEmptyCascadingSelectionDoesNotNeedPreservationOnOwnerChange(t *testing.T) {
	before := selectionDefinition()
	after := selectionDefinition()
	after.Contexts = []p.Context{{ID: contextA, Name: "Project", Projects: []string{projectA}, Overrides: []string{"options"}, Configuration: after.Configuration}}
	scope := p.Scope{ProjectID: projectA}
	removed, err := p.CascadingSelectionRequiresPreservation(before, after, scope, scope, []string{})
	if err != nil || removed {
		t.Fatalf("empty selection required preservation: removed=%v err=%v", removed, err)
	}
}

func TestSelectedDescendantChangesRequirePreservation(t *testing.T) {
	before := selectionDefinition()
	before.Configuration.Levels = append(before.Configuration.Levels, p.Level{ID: typeA, Name: "Street"})
	before.Configuration.Options = append(before.Configuration.Options, p.Option{ID: optionC, Name: "Main", Parent: optionB, Level: 2})
	for _, scenario := range []struct {
		name   string
		change func(*p.Definition)
	}{
		{"removed child", func(d *p.Definition) { d.Configuration.Options = d.Configuration.Options[:1] }},
		{"reparented child", func(d *p.Definition) {
			d.Configuration.Options = append(d.Configuration.Options, p.Option{ID: contextA, Name: "Other country"})
			d.Configuration.Options[1].Parent = contextA
		}},
		{"removed selected level", func(d *p.Definition) {
			d.Configuration.Levels = d.Configuration.Levels[:2]
			d.Configuration.Options = d.Configuration.Options[:2]
		}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			after := before
			after.Configuration.Options = slices.Clone(before.Configuration.Options)
			after.Configuration.Levels = slices.Clone(before.Configuration.Levels)
			scenario.change(&after)
			if err := p.ValidateDefinition(after, &before, true); err != nil {
				t.Fatal(err)
			}
			removed, err := p.CascadingSelectionRequiresPreservation(before, after, p.Scope{}, p.Scope{}, []any{optionA, optionB, optionC})
			if err != nil || !removed {
				t.Fatalf("selected descendant changed without preservation: %v %v", removed, err)
			}
		})
	}
}

func TestPreviouslyIncompleteRequiredPathCanSurviveUnrelatedSchemaEdit(t *testing.T) {
	before := selectionDefinition()
	before.Configuration.Levels[0].Required = true
	before.Configuration.Levels[1].Required = true
	after := before
	after.Title = "Renamed"
	removed, err := p.CascadingSelectionRequiresPreservation(before, after, p.Scope{}, p.Scope{}, []any{optionA})
	if err != nil || removed {
		t.Fatalf("required rule retroactively removed saved path: %v %v", removed, err)
	}
}

func TestCascadingPreservationRejectsMalformedOrUnavailableOldSelections(t *testing.T) {
	def := selectionDefinition()
	for _, value := range []any{[]any{optionC}, []any{7}, optionA} {
		if _, err := p.CascadingSelectionRequiresPreservation(def, def, p.Scope{}, p.Scope{}, value); err == nil {
			t.Fatalf("accepted invalid saved path: %#v", value)
		}
	}
	removed, err := p.CascadingSelectionRequiresPreservation(def, def, p.Scope{}, p.Scope{}, nil)
	if err != nil || removed {
		t.Fatalf("nil value needs preservation: %v %v", removed, err)
	}
}
