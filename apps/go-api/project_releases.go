package systemapi

import (
	"context"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (s *workItemTypesService) ReleaseCatalog(ctx context.Context, p *api.ReleaseCatalogPayload) (*api.ReleaseCatalog, error) {
	return s.projectReleases(ctx, p.Slug, p.ProjectID, nil)
}
func (s *workItemTypesService) UpdateReleaseCatalog(ctx context.Context, p *api.UpdateReleaseCatalogPayload) (*api.ReleaseCatalog, error) {
	return s.projectReleases(ctx, p.Slug, p.ProjectID, &store.ReleaseInput{Revision: p.Data.Revision, Action: p.Data.Action, ID: p.Data.ID, Values: store.ReleaseValues{Name: p.Data.Values.Name, Description: p.Data.Values.Description, TargetDate: p.Data.Values.TargetDate}})
}
func (s *workItemTypesService) projectReleases(ctx context.Context, slug, project string, input *store.ReleaseInput) (*api.ReleaseCatalog, error) {
	actor, _ := ctx.Value(workItemTypesPrincipal{}).(string)
	if actor == "" {
		return nil, &api.TypeError{Kind: "status_401", Message: unauthenticatedDetail}
	}
	var catalog store.ReleaseCatalog
	var err error
	if input == nil {
		catalog, err = s.repository.ReleaseCatalog(ctx, actor, slug, project)
	} else {
		catalog, err = s.repository.MutateReleaseCatalog(ctx, actor, slug, project, *input)
	}
	if err != nil {
		return nil, typeServiceError(err)
	}
	out := &api.ReleaseCatalog{Revision: catalog.Revision, ProjectName: catalog.ProjectName, CanManage: catalog.CanManage, Releases: make([]*api.ProjectRelease, 0, len(catalog.Releases))}
	for _, item := range catalog.Releases {
		out.Releases = append(out.Releases, &api.ProjectRelease{ID: item.ID, Name: item.Name, Description: item.Description, TargetDate: item.TargetDate, CreatedAt: item.CreatedAt})
	}
	return out, nil
}
