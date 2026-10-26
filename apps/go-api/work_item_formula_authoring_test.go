package systemapi

import (
	"context"
	"errors"
	"testing"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func TestFormulaAuthoringAdapterScopeAndFailures(t *testing.T) {
	repo := &typeServiceRepository{}
	service, err := NewWorkItemTypes(repo, typeServiceSessions{})
	if err != nil {
		t.Fatal(err)
	}
	project, issue := "project", "sample"
	payload := &api.PreviewFormulaPayload{Slug: "workspace", Data: &api.FormulaAuthoringInput{
		Revision: 7, TypeID: "task", ProjectID: &project, IssueID: &issue,
		Definition: &api.PropertyDefinition{ID: "formula", Name: "calculation", Title: "Calculation", Kind: "Formula", Configuration: &api.PropertyConfiguration{Active: true, Expression: "1-1", Result: "Number"}},
	}}
	_, err = service.PreviewFormula(context.Background(), payload)
	assertFormulaStatus(t, err, "status_401")
	if repo.actor != "" {
		t.Fatal("anonymous request dispatched")
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.PreviewFormula(ctx, &api.PreviewFormulaPayload{Slug: "workspace"})
	assertFormulaStatus(t, err, "status_400")
	if repo.actor != "" {
		t.Fatal("missing definition dispatched")
	}
	result, err := service.PreviewFormula(ctx, payload)
	if err != nil {
		t.Fatal(err)
	}
	input := repo.formulaInput
	if repo.actor != "actor" || repo.slug != "workspace" || input.Revision != 7 || input.TypeID != "task" || input.ProjectID != project || input.IssueID != issue || input.Definition.Configuration.Expression != "1-1" {
		t.Fatalf("preview scope lost: %+v %+v", repo, input)
	}
	if !result.Valid || result.Result == nil || string(result.Result.Value) != "0" {
		t.Fatalf("zero result lost: %+v", result)
	}
	for _, failure := range []struct {
		status int
		kind   string
	}{{403, "status_403"}, {409, "status_409"}} {
		repo.failure = &store.Error{Status: failure.status, Message: "Denied"}
		_, err = service.PreviewFormula(ctx, payload)
		assertFormulaStatus(t, err, failure.kind)
	}
	repo.failure = errors.New("private database details")
	_, err = service.PreviewFormula(ctx, payload)
	assertFormulaStatus(t, err, "status_500")
	if err.Error() == repo.failure.Error() {
		t.Fatal("private failure exposed")
	}
}
func assertFormulaStatus(t *testing.T, err error, kind string) {
	t.Helper()
	var failure *api.TypeError
	if !errors.As(err, &failure) || failure.Kind != kind {
		t.Fatalf("got %v, want %s", err, kind)
	}
}
