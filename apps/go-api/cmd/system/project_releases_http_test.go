package main

import (
	"context"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
)

func (r *typeHTTPRepository) ReleaseCatalog(ctx context.Context, actor, slug, project string) (store.ReleaseCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	return store.ReleaseCatalog{Revision: 8, CanManage: true, ProjectName: project, Releases: []store.Release{}}, err
}
func (r *typeHTTPRepository) MutateReleaseCatalog(ctx context.Context, actor, slug, project string, input store.ReleaseInput) (store.ReleaseCatalog, error) {
	r.releaseInput = input
	return r.ReleaseCatalog(ctx, actor, slug, project)
}
