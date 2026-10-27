package systemapi

import (
	"context"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) PreviewFormula(ctx context.Context, p *api.PreviewFormulaPayload) (*api.FormulaAuthoringResult, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	if p.Data == nil {
		return nil, &api.TypeError{Kind: "status_400", Message: "A formula preview is required."}
	}
	command, err := propertyCommand(&api.PropertyInput{Action: "update", Revision: p.Data.Revision, Definition: p.Data.Definition})
	if err != nil {
		return nil, err
	}
	input := store.FormulaAuthoringInput{Revision: command.Revision, Definition: command.Definition, TypeID: p.Data.TypeID}
	if p.Data.ProjectID != nil {
		input.ProjectID = *p.Data.ProjectID
	}
	if p.Data.IssueID != nil {
		input.IssueID = *p.Data.IssueID
	}
	result, err := s.repository.PreviewFormula(ctx, actor, p.Slug, input)
	return workItemTypesResponse[api.FormulaAuthoringResult](ctx, "work_item_types.formula_preview", result, err)
}
