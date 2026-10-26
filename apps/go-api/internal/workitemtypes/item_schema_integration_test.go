//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"strings"
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestItemSchemaReturnsOnlyEffectiveContext(t *testing.T) {
	f := newProjectTypeFixture(t)
	other := uuid.New().String()
	f.addProject(other)
	def := property.Definition{ID: uuid.New().String(), Title: "Instructions", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(101)}, Contexts: []property.Context{
		{ID: uuid.New().String(), Name: "Current project", Projects: []string{f.project}, Overrides: []string{"default_value"}, Configuration: property.Configuration{Default: float64(202)}},
		{ID: uuid.New().String(), Name: "Private context", Projects: []string{other}, Overrides: []string{"default_value"}, Configuration: property.Configuration{Default: float64(303)}},
	}}
	attachValueProperties(t, f, def)
	upperType := strings.ToUpper(f.task)
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{ProjectID: strings.ToUpper(f.project), TypeID: &upperType, Mode: "create"})
	if err != nil {
		t.Fatal(err)
	}
	if !schema.Enabled || schema.TypeID != f.task || len(schema.Properties) != 1 || schema.Properties[0].Configuration.Default != float64(202) || schema.Properties[0].ContextID != def.Contexts[0].ID {
		t.Fatalf("effective schema: %+v", schema)
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	for _, hidden := range []string{`"default_value":303`, "Private context", other, def.Contexts[1].ID, "contexts"} {
		if strings.Contains(string(raw), hidden) {
			t.Fatalf("foreign context leaked: %s", raw)
		}
	}
}

func TestItemSchemaDraftAndPublicationModes(t *testing.T) {
	f := newProjectTypeFixture(t)
	cascade := property.Definition{ID: uuid.New().String(), Title: "Area", Kind: property.Cascading, Configuration: property.Configuration{Active: true, Levels: []property.Level{{ID: uuid.New().String(), Name: "Area"}, {ID: uuid.New().String(), Name: "Component"}}, Options: []property.Option{{ID: uuid.New().String(), Name: "Platform"}}}}
	attachValueProperties(t, f, cascade)
	_, draft, _ := seedPublicationDraft(t, f)
	input := types.ItemSchemaInput{ProjectID: f.project, DraftID: draft, Mode: "draft"}
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || len(schema.Properties) != 0 {
		t.Fatalf("draft exposes cascade: %+v %v", schema, err)
	}
	input.Mode = "publish"
	schema, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || len(schema.Properties) != 1 || schema.Properties[0].ID != cascade.ID {
		t.Fatalf("publication omitted cascade: %+v %v", schema, err)
	}
	f.mutate(t, "import", f.bug)
	f.exec(`UPDATE draft_issues SET type_id=$2 WHERE id=$1`, draft, f.bug)
	f.mutate(t, "pause", f.bug)
	schema, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || schema.TypeID != "" || schema.SelectionReason == "" || len(schema.EligibleTypeIDs) != 1 || schema.EligibleTypeIDs[0] != f.task {
		t.Fatalf("paused publication must offer a usable replacement: %+v %v", schema, err)
	}
	input.Mode = "draft"
	schema, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || schema.TypeID != f.bug {
		t.Fatalf("existing paused draft unavailable: %+v %v", schema, err)
	}
}

func TestItemSchemaProjectlessDraftRetainsOrClearsType(t *testing.T) {
	f := newProjectTypeFixture(t)
	_, draft, _ := seedPublicationDraft(t, f)
	f.exec(`UPDATE draft_issues SET project_id=NULL WHERE id=$1`, draft)
	input := types.ItemSchemaInput{DraftID: draft, Mode: "draft"}
	retained, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || retained.TypeID != f.task {
		t.Fatalf("omitted type lost: %+v %v", retained, err)
	}
	empty := ""
	input.TypeID = &empty
	cleared, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || cleared.TypeID != "" || len(cleared.Properties) != 0 {
		t.Fatalf("explicit clear ignored: %+v %v", cleared, err)
	}
	input.TypeID = nil
	_, err = f.repo.ItemSchema(f.ctx, f.owner, "types", input)
	requireTypeStatus(t, err, 404)
}

func TestItemSchemaMemberEligibilityAndUnavailableLabels(t *testing.T) {
	f := newProjectTypeFixture(t)
	member := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: member, Title: "Reviewers", Kind: property.Member, Configuration: property.Configuration{Active: true, Multiple: true}})
	_, draft, _ := seedPublicationDraft(t, f)
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1 AND project_id=$2`, f.owner, f.project)
	f.exec(`UPDATE draft_issues SET custom_properties=jsonb_build_object($2::text,jsonb_build_array($3::text)) WHERE id=$1`, draft, member, f.outsider)
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{ProjectID: f.project, DraftID: draft, Mode: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]types.ItemReferenceChoice{}
	for _, choice := range schema.Members {
		byID[choice.ID] = choice
	}
	if !byID[f.member].Eligible || byID[f.owner].Eligible || byID[f.owner].Reason == "" {
		t.Fatalf("access restrictions missing: %+v", schema.Members)
	}
	if byID[f.outsider].Eligible || byID[f.outsider].Name != "Unavailable member" || byID[f.outsider].Reason == "" {
		t.Fatalf("unavailable identity disclosed: %+v", schema.Members)
	}
}

func TestItemSchemaEnforcesGuestItemAndProjectScope(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	issue := resultIssueID(t, result)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, issue, f.owner)
	f.exec(`UPDATE project_members SET role=5 WHERE member_id=$1 AND project_id=$2`, f.member, f.project)
	input := types.ItemSchemaInput{ProjectID: f.project, IssueID: issue, Mode: "edit"}
	_, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 404)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, issue, f.member)
	if _, err = f.repo.ItemSchema(f.ctx, f.member, "types", input); err != nil {
		t.Fatal(err)
	}
	input.ProjectID = uuid.New().String()
	_, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 403)
}

func TestItemSchemaOwnedDraftAfterProjectAccessRevoked(t *testing.T) {
	f := newProjectTypeFixture(t)
	release, selected, hidden := uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: release, Title: "Delivery", Kind: property.Release, Configuration: property.Configuration{Active: true}})
	_, draft, _ := seedPublicationDraft(t, f)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Private selected'),($4,$2,$3,'Private new release')`, selected, f.workspace, f.project, hidden)
	f.exec(`UPDATE draft_issues SET custom_properties=jsonb_build_object($2::text,$3::text) WHERE id=$1`, draft, release, selected)
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	input := types.ItemSchemaInput{ProjectID: f.project, DraftID: draft, Mode: "draft"}
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || len(schema.Properties) != 1 || len(schema.Releases) != 1 || schema.Releases[0].ID != selected || schema.Releases[0].Name != "Unavailable release" {
		t.Fatalf("owned draft schema or catalog restriction: %+v %v", schema, err)
	}
	input.Mode = "publish"
	_, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 403)
	input.Mode = "draft"
	input.ProjectID = uuid.New().String()
	_, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 403)
}

func TestItemSchemaInaccessibleProjectDoesNotDiscloseItemExistence(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	f.exec(`UPDATE projects SET guest_view_all_features=true WHERE id=$1`, f.project)
	for _, id := range []string{resultIssueID(t, result), uuid.New().String()} {
		_, err = f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{ProjectID: f.project, IssueID: id, Mode: "edit"})
		requireTypeStatus(t, err, 403)
	}
}

func TestItemSchemaFormulaResultsResolveEstimateCatalogAndDefaults(t *testing.T) {
	f := newProjectTypeFixture(t)
	effort, double, estimate := uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(3)}},
		property.Definition{ID: double, Title: "Double effort", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{effort}} * 2", Result: "Number"}},
		property.Definition{ID: estimate, Title: "Estimated twice", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{estimate_point}} * 2", Result: "Number"}},
	)
	created, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{Mode: "create", ProjectID: f.project})
	if err != nil || created.Computed[double].Value != float64(6) || created.Computed[estimate].Status != "empty" || created.EvaluatedAt.IsZero() {
		t.Fatalf("creation preview defaults: %+v %v", created, err)
	}
	repo, draft, _ := seedPublicationDraft(t, f)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	issue := resultIssueID(t, result)
	estimateID, point := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO estimates(id,workspace_id,project_id,name,description,type,last_used,created_at,updated_at) VALUES($1,$2,$3,'Points','','points',true,now(),now())`, estimateID, f.workspace, f.project)
	f.exec(`INSERT INTO estimate_points(id,workspace_id,project_id,estimate_id,key,value,description,created_at,updated_at) VALUES($1,$2,$3,$4,1,'2.5','',now(),now())`, point, f.workspace, f.project, estimateID)
	f.exec(`UPDATE issues SET estimate_point_id=$2 WHERE id=$1`, issue, point)
	input := types.ItemSchemaInput{Mode: "edit", ProjectID: f.project, IssueID: issue}
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || schema.Computed[estimate].Value != float64(5) || schema.Computed[double].Value != float64(6) {
		t.Fatalf("persisted numeric estimate: %+v %v", schema.Computed, err)
	}
	f.exec(`UPDATE estimate_points SET value='XL' WHERE id=$1`, point)
	f.exec(`UPDATE issues SET custom_properties='{}' WHERE id=$1`, issue)
	schema, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || schema.Computed[estimate].Status != "error" || schema.Computed[estimate].Error == "" || schema.Computed[double].Status != "empty" {
		t.Fatalf("invalid estimate or existing missing value: %+v %v", schema.Computed, err)
	}
}

func TestItemSchemaReleaseChoicesRetainUnavailableSelectionsWithoutNames(t *testing.T) {
	f := newProjectTypeFixture(t)
	field, second := uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: field, Title: "Delivery", Kind: property.Release, Configuration: property.Configuration{Active: true, Multiple: true}},
		property.Definition{ID: second, Title: "Previous delivery", Kind: property.Release, Configuration: property.Configuration{Active: true}},
	)
	_, draft, _ := seedPublicationDraft(t, f)
	current, otherProject, foreign := uuid.New().String(), uuid.New().String(), uuid.New().String()
	f.addProject(otherProject)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Available'),($4,$2,$5,'Private foreign release')`, current, f.workspace, f.project, foreign, otherProject)
	f.exec(`UPDATE draft_issues SET custom_properties=jsonb_build_object($2::text,jsonb_build_array($3::text),$4::text,$3::text) WHERE id=$1`, draft, field, foreign, second)
	schema, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{Mode: "draft", ProjectID: f.project, DraftID: draft})
	if err != nil || len(schema.Releases) != 2 {
		t.Fatalf("release choices: %+v %v", schema.Releases, err)
	}
	if !schema.Releases[0].Eligible || schema.Releases[0].ID != current || schema.Releases[1].Eligible || schema.Releases[1].ID != foreign || schema.Releases[1].Name != "Unavailable release" || schema.Releases[1].Reason == "" {
		t.Fatalf("reference access lost: %+v", schema.Releases)
	}
}
