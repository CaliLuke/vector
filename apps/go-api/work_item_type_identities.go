package systemapi

import (
	"context"

	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) TypeIdentities(ctx context.Context, p *api.TypeIdentitiesPayload) (*api.TypeIdentityCatalog, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	project := ""
	if p.ProjectID != nil {
		project = *p.ProjectID
	}
	catalog, err := s.repository.TypeIdentities(ctx, actor, p.Slug, project)
	if err != nil {
		return nil, typeServiceError(err)
	}
	out := &api.TypeIdentityCatalog{Enabled: catalog.Enabled, Revision: catalog.Revision, Types: make([]*api.WorkItemTypeIdentity, 0, len(catalog.Types))}
	for _, item := range catalog.Types {
		out.Types = append(out.Types, &api.WorkItemTypeIdentity{ID: item.ID, Name: item.Name, Icon: item.Icon, Color: item.Color, Active: item.Active, Default: item.Default, Level: item.Level})
	}
	return out, nil
}
