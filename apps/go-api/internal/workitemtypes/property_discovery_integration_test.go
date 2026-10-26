//go:build integration

package workitemtypes_test

import (
	"context"
	"encoding/json"
	"testing"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestPropertyDiscoveryScopesPlansAndRetainsPausedTypes(t *testing.T) {
	f := newProjectTypeFixture(t)
	number := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	f.exec(`INSERT INTO work_item_type_properties(type_id,property_id) VALUES($1,$2)`, f.bug, number)
	f.exec(`UPDATE issue_types SET is_active=false WHERE id=$1`, f.bug)
	privateProject := uuid.NewString()
	f.addProject(privateProject)
	_, err := types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", "", func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) (bool, error) {
		for _, typeID := range []string{f.task, f.bug} {
			projected, e := snapshot.Project(f.project, typeID, types.ItemFormulaInputs{}, map[string]any{number: float64(4)})
			if e != nil || projected.Values[number].Value != float64(4) || projected.EvaluatedAt != snapshot.EvaluatedAt {
				t.Fatalf("projection: %+v %v", projected, e)
			}
		}
		_, denied := snapshot.Project(privateProject, f.task, types.ItemFormulaInputs{}, nil)
		requireTypeStatus(t, denied, 403)
		_, missing := snapshot.Project(f.project, uuid.NewString(), types.ItemFormulaInputs{}, nil)
		requireTypeStatus(t, missing, 400)
		untyped, e := snapshot.Project(f.project, "", types.ItemFormulaInputs{}, map[string]any{number: float64(9)})
		if e != nil || len(untyped.Values) != 0 {
			t.Fatalf("untyped projection: %+v %v", untyped, e)
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = types.WithPropertyDiscoveryRead(f.ctx, f.db, f.outsider, "types", "", func(_ pgx.Tx, _ *types.PropertyDiscoverySnapshot) (bool, error) {
		t.Fatal("outsider reached projection callback")
		return false, nil
	})
	requireTypeStatus(t, err, 403)
}

func TestPropertyDiscoveryFreezesSchemaAndFormulaTimeAcrossRows(t *testing.T) {
	f := newProjectTypeFixture(t)
	fixed, clock := uuid.NewString(), uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: fixed, Title: "Instructions", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Read-only", Content: "Before"}},
		property.Definition{ID: clock, Title: "Clock", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "NOW()", Result: "Date"}},
	)
	writer, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := writer.Close(context.WithoutCancel(f.ctx)); e != nil {
			t.Error(e)
		}
	}()
	_, err = types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", f.project, func(tx pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) (bool, error) {
		before, e := snapshot.Project(f.project, f.task, types.ItemFormulaInputs{}, nil)
		if e != nil {
			return false, e
		}
		if _, e = writer.Exec(f.ctx, `UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,content}','"After"') WHERE id=$1`, fixed); e != nil {
			return false, e
		}
		after, e := snapshot.Project(f.project, f.task, types.ItemFormulaInputs{}, nil)
		if e != nil || after.Values[fixed].Value != "Before" || after.Computed[clock].Value != before.Computed[clock].Value {
			t.Fatalf("mixed projected snapshot: %+v %v", after, e)
		}
		var stored string
		if e = tx.QueryRow(f.ctx, `SELECT definition->'configuration'->>'content' FROM work_item_properties WHERE id=$1`, fixed).Scan(&stored); e != nil {
			return false, e
		}
		if stored != "Before" {
			t.Fatal("callback transaction drifted")
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", f.project, func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) (bool, error) {
		after, e := snapshot.Project(f.project, f.task, types.ItemFormulaInputs{}, nil)
		if e != nil || after.Values[fixed].Value != "After" {
			t.Fatalf("new request failed to see schema: %+v %v", after, e)
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPropertyDiscoveryRejectsOverflowWithoutCallingReader(t *testing.T) {
	f := newProjectTypeFixture(t)
	definition, _ := json.Marshal(property.Definition{Title: "Extra", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	f.exec(`WITH inserted AS (INSERT INTO work_item_properties(id,workspace_id,definition) SELECT gen_random_uuid(),$1,$2::jsonb FROM generate_series(1,1001) RETURNING id) INSERT INTO work_item_type_properties(type_id,property_id) SELECT $3,id FROM inserted`, f.workspace, string(definition), f.task)
	_, err := types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", f.project, func(_ pgx.Tx, _ *types.PropertyDiscoverySnapshot) (bool, error) {
		t.Fatal("partial catalog reached reader")
		return false, nil
	})
	requireTypeStatus(t, err, 422)
}
