package systemapi

import (
	"context"
	"errors"

	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) DiscoveryProperties(ctx context.Context, p *api.DiscoveryPropertiesPayload) (*api.PropertyDiscoveryCatalog, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	project := ""
	if p.ProjectID != nil {
		project = *p.ProjectID
	}
	catalog, err := s.repository.DiscoveryProperties(ctx, actor, p.Slug, project)
	if err != nil {
		response := typeServiceError(err)
		var failure *api.TypeError
		if errors.As(response, &failure) && failure.Kind == "status_500" {
			workItemLogError(ctx, "work_item_types.discovery_properties", "read", err)
		}
		return nil, response
	}
	out := &api.PropertyDiscoveryCatalog{Enabled: catalog.Enabled, Revision: catalog.Revision, Fields: []*api.DiscoveryProperty{}}
	for _, field := range catalog.Fields {
		item := &api.DiscoveryProperty{Key: field.Key, PropertyID: field.PropertyID, Name: field.Name, Icon: field.Icon, Kind: string(field.Kind), ResultKind: string(field.ResultKind), LevelID: field.LevelID, CatalogID: field.CatalogID, Multiple: field.Multiple, Groupable: field.Groupable, Operators: field.Operators, Options: []*api.PropertyChoice{}}
		for _, option := range field.Options {
			item.Options = append(item.Options, &api.PropertyChoice{ID: option.ID, Name: option.Name})
		}
		out.Fields = append(out.Fields, item)
	}
	return out, nil
}
