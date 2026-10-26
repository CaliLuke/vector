//go:build integration

package workitemtypes_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/intakeinboxreads"
	"github.com/CaliLuke/go-plane/internal/workitemhistory"
	"github.com/CaliLuke/go-plane/internal/workitems"
	guuid "github.com/google/uuid"
)

type hierarchyReadEffects struct{ noRequestEffects }

func (p hierarchyReadEffects) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if name == "plane.bgtasks.recent_visited_task.recent_visited_task" {
		return nil
	}
	return p.noRequestEffects.Task(ctx, name, args, kwargs)
}

func TestHierarchyRelatedVisibilityAcrossItemProjections(t *testing.T) {
	f := newProjectTypeFixture(t)
	parent := addHierarchyItem(t, f, f.task, "")
	item := addHierarchyItem(t, f, f.task, parent)
	child := addHierarchyItem(t, f, f.task, item)
	privateProject := uuid.New().String()
	f.addProject(privateProject)
	f.exec(`UPDATE issues SET project_id=$2,name='Private related title' WHERE id=ANY($1::uuid[])`, []string{parent, child}, privateProject)
	repo, err := workitems.NewRepository(f.db, hierarchyReadEffects{noRequestEffects{t}}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	read := func(api, list bool, expand string) map[string]any {
		t.Helper()
		operation := "api_workspaces_by_slug_projects_by_project_id_issues"
		if api {
			operation = "api_v1_workspaces_by_slug_projects_by_project_id_issues"
		}
		id := ""
		if !list {
			operation += "_by_pk"
			id = item
		}
		result, readErr := repo.Execute(f.ctx, workitems.Request{Operation: operation, Method: "GET", API: api, ActorID: f.member, Slug: "types", ProjectID: f.project, IssueID: id, Timezone: "UTC", Params: map[string]string{"pk": id, "expand": expand}}, nil, "")
		if readErr != nil {
			t.Fatal(readErr)
		}
		raw, _ := json.Marshal(result.Body)
		if strings.Contains(string(raw), parent) || strings.Contains(string(raw), child) || strings.Contains(string(raw), "Private related title") {
			t.Fatalf("related item leaked (api=%v list=%v expand=%s): %s", api, list, expand, raw)
		}
		return result.Body.(map[string]any)
	}
	for _, api := range []bool{false, true} {
		for _, list := range []bool{false, true} {
			for _, expand := range []string{"", "parent,sub_issues,issue_relation,issue_related"} {
				read(api, list, expand)
			}
		}
	}
	if count := read(false, false, "")["sub_issues_count"]; count != nil && count != json.Number("0") {
		t.Fatalf("inaccessible child counted: %#v", count)
	}
	for _, method := range []string{"PUT", "GET"} {
		operation := "api_workspaces_by_slug_projects_by_project_id_issues_by_pk"
		if method == "GET" {
			f.exec(`UPDATE issues SET archived_at=current_date WHERE id=$1`, item)
			operation = "api_workspaces_by_slug_projects_by_project_id_issues_by_pk_archive"
		}
		result, readErr := repo.Execute(f.ctx, workitems.Request{Operation: operation, Method: method, ActorID: f.member, Slug: "types", ProjectID: f.project, IssueID: item, Timezone: "UTC", Params: map[string]string{"pk": item, "expand": "parent"}}, strings.NewReader(`{"name":"Updated"}`), "application/json")
		if readErr != nil {
			t.Fatalf("%s response: %v", method, readErr)
		}
		assertHierarchyPrivate(t, result.Body, parent)
	}
	var saved string
	if err = f.db.QueryRow(f.ctx, `SELECT parent_id::text FROM issues WHERE id=$1`, item).Scan(&saved); err != nil || saved != parent {
		t.Fatalf("response masking changed the relationship: %s %v", saved, err)
	}
}

func assertHierarchyPrivate(t *testing.T, body any, identities ...string) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	for _, identity := range identities {
		if strings.Contains(string(raw), identity) {
			t.Fatalf("restricted relationship exposed: %s", raw)
		}
	}
}

func TestHierarchyDraftAndPublicationReplayMaskLostParentAccess(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	parent := addHierarchyItem(t, f, f.task, "")
	f.exec(`UPDATE draft_issues SET parent_id=$2 WHERE id=$1`, draft, parent)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, result)
	privateProject := uuid.New().String()
	f.addProject(privateProject)
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, parent, privateProject)
	replayed, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil || resultIssueID(t, replayed) != item {
		t.Fatalf("publication retry: %v", err)
	}
	assertHierarchyPrivate(t, replayed.Body, parent)
	// An independently retained draft keeps its link internally, while its owner
	// cannot discover the now-private parent's identity from draft detail/list.
	retainedDraft := uuid.New().String()
	f.exec(`INSERT INTO draft_issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,'Retained draft','{}','','none',0,now(),now(),$5)`, retainedDraft, f.workspace, f.project, f.task, f.member)
	f.exec(`UPDATE draft_issues SET parent_id=$2 WHERE id=$1`, retainedDraft, parent)
	reads, err := intakeinboxreads.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := reads.DraftIssueDetail(f.ctx, uuid.MustParse(f.member), "types", uuid.MustParse(retainedDraft))
	if err != nil {
		t.Fatal(err)
	}
	assertHierarchyPrivate(t, detail, parent)
	list, err := reads.DraftIssueList(f.ctx, uuid.MustParse(f.member), "types", intakeinboxreads.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertHierarchyPrivate(t, list, parent)
}

func TestHierarchyHistoryMasksIdentifiersAndHistoricalParentNames(t *testing.T) {
	f := newProjectTypeFixture(t)
	parent := addHierarchyItem(t, f, f.task, "")
	item := addHierarchyItem(t, f, f.task, parent)
	privateProject := uuid.New().String()
	f.addProject(privateProject)
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, parent, privateProject)
	f.exec(`INSERT INTO issue_activities(id,workspace_id,project_id,issue_id,actor_id,created_at,updated_at,verb,field,old_value,new_value,old_identifier,new_identifier,comment,attachments,epoch) VALUES(gen_random_uuid(),$1,$2,$3,$4,now(),now(),'updated','parent','Secret parent title','',$5,NULL,'removed Secret parent title','{}',0)`, f.workspace, f.project, item, f.owner, parent)
	version := uuid.New().String()
	f.exec(`INSERT INTO issue_versions(id,workspace_id,project_id,issue_id,parent,name,priority,assignees,sequence_id,labels,sort_order,is_draft,modules,properties,meta,last_saved_at,owned_by_id,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,'Child','none','{}',1,'{}',0,false,'{}','{}','{}',now(),$6,now(),now(),$6)`, version, f.workspace, f.project, item, parent, f.owner)
	repo, err := workitemhistory.NewRepository(f.db)
	if err != nil {
		t.Fatal(err)
	}
	request := workitemhistory.Request{Operation: "history", Slug: "types", ProjectID: guuid.MustParse(f.project), ResourceID: guuid.MustParse(item), UserID: guuid.MustParse(f.member), Timezone: "UTC"}
	for _, kind := range []string{"", "issue-property", "issue-comment"} {
		request.ActivityType = &kind
		history, readErr := repo.Read(f.ctx, request)
		if readErr != nil {
			t.Fatal(readErr)
		}
		assertHierarchyPrivate(t, history, parent, "Secret parent title")
	}
	request.Operation, request.VersionID = "version_detail", guuid.MustParse(version)
	versionBody, err := repo.Read(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	assertHierarchyPrivate(t, versionBody, parent)
}

func TestHierarchyVisibilityUsesCurrentMembershipAndGuestPolicy(t *testing.T) {
	f := newProjectTypeFixture(t)
	parent := addHierarchyItem(t, f, f.task, "")
	row := map[string]any{"parent_id": parent, "workspace_id": f.workspace}
	check := func(visible bool) {
		t.Helper()
		masked, err := djangoapi.MaskIssueParent(f.ctx, f.db, row, f.member)
		if err != nil || (masked["parent_id"] != nil) != visible {
			t.Fatalf("parent visibility want %v: %#v %v", visible, masked, err)
		}
		if row["parent_id"] != parent {
			t.Fatal("masking mutated the internal snapshot")
		}
	}
	check(true)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	check(false)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, parent, f.member)
	check(true)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, parent, f.owner)
	f.exec(`UPDATE projects SET guest_view_all_features=true WHERE id=$1`, f.project)
	check(true)
	for _, mutation := range []struct{ change, restore string }{
		{`UPDATE project_members SET deleted_at=now() WHERE member_id=$1`, `UPDATE project_members SET deleted_at=NULL WHERE member_id=$1`},
		{`UPDATE project_members SET is_active=false WHERE member_id=$1`, `UPDATE project_members SET is_active=true WHERE member_id=$1`},
		{`UPDATE workspace_members SET deleted_at=now() WHERE member_id=$1`, `UPDATE workspace_members SET deleted_at=NULL WHERE member_id=$1`},
		{`UPDATE workspace_members SET is_active=false WHERE member_id=$1`, `UPDATE workspace_members SET is_active=true WHERE member_id=$1`},
	} {
		f.exec(mutation.change, f.member)
		check(false)
		f.exec(mutation.restore, f.member)
		check(true)
	}
	f.exec(`UPDATE projects SET deleted_at=now() WHERE id=$1`, f.project)
	check(false)
	f.exec(`UPDATE projects SET deleted_at=NULL WHERE id=$1`, f.project)
	f.exec(`UPDATE issues SET is_draft=true WHERE id=$1`, parent)
	check(false)
	f.exec(`UPDATE issues SET is_draft=false WHERE id=$1`, parent)
	row["workspace_id"] = uuid.New().String()
	check(false)
}
