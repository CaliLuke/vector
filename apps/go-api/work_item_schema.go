package systemapi

import (
	"context"
	"encoding/json"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) ItemSchema(ctx context.Context, p *api.ItemSchemaPayload) (*api.ItemSchema, error) {
	return s.loadItemSchema(ctx, p.Slug, itemSchemaScope(p.Mode, p.TypeID, p.ProjectID, p.IssueID, p.DraftID, p.ParentID))
}

func (s *workItemTypesService) PreviewItemSchema(ctx context.Context, p *api.PreviewItemSchemaPayload) (*api.ItemSchema, error) {
	if p.Preview == nil {
		return nil, &api.TypeError{Kind: "status_400", Message: "Preview values are required."}
	}
	input := itemSchemaScope(p.Mode, p.TypeID, p.ProjectID, p.IssueID, p.DraftID, p.ParentID)
	preview := p.Preview
	input.Preview = &store.ItemFormulaPreview{Name: preview.Name, Priority: preview.Priority, StartDate: preview.StartDate, DueDate: preview.TargetDate, EstimatePointID: preview.EstimatePoint, Values: map[string]any{}}
	for id, raw := range preview.CustomProperties {
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, &api.TypeError{Kind: "status_400", Message: "Invalid preview value."}
		}
		input.Preview.Values[id] = value
	}
	return s.loadItemSchema(ctx, p.Slug, input)
}

func itemSchemaScope(mode string, typeID, projectID, issueID, draftID, parentID *string) store.ItemSchemaInput {
	input := store.ItemSchemaInput{Mode: mode, TypeID: typeID, ParentID: parentID}
	if projectID != nil {
		input.ProjectID = *projectID
	}
	if issueID != nil {
		input.IssueID = *issueID
	}
	if draftID != nil {
		input.DraftID = *draftID
	}
	return input
}

func (s *workItemTypesService) loadItemSchema(ctx context.Context, slug string, input store.ItemSchemaInput) (*api.ItemSchema, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	result, err := s.repository.ItemSchema(ctx, actor, slug, input)
	return workItemTypesResponse[api.ItemSchema](result, err)
}
