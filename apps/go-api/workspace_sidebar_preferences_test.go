package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_sidebar_preferences"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacesidebarpreferences"
	loom "github.com/CaliLuke/loom/pkg"
)

type workspaceSidebarPreferencesAdapterSessionStub struct{ err error }

func (s workspaceSidebarPreferencesAdapterSessionStub) Validate(context.Context, string) (djangosession.Principal, error) {
	return djangosession.Principal{UserID: "00000000-0000-4000-8000-000000000021"}, s.err
}

type workspaceSidebarPreferencesAdapterBackendStub struct {
	body    json.RawMessage
	err     error
	patches []store.Patch
}

func (b *workspaceSidebarPreferencesAdapterBackendStub) Get(context.Context, uuid.UUID, string) (json.RawMessage, error) {
	return b.body, b.err
}
func (b *workspaceSidebarPreferencesAdapterBackendStub) Patch(_ context.Context, _ uuid.UUID, _ string, patches []store.Patch) (json.RawMessage, error) {
	b.patches = patches
	return b.body, b.err
}

func workspaceSidebarPreferencesAdapterContext(t *testing.T, backend *workspaceSidebarPreferencesAdapterBackendStub) (api.Service, context.Context) {
	t.Helper()
	service, err := NewWorkspaceSidebarPreferences(backend, workspaceSidebarPreferencesAdapterSessionStub{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := service.(api.Authorizer).APIKeyAuth(context.Background(), "session", nil)
	if err != nil {
		t.Fatal(err)
	}
	return service, ctx
}

func TestWorkspaceSidebarPreferencesGetDecodesRepositoryResult(t *testing.T) {
	service, ctx := workspaceSidebarPreferencesAdapterContext(t, &workspaceSidebarPreferencesAdapterBackendStub{body: json.RawMessage(`{"views":{"is_pinned":true,"sort_order":65535}}`)})
	result, err := service.Get(ctx, &api.GetPayload{Slug: "demo"})
	if err != nil || !result["views"].IsPinned || result["views"].SortOrder != 65535 {
		t.Fatalf("Get() = %#v, %v", result, err)
	}
}

func TestWorkspaceSidebarPreferencesPatchPreservesOptionalFieldsAndIgnoresMissingKeys(t *testing.T) {
	backend := &workspaceSidebarPreferencesAdapterBackendStub{body: json.RawMessage(`{"message":"Successfully updated"}`)}
	service, ctx := workspaceSidebarPreferencesAdapterContext(t, backend)
	pinned := loom.NullableValue(true)
	result, err := service.Patch(ctx, &api.PatchPayload{Slug: "demo", Body: []*api.WorkspaceSidebarPreferencePatch{
		{IsPinned: pinned},
		{Key: sidebarStringPointer("views"), IsPinned: pinned},
	}})
	if err != nil || result.Message != "Successfully updated" {
		t.Fatalf("Patch() = %#v, %v", result, err)
	}
	if len(backend.patches) != 2 || backend.patches[0].Key != "" || backend.patches[1].Key != "views" || backend.patches[1].IsPinned == nil || !*backend.patches[1].IsPinned {
		t.Fatalf("patches = %#v", backend.patches)
	}
}

func TestWorkspaceSidebarPreferencesMapsBackendErrors(t *testing.T) {
	service, ctx := workspaceSidebarPreferencesAdapterContext(t, &workspaceSidebarPreferencesAdapterBackendStub{err: store.ErrForbidden})
	if _, err := service.Get(ctx, &api.GetPayload{Slug: "demo"}); err == nil {
		t.Fatal("Get() returned nil error for forbidden backend")
	} else {
		var forbidden *api.WorkspaceSidebarPreferenceForbidden
		if !errors.As(err, &forbidden) {
			t.Fatalf("Get() error = %T %v, want forbidden", err, err)
		}
	}
}

func sidebarStringPointer(value string) *string { return &value }
