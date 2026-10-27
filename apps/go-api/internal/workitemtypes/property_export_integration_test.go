//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func exportPropertyRows(t *testing.T, f projectTypeFixture, rows []map[string]any, projects []string) []types.PropertyExportColumn {
	t.Helper()
	columns, err := types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", "", func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) ([]types.PropertyExportColumn, error) {
		return snapshot.ExportRows(f.ctx, rows, projects)
	})
	if err != nil {
		t.Fatal(err)
	}
	return columns
}

func TestPropertyExportRetainsTypedValuesCanonicalFormulaAndOneClock(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, ids := seedDiscoveryItems(t, f)
	calculation, clock := uuid.NewString(), uuid.NewString()
	number, flag, rich, date, member, release, version := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: clock, Title: "Clock", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "NOW()", Result: "Date"}},
		property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: calculation, Title: "Calculated effort", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} + 10", Result: "Number"}},
		property.Definition{ID: flag, Title: "Verified", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: rich, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: date, Title: "Launch", Kind: property.Date, Configuration: property.Configuration{Active: true, Format: "2025/01/15"}},
		property.Definition{ID: member, Title: "Reviewer", Kind: property.Member, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: release, Title: "Release", Kind: property.Release, Configuration: property.Configuration{Active: true}},
	)
	f.exec(`UPDATE users SET display_name='Export reviewer' WHERE id=$1`, f.member)
	f.exec(`UPDATE projects SET name='Export project' WHERE id=$1`, f.project)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Release one')`, version, f.workspace, f.project)
	saved, err := json.Marshal(map[string]any{number: 0, flag: false, rich: "<p>Export notes</p>", date: "2026-10-26", member: f.member, release: version})
	if err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[0], saved)
	rows := []map[string]any{{"id": ids[0], "project_id": f.project, "type_id": f.task}, {"id": ids[1], "project_id": f.project, "type_id": f.task}}
	columns := exportPropertyRows(t, f, rows, []string{f.project})
	if len(columns) != 10 {
		t.Fatalf("columns=%+v", columns)
	}
	expected := map[string]any{number: float64(0), flag: false, rich: "<p>Export notes</p>", date: "2026-10-26", member: "Export reviewer", release: "Export project · Release one", calculation: float64(10)}
	assertExportValues(t, rows[0], expected)
	if rows[1]["customproperty_"+calculation] != nil || rows[1]["customproperty_"+number] != nil {
		t.Fatal("empty values became nonempty")
	}
	if rows[0]["customproperty_"+clock] != rows[1]["customproperty_"+clock] || rows[0]["properties_evaluated_at"] != rows[1]["properties_evaluated_at"] {
		t.Fatal("export rows used different formula clocks")
	}
	values := rows[0]["property_values"].(map[string]types.ProjectedValue)
	if values[calculation].Value != float64(10) || values[calculation].Status != "value" || values[date].Format != "2025/01/15" || values[member].Value != f.member {
		t.Fatalf("canonical projection was lost: %+v", values)
	}
	if rows[0]["work_item_type"] != "Task" || rows[0]["type_id"] != f.task {
		t.Fatalf("type identity lost: %+v", rows[0])
	}
}

func assertExportValues(t *testing.T, row map[string]any, expected map[string]any) {
	t.Helper()
	for key, want := range expected {
		if got := row["customproperty_"+key]; !reflect.DeepEqual(got, want) {
			t.Errorf("property %s=%#v; want %#v", key, got, want)
		}
	}
}

func TestPropertyExportSeparatesCascadingContextsAndRetainsErrorStates(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, ids := seedDiscoveryItems(t, f)
	cascade, contextID, root, otherRoot, level, childLevel, badFormula := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: cascade, Title: "Location", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: level, Name: "Country"}, {ID: childLevel, Name: "Region"}}, Options: []property.Option{{ID: root, Name: "Task country"}}}, Contexts: []property.Context{{ID: contextID, Name: "Bug location", Types: []string{f.bug}, Overrides: []string{"options"}, Configuration: property.Configuration{Options: []property.Option{{ID: otherRoot, Name: "Bug country"}}}}}},
		property.Definition{ID: badFormula, Title: "Broken calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "1 / 0", Result: "Number"}},
	)
	f.exec(`INSERT INTO work_item_type_properties(type_id,property_id) VALUES($1,$2)`, f.bug, cascade)
	taskSaved, _ := json.Marshal(map[string]any{cascade: []string{root}})
	bugSaved, _ := json.Marshal(map[string]any{cascade: []string{otherRoot}})
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[0], taskSaved)
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[2], bugSaved)
	rows := []map[string]any{{"id": ids[0], "project_id": f.project, "type_id": f.task}, {"id": ids[2], "project_id": f.project, "type_id": f.bug}}
	exportPropertyRows(t, f, rows, []string{f.project})
	base := "customproperty_" + cascade + "_level_" + level + "_catalog_base"
	scoped := "customproperty_" + cascade + "_level_" + level + "_catalog_" + contextID
	if rows[0][base] != "Task country" || rows[0][scoped] != "#NOT_APPLICABLE" || rows[1][base] != "#NOT_APPLICABLE" || rows[1][scoped] != "Bug country" {
		t.Fatalf("catalogs crossed: %+v", rows)
	}
	if rows[0]["customproperty_"+cascade+"_level_"+childLevel+"_catalog_base"] != nil {
		t.Fatal("empty cascade level was not empty")
	}
	problem, ok := rows[0]["customproperty_"+badFormula].(string)
	if !ok || !strings.HasPrefix(problem, "#ERROR: ") || rows[1]["customproperty_"+badFormula] != "#NOT_APPLICABLE" {
		t.Fatalf("formula error/applicability lost: %+v", rows)
	}
}

func TestPropertyExportColumnsExcludeUnselectedProjectContexts(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, ids := seedDiscoveryItems(t, f)
	other, cascade, contextID, level, childLevel := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	f.addProject(other)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, other, f.member)
	attachValueProperties(t, f, property.Definition{ID: cascade, Title: "Location", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: level, Name: "Country"}, {ID: childLevel, Name: "Region"}}}, Contexts: []property.Context{{ID: contextID, Name: "Unselected context", Projects: []string{other}, Overrides: []string{"options"}, Configuration: property.Configuration{Options: []property.Option{{ID: uuid.NewString(), Name: "Unselected option"}}}}}})
	rows := []map[string]any{{"id": ids[0], "project_id": f.project, "type_id": f.task}}
	columns := exportPropertyRows(t, f, rows, []string{f.project})
	encoded, err := json.Marshal(columns)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), contextID) || strings.Contains(string(encoded), "Unselected") || len(columns) != 4 {
		t.Fatalf("unselected context in columns: %s", encoded)
	}
}

func TestPropertyExportRejectsInaccessibleEmptySelectedProject(t *testing.T) {
	f := newProjectTypeFixture(t)
	private := uuid.NewString()
	f.addProject(private)
	_, err := types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", "", func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) ([]types.PropertyExportColumn, error) {
		return snapshot.ExportRows(f.ctx, []map[string]any{}, []string{f.project, private})
	})
	requireTypeStatus(t, err, 403)
}
