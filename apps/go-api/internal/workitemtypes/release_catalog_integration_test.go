//go:build integration

package workitemtypes_test

import (
	"strings"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func releaseCatalog(t *testing.T, f projectTypeFixture) types.ReleaseCatalog {
	t.Helper()
	out, err := f.repo.ReleaseCatalog(f.ctx, f.owner, "types", f.project)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func createTestRelease(t *testing.T, f projectTypeFixture) types.ReleaseInput {
	t.Helper()
	input := types.ReleaseInput{Action: "create", ID: uuid.New().String(), Revision: releaseCatalog(t, f).Revision, Values: types.ReleaseValues{Name: "October", Description: "Release notes", TargetDate: "2026-10-31"}}
	if _, err := f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input); err != nil {
		t.Fatal(err)
	}
	return input
}
func TestReleaseCatalogLifecycleAndCanonicalRetry(t *testing.T) {
	f := newProjectTypeFixture(t)
	input := types.ReleaseInput{Action: "create", ID: strings.ToUpper(uuid.New().String()), Revision: releaseCatalog(t, f).Revision, Values: types.ReleaseValues{Name: " October ", TargetDate: "2026-10-31"}}
	created, err := f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Releases) != 1 || created.Releases[0].ID != strings.ToLower(input.ID) || created.Releases[0].Name != "October" || created.Revision != input.Revision+1 {
		t.Fatalf("creation: %+v", created)
	}
	retried, err := f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input)
	if err != nil || retried.Revision != created.Revision || len(retried.Releases) != 1 {
		t.Fatalf("retry changed catalog: %+v %v", retried, err)
	}
	input.Action = "update"
	input.Revision = created.Revision
	input.Values = types.ReleaseValues{Name: "November", Description: "Updated"}
	updated, err := f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input)
	if err != nil || updated.Releases[0].Name != "November" || updated.Releases[0].TargetDate != "" {
		t.Fatalf("update: %+v %v", updated, err)
	}
	input.Action = "delete"
	input.Revision = updated.Revision
	deleted, err := f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input)
	if err != nil || len(deleted.Releases) != 0 {
		t.Fatalf("delete: %+v %v", deleted, err)
	}
}
func TestReleaseCatalogAuthorizationAndProjectIsolation(t *testing.T) {
	f := newProjectTypeFixture(t)
	input := createTestRelease(t, f)
	member, err := f.repo.ReleaseCatalog(f.ctx, f.member, "types", f.project)
	if err != nil || member.CanManage || len(member.Releases) != 1 {
		t.Fatalf("member read: %+v %v", member, err)
	}
	_, err = f.repo.MutateReleaseCatalog(f.ctx, f.member, "types", f.project, input)
	requireTypeStatus(t, err, 403)
	_, err = f.repo.ReleaseCatalog(f.ctx, f.outsider, "types", f.project)
	requireTypeStatus(t, err, 403)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	guest, err := f.repo.ReleaseCatalog(f.ctx, f.member, "types", f.project)
	if err != nil || guest.CanManage || len(guest.Releases) != 1 {
		t.Fatalf("guest catalog: %+v %v", guest, err)
	}
	other := uuid.New().String()
	f.addProject(other)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0)`, f.workspace, other, f.owner)
	foreign, err := f.repo.ReleaseCatalog(f.ctx, f.owner, "types", other)
	if err != nil || len(foreign.Releases) != 0 {
		t.Fatalf("other project leaked release: %+v %v", foreign, err)
	}
	input.Revision = foreign.Revision
	_, err = f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", other, input)
	requireTypeStatus(t, err, 409)
	input.Action = "update"
	_, err = f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", other, input)
	requireTypeStatus(t, err, 404)
}
func TestReleaseCatalogRejectedChangesAreAtomic(t *testing.T) {
	f := newProjectTypeFixture(t)
	input := createTestRelease(t, f)
	before := releaseCatalog(t, f)
	tests := []struct {
		name   string
		input  types.ReleaseInput
		status int
	}{
		{"stale", types.ReleaseInput{Action: "update", ID: input.ID, Revision: input.Revision, Values: types.ReleaseValues{Name: "Changed"}}, 409},
		{"duplicate name", types.ReleaseInput{Action: "create", ID: uuid.New().String(), Revision: before.Revision, Values: input.Values}, 422},
		{"invalid date", types.ReleaseInput{Action: "update", ID: input.ID, Revision: before.Revision, Values: types.ReleaseValues{Name: "Changed", TargetDate: "2026-02-30"}}, 422},
		{"empty name", types.ReleaseInput{Action: "update", ID: input.ID, Revision: before.Revision}, 422},
		{"id reused", types.ReleaseInput{Action: "create", ID: input.ID, Revision: before.Revision, Values: types.ReleaseValues{Name: "Different"}}, 409},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, test.input)
			requireTypeStatus(t, err, test.status)
			after := releaseCatalog(t, f)
			if after.Revision != before.Revision || len(after.Releases) != 1 || after.Releases[0] != before.Releases[0] {
				t.Fatalf("rejected mutation changed catalog: %+v", after)
			}
		})
	}
}
func TestReleaseDeletionProtectsRetainedItemAndPrivateDraftValues(t *testing.T) {
	f := newProjectTypeFixture(t)
	field := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: field, Title: "Delivery", Kind: property.Release, Configuration: property.Configuration{Active: true}})
	input := createTestRelease(t, f)
	item := addHierarchyItem(t, f, f.task, "")
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($2::text,$3::text),deleted_at=now() WHERE id=$1`, item, field, input.ID)
	f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,active}','false') WHERE id=$1`, field)
	input.Action = "delete"
	input.Revision = releaseCatalog(t, f).Revision
	_, err := f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input)
	requireTypeStatus(t, err, 422)
	f.exec(`UPDATE issues SET custom_properties='{}' WHERE id=$1`, item)
	draft := uuid.New().String()
	f.exec(`INSERT INTO draft_issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,created_at,updated_at,created_by_id,custom_properties) VALUES($1,$2,$3,$4,'Private','{}','','none',0,now(),now(),$5,jsonb_build_object($6::text,jsonb_build_array($7::text)))`, draft, f.workspace, f.project, f.task, f.member, field, input.ID)
	_, err = f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input)
	requireTypeStatus(t, err, 422)
	f.exec(`UPDATE draft_issues SET custom_properties='{}' WHERE id=$1`, draft)
	removed, err := f.repo.MutateReleaseCatalog(f.ctx, f.owner, "types", f.project, input)
	if err != nil || len(removed.Releases) != 0 {
		t.Fatalf("unused release retained: %+v %v", removed, err)
	}
}
