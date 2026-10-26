//go:build integration

package workitemtypes_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func configureCreationHierarchy(t *testing.T, f projectTypeFixture, level, defaultLevel int) {
	t.Helper()
	input := hierarchyInput(t, f, level)
	input.DefaultLevel = defaultLevel
	review, err := f.repo.PreviewHierarchy(f.ctx, f.owner, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	input.ReviewID = review.ID
	if review.Links != 0 {
		t.Fatalf("fixture unexpectedly invalid: %+v", review)
	}
	if _, err = f.repo.CommitHierarchy(f.ctx, f.owner, "types", input); err != nil {
		t.Fatal(err)
	}
}
func creationSchema(t *testing.T, f projectTypeFixture, parent string) types.ItemSchema {
	t.Helper()
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{ProjectID: f.project, Mode: "create", ParentID: &parent})
	if err != nil {
		t.Fatal(err)
	}
	return schema
}
func TestCreationDefaultsRespectSuggestedLevelButPreserveExplicitStandaloneChoice(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	configureCreationHierarchy(t, f, 1, 1)
	schema := creationSchema(t, f, "")
	if schema.TypeID != f.bug || !slices.Contains(schema.EligibleTypeIDs, f.task) {
		t.Fatalf("suggested level changed explicit choices: %+v", schema)
	}
	explicit, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{ProjectID: f.project, Mode: "create", TypeID: &f.task})
	if err != nil || explicit.TypeID != f.task {
		t.Fatalf("explicit standalone type lost: %+v %v", explicit, err)
	}
	repo, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Suggested type"}`)
	if err != nil {
		t.Fatal(err)
	}
	var selected string
	if err = f.db.QueryRow(f.ctx, `SELECT type_id::text FROM issues WHERE id=$1`, resultIssueID(t, created)).Scan(&selected); err != nil || selected != f.bug {
		t.Fatalf("ordinary default=%s err=%v", selected, err)
	}
	apiResult, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_v1_workspaces_by_slug_projects_by_project_id_issues", Method: "POST", ActorID: f.owner, Slug: "types", ProjectID: f.project, API: true, Timezone: "UTC", Params: map[string]string{}}, strings.NewReader(`{"name":"API suggestion"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT type_id::text FROM issues WHERE id=$1`, resultIssueID(t, apiResult)).Scan(&selected); err != nil || selected != f.bug {
		t.Fatalf("API default=%s err=%v", selected, err)
	}
}
func TestCreationDefaultsParentAdjacencyAndLeafEmptySchema(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	parent := addHierarchyItem(t, f, f.bug, "")
	leaf := addHierarchyItem(t, f, f.task, "")
	configureCreationHierarchy(t, f, 1, 1)
	f.mutate(t, "set_default", f.bug)
	child := creationSchema(t, f, parent)
	if child.TypeID != f.task || len(child.EligibleTypeIDs) != 1 || child.EligibleTypeIDs[0] != f.task {
		t.Fatalf("parent suggestion: %+v", child)
	}
	empty := creationSchema(t, f, leaf)
	if empty.TypeID != "" || len(empty.EligibleTypeIDs) != 0 || empty.SelectionReason == "" {
		t.Fatalf("leaf must have useful empty schema: %+v", empty)
	}
	repo, _, _ := seedPublicationDraft(t, f)
	_, err := writeValueItem(f, repo, "POST", "", `{"name":"Impossible child","parent_id":"`+leaf+`"}`)
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("leaf child saved: %v", err)
	}
}
func TestCreationProjectPreferenceSurvivesPauseRemovalAndResume(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	selected := f.mutate(t, "set_default", f.bug)
	if selected.DefaultTypeID != f.bug || creationSchema(t, f, "").TypeID != f.bug {
		t.Fatal("project preference ignored")
	}
	paused := f.mutate(t, "pause", f.bug)
	if paused.DefaultTypeID != f.bug || creationSchema(t, f, "").TypeID != f.task {
		t.Fatal("pause erased preference or selected unavailable type")
	}
	f.mutate(t, "resume", f.bug)
	if creationSchema(t, f, "").TypeID != f.bug {
		t.Fatal("resume did not restore preference")
	}
	removed := f.mutate(t, "remove", f.bug)
	if removed.DefaultTypeID != f.bug || creationSchema(t, f, "").TypeID != f.task {
		t.Fatal("removing import erased preference or selected unavailable type")
	}
	f.mutate(t, "import", f.bug)
	if creationSchema(t, f, "").TypeID != f.bug {
		t.Fatal("reimport did not restore preference")
	}
	cleared := f.mutate(t, "set_default")
	if cleared.DefaultTypeID != "" || creationSchema(t, f, "").TypeID != f.task {
		t.Fatal("automatic did not clear preference")
	}
}
func TestCreationDefaultsAmbiguityRequiresSelection(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	configureCreationHierarchy(t, f, 1, 1)
	if creationSchema(t, f, "").TypeID != f.bug {
		t.Fatal("sole candidate not suggested")
	}
	// Introduce a second available type at the default level without any preference.
	id := uuid.New().String()
	f.exec(`INSERT INTO issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,mandatory,level,is_epic,created_at,updated_at) VALUES($1,$2,'Feature','','{}',true,false,true,1,false,now(),now())`, id, f.workspace)
	schema := creationSchema(t, f, "")
	if schema.TypeID != "" || len(schema.Properties) != 0 || schema.SelectionReason == "" {
		t.Fatalf("ambiguous schema chose arbitrarily: %+v", schema)
	}
	repo, _, _ := seedPublicationDraft(t, f)
	_, err := writeValueItem(f, repo, "POST", "", `{"name":"Ambiguous"}`)
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("ambiguous creation saved: %v", err)
	}
}
func TestCreationSavedUntypedDraftNeverAcquiresSuggestedType(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	f.exec(`UPDATE draft_issues SET type_id=NULL WHERE id=$1`, draft)
	input := types.ItemSchemaInput{ProjectID: f.project, DraftID: draft, Mode: "draft"}
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || schema.TypeID != "" || len(schema.Properties) != 0 {
		t.Fatalf("saved untyped draft defaulted: %+v %v", schema, err)
	}
	input.Mode = "publish"
	schema, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || schema.TypeID != "" || schema.SelectionReason == "" {
		t.Fatalf("publication schema defaulted: %+v %v", schema, err)
	}
	_, err = publishDraftForTest(f, repo, draft, f.member, `{}`)
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("unselected saved draft published: %v", err)
	}
	var unchanged bool
	if err = f.db.QueryRow(f.ctx, `SELECT type_id IS NULL AND deleted_at IS NULL FROM draft_issues WHERE id=$1`, draft).Scan(&unchanged); err != nil || !unchanged {
		t.Fatalf("failed publication changed draft: %v %v", unchanged, err)
	}
	if _, err = publishDraftForTest(f, repo, draft, f.member, `{"type_id":"`+f.task+`"}`); err != nil {
		t.Fatal(err)
	}
}
func TestCreationUntypedDraftParentSurvivesHierarchyReview(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	repo, draft, _ := seedPublicationDraft(t, f)
	parent := addHierarchyItem(t, f, f.bug, "")
	f.exec(`UPDATE draft_issues SET type_id=NULL,parent_id=$2 WHERE id=$1`, draft, parent)
	configureCreationHierarchy(t, f, 2, 0)
	var savedParent string
	if err := f.db.QueryRow(f.ctx, `SELECT parent_id::text FROM draft_issues WHERE id=$1`, draft).Scan(&savedParent); err != nil || savedParent != parent {
		t.Fatalf("untyped link lost: %s %v", savedParent, err)
	}
	// A parent type change keeps unfinished private intent; choosing a type must validate it.
	f.exec(`UPDATE issues SET type_id=$2 WHERE id=$1`, parent, f.task)
	if _, err := f.db.Exec(f.ctx, `UPDATE draft_issues SET type_id=$2 WHERE id=$1`, draft, f.task); err == nil {
		t.Fatal("incompatible typed draft accepted")
	}
	if _, err := publishDraftForTest(f, repo, draft, f.member, `{}`); err == nil {
		t.Fatal("untyped draft under leaf published")
	}
}

func TestCreationDefaultsIntakeAndPublicFeedbackUseProjectSuggestion(t *testing.T) {
	f, repo, _, _, intake, _ := seedIntakeWriter(t)
	f.mutate(t, "import", f.bug)
	f.mutate(t, "set_default", f.bug)
	_, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_intake_issues", Method: "POST", ActorID: f.owner, Slug: "types", ProjectID: f.project, Timezone: "UTC", Params: map[string]string{"intake_id": intake}}, strings.NewReader(`{"issue":{"name":"Typed private intake"}}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	scope := workitems.PublicFeedbackScope{WorkspaceID: f.workspace, WorkspaceSlug: "types", ProjectID: f.project, IntakeID: &intake}
	if _, err = repo.PublicIntakeCreate(f.ctx, scope, intake, f.outsider, strings.NewReader(`{"issue":{"name":"Typed public feedback"}}`), "application/json"); err != nil {
		t.Fatal(err)
	}
	var typed int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM issues WHERE name IN ('Typed private intake','Typed public feedback') AND type_id=$1`, f.bug).Scan(&typed); err != nil || typed != 2 {
		t.Fatalf("intake defaults: %d %v", typed, err)
	}
}
func TestCreationSchemaRetainsMaskedParentUnlessExplicitlyCleared(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	parent := addHierarchyItem(t, f, f.bug, "")
	child := addHierarchyItem(t, f, f.task, parent)
	configureCreationHierarchy(t, f, 1, 0)
	hidden := uuid.New().String()
	f.addProject(hidden)
	f.exec(`INSERT INTO project_issue_types(id,workspace_id,project_id,issue_type_id,level,is_default,is_active,created_at,updated_at) VALUES(gen_random_uuid(),$1,$2,$3,1,false,true,now(),now())`, f.workspace, hidden, f.bug)
	f.exec(`UPDATE issues SET project_id=$2 WHERE id=$1`, parent, hidden)
	input := types.ItemSchemaInput{ProjectID: f.project, IssueID: child, Mode: "edit"}
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || len(schema.EligibleTypeIDs) != 1 || schema.EligibleTypeIDs[0] != f.task {
		t.Fatalf("masked parent scope lost: %+v %v", schema, err)
	}
	input.ParentID = &parent
	_, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 422)
	cleared := ""
	input.ParentID = &cleared
	schema, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || !slices.Contains(schema.EligibleTypeIDs, f.bug) {
		t.Fatalf("explicit parent clear ignored: %+v %v", schema, err)
	}
}
func TestCreationSchemaInvalidSelectedTypeStillOffersReplacements(t *testing.T) {
	f := newProjectTypeFixture(t)
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{ProjectID: f.project, Mode: "create", TypeID: &f.bug})
	if err != nil || schema.TypeID != "" || len(schema.Properties) != 0 || schema.SelectionReason == "" || !slices.Contains(schema.EligibleTypeIDs, f.task) {
		t.Fatalf("invalid selection trapped editor: %+v %v", schema, err)
	}
}

func TestCreationExplicitEmptyTypeDoesNotRequestSuggestion(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, _, _ := seedPublicationDraft(t, f)
	for _, api := range []bool{false, true} {
		for _, field := range []string{"type_id", "type"} {
			operation := "api_workspaces_by_slug_projects_by_project_id_issues"
			if api {
				operation = "api_v1_workspaces_by_slug_projects_by_project_id_issues"
			}
			_, err := repo.Execute(f.ctx, workitems.Request{Operation: operation, Method: "POST", API: api, ActorID: f.owner, Slug: "types", ProjectID: f.project, Timezone: "UTC", Params: map[string]string{}}, strings.NewReader(`{"name":"Explicitly untyped","`+field+`":null}`), "application/json")
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || failure.Status != 400 {
				t.Fatalf("api=%v field=%s: cleared type replaced: %v", api, field, err)
			}
		}
	}
	result, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_draft_issues", Method: "POST", ActorID: f.member, Slug: "types", Timezone: "UTC", Params: map[string]string{}}, strings.NewReader(`{"name":"Deliberately untyped","project_id":"`+f.project+`","type_id":null}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var untyped bool
	if err = f.db.QueryRow(f.ctx, `SELECT type_id IS NULL FROM draft_issues WHERE id=$1`, resultIssueID(t, result)).Scan(&untyped); err != nil || !untyped {
		t.Fatalf("new draft explicit clear defaulted: %v %v", untyped, err)
	}
}
func TestCreationAPIIntakeHonorsExplicitType(t *testing.T) {
	f, repo, _, _, intake, _ := seedIntakeWriter(t)
	f.mutate(t, "import", f.bug)
	_, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_v1_workspaces_by_slug_projects_by_project_id_intake_issues", Method: "POST", API: true, ActorID: f.owner, Slug: "types", ProjectID: f.project, Timezone: "UTC", Params: map[string]string{"intake_id": intake}}, strings.NewReader(`{"issue":{"name":"Explicit API intake","type":"`+f.bug+`"}}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	var selected string
	if err = f.db.QueryRow(f.ctx, `SELECT type_id::text FROM issues WHERE name='Explicit API intake'`).Scan(&selected); err != nil || selected != f.bug {
		t.Fatalf("explicit API intake type=%s err=%v", selected, err)
	}
}
func TestCreationPreferenceClearedByTypeDeletion(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	before := f.mutate(t, "set_default", f.bug)
	_, err := f.repo.Mutate(f.ctx, f.owner, "types", "delete", f.bug, types.Input{Revision: before.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if f.catalog(t).DefaultTypeID != "" {
		t.Fatal("soft-deleted type preference retained")
	}
}
