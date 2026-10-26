//go:build integration

package workitemtypes_test

import (
	"testing"
	"uuid"

	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
)

func TestFormulaPreviewUsesUnsavedInputsWithoutDurableChanges(t *testing.T) {
	f := newProjectTypeFixture(t)
	effort, flag, calculation, priority := uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: effort, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true, Default: float64(3)}},
		property.Definition{ID: flag, Title: "Verified", Kind: property.Boolean, Configuration: property.Configuration{Active: true}},
		property.Definition{ID: calculation, Title: "Calculation", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "IF({{verified}}, {{effort}} * 2, 0)", Result: "Number"}},
		property.Definition{ID: priority, Title: "Priority label", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{priority}}", Result: "Text"}},
	)
	for _, mode := range []string{"create", "draft"} {
		preview, err := f.repo.ItemSchema(f.ctx, f.member, "types", types.ItemSchemaInput{Mode: mode, ProjectID: f.project})
		if err != nil || preview.Computed[priority].Value != "NONE" {
			t.Fatalf("%s priority default: %+v %v", mode, preview.Computed, err)
		}
	}
	repo, draft, _ := seedPublicationDraft(t, f)
	result, err := publishDraftForTest(f, repo, draft, f.member, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	issue := resultIssueID(t, result)
	var effects int
	if err = f.db.QueryRow(f.ctx, `SELECT count(*) FROM work_item_effects`).Scan(&effects); err != nil {
		t.Fatal(err)
	}
	input := types.ItemSchemaInput{Mode: "edit", ProjectID: f.project, IssueID: issue, Preview: &types.ItemFormulaPreview{Priority: new("urgent"), Values: map[string]any{effort: float64(7), flag: true}}}
	preview, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || preview.Computed[calculation].Value != float64(14) || preview.Computed[priority].Value != "URGENT" {
		t.Fatalf("unsaved input preview: %+v %v", preview.Computed, err)
	}
	input.Preview.Values[effort] = "unfinished number"
	input.Preview.Values[flag] = false
	preview, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || preview.Computed[calculation].Value != float64(0) {
		t.Fatalf("preview evaluated unused invalid input: %+v %v", preview.Computed, err)
	}
	input.Preview.Values[flag] = true
	preview, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || preview.Computed[calculation].Status != "error" {
		t.Fatalf("invalid input became a value: %+v %v", preview.Computed, err)
	}
	input.Preview.Values = map[string]any{effort: nil, flag: true}
	preview, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || preview.Computed[calculation].Status != "empty" {
		t.Fatalf("explicit clear became default: %+v %v", preview.Computed, err)
	}
	saved := draftValues(t, f, "issues", issue)
	var afterEffects int
	var savedPriority string
	if err = f.db.QueryRow(f.ctx, `SELECT priority,(SELECT count(*) FROM work_item_effects) FROM issues WHERE id=$1`, issue).Scan(&savedPriority, &afterEffects); err != nil {
		t.Fatal(err)
	}
	if saved[effort] != float64(3) || saved[flag] != false || savedPriority != "high" || afterEffects != effects {
		t.Fatalf("preview mutated saved item/effects: %+v %s %d/%d", saved, savedPriority, afterEffects, effects)
	}
	input.Preview.Values = map[string]any{uuid.New().String(): float64(1)}
	_, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 422)
}

func TestFormulaPreviewUsesWriterCanonicalizationAndLazyValidation(t *testing.T) {
	f := newProjectTypeFixture(t)
	text, date, titleLength, textLength, dateResult, lazy := uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()
	attachValueProperties(t, f,
		property.Definition{ID: text, Title: "Code", Kind: property.Text, Configuration: property.Configuration{Active: true, Format: "Single line"}},
		property.Definition{ID: date, Title: "Launch", Kind: property.Date, Configuration: property.Configuration{Active: true, Format: "2025/01/15"}},
		property.Definition{ID: titleLength, Title: "Title length", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "LEN({{name}})", Result: "Number"}},
		property.Definition{ID: textLength, Title: "Code length", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "LEN({{code}})", Result: "Number"}},
		property.Definition{ID: dateResult, Title: "Launch result", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{launch}}", Result: "Date"}},
		property.Definition{ID: lazy, Title: "Lazy result", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "IF(false, LEN({{code}}), 4)", Result: "Number"}},
	)
	input := types.ItemSchemaInput{Mode: "create", ProjectID: f.project, Preview: &types.ItemFormulaPreview{Name: new("\u001c Bug \u001f"), Values: map[string]any{text: "a\nb", date: "   "}}}
	preview, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Computed[titleLength].Value != float64(3) || preview.Computed[textLength].Status != "error" || preview.Computed[dateResult].Status != "empty" || preview.Computed[lazy].Value != float64(4) {
		t.Fatalf("preview canonicalization differs: %+v", preview.Computed)
	}
}

func TestDraftFormulaPreviewDoesNotReadInaccessibleEstimateCatalogs(t *testing.T) {
	f := newProjectTypeFixture(t)
	calculation := uuid.New().String()
	attachValueProperties(t, f, property.Definition{ID: calculation, Title: "Estimate value", Kind: property.Formula, Configuration: property.Configuration{Active: true, Expression: "{{estimate_point}}", Result: "Number"}})
	_, draft, _ := seedPublicationDraft(t, f)
	estimate, point := uuid.New().String(), uuid.New().String()
	f.exec(`INSERT INTO estimates(id,workspace_id,project_id,name,description,type,last_used,created_at,updated_at) VALUES($1,$2,$3,'Points','','points',true,now(),now())`, estimate, f.workspace, f.project)
	f.exec(`INSERT INTO estimate_points(id,workspace_id,project_id,estimate_id,key,value,description,created_at,updated_at) VALUES($1,$2,$3,$4,1,'5','',now(),now())`, point, f.workspace, f.project, estimate)
	f.exec(`UPDATE draft_issues SET estimate_point_id=$2 WHERE id=$1`, draft, point)
	input := types.ItemSchemaInput{Mode: "draft", ProjectID: f.project, DraftID: draft}
	before, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || before.Computed[calculation].Value != float64(5) {
		t.Fatalf("accessible estimate: %+v %v", before.Computed, err)
	}
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	hidden, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || hidden.Computed[calculation].Status != "empty" {
		t.Fatalf("revoked project catalog exposed: %+v %v", hidden.Computed, err)
	}
	input.Preview = &types.ItemFormulaPreview{EstimatePointID: &point}
	_, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 403)
	target := uuid.New().String()
	f.addProject(target)
	f.exec(`INSERT INTO project_members(id,workspace_id,project_id,member_id,role,is_active,created_at,updated_at,view_props,default_props,preferences,sort_order) VALUES(gen_random_uuid(),$1,$2,$3,15,true,now(),now(),'{}','{}','{}',0)`, f.workspace, target, f.member)
	input.ProjectID, input.Preview = target, nil
	moved, err := f.repo.ItemSchema(f.ctx, f.member, "types", input)
	if err != nil || moved.Computed[calculation].Status != "empty" {
		t.Fatalf("target preview used old project estimate: %+v %v", moved.Computed, err)
	}
	input.Preview = &types.ItemFormulaPreview{EstimatePointID: &point}
	_, err = f.repo.ItemSchema(f.ctx, f.member, "types", input)
	requireTypeStatus(t, err, 422)
}
