//go:build integration

package workitemtypes_test

import (
	"context"
	"reflect"
	"testing"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type discoveryEstimateFixture struct {
	projectTypeFixture
	item, other, calculation, clock, estimate, point string
}

func newDiscoveryEstimateFixture(t *testing.T) discoveryEstimateFixture {
	t.Helper()
	f := newProjectTypeFixture(t)
	_, ids := seedDiscoveryItems(t, f)
	out := discoveryEstimateFixture{projectTypeFixture: f, item: ids[0], other: ids[1], calculation: uuid.NewString(), clock: uuid.NewString(), estimate: uuid.NewString(), point: uuid.NewString()}
	attachValueProperties(t, f,
		property.Definition{ID: out.calculation, Title: "Estimate", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{estimate_point}} * 2", Result: "Number"}},
		property.Definition{ID: out.clock, Title: "Clock", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "NOW()", Result: "Date"}},
	)
	f.exec(`INSERT INTO estimates(id,workspace_id,project_id,name,description,type,last_used,created_at,updated_at) VALUES($1,$2,$3,'Batch points','','points',true,now(),now())`, out.estimate, f.workspace, f.project)
	f.exec(`INSERT INTO estimate_points(id,workspace_id,project_id,estimate_id,key,value,description,created_at,updated_at) VALUES($1,$2,$3,$4,1,'5','',now(),now())`, out.point, f.workspace, f.project, out.estimate)
	f.exec(`UPDATE issues SET estimate_point_id=$2 WHERE id=$1`, out.item, out.point)
	return out
}

func TestPropertyDiscoveryItemsPreserveOrderDuplicatesAndSnapshot(t *testing.T) {
	f := newDiscoveryEstimateFixture(t)
	writer, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := writer.Close(context.WithoutCancel(f.ctx)); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	stale := "99"
	item := types.DiscoveryItem{ID: f.item, ProjectID: f.project, TypeID: f.task, EstimatePointID: uuid.NewString(), Standard: types.ItemFormulaInputs{Estimate: &stale}}
	missing := types.DiscoveryItem{ID: f.other, ProjectID: f.project, TypeID: f.task, Standard: types.ItemFormulaInputs{Estimate: &stale}}
	items := []types.DiscoveryItem{missing, item, item}
	original := append([]types.DiscoveryItem(nil), items...)
	_, err = types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", f.project, func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) (bool, error) {
		first, readErr := snapshot.ProjectItems(f.ctx, items)
		if readErr != nil {
			return false, readErr
		}
		if len(first) != 3 || first[0].Values[f.calculation].Status != "empty" || first[1].Values[f.calculation].Value != float64(10) || !reflect.DeepEqual(first[1], first[2]) {
			t.Fatalf("ordered duplicate projections: %+v", first)
		}
		if first[0].Values[f.clock].Value != first[1].Values[f.clock].Value || first[0].EvaluatedAt != snapshot.EvaluatedAt {
			t.Fatal("batch used different evaluation instants")
		}
		if _, readErr = writer.Exec(f.ctx, `UPDATE estimate_points SET value='7' WHERE id=$1`, f.point); readErr != nil {
			return false, readErr
		}
		second, readErr := snapshot.ProjectItems(f.ctx, items)
		if readErr != nil {
			return false, readErr
		}
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(items, original) || stale != "99" {
			t.Fatal("projection changed snapshot values or caller inputs")
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPropertyDiscoveryItemsDropRemovedEstimateValues(t *testing.T) {
	f := newDiscoveryEstimateFixture(t)
	stale := "99"
	item := types.DiscoveryItem{ID: f.item, ProjectID: f.project, TypeID: f.task, EstimatePointID: f.point, Standard: types.ItemFormulaInputs{Estimate: &stale}}
	for _, table := range []string{"estimate_points", "estimates"} {
		t.Run(table, func(t *testing.T) {
			id := f.point
			if table == "estimates" {
				id = f.estimate
				f.exec(`UPDATE estimate_points SET deleted_at=NULL WHERE id=$1`, f.point)
			}
			f.exec(`UPDATE `+table+` SET deleted_at=now() WHERE id=$1`, id)
			_, err := types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", f.project, func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) (bool, error) {
				values, readErr := snapshot.ProjectItems(f.ctx, []types.DiscoveryItem{item})
				if readErr == nil && (len(values) != 1 || values[0].Values[f.calculation].Status != "empty") {
					t.Fatalf("removed estimate retained a numeric value: %+v", values)
				}
				return true, readErr
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPropertyDiscoveryItemsValidateEveryScopeBeforeHydration(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, err := types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", f.project, func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) (bool, error) {
		// An attempted hydration of the first item would fail its UUID cast. Scope
		// validation must reject the later item first, without a partial projection.
		first := types.DiscoveryItem{ID: "invalid-id", ProjectID: f.project, TypeID: f.task, EstimatePointID: "hint"}
		values, denied := snapshot.ProjectItems(f.ctx, []types.DiscoveryItem{first, {ProjectID: uuid.NewString(), TypeID: f.task}})
		requireTypeStatus(t, denied, 403)
		if values != nil {
			t.Fatal("denied batch returned partial projections")
		}
		values, invalid := snapshot.ProjectItems(f.ctx, []types.DiscoveryItem{first, {ProjectID: f.project, TypeID: uuid.NewString()}})
		requireTypeStatus(t, invalid, 400)
		if values != nil {
			t.Fatal("invalid batch returned partial projections")
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
