//go:build integration

package workitemtypes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/cyclereads"
	"github.com/CaliLuke/go-plane/internal/modulereads"
)

func TestModulePropertyProjectionPreservesPagesAndGroups(t *testing.T) {
	f := newModulePropertyFixture(t)
	for _, options := range []modulereads.IssueListOptions{
		{Limit: 1, OrderBy: "created_at"},
		{Limit: 1, Page: 1, OrderBy: "created_at"},
		{Limit: 10, GroupBy: "priority", SubGroupBy: "state_id"},
		{Limit: 10, Filters: fmt.Sprintf(`{"customproperty_%s__exact":18}`, f.calculation)},
	} {
		baseline := f.read(t, options)
		options.IncludeProperties = true
		actual := f.read(t, options)
		assertProjectionEnvelope(t, baseline.Items, actual.Items, f.field, f.calculation)
		actual.Items = baseline.Items
		if !reflect.DeepEqual(actual, baseline) {
			t.Fatal("projection changed module pagination/group metadata")
		}
	}
}

func TestCyclePropertyProjectionPreservesPagesAndGroups(t *testing.T) {
	f := newCyclePropertyFixture(t)
	for _, options := range []cyclereads.IssueListOptions{
		{PerPage: 1, OrderBy: "created_at"},
		{PerPage: 1, Cursor: "1:1:0", OrderBy: "created_at"},
		{PerPage: 10, GroupBy: "priority"},
		{PerPage: 10, GroupBy: "priority", SubGroupBy: "state_id"},
		{PerPage: 10, ComplexFilter: fmt.Sprintf(`{"customproperty_%s__exact":18}`, f.calculation)},
	} {
		baseline := f.read(t, options)
		options.IncludeProperties = true
		actual := f.read(t, options)
		assertProjectionEnvelope(t, baseline, actual, f.field, f.calculation)
	}
}

func assertProjectionEnvelope(t *testing.T, before, after json.RawMessage, field, calculation string) {
	t.Helper()
	var baseline, actual any
	beforeDecoder := json.NewDecoder(bytes.NewReader(before))
	beforeDecoder.UseNumber()
	if err := beforeDecoder.Decode(&baseline); err != nil {
		t.Fatal(err)
	}
	afterDecoder := json.NewDecoder(bytes.NewReader(after))
	afterDecoder.UseNumber()
	if err := afterDecoder.Decode(&actual); err != nil {
		t.Fatal(err)
	}
	rows := 0
	var visit func(any)
	visit = func(value any) {
		switch node := value.(type) {
		case []any:
			for _, child := range node {
				visit(child)
			}
		case map[string]any:
			if _, ok := node["project_id"]; ok {
				rows++
				fields, ok := node["property_values"].(map[string]any)
				if !ok || (node["properties_evaluated_at"] == nil || node["properties_revision"] == nil) {
					t.Fatalf("missing projection: %+v", node)
				}
				if len(fields) > 0 {
					number := fields[field].(map[string]any)
					formula := fields[calculation].(map[string]any)
					numberValue, numberErr := number["value"].(json.Number).Float64()
					formulaValue, formulaErr := formula["value"].(json.Number).Float64()
					if numberErr != nil || formulaErr != nil || formula["status"] != "value" || formulaValue != numberValue*2 {
						t.Fatalf("wrong projection: %+v", fields)
					}
				}
				delete(node, "property_values")
				delete(node, "properties_evaluated_at")
				delete(node, "properties_revision")
			} else {
				for _, child := range node {
					visit(child)
				}
			}
		}
	}
	visit(actual)
	if rows == 0 {
		t.Fatal("projection check exercised no returned rows")
	}
	if !reflect.DeepEqual(actual, baseline) {
		t.Fatalf("projection changed legacy envelope:\n%s\n%s", before, after)
	}
}

func TestRelationPropertyProjectionPreservesElevatedProjectAccess(t *testing.T) {
	t.Run("module", func(t *testing.T) {
		f := newModulePropertyFixture(t)
		assertElevatedProjectionAccess(t, f.projectTypeFixture, func(include bool) error {
			_, err := f.repo.Issues(f.ctx, uuid.MustParse(f.member), "types", uuid.MustParse(f.project), f.module, modulereads.IssueListOptions{Limit: 10, IncludeProperties: include})
			return err
		})
	})
	t.Run("cycle", func(t *testing.T) {
		f := newCyclePropertyFixture(t)
		assertElevatedProjectionAccess(t, f.projectTypeFixture, func(include bool) error {
			_, err := f.repo.Issues(f.ctx, uuid.MustParse(f.member), "types", uuid.MustParse(f.project), f.cycle, cyclereads.IssueListOptions{PerPage: 10, IncludeProperties: include})
			return err
		})
	})
}

func assertElevatedProjectionAccess(t *testing.T, f projectTypeFixture, read func(bool) error) {
	t.Helper()
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, f.member)
	for _, scenario := range []struct {
		name                                    string
		workspaceRole                           int
		workspaceActive, projectActive, allowed bool
	}{
		{"guest", 15, true, true, false},
		{"workspace admin", 20, true, true, true},
		{"inactive workspace", 20, false, true, false},
		{"inactive project", 20, true, false, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f.exec(`UPDATE workspace_members SET role=$2,is_active=$3 WHERE member_id=$1`, f.member, scenario.workspaceRole, scenario.workspaceActive)
			f.exec(`UPDATE project_members SET is_active=$2 WHERE member_id=$1`, f.member, scenario.projectActive)
			for _, include := range []bool{false, true} {
				err := read(include)
				if (err == nil) != scenario.allowed {
					t.Fatalf("include_properties=%v error=%v allowed=%v", include, err, scenario.allowed)
				}
			}
		})
	}
}
