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
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	store "github.com/CaliLuke/go-plane/internal/workspaceuserinsights"
	api "github.com/CaliLuke/go-plane/workspaceuserinsights/gen/workspace_user_insights"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	csv "github.com/JensRantil/go-csv"
	"github.com/google/uuid"
)

type workspaceUserInsightsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type workspaceUserInsightsRepository interface {
	AuthorizeWorkspace(context.Context, string, string, bool) error
	Profile(context.Context, string, string, string) (json.RawMessage, error)
	Activity(context.Context, string, string, string, []string, djangoapi.Query) (json.RawMessage, error)
	Stats(context.Context, string, string, string, map[string]string) (json.RawMessage, error)
	ExportActivity(context.Context, string, string, string, string) ([]store.ActivityCSVRow, error)
	CanExportActivity(context.Context, string, string) (bool, error)
}

type workspaceUserInsightsActor struct{ userID, timezone string }
type workspaceUserInsightsActorKey struct{}

type workspaceUserInsightsService struct {
	repo     workspaceUserInsightsRepository
	sessions workspaceUserInsightsSessions
}

func NewWorkspaceUserInsights(repo workspaceUserInsightsRepository, sessions workspaceUserInsightsSessions) (api.Service, error) {
	if repo == nil || sessions == nil {
		return nil, fmt.Errorf("workspace user insights dependencies are required")
	}
	return &workspaceUserInsightsService{repo: repo, sessions: sessions}, nil
}

func (s *workspaceUserInsightsService) APIKeyAuth(ctx context.Context, credential string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, sessionErr := serviceSessionIdentityFromRequest(ctx, credential, s.sessions)
	if sessionErr != nil {
		userID, timezone = "", ""
	}
	if _, err := uuid.Parse(userID); err != nil {
		return ctx, s.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	return context.WithValue(ctx, workspaceUserInsightsActorKey{}, workspaceUserInsightsActor{userID, timezone}), nil
}

func (s *workspaceUserInsightsService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	ctx, err := s.APIKeyAuth(ctx, credential, nil)
	if status, body := workspaceUserInsightsProtocolError(err); status != 0 {
		return status, body
	}
	actor, _ := ctx.Value(workspaceUserInsightsActorKey{}).(workspaceUserInsightsActor)
	if err := workspaceUserInsightsValidateTimezone(actor.timezone); err != nil {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	return 0, nil
}

func (s *workspaceUserInsightsService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, resource SessionProtocolResource) (int, map[string]any) {
	ctx, err := s.APIKeyAuth(ctx, credential, nil)
	if status, body := workspaceUserInsightsProtocolError(err); status != 0 {
		return status, body
	}
	actor, ok := ctx.Value(workspaceUserInsightsActorKey{}).(workspaceUserInsightsActor)
	if !ok {
		return http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail}
	}
	if strings.Contains(resource.Pattern, "/user-activity/") {
		// WorkspaceEntityPermission runs before TimezoneMixin.initial. GET,
		// HEAD and OPTIONS require any active member; other methods require
		// an active Admin or Member before method dispatch.
		requireWriteRole := resource.Method != http.MethodGet && resource.Method != http.MethodHead && resource.Method != http.MethodOptions
		if err := s.repo.AuthorizeWorkspace(ctx, actor.userID, resource.Slug, requireWriteRole); err != nil {
			return workspaceUserInsightsPermissionError(err)
		}
	}
	if err := workspaceUserInsightsValidateTimezone(actor.timezone); err != nil {
		return workspaceUserInsightsProtocolError(s.error(500, map[string]any{"error": "Something went wrong please try again later"}))
	}
	return 0, nil
}

func workspaceUserInsightsPermissionError(err error) (int, map[string]any) {
	if errors.Is(err, store.ErrForbidden) {
		return http.StatusForbidden, map[string]any{"detail": "You do not have permission to perform this action."}
	}
	return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
}

func workspaceUserInsightsValidateTimezone(timezone string) error {
	if err := djangosession.ValidateUserTimezone(timezone); err != nil {
		return fmt.Errorf("invalid user timezone: %w", err)
	}
	return nil
}

func workspaceUserInsightsProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.WorkspaceUserInsightsErrorResponse
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

func (s *workspaceUserInsightsService) error(status int, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		raw = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(raw))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.WorkspaceUserInsightsErrorResponse{Name: "status_" + strconv.Itoa(status), Body: value}
}

func (s *workspaceUserInsightsService) actor(ctx context.Context) (workspaceUserInsightsActor, error) {
	actor, ok := ctx.Value(workspaceUserInsightsActorKey{}).(workspaceUserInsightsActor)
	if !ok {
		return actor, s.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	return actor, nil
}

func workspaceUserInsightsJSON(raw json.RawMessage, contentType string) (*api.WorkspaceUserInsightsRawJSON, io.ReadCloser, error) {
	if !json.Valid(raw) {
		return nil, nil, fmt.Errorf("invalid workspace user insights JSON")
	}
	return &api.WorkspaceUserInsightsRawJSON{ContentType: contentType}, io.NopCloser(bytes.NewReader(raw)), nil
}

func (s *workspaceUserInsightsService) Profile(ctx context.Context, p *api.ProfilePayload) (*api.WorkspaceUserInsightsRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	if workspaceUserErr := workspaceUserInsightsValidateTimezone(actor.timezone); workspaceUserErr != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	body, err := s.repo.Profile(ctx, actor.userID, p.Slug, p.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, s.error(404, map[string]any{"error": "The required object does not exist."})
	}
	if err != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return workspaceUserInsightsJSON(body, "application/json")
}

func (s *workspaceUserInsightsService) Activity(ctx context.Context, p *api.ActivityPayload) (*api.WorkspaceUserInsightsRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	if authorizeWorkspaceErr := s.repo.AuthorizeWorkspace(ctx, actor.userID, p.Slug, false); authorizeWorkspaceErr != nil {
		status, response := workspaceUserInsightsPermissionError(authorizeWorkspaceErr)
		return nil, nil, s.error(status, response)
	}
	if workspaceUserErr := workspaceUserInsightsValidateTimezone(actor.timezone); workspaceUserErr != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	params := map[string]string{}
	present := map[string]bool{}
	for name, value := range map[string]*string{"order_by": p.OrderBy, "per_page": p.PerPage, "cursor": p.Cursor} {
		if value != nil {
			params[name] = *value
			present[name] = true
		}
	}
	projects, hasProjects := ReadServiceQueryValues(ctx)["project"]
	if !hasProjects && p.Project != nil {
		projects = []string{*p.Project}
	}
	for _, project := range projects {
		if _, parseErr := djangoinput.UUID(project); parseErr != nil {
			return nil, nil, s.error(400, map[string]any{"error": "Please provide valid detail"})
		}
	}
	body, err := s.repo.Activity(ctx, actor.userID, p.Slug, p.UserID, projects, djangoapi.Query{Params: params, Present: present})
	if errors.Is(err, store.ErrForbidden) {
		return nil, nil, s.error(403, map[string]any{"error": "You don't have the required permissions."})
	}
	var httpErr *djangoapi.HTTPError
	if errors.As(err, &httpErr) {
		return nil, nil, s.error(httpErr.Status, httpErr.Body)
	}
	if err != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return workspaceUserInsightsJSON(body, "application/json")
}

func (s *workspaceUserInsightsService) Stats(ctx context.Context, p *api.StatsPayload) (*api.WorkspaceUserInsightsRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	if workspaceUserErr := workspaceUserInsightsValidateTimezone(actor.timezone); workspaceUserErr != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	filters := map[string]string{}
	for name, value := range map[string]*string{"state": p.State, "state_group": p.StateGroup, "estimate_point": p.EstimatePoint, "priority": p.Priority, "parent": p.Parent, "labels": p.Labels, "assignees": p.Assignees, "mentions": p.Mentions, "created_by": p.CreatedBy, "logged_by": p.LoggedBy, "name": p.Name, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt, "start_date": p.StartDate, "target_date": p.TargetDate, "completed_at": p.CompletedAt, "type": p.Type, "project": p.Project, "cycle": p.Cycle, "module": p.Module, "intake_status": p.IntakeStatus, "inbox_status": p.InboxStatus, "sub_issue": p.SubIssue, "subscriber": p.Subscriber, "start_target_date": p.StartTargetDate} {
		if value != nil {
			filters[name] = *value
		}
	}
	body, err := s.repo.Stats(ctx, actor.userID, p.UserID, p.Slug, filters)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, s.error(404, map[string]any{"error": "The required object does not exist."})
	}
	if err != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return workspaceUserInsightsJSON(body, "application/json")
}

func (s *workspaceUserInsightsService) ExportActivity(ctx context.Context, p *api.ExportActivityPayload, body io.ReadCloser) (*api.WorkspaceUserInsightsCSV, io.ReadCloser, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	allowed, err := s.repo.CanExportActivity(ctx, actor.userID, p.Slug)
	if err != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if !allowed {
		return nil, nil, s.error(403, map[string]any{"detail": "You do not have permission to perform this action."})
	}
	if workspaceUserErr := workspaceUserInsightsValidateTimezone(actor.timezone); workspaceUserErr != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	contentType := ""
	defer func() {
		if body != nil {
			_ = body.Close()
		}
	}()
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		var httpErr *djangoapi.HTTPError
		if errors.As(err, &httpErr) {
			return nil, nil, s.error(httpErr.Status, httpErr.Body)
		}
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if input.Kind != "" {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	date, err := workspaceExportStringOrFalse(input, "date")
	if err != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	if date == "" {
		return nil, nil, s.error(400, map[string]any{"error": "Date is required"})
	}
	rows, err := s.repo.ExportActivity(ctx, actor.userID, p.Slug, p.UserID, date)
	if errors.Is(err, store.ErrForbidden) {
		return nil, nil, s.error(403, map[string]any{"error": "You don't have the required permissions."})
	}
	if err != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	data, err := encodeWorkspaceActivityCSV(rows)
	if err != nil {
		return nil, nil, s.error(500, map[string]any{"error": "Something went wrong please try again later"})
	}
	return &api.WorkspaceUserInsightsCSV{ContentDisposition: `attachment; filename="workspace-user-activity.csv"`}, io.NopCloser(strings.NewReader(data)), nil
}

func encodeWorkspaceActivityCSV(rows []store.ActivityCSVRow) (string, error) {
	var out strings.Builder
	// Python's source contract uses QUOTE_ALL. encoding/csv cannot configure this
	// dialect, so the CSV library owns quoting, escaping and record termination.
	writer := csv.NewDialectWriter(&out, csv.Dialect{Quoting: csv.QuoteAll, LineTerminator: "\r\n"})
	if err := writer.Write([]string{"Actor name", "Issue ID", "Project", "Created at", "Updated at", "Action", "Field", "Old value", "New value"}); err != nil {
		return "", err
	}
	for _, row := range rows {
		values := []string{row.Actor, row.IssueID, row.Project, csvActivityValue(row.CreatedAt), csvActivityValue(row.UpdatedAt), row.Verb, row.Field, row.OldValue, row.NewValue}
		for i, value := range values {
			// Retain Plane's spreadsheet formula-injection policy before encoding.
			if len(value) > 0 && strings.ContainsRune("=+-@\t\r\n", rune(value[0])) {
				values[i] = "'" + value
			}
		}
		if err := writer.Write(values); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func csvActivityValue(value any) string {
	stamp, ok := value.(time.Time)
	if !ok {
		if value == nil {
			return ""
		}
		return fmt.Sprint(value)
	}
	return pythoncompat.PythonDateTimeString(stamp)
}

var _ api.Service = (*workspaceUserInsightsService)(nil)
