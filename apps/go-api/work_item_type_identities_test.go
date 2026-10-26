package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	store "github.com/CaliLuke/go-plane/internal/workitemtypes"
	api "github.com/CaliLuke/go-plane/workitemtypes/gen/work_item_types"
)

func TestTypeIdentityServiceScopePrivacyAndErrors(t *testing.T) {
	repo := &typeServiceRepository{}
	service, err := NewWorkItemTypes(repo, typeServiceSessions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.TypeIdentities(context.Background(), &api.TypeIdentitiesPayload{Slug: "workspace"})
	var failure *api.TypeError
	if !errors.As(err, &failure) || failure.Kind != "status_401" || repo.actor != "" {
		t.Fatalf("anonymous dispatch: %v %+v", err, repo)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "valid", nil)
	if err != nil {
		t.Fatal(err)
	}
	project := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	for _, scope := range []*string{nil, &project} {
		result, readErr := service.TypeIdentities(ctx, &api.TypeIdentitiesPayload{Slug: "workspace", ProjectID: scope})
		if readErr != nil {
			t.Fatal(readErr)
		}
		expected := ""
		if scope != nil {
			expected = *scope
		}
		if repo.project != expected || repo.actor != "actor" || repo.slug != "workspace" || result.Revision != 7 || len(result.Types) != 1 || result.Types[0].Active {
			t.Fatalf("scope/result lost: %+v %+v", repo, result)
		}
		raw, encodeErr := json.Marshal(result)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		for _, private := range []string{"usage", "project_count", "can_manage"} {
			if strings.Contains(string(raw), `"`+private+`"`) {
				t.Fatalf("private catalog metadata: %s", raw)
			}
		}
	}
	repo.failure = &store.Error{Status: 403, Message: "Access denied"}
	_, err = service.TypeIdentities(ctx, &api.TypeIdentitiesPayload{Slug: "workspace"})
	if !errors.As(err, &failure) || failure.Kind != "status_403" {
		t.Fatalf("access mapping: %v", err)
	}
}
