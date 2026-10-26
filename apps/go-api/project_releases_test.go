package systemapi

import (
	"context"
	"testing"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func (r *typeServiceRepository) ReleaseCatalog(ctx context.Context, actor, slug, project string) (store.ReleaseCatalog, error) {
	_, err := r.Catalog(ctx, actor, slug)
	r.project = project
	return store.ReleaseCatalog{Revision: 8, CanManage: true, ProjectName: project, Releases: []store.Release{{ID: "release", ReleaseValues: store.ReleaseValues{Name: "October", Description: "Notes", TargetDate: "2026-10-31"}, CreatedAt: "2026-10-01T00:00:00Z"}}}, err
}
func (r *typeServiceRepository) MutateReleaseCatalog(ctx context.Context, actor, slug, project string, input store.ReleaseInput) (store.ReleaseCatalog, error) {
	r.releaseInput = input
	return r.ReleaseCatalog(ctx, actor, slug, project)
}
func TestReleaseServiceCarriesAuthenticatedScopeAndValues(t *testing.T) {
	repository := &typeServiceRepository{}
	service, err := NewWorkItemTypes(repository, typeServiceSessions{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	input := &api.ReleaseInput{Revision: 7, Action: "update", ID: "release", Values: &api.ReleaseValues{Name: "October", Description: "Notes", TargetDate: "2026-10-31"}}
	out, err := service.UpdateReleaseCatalog(ctx, &api.UpdateReleaseCatalogPayload{Slug: "workspace", ProjectID: "project", Data: input})
	if err != nil {
		t.Fatal(err)
	}
	if repository.actor != "actor" || repository.slug != "workspace" || repository.project != "project" || repository.releaseInput.Revision != 7 || repository.releaseInput.ID != "release" || repository.releaseInput.Action != "update" || repository.releaseInput.Values.TargetDate != "2026-10-31" {
		t.Fatalf("lost authenticated request: %+v", repository)
	}
	if out.Revision != 8 || len(out.Releases) != 1 || out.Releases[0].Description != "Notes" || out.Releases[0].CreatedAt != "2026-10-01T00:00:00Z" {
		t.Fatalf("lost catalog response: %+v", out)
	}
}
