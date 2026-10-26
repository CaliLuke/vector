//go:build integration

package workitemtypes_test

import (
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/workitems"
)

func TestProjectlessDraftTypeSelectionAndProjectAvailability(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	if err := editValueDraft(f, repo, draft, `{"project_id":null,"type_id":"`+f.bug+`"}`); err != nil {
		t.Fatalf("select active workspace type without a project: %v", err)
	}
	var selected string
	var project *string
	if err := f.db.QueryRow(f.ctx, `SELECT type_id::text,project_id::text FROM draft_issues WHERE id=$1`, draft).Scan(&selected, &project); err != nil || selected != f.bug || project != nil {
		t.Fatalf("draft selection: type=%s project=%v error=%v", selected, project, err)
	}
	if err := editValueDraft(f, repo, draft, `{"project_id":"`+f.project+`"}`); err == nil {
		t.Fatal("assigned an unimported type to a project")
	}
	f.mutate(t, "import", f.bug)
	if err := editValueDraft(f, repo, draft, `{"project_id":"`+f.project+`"}`); err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE issue_types SET is_active=false WHERE id=$1`, f.bug)
	_, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_draft_issues", Method: "POST", ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{}}, strings.NewReader(`{"type_id":"`+f.bug+`"}`), "application/json")
	if err == nil {
		t.Fatal("new projectless draft accepted an inactive type")
	}
	// The database guard also protects writers that do not use the repository.
	if _, err = f.db.Exec(f.ctx, `UPDATE draft_issues SET project_id=NULL WHERE id=$1`, draft); err == nil {
		t.Fatal("database allowed changing destination with an inactive type")
	}
}

func TestProjectlessDraftCanBeCreatedWithAnOptionalWorkspaceType(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, err := workitems.NewRepository(f.db, noRequestEffects{t}, "http://example.test")
	if err != nil {
		t.Fatal(err)
	}
	result, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_draft_issues", Method: "POST", ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{}}, strings.NewReader(`{"name":"","type_id":"`+f.bug+`"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	id := resultIssueID(t, result)
	var selected string
	if err = f.db.QueryRow(f.ctx, `SELECT type_id::text FROM draft_issues WHERE id=$1 AND project_id IS NULL`, id).Scan(&selected); err != nil || selected != f.bug {
		t.Fatalf("created draft type=%s error=%v", selected, err)
	}
}
