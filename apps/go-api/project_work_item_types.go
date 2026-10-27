package systemapi

import (
	"context"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) ProjectCatalog(ctx context.Context, p *api.ProjectCatalogPayload) (*api.ProjectCatalog, error) {
	return s.projectTypes(ctx, p.Slug, p.ProjectID, nil)
}
func (s *workItemTypesService) UpdateProject(ctx context.Context, p *api.UpdateProjectPayload) (*api.ProjectCatalog, error) {
	return s.projectTypes(ctx, p.Slug, p.ProjectID, &store.ProjectInput{Revision: p.Data.Revision, Action: p.Data.Action, TypeIDs: p.Data.TypeIds})
}
func (s *workItemTypesService) projectTypes(ctx context.Context, slug, project string, input *store.ProjectInput) (*api.ProjectCatalog, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	if input != nil && input.Action != "set_default" && len(input.TypeIDs) == 0 {
		return nil, &api.TypeError{Kind: "status_400", Message: "Choose at least one work item type."}
	}
	var result store.ProjectCatalog
	var err error
	if input == nil {
		result, err = s.repository.ProjectCatalog(ctx, actor, slug, project)
	} else {
		result, err = s.repository.MutateProject(ctx, actor, slug, project, *input)
	}
	if err != nil {
		return nil, typeServiceError(ctx, "work_item_types.project", err)
	}
	out := &api.ProjectCatalog{Enabled: result.Enabled, CanManage: result.CanManage, Revision: result.Revision, ProjectName: result.ProjectName, DefaultTypeID: result.DefaultTypeID, Types: make([]*api.ProjectWorkItemType, 0, len(result.Types))}
	for _, item := range result.Types {
		out.Types = append(out.Types, &api.ProjectWorkItemType{ID: item.ID, Name: item.Name, Description: item.Description, Icon: item.Icon, Color: item.Color, Active: item.Active, Default: item.Default, Mandatory: item.Mandatory, Level: item.Level, Imported: item.Imported, ProjectActive: item.ProjectActive, Available: item.Available})
	}
	return out, nil
}
