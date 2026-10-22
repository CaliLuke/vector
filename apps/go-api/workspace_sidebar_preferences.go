package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_sidebar_preferences"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacesidebarpreferences"
	"github.com/CaliLuke/loom/security"
)

type workspaceSidebarPreferencesBackend interface {
	Get(context.Context, uuid.UUID, string) (json.RawMessage, error)
	Patch(context.Context, uuid.UUID, string, []store.Patch) (json.RawMessage, error)
}

type workspaceSidebarPreferencesSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceSidebarPreferencesService struct {
	backend  workspaceSidebarPreferencesBackend
	sessions workspaceSidebarPreferencesSessions
}

type workspaceSidebarPreferencesPrincipal struct{}

// NewWorkspaceSidebarPreferences constructs the sidebar preference service.
func NewWorkspaceSidebarPreferences(backend workspaceSidebarPreferencesBackend, sessions workspaceSidebarPreferencesSessions) (api.Service, error) {
	if backend == nil || sessions == nil {
		return nil, fmt.Errorf("workspace sidebar preferences dependencies are required")
	}
	return &workspaceSidebarPreferencesService{backend: backend, sessions: sessions}, nil
}

func (s *workspaceSidebarPreferencesService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.WorkspaceSidebarPreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.WorkspaceSidebarPreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, workspaceSidebarPreferencesPrincipal{}, userID), nil
}

func (s *workspaceSidebarPreferencesService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(workspaceSidebarPreferencesPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WorkspaceSidebarPreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func (s *workspaceSidebarPreferencesService) Get(ctx context.Context, payload *api.GetPayload) (map[string]*api.WorkspaceSidebarPreference, error) {
	userID, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	body, err := s.backend.Get(ctx, userID, payload.Slug)
	if err != nil {
		return nil, workspaceSidebarPreferencesError(err)
	}
	var result map[string]*api.WorkspaceSidebarPreference
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, workspaceSidebarPreferencesInternalError()
	}
	return result, nil
}

func (s *workspaceSidebarPreferencesService) Patch(ctx context.Context, payload *api.PatchPayload) (*api.WorkspaceSidebarPreferenceMessage, error) {
	userID, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}
	patches := make([]store.Patch, 0, len(payload.Body))
	for _, item := range payload.Body {
		if item == nil || item.Key == nil {
			patches = append(patches, store.Patch{})
			continue
		}
		patch := store.Patch{Key: *item.Key}
		if item.IsPinned.Present() && !item.IsPinned.IsNull() {
			value, ok := item.IsPinned.Value()
			if ok {
				patch.IsPinned = &value
			}
		}
		if item.SortOrder.Present() && !item.SortOrder.IsNull() {
			value, ok := item.SortOrder.Value()
			if ok {
				patch.SortOrder = &value
			}
		}
		patches = append(patches, patch)
	}
	body, err := s.backend.Patch(ctx, userID, payload.Slug, patches)
	if err != nil {
		return nil, workspaceSidebarPreferencesError(err)
	}
	var result api.WorkspaceSidebarPreferenceMessage
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, workspaceSidebarPreferencesInternalError()
	}
	return &result, nil
}

func workspaceSidebarPreferencesError(err error) error {
	if errors.Is(err, store.ErrUnauthenticated) {
		return &api.WorkspaceSidebarPreferenceUnauthenticated{Detail: unauthenticatedDetail}
	}
	if errors.Is(err, store.ErrForbidden) {
		return &api.WorkspaceSidebarPreferenceForbidden{Detail: "You do not have the required permissions."}
	}
	return workspaceSidebarPreferencesInternalError()
}

func workspaceSidebarPreferencesInternalError() error {
	return &api.WorkspaceSidebarPreferenceInternal{Message: "Something went wrong please try again later"}
}

var _ workspaceSidebarPreferencesBackend = (*store.Service)(nil)
