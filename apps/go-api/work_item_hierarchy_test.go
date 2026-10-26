package systemapi

import (
	"context"
	"errors"
	"testing"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func TestHierarchyAdapterAuthorizationAndFailures(t *testing.T) {
	repo := &typeServiceRepository{}
	service, err := NewWorkItemTypes(repo, typeServiceSessions{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name string
		call func(context.Context) error
	}{
		{"catalog", func(ctx context.Context) error {
			_, callErr := service.HierarchyCatalog(ctx, &api.HierarchyCatalogPayload{Slug: "workspace"})
			return callErr
		}},
		{"preview", func(ctx context.Context) error {
			_, callErr := service.PreviewHierarchy(ctx, &api.PreviewHierarchyPayload{Slug: "workspace", Data: &api.HierarchyInput{}})
			return callErr
		}},
		{"commit", func(ctx context.Context) error {
			_, callErr := service.CommitHierarchy(ctx, &api.CommitHierarchyPayload{Slug: "workspace", Data: &api.HierarchyInput{}})
			return callErr
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			var failure *api.TypeError
			if err = check.call(context.Background()); !errors.As(err, &failure) || failure.Kind != "status_401" {
				t.Fatalf("anonymous call: %v", err)
			}
			for _, status := range []int{403, 409} {
				repo.failure = &store.Error{Status: status, Message: "Denied"}
				err = check.call(ctx)
				if !errors.As(err, &failure) || failure.Message != "Denied" {
					t.Fatalf("domain failure lost: %v", err)
				}
			}
			repo.failure = errors.New("private database detail")
			err = check.call(ctx)
			if !errors.As(err, &failure) || failure.Kind != "status_500" || failure.Message == repo.failure.Error() {
				t.Fatalf("internal failure exposed: %v", err)
			}
			repo.failure = nil
		})
	}
	_, err = service.PreviewHierarchy(ctx, &api.PreviewHierarchyPayload{Slug: "workspace"})
	var failure *api.TypeError
	if !errors.As(err, &failure) || failure.Kind != "status_400" {
		t.Fatalf("missing configuration: %v", err)
	}
	_, err = service.CommitHierarchy(ctx, &api.CommitHierarchyPayload{Slug: "workspace", Data: &api.HierarchyInput{Placements: []*api.HierarchyPlacement{nil}}})
	if !errors.As(err, &failure) || failure.Kind != "status_400" {
		t.Fatalf("nil placement: %v", err)
	}
}
