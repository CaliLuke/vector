package systemapi

import (
	"context"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) PreviewTypeRemoval(ctx context.Context, p *api.PreviewTypeRemovalPayload) (*api.TypeRemovalReview, error) {
	return typeRemovalMutation[api.TypeRemovalReview](ctx, p.Slug, p.Data, s.repository.PreviewTypeRemoval)
}
func (s *workItemTypesService) CommitTypeRemoval(ctx context.Context, p *api.CommitTypeRemovalPayload) (*api.Catalog, error) {
	return typeRemovalMutation[api.Catalog](ctx, p.Slug, p.Data, s.repository.CommitTypeRemoval)
}
func typeRemovalMutation[Result, Stored any](ctx context.Context, slug string, data *api.TypeRemovalInput, write func(context.Context, string, string, store.TypeRemovalInput) (Stored, error)) (*Result, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	if data == nil {
		return nil, &api.TypeError{Kind: "status_400", Message: "Choose the type and its replacement."}
	}
	in := store.TypeRemovalInput{Revision: data.Revision, TypeID: data.TypeID, ReplacementID: data.ReplacementID}
	if data.ReviewID != nil {
		in.ReviewID = *data.ReviewID
	}
	out, err := write(ctx, actor, slug, in)
	return workItemTypesResponse[Result](out, err)
}
