package systemapi

import (
	"context"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) HierarchyCatalog(ctx context.Context, p *api.HierarchyCatalogPayload) (*api.HierarchyCatalog, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	result, err := s.repository.HierarchyCatalog(ctx, actor, p.Slug)
	return workItemTypesResponse[api.HierarchyCatalog](result, err)
}
func (s *workItemTypesService) PreviewHierarchy(ctx context.Context, p *api.PreviewHierarchyPayload) (*api.HierarchyReview, error) {
	return hierarchyMutation[api.HierarchyReview](ctx, p.Slug, p.Data, s.repository.PreviewHierarchy)
}
func (s *workItemTypesService) CommitHierarchy(ctx context.Context, p *api.CommitHierarchyPayload) (*api.HierarchyCatalog, error) {
	return hierarchyMutation[api.HierarchyCatalog](ctx, p.Slug, p.Data, s.repository.CommitHierarchy)
}
func hierarchyMutation[Result, Stored any](ctx context.Context, slug string, data *api.HierarchyInput, write func(context.Context, string, string, store.HierarchyInput) (Stored, error)) (*Result, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	if data == nil {
		return nil, &api.TypeError{Kind: "status_400", Message: "A hierarchy configuration is required."}
	}
	input := store.HierarchyInput{Revision: data.Revision, DefaultLevel: data.DefaultLevel, Placements: []store.HierarchyPlacement{}}
	for _, placement := range data.Placements {
		if placement == nil {
			return nil, &api.TypeError{Kind: "status_400", Message: "Choose a valid type placement."}
		}
		input.Placements = append(input.Placements, store.HierarchyPlacement{TypeID: placement.TypeID, Level: placement.Level})
	}
	if data.ReviewID != nil {
		input.ReviewID = *data.ReviewID
	}
	if data.Confirm != nil {
		input.Confirm = *data.Confirm
	}
	result, err := write(ctx, actor, slug, input)
	return workItemTypesResponse[Result](result, err)
}
