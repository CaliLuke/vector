//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
)

func searchHierarchyItems(t *testing.T, f projectTypeFixture, repo *workitems.Repository, params map[string]string) []map[string]any {
	t.Helper()
	result, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_search_issues", Method: "GET", ActorID: f.member, Slug: "types", ProjectID: f.project, Params: params}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	return result.Body.([]map[string]any)
}

func TestHierarchyParentSearchChecksLevelBeforeLimitAndProspectiveType(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	parent := addHierarchyItem(t, f, f.bug, "")
	child := addHierarchyItem(t, f, f.task, "")
	// The only eligible parent is older than a complete page of wrong-level items.
	f.exec(`UPDATE issues SET created_at=now()-interval '1 day' WHERE id=$1`, parent)
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,type_id,name,description_html,description_json,priority,sequence_id,sort_order,is_draft,created_at,updated_at,created_by_id) SELECT gen_random_uuid(),$1,$2,$3,'Task candidate','','{}','none',1000+n,n,false,now(),now(),$4 FROM generate_series(1,101) n`, f.workspace, f.project, f.task, f.member)
	input := hierarchyInput(t, f, 1)
	review, err := f.repo.PreviewHierarchy(f.ctx, f.owner, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = review.ID
	if _, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", input); err != nil {
		t.Fatal(err)
	}
	repo, err := workitems.NewRepository(f.db, hierarchyReadEffects{noRequestEffects{t}}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, anchor := range []string{"", child} {
		found := searchHierarchyItems(t, f, repo, map[string]string{"parent": "true", "child_type_id": f.task, "issue_id": anchor, "workspace_search": "true"})
		if len(found) != 1 || found[0]["id"] != parent {
			t.Fatalf("parent candidates: %#v", found)
		}
	}
	found := searchHierarchyItems(t, f, repo, map[string]string{"parent": "true", "child_type_id": f.bug, "issue_id": child, "workspace_search": "true"})
	if len(found) != 0 {
		t.Fatalf("prospective type ignored: %#v", found)
	}
	found = searchHierarchyItems(t, f, repo, map[string]string{"parent": "true", "workspace_search": "true"})
	if len(found) != 0 {
		t.Fatalf("unselected child type has configured parents: %#v", found)
	}
}

func TestHierarchySearchAndWritesAllowOnlyReadableCrossProjectParents(t *testing.T) {
	f := newProjectTypeFixture(t)
	parent := addHierarchyItem(t, f, f.task, "")
	project := uuid.New().String()
	f.addProject(project)
	f.exec(`UPDATE issues SET project_id=$2,name='Cross project' WHERE id=$1`, parent, project)
	repo, _, _ := seedPublicationDraft(t, f)
	params := map[string]string{"parent": "true", "workspace_search": "true", "search": "Cross project"}
	if found := searchHierarchyItems(t, f, repo, params); len(found) != 0 {
		t.Fatalf("private parent found: %#v", found)
	}
	_, err := writeValueItem(f, repo, "POST", "", `{"name":"Child","parent_id":"`+parent+`"}`)
	var httpErr *djangoapi.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != 400 {
		t.Fatalf("private parent accepted: %v", err)
	}
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, project, f.member)
	found := searchHierarchyItems(t, f, repo, params)
	if len(found) != 1 || found[0]["id"] != parent {
		t.Fatalf("visible cross-project parent absent: %#v", found)
	}
	result, err := writeValueItem(f, repo, "POST", "", `{"name":"Child","parent_id":"`+parent+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	child := resultIssueID(t, result)
	var saved string
	if err = f.db.QueryRow(f.ctx, `SELECT parent_id::text FROM issues WHERE id=$1`, child).Scan(&saved); err != nil || saved != parent {
		t.Fatalf("parent=%s err=%v", saved, err)
	}
	f.exec(`UPDATE project_members SET deleted_at=now() WHERE project_id=$1 AND member_id=$2`, project, f.member)
	if found = searchHierarchyItems(t, f, repo, params); len(found) != 0 {
		t.Fatal("revoked parent returned")
	}
	// Existing hidden links survive unrelated edits, but an explicit assignment is rejected.
	if _, err = writeValueItem(f, repo, "PATCH", child, `{"name":"Retained"}`); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"PATCH", "PUT"} {
		_, err = writeValueItem(f, repo, method, child, `{"name":"Rejected","parent_id":"`+parent+`"}`)
		if !errors.As(err, &httpErr) || httpErr.Status != 400 {
			t.Fatalf("%s private parent accepted: %v", method, err)
		}
	}
}

func TestHierarchySearchExcludesCyclesThroughHiddenNodesAndRejectsPrivateAnchor(t *testing.T) {
	f := newProjectTypeFixture(t)
	root := addHierarchyItem(t, f, f.task, "")
	hidden := addHierarchyItem(t, f, f.task, root)
	descendant := addHierarchyItem(t, f, f.task, hidden)
	unrelated := addHierarchyItem(t, f, f.task, "")
	project := uuid.New().String()
	f.addProject(project)
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, hidden, project)
	repo, err := workitems.NewRepository(f.db, hierarchyReadEffects{noRequestEffects{t}}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	found := searchHierarchyItems(t, f, repo, map[string]string{"parent": "true", "issue_id": root, "workspace_search": "true"})
	if len(found) != 1 || found[0]["id"] != unrelated {
		t.Fatalf("cycle candidate returned: %#v descendant=%s", found, descendant)
	}
	// Ancestor exclusion is equally recursive when selecting children.
	found = searchHierarchyItems(t, f, repo, map[string]string{"sub_issue": "true", "issue_id": descendant, "workspace_search": "true"})
	if len(found) != 1 || found[0]["id"] != unrelated {
		t.Fatalf("ancestor candidate returned: %#v", found)
	}
	result, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_search_issues", Method: "GET", ActorID: f.member, Slug: "types", ProjectID: f.project, Params: map[string]string{"parent": "true", "issue_id": hidden, "workspace_search": "true"}}, nil, "")
	var httpErr *djangoapi.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != 404 {
		t.Fatalf("private anchor: %#v %v", result, err)
	}
	raw, _ := json.Marshal(httpErr.Body)
	if strings.Contains(string(raw), hidden) {
		t.Fatal("error exposed private anchor")
	}
}

func attachHierarchyChildren(f projectTypeFixture, repo *workitems.Repository, parent string, children ...string) error {
	raw, _ := json.Marshal(map[string]any{"sub_issue_ids": children})
	_, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_sub_issues", Method: "POST", ActorID: f.member, Slug: "types", ProjectID: f.project, IssueID: parent, Params: map[string]string{}}, strings.NewReader(string(raw)), "application/json")
	return err
}

func TestHierarchyBulkAttachmentRejectsDeletedProjectWithoutPartialWrites(t *testing.T) {
	f := newProjectTypeFixture(t)
	parent := addHierarchyItem(t, f, f.task, "")
	child := addHierarchyItem(t, f, f.task, "")
	hidden := addHierarchyItem(t, f, f.task, "")
	project := uuid.New().String()
	f.addProject(project)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, project, f.member)
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, hidden, project)
	f.exec(`UPDATE projects SET deleted_at=now() WHERE id=$1`, project)
	repo, err := workitems.NewRepository(f.db, hierarchyReadEffects{noRequestEffects{t}}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	err = attachHierarchyChildren(f, repo, parent, child, hidden)
	var denied *djangoapi.HTTPError
	if !errors.As(err, &denied) || denied.Status != 400 {
		t.Fatalf("deleted-project child accepted: %v", err)
	}
	var links int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM issues WHERE parent_id=$1`, parent).Scan(&links); err != nil || links != 0 {
		t.Fatalf("partial bulk change: %d %v", links, err)
	}
}

func TestHierarchyParentWritersReauthorizeAfterSchemaWait(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "bulk"}[bulk], func(t *testing.T) {
			f := newProjectTypeFixture(t)
			parent := addHierarchyItem(t, f, f.task, "")
			child := addHierarchyItem(t, f, f.task, "")
			connection := hierarchyConnection(t, f)
			repo, err := workitems.NewRepository(connection, hierarchyReadEffects{noRequestEffects{t}}, "http://example.test")
			if err != nil {
				t.Fatal(err)
			}
			lock := hierarchyTransaction(t, f)
			done := make(chan error, 1)
			go func() {
				if bulk {
					done <- attachHierarchyChildren(f, repo, parent, child)
					return
				}
				_, writeErr := writeValueItem(f, repo, "PATCH", child, `{"parent_id":"`+parent+`"}`)
				done <- writeErr
			}()
			waitForSchemaWriter(t, f, lock, int(connection.PgConn().PID()), done)
			if _, err = lock.Exec(f.ctx, `UPDATE workspace_members SET is_active=false WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.member); err != nil {
				t.Fatal(err)
			}
			if err = lock.Commit(f.ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err = <-done:
				var denied *djangoapi.HTTPError
				if !errors.As(err, &denied) || denied.Status != 403 {
					t.Fatalf("revoked writer accepted: %v", err)
				}
			case <-f.ctx.Done():
				t.Fatal("writer did not finish")
			}
			var linked bool
			if err = f.db.QueryRow(f.ctx, `SELECT parent_id IS NOT NULL FROM issues WHERE id=$1`, child).Scan(&linked); err != nil || linked {
				t.Fatalf("revoked writer changed parent: %t %v", linked, err)
			}
		})
	}
}

func TestHierarchyChildSearchRequiresWriteAccessButParentSearchOnlyReadAccess(t *testing.T) {
	f := newProjectTypeFixture(t)
	anchor := addHierarchyItem(t, f, f.task, "")
	guestItem := addHierarchyItem(t, f, f.task, "")
	project := uuid.New().String()
	f.addProject(project)
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, guestItem, project)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,5,true,now(),now(),'{}','{}','{}',0)`, f.workspace, project, f.member)
	f.exec(`UPDATE projects SET guest_view_all_features=true WHERE id=$1`, project)
	repo, err := workitems.NewRepository(f.db, hierarchyReadEffects{noRequestEffects{t}}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	parents := searchHierarchyItems(t, f, repo, map[string]string{"parent": "true", "issue_id": anchor, "workspace_search": "true"})
	if len(parents) != 1 || parents[0]["id"] != guestItem {
		t.Fatalf("readable guest parent missing: %#v", parents)
	}
	children := searchHierarchyItems(t, f, repo, map[string]string{"sub_issue": "true", "issue_id": anchor, "workspace_search": "true"})
	if len(children) != 0 {
		t.Fatalf("read-only guest child offered: %#v", children)
	}
	f.exec(`UPDATE project_members SET role=15 WHERE project_id=$1 AND member_id=$2`, project, f.member)
	children = searchHierarchyItems(t, f, repo, map[string]string{"sub_issue": "true", "issue_id": anchor, "workspace_search": "true"})
	if len(children) != 1 || children[0]["id"] != guestItem {
		t.Fatalf("writable child missing: %#v", children)
	}
	if err = attachHierarchyChildren(f, repo, anchor, guestItem); err != nil {
		t.Fatal(err)
	}
}
