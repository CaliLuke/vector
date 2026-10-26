//go:build integration

package workitemtypes_test

import (
	"errors"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
)

type projectTypeFixture struct {
	typeFixture
	repo      *types.Repository
	task, bug string
}

func newProjectTypeFixture(t *testing.T) projectTypeFixture {
	t.Helper()
	f := newTypeFixture(t)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0),(gen_random_uuid(),$1,$2,$4,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, f.project, f.owner, f.member)
	repo := types.NewRepository(f.db)
	catalog, err := repo.Mutate(f.ctx, f.owner, "types", "enable", "", types.Input{})
	if err != nil {
		t.Fatal(err)
	}
	task := catalog.Types[0].ID
	catalog, err = repo.Mutate(f.ctx, f.owner, "types", "create", "", types.Input{Revision: catalog.Revision, Name: "Bug", Icon: "bug", Color: "#123456"})
	if err != nil {
		t.Fatal(err)
	}
	bug := catalog.Types[1].ID
	_, err = repo.Mutate(f.ctx, f.owner, "types", "update", bug, types.Input{Revision: catalog.Revision, Name: "Bug", Icon: "bug", Color: "#123456", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	return projectTypeFixture{f, repo, task, bug}
}
func requireTypeStatus(t *testing.T, err error, status int) {
	t.Helper()
	var failure *types.Error
	if !errors.As(err, &failure) || failure.Status != status {
		t.Fatalf("want status %d, got %v", status, err)
	}
}
func (f projectTypeFixture) catalog(t *testing.T) types.ProjectCatalog {
	t.Helper()
	out, err := f.repo.ProjectCatalog(f.ctx, f.owner, "types", f.project)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func (f projectTypeFixture) mutate(t *testing.T, action string, ids ...string) types.ProjectCatalog {
	t.Helper()
	before := f.catalog(t)
	out, err := f.repo.MutateProject(f.ctx, f.owner, "types", f.project, types.ProjectInput{Revision: before.Revision, Action: action, TypeIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestProjectTypeAccessAndAtomicValidation(t *testing.T) {
	f := newProjectTypeFixture(t)
	before := f.catalog(t)
	if !before.CanManage || len(before.Types) != 2 || !before.Types[0].Available || before.Types[1].Imported {
		t.Fatalf("initial availability: %+v", before)
	}
	member, err := f.repo.ProjectCatalog(f.ctx, f.member, "types", f.project)
	if err != nil || member.CanManage {
		t.Fatalf("member read access: %+v %v", member, err)
	}
	_, err = f.repo.MutateProject(f.ctx, f.member, "types", f.project, types.ProjectInput{Revision: before.Revision, Action: "import", TypeIDs: []string{f.bug}})
	requireTypeStatus(t, err, 403)
	_, err = f.repo.ProjectCatalog(f.ctx, f.outsider, "types", f.project)
	requireTypeStatus(t, err, 403)
	_, err = f.repo.ProjectCatalog(f.ctx, f.owner, "types", uuid.New().String())
	requireTypeStatus(t, err, 403)
	for _, test := range []struct {
		name, action string
		ids          []string
		status       int
	}{
		{"default protected", "pause", []string{f.task}, 422},
		{"missing id rejects entire import", "import", []string{f.bug, uuid.New().String()}, 404},
		{"duplicate selection", "import", []string{f.bug, f.bug}, 422},
		{"empty selection", "import", nil, 422},
		{"missing import", "resume", []string{f.bug}, 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, mutationErr := f.repo.MutateProject(f.ctx, f.owner, "types", f.project, types.ProjectInput{Revision: before.Revision, Action: test.action, TypeIDs: test.ids})
			requireTypeStatus(t, mutationErr, test.status)
			after := f.catalog(t)
			if after.Revision != before.Revision || after.Types[1].Imported {
				t.Fatalf("rejected write changed catalog: %+v", after)
			}
		})
	}
	// A project administrator need not be a workspace administrator.
	f.exec(`UPDATE workspace_members SET role=15 WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.owner)
	imported := f.mutate(t, "import", f.bug)
	if !imported.Types[1].Available {
		t.Fatal("project administrator could not import")
	}
	_, err = f.repo.Mutate(f.ctx, f.owner, "types", "delete", f.bug, types.Input{Revision: imported.Revision})
	requireTypeStatus(t, err, 403)
}
func TestProjectImportsPreserveExistingWorkAndRejectStaleWrites(t *testing.T) {
	f := newProjectTypeFixture(t)
	before := f.catalog(t)
	imported := f.mutate(t, "import", f.bug)
	if !imported.Types[1].Imported || !imported.Types[1].Available {
		t.Fatalf("import not available: %+v", imported)
	}
	draft := uuid.New().String()
	f.exec(`INSERT INTO draft_issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,created_at,updated_at) VALUES($1,$2,$3,$4,'Keep me','{}','<p>Details</p>','none',0,now(),now())`, draft, f.workspace, f.project, f.bug)
	_, err := f.repo.MutateProject(f.ctx, f.owner, "types", f.project, types.ProjectInput{Revision: before.Revision, Action: "remove", TypeIDs: []string{f.bug}})
	requireTypeStatus(t, err, 409)
	paused := f.mutate(t, "pause", f.bug)
	if !paused.Types[1].Imported || paused.Types[1].Available || paused.Types[1].ProjectActive {
		t.Fatalf("pause changed wrong state: %+v", paused)
	}
	// Existing drafts remain editable while new assignments are rejected.
	f.exec(`UPDATE draft_issues SET name='Still editable' WHERE id=$1`, draft)
	_, err = f.db.Exec(f.ctx, `INSERT INTO draft_issues(id,workspace_id,project_id,type_id,description_json,description_html,priority,sort_order,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,'{}','','none',0,now(),now())`, f.workspace, f.project, f.bug)
	if err == nil {
		t.Fatal("paused type accepted a new draft")
	}
	resumed := f.mutate(t, "resume", f.bug)
	if !resumed.Types[1].Available {
		t.Fatal("resume did not restore availability")
	}
	removed := f.mutate(t, "remove", f.bug)
	if removed.Types[1].Imported || removed.Types[1].Available {
		t.Fatalf("removal failed: %+v", removed)
	}
	var typeID, html string
	if err = f.db.QueryRow(f.ctx, `SELECT type_id::text,description_html FROM draft_issues WHERE id=$1`, draft).Scan(&typeID, &html); err != nil {
		t.Fatal(err)
	}
	if typeID != f.bug || html != "<p>Details</p>" {
		t.Fatalf("removal altered existing draft: %s %s", typeID, html)
	}
	f.exec(`UPDATE draft_issues SET name='After removal' WHERE id=$1`, draft)
	f.mutate(t, "import", f.bug)
	reloaded := f.catalog(t)
	if !reloaded.Types[1].Available || reloaded.Types[1].ID != f.bug || reloaded.Types[1].Usage != 0 {
		t.Fatalf("reimport lost identity or leaked usage: %+v", reloaded)
	}
}
func TestProjectImportsRespectWorkspaceChanges(t *testing.T) {
	f := newProjectTypeFixture(t)
	imported := f.mutate(t, "import", f.bug)
	_, err := f.repo.Mutate(f.ctx, f.owner, "types", "update", f.bug, types.Input{Revision: imported.Revision, Name: "Defect", Icon: "Database", Color: "#02b55c", Active: false})
	if err != nil {
		t.Fatal(err)
	}
	changed := f.catalog(t)
	if changed.Types[1].Name != "Defect" || changed.Types[1].Available || !changed.Types[1].Imported {
		t.Fatalf("workspace definition not shared: %+v", changed)
	}
	_, err = f.repo.MutateProject(f.ctx, f.owner, "types", f.project, types.ProjectInput{Revision: changed.Revision, Action: "resume", TypeIDs: []string{f.bug}})
	requireTypeStatus(t, err, 422)
	_, err = f.repo.Mutate(f.ctx, f.owner, "types", "update", f.bug, types.Input{Revision: changed.Revision, Name: "Defect", Icon: "Database", Color: "#02b55c", Active: true, Mandatory: true})
	if err != nil {
		t.Fatal(err)
	}
	mandatory := f.catalog(t)
	_, err = f.repo.MutateProject(f.ctx, f.owner, "types", f.project, types.ProjectInput{Revision: mandatory.Revision, Action: "remove", TypeIDs: []string{f.bug}})
	requireTypeStatus(t, err, 422)
	if !mandatory.Types[1].Available || !mandatory.Types[1].Mandatory {
		t.Fatal("mandatory rollout not reflected")
	}
}
