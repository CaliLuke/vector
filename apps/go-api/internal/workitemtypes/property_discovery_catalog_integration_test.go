//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
)

func TestPropertyDiscoveryCatalogExposesOnlyAuthorizedEffectiveFields(t *testing.T) {
	f := newProjectTypeFixture(t)
	visible, hidden, cascade, first, second, root, child, contextID, privateProject := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	f.addProject(privateProject)
	options := []property.Option{{ID: root, Name: "Allowed country", Icon: "task"}, {ID: child, Name: "Allowed region", Parent: root, Level: 1, Icon: "bug"}}
	dropdown := uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: dropdown, Title: "Area", Kind: property.Dropdown, Configuration: property.Configuration{Active: true, Options: options}},
		property.Definition{ID: visible, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(7)}},
		property.Definition{ID: hidden, Title: "Hidden field", Kind: property.Text, Configuration: property.Configuration{Active: false, Format: "Read-only", Content: "Hidden content"}},
		property.Definition{ID: cascade, Title: "Location", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: first, Name: "Country"}, {ID: second, Name: "Region"}}, Options: options}, Contexts: []property.Context{{ID: contextID, Name: "Private context", Projects: []string{privateProject}, Overrides: []string{"options"}, Configuration: property.Configuration{Options: []property.Option{{ID: uuid.NewString(), Name: "Private option"}}}}}},
	)
	catalog, err := f.repo.DiscoveryProperties(f.ctx, f.member, "types", f.project)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Enabled || len(catalog.Fields) != 4 {
		t.Fatalf("wrong field catalog: %+v", catalog)
	}
	raw, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"Private context", "Private option", privateProject, contextID, hidden, "default_value", "configuration", "scope", "usage", "Hidden content"} {
		if strings.Contains(string(raw), private) {
			t.Fatalf("private configuration exposed: %s", private)
		}
	}
	for _, field := range catalog.Fields {
		if field.PropertyID == dropdown && (len(field.Options) != 2 || field.Options[0].Icon != "task" || field.Options[1].Parent != root || field.Options[1].Icon != "bug") {
			t.Fatalf("missing option presentation: %+v", field)
		}
		if field.Kind == property.Cascading && (!strings.Contains(field.Key, "_catalog_base") || field.LevelID == "" || len(field.Options) != 1) {
			t.Fatalf("missing level identity: %+v", field)
		}
	}
	_, err = f.repo.DiscoveryProperties(f.ctx, f.member, "types", privateProject)
	requireTypeStatus(t, err, 403)
	_, err = f.repo.DiscoveryProperties(f.ctx, f.outsider, "types", "")
	requireTypeStatus(t, err, 403)
}

func TestPropertyDiscoveryCatalogRetainsDisplayOnlyFieldsAndScopedReferences(t *testing.T) {
	f := newProjectTypeFixture(t)
	member, release, rich, formulaID, own, foreign, private := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: member, Title: "Reviewers", Kind: property.Member, Configuration: property.Configuration{Active: true, Multiple: true}},
		property.Definition{ID: release, Title: "Release", Kind: property.Release, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: rich, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: formulaID, Title: "Calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "1 + 2", Result: "Number"}},
	)
	f.addProject(private)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Visible release'),($4,$2,$5,'Private release')`, own, f.workspace, f.project, foreign, private)
	catalog, err := f.repo.DiscoveryProperties(f.ctx, f.member, "types", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Fields) != 4 {
		t.Fatalf("wrong fields: %+v", catalog)
	}
	for _, field := range catalog.Fields {
		switch field.PropertyID {
		case rich:
			if len(field.Operators) != 0 || field.Groupable {
				t.Fatal("rich text gained filtering/grouping")
			}
		case member:
			if field.Groupable || !field.Multiple || len(field.Options) == 0 {
				t.Fatalf("member cardinality: %+v", field)
			}
		case release:
			if !field.Groupable || len(field.Options) != 1 || field.Options[0].ID != own {
				t.Fatalf("release scope: %+v", field)
			}
		case formulaID:
			if field.ResultKind != "Number" || len(field.Operators) != 7 {
				t.Fatalf("formula controls: %+v", field)
			}
		}
	}
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	disabled, err := f.repo.DiscoveryProperties(f.ctx, f.member, "types", "")
	if err != nil || disabled.Enabled || len(disabled.Fields) != 0 {
		t.Fatalf("disabled catalog leaked: %+v %v", disabled, err)
	}
}
