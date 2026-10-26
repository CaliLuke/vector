//go:build integration

package workitemtypes_test

import (
	"context"
	"reflect"
	"testing"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestStoredPropertyProjectionUsesCanonicalSnapshotAndDuplicates(t *testing.T) {
	f := newDiscoveryEstimateFixture(t)
	writer, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close(context.WithoutCancel(f.ctx)) }()
	_, err = types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", f.project, func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) (bool, error) {
		refs := []types.DiscoveryItemReference{{ID: f.other, ProjectID: f.project}, {ID: f.item, ProjectID: f.project}, {ID: f.item, ProjectID: f.project}}
		first, readErr := snapshot.ProjectStoredItems(f.ctx, refs, types.DiscoveryItemVisibility)
		if readErr != nil {
			return false, readErr
		}
		if len(first) != 3 || first[0].Values[f.calculation].Status != "empty" || first[1].Values[f.calculation].Value != float64(10) || !reflect.DeepEqual(first[1], first[2]) {
			t.Fatalf("canonical batch: %+v", first)
		}
		if _, readErr = writer.Exec(f.ctx, `UPDATE issues SET name='Changed'; UPDATE estimate_points SET value='9'`); readErr != nil {
			return false, readErr
		}
		second, readErr := snapshot.ProjectStoredItems(f.ctx, refs, types.DiscoveryItemVisibility)
		if readErr == nil && !reflect.DeepEqual(first, second) {
			t.Fatal("canonical inputs escaped snapshot")
		}
		return true, readErr
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStoredPropertyProjectionRejectsWholeInvalidBatch(t *testing.T) {
	f := newDiscoveryEstimateFixture(t)
	otherProject := uuid.NewString()
	f.addProject(otherProject)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, otherProject, f.member)
	for _, scenario := range []string{"missing", "draft", "deleted", "hidden guest", "hidden workspace admin", "conflicting project", "mismatched accessible project"} {
		t.Run(scenario, func(t *testing.T) {
			f.exec(`UPDATE issues SET is_draft=false,deleted_at=NULL; UPDATE project_members SET role=15; UPDATE workspace_members SET role=15`)
			target, project := f.other, f.project
			switch scenario {
			case "missing":
				target = uuid.NewString()
			case "draft":
				f.exec(`UPDATE issues SET is_draft=true WHERE id=$1`, target)
			case "deleted":
				f.exec(`UPDATE issues SET deleted_at=now() WHERE id=$1`, target)
			case "hidden guest", "hidden workspace admin":
				if scenario == "hidden workspace admin" {
					f.exec(`UPDATE workspace_members SET role=20 WHERE member_id=$1`, f.member)
				}
				f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1`, f.member)
				f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, f.item, f.member)
				f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, target, f.owner)
			case "conflicting project":
				target, project = f.item, uuid.NewString()
			case "mismatched accessible project":
				target, project = f.item, otherProject
			}
			_, err := types.WithPropertyDiscoveryRead(f.ctx, f.db, f.member, "types", "", func(_ pgx.Tx, snapshot *types.PropertyDiscoverySnapshot) (bool, error) {
				rows := []map[string]any{{"id": f.item, "project_id": f.project}, {"id": target, "project_id": project}}
				denied := snapshot.ProjectStoredRows(f.ctx, rows, types.DiscoveryItemVisibility)
				requireTypeStatus(t, denied, 403)
				for _, row := range rows {
					if _, ok := row["property_values"]; ok {
						t.Fatal("failed batch partially enriched rows")
					}
				}
				return true, nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
