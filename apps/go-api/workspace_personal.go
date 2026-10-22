package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	api "github.com/CaliLuke/go-plane/workspacepersonal/gen/personal_dashboard"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type personalDashboardSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type personalDashboardRepository interface {
	Dashboard(context.Context, string, string, string) (json.RawMessage, error)
	ActivityGraph(context.Context, string, string) (json.RawMessage, error)
	CompletedGraph(context.Context, string, string, string) (json.RawMessage, error)
}
type personalDashboardActorKey struct{}
type personalDashboardActor struct{ id string }
type personalDashboardService struct {
	repository personalDashboardRepository
	sessions   personalDashboardSessions
}

func NewPersonalDashboard(repository personalDashboardRepository, sessions personalDashboardSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("personal dashboard repository and sessions are required")
	}
	return &personalDashboardService{repository: repository, sessions: sessions}, nil
}

func (s *personalDashboardService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
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
	if _, err := uuid.Parse(userID); userID == "" || err != nil {
		return ctx, s.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	if err := djangosession.ValidateUserTimezone(timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return ctx, s.error(http.StatusBadRequest, map[string]any{"error": "The required key does not exist."})
		}
		return ctx, s.error(http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"})
	}
	return context.WithValue(ctx, personalDashboardActorKey{}, personalDashboardActor{id: userID}), nil
}

func (s *personalDashboardService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, nil)
	return personalDashboardProtocolError(err)
}

func (s *personalDashboardService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, _ SessionProtocolResource) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, nil)
	return personalDashboardProtocolError(err)
}

func personalDashboardProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.PersonalDashboardErrorResponse
	if !errors.As(err, &failure) {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	code, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	raw, _ := json.Marshal(failure.Body)
	if json.Unmarshal(raw, &body) != nil || body == nil {
		body = map[string]any{}
	}
	return code, body
}

func (s *personalDashboardService) error(status int, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		encoded = []byte(`{"detail":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(encoded))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.PersonalDashboardErrorResponse{Name: "status_" + strconv.Itoa(status), Body: value}
}
func (s *personalDashboardService) internal() error {
	return s.error(500, map[string]any{"error": "Something went wrong please try again later"})
}
func personalDashboardRaw(data json.RawMessage) (*api.PersonalDashboardRawJSON, io.ReadCloser, error) {
	if !json.Valid(data) {
		return nil, nil, fmt.Errorf("invalid dashboard JSON")
	}
	return &api.PersonalDashboardRawJSON{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(data)), nil
}
func (s *personalDashboardService) actor(ctx context.Context) (personalDashboardActor, error) {
	actor, ok := ctx.Value(personalDashboardActorKey{}).(personalDashboardActor)
	if !ok {
		return actor, s.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	return actor, nil
}
func (s *personalDashboardService) ActivityGraph(ctx context.Context, p *api.ActivityGraphPayload) (*api.PersonalDashboardRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.repository.ActivityGraph(ctx, actor.id, p.Slug)
	if err != nil {
		return nil, nil, s.internal()
	}
	return personalDashboardRaw(body)
}
func (s *personalDashboardService) Dashboard(ctx context.Context, p *api.DashboardPayload) (*api.PersonalDashboardRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	month := "1"
	if p.Month != nil {
		parsed, err := pythoncompat.DecimalInteger(*p.Month)
		if err != nil {
			return nil, nil, s.error(http.StatusBadRequest, map[string]any{"error": "Invalid month."})
		}
		month = parsed.String()
	}
	body, err := s.repository.Dashboard(ctx, actor.id, p.Slug, month)
	if err != nil {
		return nil, nil, s.internal()
	}
	return personalDashboardRaw(body)
}
func (s *personalDashboardService) IssuesCompletedGraph(ctx context.Context, p *api.IssuesCompletedGraphPayload) (*api.PersonalDashboardRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	month := "1"
	if p.Month != nil {
		parsed, err := pythoncompat.DecimalInteger(*p.Month)
		if err != nil {
			return nil, nil, s.error(http.StatusBadRequest, map[string]any{"error": "Invalid month."})
		}
		month = parsed.String()
	}
	body, err := s.repository.CompletedGraph(ctx, actor.id, p.Slug, month)
	if err != nil {
		return nil, nil, s.internal()
	}
	return personalDashboardRaw(body)
}

var _ api.Service = (*personalDashboardService)(nil)
