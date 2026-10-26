//go:build integration

package workitemtypes_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func patchTypedIntake(f projectTypeFixture, repo *workitems.Repository, item, intake string, api bool, body string) error {
	_, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_intake_issues", Method: "PATCH", API: api, ActorID: f.owner, Slug: "types", ProjectID: f.project, IssueID: item, Timezone: "UTC", Params: map[string]string{"intake_id": intake}}, strings.NewReader(body), "application/json")
	return err
}

func TestTypedIntakePatchPersistsCustomValues(t *testing.T) {
	for _, api := range []bool{false, true} {
		t.Run(fmt.Sprintf("api=%t", api), func(t *testing.T) {
			f, repo, item, field, intake, _ := seedIntakeWriter(t)
			err := patchTypedIntake(f, repo, item, intake, api, `{"issue":{"custom_properties":{"`+field+`":12}}}`)
			if err != nil {
				t.Fatal(err)
			}
			if got := draftValues(t, f, "issues", item)[field]; got != float64(12) {
				t.Fatalf("saved value = %v", got)
			}
		})
	}
}

func TestTypedIntakePatchRejectsMalformedValuesAtomically(t *testing.T) {
	for _, api := range []bool{false, true} {
		t.Run(fmt.Sprintf("api=%t", api), func(t *testing.T) {
			f, repo, item, field, intake, _ := seedIntakeWriter(t)
			var before int
			if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects`).Scan(&before); err != nil {
				t.Fatal(err)
			}
			err := patchTypedIntake(f, repo, item, intake, api, `{"issue":{"name":"Rejected","custom_properties":{"`+field+`":"twelve"}},"status":1}`)
			var invalid *djangoapi.HTTPError
			if !errors.As(err, &invalid) || invalid.Status != 400 {
				t.Fatalf("want 400, got %v", err)
			}
			var name string
			var status, after int
			if err = f.db.QueryRow(f.ctx, `SELECT i.name,t.status,(SELECT count(*) FROM work_item_effects) FROM issues i JOIN intake_issues t ON t.issue_id=i.id WHERE i.id=$1`, item).Scan(&name, &status, &after); err != nil {
				t.Fatal(err)
			}
			if name != "Before" || status != -2 || after != before || draftValues(t, f, "issues", item)[field] != float64(7) {
				t.Fatalf("rejected edit changed item/intake/effects: %q %d %d/%d", name, status, after, before)
			}
		})
	}
}

func TestTypedIntakePatchRequiresPreservationReview(t *testing.T) {
	for _, api := range []bool{false, true} {
		t.Run(fmt.Sprintf("api=%t", api), func(t *testing.T) {
			f, repo, item, field, intake, _ := seedIntakeWriter(t)
			f.mutate(t, "import", f.bug)
			err := patchTypedIntake(f, repo, item, intake, api, `{"issue":{"type_id":"`+f.bug+`"}}`)
			var invalid *djangoapi.HTTPError
			if !errors.As(err, &invalid) || invalid.Status != 409 {
				t.Fatalf("want preservation conflict, got %v", err)
			}
			var itemType string
			if err = f.db.QueryRow(f.ctx, `SELECT type_id::text FROM issues WHERE id=$1`, item).Scan(&itemType); err != nil {
				t.Fatal(err)
			}
			if itemType != f.task || draftValues(t, f, "issues", item)[field] != float64(7) {
				t.Fatal("rejected transition lost original type/value")
			}
		})
	}
}

func TestTypedIntakePatchRecordsHistoryAndSubscribers(t *testing.T) {
	for _, api := range []bool{false, true} {
		t.Run(fmt.Sprintf("api=%t", api), func(t *testing.T) {
			f, repo, item, risk, intake, _ := seedIntakeWriter(t)
			member, notes := uuid.New().String(), uuid.New().String()
			appendIntakeProperties(t, f, risk,
				property.Definition{ID: member, Title: "Reviewer", Kind: property.Member, Configuration: property.Configuration{Active: true}},
				property.Definition{ID: notes, Title: "Notes", Kind: property.RichText, Configuration: property.Configuration{Active: true}},
			)
			err := patchTypedIntake(f, repo, item, intake, api, `{"issue":{"custom_properties":{"`+member+`":"`+f.member+`","`+notes+`":"<p>Intake notes</p>"}}}`)
			if err != nil {
				t.Fatal(err)
			}
			var subscribers, versions, search int
			err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM issue_subscribers WHERE issue_id=$1 AND subscriber_id=$2 AND deleted_at IS NULL),(SELECT count(*) FROM work_item_property_versions WHERE issue_id=$1 AND property_id=$3 AND value='"<p>Intake notes</p>"'::jsonb AND actor_id=$4),(SELECT count(*) FROM work_item_property_search WHERE issue_id=$1 AND property_id=$3 AND search_text LIKE '%Intake notes%')`, item, f.member, notes, f.owner).Scan(&subscribers, &versions, &search)
			if err != nil {
				t.Fatal(err)
			}
			if subscribers != 1 || versions != 1 || search != 1 {
				t.Fatalf("missing subscriber/history/search: %d %d %d", subscribers, versions, search)
			}
		})
	}
}

func TestAPIIntakePatchAcceptsVisibleCrossProjectParent(t *testing.T) {
	f, repo, item, _, intake, _ := seedIntakeWriter(t)
	other := uuid.New().String()
	f.addProject(other)
	parentFixture := f
	parentFixture.project = other
	parent := addHierarchyItem(t, parentFixture, f.task, "")
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0)`, f.workspace, other, f.owner)
	if err := patchTypedIntake(f, repo, item, intake, true, `{"issue":{"parent":"`+parent+`"}}`); err != nil {
		t.Fatal(err)
	}
	var saved string
	if err := f.db.QueryRow(f.ctx, `SELECT parent_id::text FROM issues WHERE id=$1`, item).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if saved != parent {
		t.Fatalf("parent = %s", saved)
	}
}

func TestAPIIntakePatchRejectsUnknownParent(t *testing.T) {
	f, repo, item, _, intake, _ := seedIntakeWriter(t)
	err := patchTypedIntake(f, repo, item, intake, true, `{"issue":{"parent":"`+uuid.New().String()+`","name":"Rejected"}}`)
	var invalid *djangoapi.HTTPError
	if !errors.As(err, &invalid) || invalid.Status != 400 {
		t.Fatalf("want invalid parent, got %v", err)
	}
	var name string
	var parent *string
	if err = f.db.QueryRow(f.ctx, `SELECT name,parent_id::text FROM issues WHERE id=$1`, item).Scan(&name, &parent); err != nil {
		t.Fatal(err)
	}
	if name != "Before" || parent != nil {
		t.Fatal("invalid parent changed item")
	}
}

func appendIntakeProperties(t *testing.T, f projectTypeFixture, existing string, definitions ...property.Definition) {
	t.Helper()
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{existing}
	for _, definition := range definitions {
		catalog, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: definition})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, definition.ID)
	}
	if _, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: f.task, PropertyIDs: ids}); err != nil {
		t.Fatal(err)
	}
}

func TestTypedIntakeAcceptanceRetainsCustomEdit(t *testing.T) {
	for _, api := range []bool{false, true} {
		t.Run(fmt.Sprintf("api=%t", api), func(t *testing.T) {
			f, repo, item, field, intake, _ := seedIntakeWriter(t)
			triage, backlog := uuid.New().String(), uuid.New().String()
			f.exec(`INSERT INTO states(id,name,description,color,slug,sequence,"group",is_triage,"default",project_id,workspace_id,created_at,updated_at) VALUES($1,'Triage','','#000000','triage',0,'triage',true,false,$3,$4,now(),now()),($2,'Backlog','','#000000','backlog',1,'backlog',false,true,$3,$4,now(),now())`, triage, backlog, f.project, f.workspace)
			f.exec(`UPDATE issues SET state_id=$2 WHERE id=$1`, item, triage)
			if err := patchTypedIntake(f, repo, item, intake, api, `{"issue":{"custom_properties":{"`+field+`":12}},"status":1}`); err != nil {
				t.Fatal(err)
			}
			var state string
			if err := f.db.QueryRow(f.ctx, `SELECT state_id::text FROM issues WHERE id=$1`, item).Scan(&state); err != nil {
				t.Fatal(err)
			}
			if state != backlog || draftValues(t, f, "issues", item)[field] != float64(12) {
				t.Fatal("acceptance discarded value or state transition")
			}
		})
	}
}

func TestAPIIntakePatchRejectsHiddenParent(t *testing.T) {
	f, repo, item, _, intake, _ := seedIntakeWriter(t)
	other := uuid.New().String()
	f.addProject(other)
	parentFixture := f
	parentFixture.project = other
	parent := addHierarchyItem(t, parentFixture, f.task, "")
	err := patchTypedIntake(f, repo, item, intake, true, `{"issue":{"parent":"`+parent+`"}}`)
	var invalid *djangoapi.HTTPError
	if !errors.As(err, &invalid) || invalid.Status != 400 {
		t.Fatalf("want hidden parent rejection, got %v", err)
	}
}

func TestAPIIntakePatchRejectsForeignWorkspaceParent(t *testing.T) {
	f, repo, item, _, intake, _ := seedIntakeWriter(t)
	workspace, project, parent := uuid.New().String(), uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Other',$2,'other-types','UTC','',now(),now())`, workspace, f.owner)
	f.addProject(project)
	f.exec(`UPDATE projects SET workspace_id=$2 WHERE id=$1`, project, workspace)
	f.exec(`INSERT INTO workspace_members(id,workspace_id,member_id,role,created_at,updated_at,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features) VALUES(gen_random_uuid(),$1,$2,20,now(),now(),'{}','{}','{}',true,'{}','{}','{}')`, workspace, f.owner)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,20,true,now(),now(),'{}','{}','{}',0)`, workspace, project, f.owner)
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at,created_by_id) VALUES($1,$2,$3,'Foreign parent','{}','','none',0,1,false,now(),now(),$4)`, parent, workspace, project, f.owner)
	err := patchTypedIntake(f, repo, item, intake, true, `{"issue":{"parent":"`+parent+`"}}`)
	var invalid *djangoapi.HTTPError
	if !errors.As(err, &invalid) || invalid.Status != 400 {
		t.Fatalf("want cross-workspace parent rejection, got %v", err)
	}
}

func TestAPIIntakePatchEnforcesHierarchy(t *testing.T) {
	f, repo, item, _, intake, _ := seedIntakeWriter(t)
	parent := addHierarchyItem(t, f, f.task, "")
	configureHierarchy(t, f)
	err := patchTypedIntake(f, repo, item, intake, true, `{"issue":{"parent":"`+parent+`"}}`)
	requireHierarchyConstraint(t, err)
	var saved *string
	if err = f.db.QueryRow(f.ctx, `SELECT parent_id::text FROM issues WHERE id=$1`, item).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	if saved != nil {
		t.Fatal("invalid hierarchy parent was saved")
	}
}
