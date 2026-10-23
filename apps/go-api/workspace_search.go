package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	store "github.com/CaliLuke/go-plane/internal/workspacesearch"
	api "github.com/CaliLuke/go-plane/workspacesearch/gen/workspace_search"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type workspaceSearchSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type workspaceSearchRepository interface {
	WorkspaceAllowed(context.Context, uuid.UUID, string) (bool, error)
	Global(context.Context, uuid.UUID, string, string, string, string, string) (json.RawMessage, error)
	Entity(context.Context, uuid.UUID, string, string, string, string, int) (json.RawMessage, error)
}
type workspaceSearchActorKey struct{}
type workspaceSearchService struct {
	repository workspaceSearchRepository
	sessions   workspaceSearchSessions
}
type workspaceSearchActor struct {
	id       uuid.UUID
	timezone string
}

func NewWorkspaceSearch(repository workspaceSearchRepository, sessions workspaceSearchSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("workspace search repository and sessions are required")
	}
	return &workspaceSearchService{repository: repository, sessions: sessions}, nil
}
func (s *workspaceSearchService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		if key == "" {
			key = sessionCredentialFromContext(ctx)
		}
		principal, err := s.sessions.Validate(ctx, key)
		if err == nil {
			userID, timezone = principal.UserID, principal.UserTimezone
		}
	}
	id, err := uuid.Parse(userID)
	if userID == "" || err != nil {
		return ctx, s.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	return context.WithValue(ctx, workspaceSearchActorKey{}, workspaceSearchActor{id: id, timezone: timezone}), nil
}
func (s *workspaceSearchService) AuthorizeSessionProtocol(ctx context.Context, key string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, key, nil)
	return workspaceSearchProtocolError(err)
}
func (s *workspaceSearchService) AuthorizeSessionResourceProtocol(ctx context.Context, key string, resource SessionProtocolResource) (int, map[string]any) {
	ctx, err := s.APIKeyAuth(ctx, key, nil)
	if status, body := workspaceSearchProtocolError(err); status != 0 {
		return status, body
	}
	if resource.Pattern != "/api/workspaces/{slug}/entity-search/" {
		return 0, nil
	}
	actor, ok := ctx.Value(workspaceSearchActorKey{}).(workspaceSearchActor)
	if !ok {
		return 401, map[string]any{"detail": unauthenticatedDetail}
	}
	allowed, err := s.repository.WorkspaceAllowed(ctx, actor.id, resource.Slug)
	if err != nil {
		return 500, map[string]any{"detail": "Something went wrong please try again later"}
	}
	if !allowed {
		return 403, map[string]any{"detail": "You do not have permission to perform this action."}
	}
	return workspaceSearchTimezoneError(actor.timezone)
}

func workspaceSearchTimezoneError(timezone string) (int, map[string]any) {
	if err := djangosession.ValidateUserTimezone(timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return 400, map[string]any{"error": "The required key does not exist."}
		}
		return 500, map[string]any{"detail": "Something went wrong please try again later"}
	}
	return 0, nil
}
func (s *workspaceSearchService) actor(ctx context.Context, slug string, workspacePermission bool) (workspaceSearchActor, error) {
	actor, ok := ctx.Value(workspaceSearchActorKey{}).(workspaceSearchActor)
	if !ok {
		return actor, s.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	if workspacePermission {
		allowed, err := s.repository.WorkspaceAllowed(ctx, actor.id, slug)
		if err != nil {
			return actor, s.internal()
		}
		if !allowed {
			return actor, s.error(403, map[string]any{"detail": "You do not have permission to perform this action."})
		}
		if status, body := workspaceSearchTimezoneError(actor.timezone); status != 0 {
			return actor, s.error(status, body)
		}
	}
	return actor, nil
}
func (s *workspaceSearchService) GlobalSearch(ctx context.Context, p *api.GlobalSearchPayload) (*api.WorkspaceSearchRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx, p.Slug, false)
	if err != nil {
		return nil, nil, err
	}
	projectID := value(p.ProjectID, "")
	workspaceSearch := value(p.WorkspaceSearch, "false")
	if projectID != "" && workspaceSearch == "false" {
		if _, parseErr := djangoinput.UUID(projectID); parseErr != nil {
			return nil, nil, s.error(400, map[string]any{"error": "Please provide valid detail"})
		}
	}
	body, err := s.repository.Global(ctx, actor.id, p.Slug, value(p.Search, ""), value(p.Entities, ""), projectID, workspaceSearch)
	if err != nil {
		return nil, nil, s.internal()
	}
	return workspaceSearchRaw(body)
}
func (s *workspaceSearchService) EntitySearch(ctx context.Context, p *api.EntitySearchPayload) (*api.WorkspaceSearchRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx, p.Slug, true)
	if err != nil {
		return nil, nil, err
	}
	count := 5
	if p.Count != nil {
		parsed, parseErr := pythoncompat.DecimalInteger(*p.Count)
		if parseErr != nil || !parsed.IsInt64() || parsed.Sign() < 0 {
			return nil, nil, s.error(400, map[string]any{"count": []string{"A valid non-negative integer is required."}})
		}
		count = int(parsed.Int64())
	}
	projectID := value(p.ProjectID, "")
	if projectID != "" {
		if _, parseErr := djangoinput.UUID(projectID); parseErr != nil {
			return nil, nil, s.error(400, map[string]any{"error": "Please provide valid detail"})
		}
	}
	body, err := s.repository.Entity(ctx, actor.id, p.Slug, value(p.Query, ""), value(p.QueryType, "user_mention"), projectID, count)
	if err != nil {
		slog.ErrorContext(ctx, "workspace search entity query failed", "error", err)
		return nil, nil, s.internal()
	}
	return workspaceSearchRaw(body)
}
func (s *workspaceSearchService) error(status int, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		encoded = []byte(`{"detail":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(encoded))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.WorkspaceSearchErrorResponse{Name: "status_" + strconv.Itoa(status), Body: value}
}
func (s *workspaceSearchService) internal() error {
	return s.error(500, map[string]any{"detail": "Something went wrong please try again later"})
}
func workspaceSearchProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.WorkspaceSearchErrorResponse
	if !errors.As(err, &failure) {
		return 500, map[string]any{"detail": "Something went wrong please try again later"}
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	encoded, _ := json.Marshal(failure.Body)
	var body map[string]any
	if json.Unmarshal(encoded, &body) != nil || body == nil {
		body = map[string]any{}
	}
	if status == 0 {
		status = 500
	}
	return status, body
}
func workspaceSearchRaw(body json.RawMessage) (*api.WorkspaceSearchRawJSON, io.ReadCloser, error) {
	if !json.Valid(body) {
		return nil, nil, fmt.Errorf("invalid workspace search JSON")
	}
	return &api.WorkspaceSearchRawJSON{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(body)), nil
}
func value(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

var _ = store.ErrForbidden
