//go:build integration

package workitemtypes_test

import (
	"errors"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func validateTestItemValues(t *testing.T, f projectTypeFixture, input types.ItemValueInput) (types.ItemValueResult, error) {
	t.Helper()
	tx, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { testsupport.Rollback(t, tx, f.ctx) }()
	return types.ValidateItemValues(f.ctx, tx, input)
}

func TestItemValueDefaultsAndScopedReferences(t *testing.T) {
	f := newProjectTypeFixture(t)
	number, member, release := uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(7)}},
		property.Definition{ID: member, Title: "Reviewers", Kind: property.Member, Configuration: property.Configuration{Active: true, Multiple: true, Default: []string{f.member}}},
		property.Definition{ID: release, Title: "Delivery", Kind: property.Release, Configuration: property.Configuration{Active: true}},
	)
	releaseID, otherRelease, otherProject := uuid.New().String(), uuid.New().String(), uuid.New().String()
	f.addProject(otherProject)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Current'),($4,$2,$5,'Other')`, releaseID, f.workspace, f.project, otherRelease, otherProject)
	input := types.ItemValueInput{WorkspaceID: f.workspace, ProjectID: f.project, CreatorID: f.owner, Creating: true, Submitted: map[string]any{release: releaseID}}
	result, err := validateTestItemValues(t, f, input)
	if err != nil || result.TypeID != f.task || result.Values[number] != float64(7) || len(result.Subscribers) != 1 || result.Subscribers[0] != f.member {
		t.Fatalf("creation defaults/references: %+v %v", result, err)
	}
	for _, test := range []struct {
		name, property string
		value          any
	}{
		{"other project release", release, otherRelease},
		{"outsider member", member, []any{f.outsider}},
		{"wrong numeric shape", number, "7"},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := input
			candidate.Submitted = map[string]any{test.property: test.value}
			_, validationErr := validateTestItemValues(t, f, candidate)
			var invalid *types.ItemValueError
			if !errors.As(validationErr, &invalid) || invalid.Fields[test.property] == "" {
				t.Fatalf("invalid reference/value accepted: %v", validationErr)
			}
		})
	}
	// Restricted guests can subscribe to their own item, not another creator's.
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1 AND project_id=$2`, f.member, f.project)
	_, err = validateTestItemValues(t, f, input)
	var invalid *types.ItemValueError
	if !errors.As(err, &invalid) || invalid.Fields[member] == "" {
		t.Fatalf("restricted guest got access through default: %v", err)
	}
	input.CreatorID = f.member
	if _, err = validateTestItemValues(t, f, input); err != nil {
		t.Fatalf("creator guest cannot subscribe: %v", err)
	}
	input.Draft = true
	result, err = validateTestItemValues(t, f, input)
	if err != nil || len(result.Subscribers) != 0 {
		t.Fatalf("draft emitted subscribers: %+v %v", result, err)
	}
}

func TestItemValueCascadesRequireContextPreservation(t *testing.T) {
	f := newProjectTypeFixture(t)
	other := uuid.New().String()
	f.addProject(other)
	root, leaf := uuid.New().String(), uuid.New().String()
	def := property.Definition{ID: uuid.New().String(), Title: "Area", Kind: property.Cascading, Configuration: property.Configuration{Active: true,
		Levels:  []property.Level{{ID: uuid.New().String(), Name: "Product", Required: true}, {ID: uuid.New().String(), Name: "Component"}},
		Options: []property.Option{{ID: root, Name: "Platform"}, {ID: leaf, Name: "API", Parent: root, Level: 1}},
	}}
	def.Contexts = []property.Context{{ID: uuid.New().String(), Name: "Other project", Projects: []string{other}, Overrides: []string{"options"}, Configuration: property.Configuration{Options: []property.Option{{ID: root, Name: "Mobile"}, {ID: leaf, Name: "iOS", Parent: root, Level: 1}}}}}
	attachValueProperties(t, f, def)
	input := types.ItemValueInput{WorkspaceID: f.workspace, ProjectID: f.project, TypeID: f.task, PreviousProjectID: f.project, PreviousTypeID: f.task, CreatorID: f.member, Stored: map[string]any{def.ID: []any{root, leaf}}}
	if _, err := validateTestItemValues(t, f, input); err != nil {
		t.Fatal(err)
	}
	input.ProjectID, input.ProjectChanged = other, true
	_, err := validateTestItemValues(t, f, input)
	assertTypeError(t, err, 409)
	input.ProjectID, input.ProjectChanged, input.Draft = f.project, false, true
	input.Stored = nil
	result, err := validateTestItemValues(t, f, input)
	if err != nil || len(result.Values) != 0 {
		t.Fatalf("draft requires cascade: %+v %v", result, err)
	}
	input.Submitted = map[string]any{def.ID: []any{root}}
	_, err = validateTestItemValues(t, f, input)
	var invalid *types.ItemValueError
	if !errors.As(err, &invalid) || invalid.Fields[def.ID] == "" {
		t.Fatalf("draft cascade accepted: %v", err)
	}
	input.Draft, input.Creating = false, true
	if _, err = validateTestItemValues(t, f, input); err != nil {
		t.Fatalf("publication cascade rejected: %v", err)
	}
}
