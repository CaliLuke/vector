//go:build integration

package workitemtypes_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/formula"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func numericAuthoringFormula(expression string) property.Definition {
	return property.Definition{ID: uuid.New().String(), Title: "Calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: expression, Result: formula.Number}}
}

func TestFormulaAuthoringEvaluatesUnsavedDefinitionWithoutChangingItsSample(t *testing.T) {
	f := newProjectTypeFixture(t)
	effort := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(7)}})
	repo, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Sample","description_html":"<p>Keep</p>","custom_properties":{"`+effort+`":2}}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, created)
	search, err := repo.Execute(f.ctx, workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_search_issues", Method: "GET", ActorID: f.owner, Slug: "types", ProjectID: f.project, Params: map[string]string{"search": "Sample", "workspace_search": "false"}}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(search.Body)
	if err != nil {
		t.Fatal(err)
	}
	var samples []struct {
		ID     string `json:"id"`
		TypeID string `json:"type_id"`
	}
	if err = json.Unmarshal(raw, &samples); err != nil || len(samples) != 1 || samples[0].ID != item || samples[0].TypeID != f.task {
		t.Fatalf("search omits sample type: %s %v", raw, err)
	}

	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.FormulaAuthoringInput{Revision: catalog.Revision, Definition: numericAuthoringFormula("{{effort}} * 2"), TypeID: f.task, ProjectID: f.project, IssueID: item}
	var original, after string
	if err = f.db.QueryRow(f.ctx, `SELECT row_to_json(i)::text FROM issues i WHERE id=$1`, item).Scan(&original); err != nil {
		t.Fatal(err)
	}
	preview, err := f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	if err != nil || !preview.Valid || preview.Result == nil || preview.Result.Value != float64(4) || preview.EvaluatedAt.IsZero() {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	if err = f.db.QueryRow(f.ctx, `SELECT row_to_json(i)::text FROM issues i WHERE id=$1`, item).Scan(&after); err != nil || original != after {
		t.Fatalf("preview changed sample: %v", err)
	}
	post, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || !reflect.DeepEqual(post, catalog) {
		t.Fatalf("preview changed configuration: %+v %v", post, err)
	}
	command.IssueID = ""
	preview, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	if err != nil || preview.Result == nil || preview.Result.Value != float64(14) {
		t.Fatalf("new item defaults: %+v %v", preview, err)
	}
	f.exec(`UPDATE issues SET custom_properties='{}' WHERE id=$1`, item)
	command.IssueID = item
	preview, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	if err != nil || preview.Result == nil || preview.Result.Status != "empty" {
		t.Fatalf("existing item incorrectly filled default: %+v %v", preview, err)
	}
	for _, test := range []struct {
		expression string
		kind       formula.Kind
		valid      bool
		status     string
	}{
		{"{{unknown}}", formula.Number, false, ""}, {"TODAY()", formula.Number, false, ""}, {"1 / 0", formula.Number, true, "error"}, {"IF(false, 1 / 0, 0)", formula.Number, true, "value"}, {"TODAY()", formula.Date, true, "value"},
	} {
		command.Definition.Configuration.Expression, command.Definition.Configuration.Result = test.expression, test.kind
		preview, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
		if err != nil || preview.Valid != test.valid {
			t.Fatalf("%s validation: %+v %v", test.expression, preview, err)
		}
		if test.valid && (preview.Result == nil || preview.Result.Status != test.status) {
			t.Fatalf("%s result: %+v", test.expression, preview)
		}
		if !test.valid && (preview.ValidationError == "" || preview.Result != nil) {
			t.Fatalf("invalid formula evaluated: %+v", preview)
		}
	}
}

func TestFormulaAuthoringUsesContextAndValidatesEveryAttachedType(t *testing.T) {
	f := newProjectTypeFixture(t)
	effort, calculation := uuid.New().String(), numericAuthoringFormula("LEN({{name}})")
	attachValueProperties(t, f,
		property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(7)}, Contexts: []property.Context{{ID: uuid.New().String(), Name: "Project effort", Projects: []string{f.project}, Overrides: []string{"default_value"}, Configuration: property.Configuration{Default: float64(3)}}}}, calculation)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range catalog.Properties {
		if record.Definition.ID == calculation.ID {
			calculation = record.Definition
		}
	}
	command := types.FormulaAuthoringInput{Revision: catalog.Revision, Definition: calculation, TypeID: f.task, ProjectID: f.project}
	command.Definition.Configuration.Expression = "{{effort}} * 2"
	preview, err := f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	if err != nil || !preview.Valid || preview.Result == nil || preview.Result.Value != float64(6) {
		t.Fatalf("effective default: %+v %v", preview, err)
	}
	if !strings.Contains(preview.Contexts[0].Name, "Project effort") {
		t.Fatalf("missing effective context: %+v", preview.Contexts)
	}
	catalog, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "attach", TypeID: f.bug, PropertyIDs: []string{calculation.ID}})
	if err != nil {
		t.Fatal(err)
	}
	command.Revision = catalog.Revision
	preview, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	if err != nil || preview.Valid || preview.ValidationError == "" {
		t.Fatalf("other attached type bypassed: %+v %v", preview, err)
	}
	command.Definition.Configuration.Expression = "{{calculation}}"
	preview, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	if err != nil || preview.Valid {
		t.Fatalf("self-reference allowed: %+v %v", preview, err)
	}
	command.Definition.Configuration.Expression = "LEN({{name}})"
	command.Definition.Contexts = []property.Context{{ID: uuid.New().String(), Name: "Hidden here", Projects: []string{f.project}, Overrides: []string{"active"}, Configuration: property.Configuration{Active: false}}}
	preview, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	if err != nil || !preview.Valid || preview.Active || preview.Result != nil {
		t.Fatalf("inactive formula masqueraded as empty: %+v %v", preview, err)
	}
}

func TestFormulaAuthoringRejectsPrivateSamplesAndStaleConfiguration(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, draft, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Private sample"}`)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	command := types.FormulaAuthoringInput{Revision: catalog.Revision, Definition: numericAuthoringFormula("LEN({{name}})"), TypeID: f.task, ProjectID: f.project, IssueID: resultIssueID(t, created)}
	for _, actor := range []string{f.member, f.outsider} {
		_, err = f.repo.PreviewFormula(f.ctx, actor, "types", command)
		requireTypeStatus(t, err, 403)
	}
	command.TypeID = f.bug
	_, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 422)
	command.TypeID, command.IssueID = f.task, draft
	_, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 404)
	command.IssueID = resultIssueID(t, created)
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.owner)
	_, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 404)
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.owner)
	_, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 403)
	command.Revision--
	_, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", command)
	requireTypeStatus(t, err, 409)
}

func TestFormulaAuthoringSharesWriterCatalogValidation(t *testing.T) {
	f := newProjectTypeFixture(t)
	number := property.Definition{ID: uuid.New().String(), Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}}
	attachValueProperties(t, f, number)
	catalog, err := f.repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil {
		t.Fatal(err)
	}
	duplicate := numericAuthoringFormula("{{effort}} * 2")
	duplicate.Title = "Effort"
	input := types.FormulaAuthoringInput{Revision: catalog.Revision, Definition: duplicate, TypeID: f.task}
	preview, err := f.repo.PreviewFormula(f.ctx, f.owner, "types", input)
	if err != nil || preview.Valid || !strings.Contains(preview.ValidationError, "internal name") {
		t.Fatalf("duplicate name preview: %+v %v", preview, err)
	}
	dependent := numericAuthoringFormula("{{effort}} * 2")
	catalog, err = f.repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: dependent})
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range catalog.Properties {
		if record.Definition.ID == number.ID {
			number = record.Definition
		}
	}
	number.Kind = property.Formula
	number.Configuration.Expression, number.Configuration.Result = "1", formula.Number
	input.Definition, input.Revision = number, catalog.Revision
	preview, err = f.repo.PreviewFormula(f.ctx, f.owner, "types", input)
	if err != nil || preview.Valid || !strings.Contains(preview.ValidationError, "effort") {
		t.Fatalf("unattached dependent ignored: %+v %v", preview, err)
	}
}

func TestFormulaSampleSearchFiltersBeforeTheResultLimit(t *testing.T) {
	f := newProjectTypeFixture(t)
	f.mutate(t, "import", f.bug)
	repo, _, _ := seedPublicationDraft(t, f)
	created, err := writeValueItem(f, repo, "POST", "", `{"name":"Find sample"}`)
	if err != nil {
		t.Fatal(err)
	}
	item := resultIssueID(t, created)
	f.exec(`INSERT INTO issues SELECT (jsonb_populate_record(NULL::issues,to_jsonb(i)||jsonb_build_object('id',gen_random_uuid(),'sequence_id',1000+n,'created_at',now()+interval '1 day','type_id',$2::text))).* FROM issues i CROSS JOIN generate_series(1,101) n WHERE i.id=$1`, item, f.bug)
	request := workitems.Request{Operation: "api_workspaces_by_slug_projects_by_project_id_search_issues", Method: "GET", ActorID: f.owner, Slug: "types", ProjectID: f.project, Params: map[string]string{"search": "Find sample", "workspace_search": "false", "type_id": f.task}}
	result, err := repo.Execute(f.ctx, request, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	var items []struct {
		ID     string `json:"id"`
		TypeID string `json:"type_id"`
	}
	if err = json.Unmarshal(raw, &items); err != nil || len(items) != 1 || items[0].ID != item || items[0].TypeID != f.task {
		t.Fatalf("sample hidden behind other types: %s %v", raw, err)
	}
	request.Params["type_id"] = "invalid"
	_, err = repo.Execute(f.ctx, request, nil, "")
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 {
		t.Fatalf("invalid type accepted: %v", err)
	}
}
