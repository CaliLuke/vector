package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/CaliLuke/go-plane/internal/pythoncompat"

	"uuid"

	api "github.com/CaliLuke/go-plane/gen/workspace_quick_links"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/workspacequicklinks"
	"github.com/CaliLuke/loom/security"
)

type workspaceQuickLinksRepository interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	List(context.Context, uuid.UUID, string) (json.RawMessage, error)
	Get(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	Create(context.Context, uuid.UUID, string, string, *string, json.RawMessage) (json.RawMessage, error)
	Update(context.Context, uuid.UUID, string, uuid.UUID, bool, *string, *string, bool, json.RawMessage) (json.RawMessage, error)
	Delete(context.Context, uuid.UUID, string, uuid.UUID) error
}

type workspaceQuickLinksSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceQuickLinksService struct {
	repository workspaceQuickLinksRepository
	sessions   workspaceQuickLinksSessions
}

func NewWorkspaceQuickLinks(repository workspaceQuickLinksRepository, sessions workspaceQuickLinksSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("workspace quick links dependencies are required")
	}
	return &workspaceQuickLinksService{repository: repository, sessions: sessions}, nil
}

func (s *workspaceQuickLinksService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := s.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &api.WorkspaceQuickLinkUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &api.WorkspaceQuickLinkUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, workspaceQuickLinksPrincipal{}, userID), nil
}

func (s *workspaceQuickLinksService) authorize(ctx context.Context, slug string) (uuid.UUID, error) {
	userID, ok := ctx.Value(workspaceQuickLinksPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.WorkspaceQuickLinkUnauthenticated{Detail: unauthenticatedDetail}
	}
	authorized, err := s.repository.Authorized(ctx, userID, slug)
	if err != nil {
		return uuid.Nil(), quickLinkInternalError()
	}
	if !authorized {
		return uuid.Nil(), &api.WorkspaceQuickLinkForbidden{Detail: "You do not have the required permissions."}
	}
	return userID, nil
}

func (s *workspaceQuickLinksService) List(ctx context.Context, payload *api.ListPayload) ([]*api.WorkspaceQuickLink, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.List(ctx, userID, payload.Slug)
	if err != nil {
		return nil, quickLinkInternalError()
	}
	var result []*api.WorkspaceQuickLink
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, quickLinkInternalError()
	}
	return result, nil
}

func (s *workspaceQuickLinksService) Create(ctx context.Context, payload *api.CreatePayload) (*api.WorkspaceQuickLink, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	normalizedURL, valid := normalizeQuickLinkURL(payload.URL)
	if !valid {
		return nil, quickLinkURLValidationError()
	}
	var title *string
	if value, ok := payload.Title.Value(); ok {
		title = &value
	}
	metadata := json.RawMessage("{}")
	if payload.Metadata.Kind() == 'n' {
		return nil, &api.WorkspaceQuickLinkValidation{Metadata: []string{"This field may not be null."}}
	}
	if len(payload.Metadata) > 0 {
		metadata, err = pythoncompat.NormalizeJSONNumbers(payload.Metadata)
		if err != nil {
			return nil, &api.WorkspaceQuickLinkValidation{Metadata: []string{"Value must be valid JSON."}}
		}
	}
	body, err := s.repository.Create(ctx, userID, payload.Slug, normalizedURL, title, metadata)
	if err != nil {
		return nil, quickLinkInternalError()
	}
	return decodeQuickLink(body)
}

func (s *workspaceQuickLinksService) Retrieve(ctx context.Context, payload *api.RetrievePayload) (*api.WorkspaceQuickLink, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	linkID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, quickLinkRetrieveNotFound()
	}
	body, err := s.repository.Get(ctx, userID, payload.Slug, linkID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, quickLinkRetrieveNotFound()
	}
	if err != nil {
		return nil, quickLinkInternalError()
	}
	return decodeQuickLink(body)
}

func (s *workspaceQuickLinksService) Patch(ctx context.Context, payload *api.PatchPayload) (*api.WorkspaceQuickLink, error) {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return nil, err
	}
	linkID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, quickLinkPatchNotFound()
	}
	var normalizedURL *string
	if payload.URL != nil {
		value, valid := normalizeQuickLinkURL(*payload.URL)
		if !valid {
			return nil, quickLinkURLValidationError()
		}
		normalizedURL = &value
	}
	var title *string
	if value, ok := payload.Title.Value(); ok {
		title = &value
	}
	metadata := json.RawMessage(nil)
	if payload.Metadata.Kind() == 'n' {
		return nil, &api.WorkspaceQuickLinkValidation{Metadata: []string{"This field may not be null."}}
	}
	if len(payload.Metadata) > 0 {
		metadata, err = pythoncompat.NormalizeJSONNumbers(payload.Metadata)
		if err != nil {
			if _, lookupErr := s.repository.Get(ctx, userID, payload.Slug, linkID); lookupErr != nil {
				if errors.Is(lookupErr, store.ErrNotFound) {
					return nil, quickLinkPatchNotFound()
				}
				return nil, quickLinkInternalError()
			}
			return nil, &api.WorkspaceQuickLinkValidation{Metadata: []string{"Value must be valid JSON."}}
		}
	}
	body, err := s.repository.Update(ctx, userID, payload.Slug, linkID, payload.Title.Present(), title, normalizedURL, len(payload.Metadata) > 0, metadata)
	if errors.Is(err, store.ErrNotFound) {
		return nil, quickLinkPatchNotFound()
	}
	if err != nil {
		return nil, quickLinkInternalError()
	}
	return decodeQuickLink(body)
}

func (s *workspaceQuickLinksService) Delete(ctx context.Context, payload *api.DeletePayload) error {
	userID, err := s.authorize(ctx, payload.Slug)
	if err != nil {
		return err
	}
	linkID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return quickLinkDeleteNotFound()
	}
	err = s.repository.Delete(ctx, userID, payload.Slug, linkID)
	if errors.Is(err, store.ErrNotFound) {
		return quickLinkDeleteNotFound()
	}
	if err != nil {
		return quickLinkInternalError()
	}
	return nil
}

func normalizeQuickLinkURL(value string) (string, bool) {
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "http://" + value
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Hostname() == "" || strings.ContainsAny(parsed.Host, " \t\r\n") {
		return "", false
	}
	return value, true
}

func decodeQuickLink(body json.RawMessage) (*api.WorkspaceQuickLink, error) {
	var result api.WorkspaceQuickLink
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, quickLinkInternalError()
	}
	return &result, nil
}

func quickLinkURLValidationError() error {
	return &api.WorkspaceQuickLinkValidation{URL: &api.WorkspaceQuickLinkURLValidationDetail{Message: "Invalid URL format."}}
}

func quickLinkRetrieveNotFound() error {
	return &api.WorkspaceQuickLinkRetrieveNotFound{Message: "Quick link not found."}
}

func quickLinkPatchNotFound() error {
	return &api.WorkspaceQuickLinkPatchNotFound{Detail: "Quick link not found."}
}

func quickLinkDeleteNotFound() error {
	return &api.WorkspaceQuickLinkDeleteNotFound{Message: "The required object does not exist."}
}

func quickLinkInternalError() error {
	return &api.WorkspaceQuickLinkInternal{Message: "Something went wrong please try again later"}
}

type workspaceQuickLinksPrincipal struct{}

var _ workspaceQuickLinksRepository = (*store.Repository)(nil)
