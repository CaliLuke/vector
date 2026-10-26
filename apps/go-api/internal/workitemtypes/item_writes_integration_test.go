//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func writeValueItem(f projectTypeFixture, repo *workitems.Repository, method, id, body string) (workitems.Result, error) {
	operation := "api_workspaces_by_slug_projects_by_project_id_issues"
	if id != "" {
		operation += "_by_pk"
	}
	return repo.Execute(f.ctx, workitems.Request{Operation: operation, Method: method, ActorID: f.member, Slug: "types", ProjectID: f.project, IssueID: id, Timezone: "UTC", Params: map[string]string{"pk": id}}, strings.NewReader(body), "application/json")
}

func TestOrdinaryItemValuesPersistAndRejectAtomically(t *testing.T) {
	f := newProjectTypeFixture(t)
	number, text, member := uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(5)}},
		property.Definition{ID: text, Title: "Acceptance", Kind: property.Text, Configuration: property.Configuration{Active: true, Required: true, Format: "Single line"}},
		property.Definition{ID: member, Title: "Reviewer", Kind: property.Member, Configuration: property.Configuration{Active: true}},
	)
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	result, err := writeValueItem(f, repo, "POST", "", fmt.Sprintf(`{"name":"Created","type_id":"%s","custom_properties":{"%s":"Ready","%s":"%s"}}`, f.task, text, member, f.member))
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, result)
	response := result.Body.(map[string]any)
	if response["type_id"] != f.task || response["custom_properties"] == nil {
		t.Fatalf("authenticated creation omits type/values: %#v", response)
	}
	values := draftValues(t, f, "issues", id)
	if values[number] != float64(5) || values[text] != "Ready" {
		t.Fatalf("creation values: %#v", values)
	}
	_, err = writeValueItem(f, repo, "PATCH", id, fmt.Sprintf(`{"name":"Updated","custom_properties":{"%s":0}}`, number))
	if err != nil {
		t.Fatal(err)
	}
	values = draftValues(t, f, "issues", id)
	if values[number] != float64(0) || values[text] != "Ready" || values[member] != f.member {
		t.Fatalf("partial edit lost values: %#v", values)
	}
	var effects, subscribers int
	if err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM work_item_effects),(SELECT count(*) FROM issue_subscribers WHERE issue_id=$1)`, id).Scan(&effects, &subscribers); err != nil || effects != 6 || subscribers != 1 {
		t.Fatalf("durable effects/subscribers: %d %d %v", effects, subscribers, err)
	}
	_, err = writeValueItem(f, repo, "PATCH", id, fmt.Sprintf(`{"name":"Must not persist","custom_properties":{"%s":null}}`, text))
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("required clear accepted: %v", err)
	}
	var name string
	var afterEffects int
	if err = f.db.QueryRow(f.ctx, `SELECT name,(SELECT count(*) FROM work_item_effects) FROM issues WHERE id=$1`, id).Scan(&name, &afterEffects); err != nil || name != "Updated" || afterEffects != effects {
		t.Fatalf("failed edit persisted: %q %d %v", name, afterEffects, err)
	}
	// An unrelated edit must not recreate a subscription the user removed.
	f.exec(`UPDATE issue_subscribers SET deleted_at=now() WHERE issue_id=$1`, id)
	if _, err = writeValueItem(f, repo, "PATCH", id, `{"priority":"high"}`); err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL`, id).Scan(&subscribers); err != nil || subscribers != 0 {
		t.Fatalf("unrelated save resubscribed user: %d %v", subscribers, err)
	}
}

func TestOrdinaryItemEffectFailureRollsBackValuesAndSubscriptions(t *testing.T) {
	f := newProjectTypeFixture(t)
	member := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: member, Title: "Reviewer", Kind: property.Member, Configuration: property.Configuration{Active: true}})
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	f.exec(`CREATE FUNCTION reject_webhook_effect() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.kind='webhook' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_webhook_effect BEFORE INSERT ON work_item_effects FOR EACH ROW EXECUTE FUNCTION reject_webhook_effect()`)
	body, _ := json.Marshal(map[string]any{"name": "Must rollback", "custom_properties": map[string]any{member: f.member}})
	if _, err = writeValueItem(f, repo, "POST", "", string(body)); err == nil {
		t.Fatal("injected late failure did not fail")
	}
	var items, sequences, subscribers, effects int
	if err = f.db.QueryRow(f.ctx, `SELECT (SELECT count(*) FROM issues),(SELECT count(*) FROM issue_sequences),(SELECT count(*) FROM issue_subscribers),(SELECT count(*) FROM work_item_effects)`).Scan(&items, &sequences, &subscribers, &effects); err != nil || items+sequences+subscribers+effects != 0 {
		t.Fatalf("failed request partially committed: %d %d %d %d %v", items, sequences, subscribers, effects, err)
	}
}

func TestMemberEligibilityUsesResultingItemCreator(t *testing.T) {
	f := newProjectTypeFixture(t)
	member := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: member, Title: "Reviewer", Kind: property.Member, Configuration: property.Configuration{Active: true}})
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Guest-created item"}`)
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, created)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	request := workitems.Request{Operation: "api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk", Method: "PATCH", API: true, ActorID: f.owner, Slug: "types", ProjectID: f.project, IssueID: id, Timezone: "UTC", Params: map[string]string{"pk": id}}
	// The guest loses creator visibility in this same write, so selecting them
	// must fail despite their access to the old row.
	body := fmt.Sprintf(`{"created_by":"%s","custom_properties":{"%s":"%s"}}`, f.owner, member, f.member)
	_, err = repo.Execute(f.ctx, request, strings.NewReader(body), "application/json")
	var validation *djangoapi.HTTPError
	if !errors.As(err, &validation) || validation.Status != 400 {
		t.Fatalf("former creator subscribed without resulting access: %v", err)
	}
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, id, f.owner)
	body = fmt.Sprintf(`{"created_by":"%s","custom_properties":{"%s":"%s"}}`, f.member, member, f.member)
	updated, err := repo.Execute(f.ctx, request, strings.NewReader(body), "application/json")
	if err != nil {
		t.Fatalf("new creator rejected: %v", err)
	}
	projection, ok := updated.Body.(map[string]any)
	if !ok || projection["custom_properties"] == nil {
		t.Fatalf("API update omits values: %#v", updated.Body)
	}
	request.Method = "GET"
	request.Params["fields"] = "id,custom_properties"
	read, err := repo.Execute(f.ctx, request, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	projection, ok = read.Body.(map[string]any)
	if !ok || projection["custom_properties"] == nil || len(projection) != 2 {
		t.Fatalf("API read field selection: %#v", read.Body)
	}
	var creator string
	var subscribers int
	if err = f.db.QueryRow(f.ctx, `SELECT created_by_id::text,(SELECT count(*) FROM issue_subscribers WHERE issue_id=i.id AND subscriber_id=$2 AND deleted_at IS NULL) FROM issues i WHERE id=$1`, id, f.member).Scan(&creator, &subscribers); err != nil || creator != f.member || subscribers != 1 {
		t.Fatalf("creator/subscriber write: %s %d %v", creator, subscribers, err)
	}
}

func TestAPIValuesRespectGuestVisibilityAcrossReadPaths(t *testing.T) {
	f := newProjectTypeFixture(t)
	propertyID := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: propertyID, Title: "Private effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	created, err := writeValueItem(f, repo, "POST", "", fmt.Sprintf(`{"name":"Hidden","external_id":"external","external_source":"test","custom_properties":{"%s":7}}`, propertyID))
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, created)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, id, f.owner)
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1 AND project_id=$2`, f.member, f.project)
	request := workitems.Request{Operation: "api_v1_workspaces_by_slug_projects_by_project_id_issues_by_pk", Method: "GET", API: true, ActorID: f.member, Slug: "types", ProjectID: f.project, IssueID: id, Timezone: "UTC", Params: map[string]string{}}
	_, err = repo.Execute(f.ctx, request, nil, "")
	var denied *djangoapi.HTTPError
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("guest read another creator's item: %v", err)
	}
	request.Params = map[string]string{"identifier": f.project[:12] + "-1"}
	_, err = repo.Execute(f.ctx, request, nil, "")
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("identifier bypassed guest visibility: %v", err)
	}
	request.Operation, request.IssueID = "api_v1_workspaces_by_slug_projects_by_project_id_issues", ""
	request.Params = map[string]string{"external_id": "external", "external_source": "test"}
	_, err = repo.Execute(f.ctx, request, nil, "")
	if !errors.As(err, &denied) || denied.Status != 403 {
		t.Fatalf("external lookup bypassed guest visibility: %v", err)
	}
	request.Params = map[string]string{}
	listed, err := repo.Execute(f.ctx, request, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(listed.Body)
	if strings.Contains(string(raw), id) || strings.Contains(string(raw), propertyID) {
		t.Fatalf("list exposed restricted item: %s", raw)
	}
	f.exec(`UPDATE projects SET guest_view_all_features=true WHERE id=$1`, f.project)
	listed, err = repo.Execute(f.ctx, request, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(listed.Body)
	if !strings.Contains(string(raw), propertyID) {
		t.Fatalf("authorized guest list omitted values: %s", raw)
	}
}

func TestSubissueValuesRespectAnchorAndChildVisibility(t *testing.T) {
	f := newProjectTypeFixture(t)
	propertyID := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: propertyID, Title: "Private effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	parentResult, err := writeValueItem(f, repo, "POST", "", `{"name":"Visible parent"}`)
	if err != nil {
		t.Fatal(err)
	}
	parent := resultIssueID(t, parentResult)
	childResult, err := writeValueItem(f, repo, "POST", "", fmt.Sprintf(`{"name":"Hidden child","parent_id":"%s","custom_properties":{"%s":7}}`, parent, propertyID))
	if err != nil {
		t.Fatal(err)
	}
	child := resultIssueID(t, childResult)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, child, f.owner)
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1 AND project_id=$2`, f.member, f.project)
	request := workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_issues_by_issue_id_sub_issues", Method: "GET", ActorID: f.member, Slug: "types", ProjectID: f.project, IssueID: parent, Params: map[string]string{}}
	assertHidden := func() {
		t.Helper()
		result, readErr := repo.Execute(f.ctx, request, nil, "")
		if readErr != nil {
			t.Fatal(readErr)
		}
		raw, _ := json.Marshal(result.Body)
		if strings.Contains(string(raw), child) || strings.Contains(string(raw), propertyID) {
			t.Fatalf("child exposed in result/distribution: %s", raw)
		}
	}
	assertHidden()
	otherProject := uuid.New().String()
	f.addProject(otherProject)
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, child, otherProject)
	f.exec(`UPDATE project_members SET role=15 WHERE member_id=$1 AND project_id=$2`, f.member, f.project)
	assertHidden() // Normal membership in the parent project grants no child access.
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, parent, otherProject)
	_, err = repo.Execute(f.ctx, request, nil, "")
	var missing *djangoapi.HTTPError
	if !errors.As(err, &missing) || missing.Status != 404 {
		t.Fatalf("mismatched URL project accepted parent: %v", err)
	}
}
