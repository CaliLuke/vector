package systemapi

import (
	"context"
	"encoding/json"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) PropertyCatalog(ctx context.Context, p *api.PropertyCatalogPayload) (*api.PropertyCatalog, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	result, err := s.repository.PropertyCatalog(ctx, actor, p.Slug)
	return propertyCatalogResponse(result, err)
}
func (s *workItemTypesService) MutateProperty(ctx context.Context, p *api.MutatePropertyPayload) (*api.PropertyCatalog, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	input, err := propertyCommand(p.Data)
	if err != nil {
		return nil, err
	}
	result, err := s.repository.MutateProperty(ctx, actor, p.Slug, input)
	return propertyCatalogResponse(result, err)
}
func (s *workItemTypesService) PreviewPropertyRemoval(ctx context.Context, p *api.PreviewPropertyRemovalPayload) (*api.PropertyReview, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	input, err := propertyCommand(p.Data)
	if err != nil {
		return nil, err
	}
	result, err := s.repository.PreviewPropertyRemoval(ctx, actor, p.Slug, input)
	if err != nil {
		return nil, typeServiceError(err)
	}
	return &api.PropertyReview{ID: result.ID, Items: result.Items, Types: result.Types, Projects: result.Projects}, nil
}
func propertyCommand(data *api.PropertyInput) (store.PropertyInput, error) {
	if data == nil {
		return store.PropertyInput{}, &api.TypeError{Kind: "status_400", Message: "A property command is required."}
	}
	input := store.PropertyInput{Action: data.Action, Revision: data.Revision, PropertyIDs: data.PropertyIds}
	if data.TypeID != nil {
		input.TypeID = *data.TypeID
	}
	if data.ID != nil {
		input.ID = *data.ID
	}
	if data.ReviewID != nil {
		input.ReviewID = *data.ReviewID
	}
	if data.Permanent != nil {
		input.Permanent = *data.Permanent
	}
	if ((input.Action == "attach" || input.Action == "detach") && input.TypeID == "") || ((input.Action == "delete" || input.Action == "detach") && input.ID == "") || ((input.Action == "create" || input.Action == "update") && data.Definition == nil) {
		return input, &api.TypeError{Kind: "status_400", Message: "The property command is incomplete."}
	}
	if data.Definition != nil {
		raw, err := json.Marshal(data.Definition)
		if err == nil {
			err = json.Unmarshal(raw, &input.Definition)
		}
		if err != nil {
			return input, &api.TypeError{Kind: "status_400", Message: "Invalid property definition."}
		}
	}
	return input, nil
}
func propertyCatalogResponse(result store.PropertyCatalog, err error) (*api.PropertyCatalog, error) {
	return workItemTypesResponse[api.PropertyCatalog](result, err)
}

func workItemTypesResponse[T any](result any, err error) (*T, error) {
	if err != nil {
		return nil, typeServiceError(err)
	}
	// The domain JSON contract is also the persistence contract. Keep the
	// generated HTTP structs at this boundary, including nullable default values.
	raw, err := json.Marshal(result)
	var out T
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	if err != nil {
		return nil, typeServiceError(err)
	}
	return &out, nil
}
