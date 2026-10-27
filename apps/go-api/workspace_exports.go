package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	store "github.com/CaliLuke/go-plane/internal/workspaceexports"
	api "github.com/CaliLuke/go-plane/workspaceexports/gen/workspace_exports"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type workspaceExportsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type workspaceExportsPublisher interface {
	Task(context.Context, string, []any, map[string]any) error
}
type workspaceExportsRepository interface {
	Authorized(context.Context, string, string) (bool, error)
	UserEmail(context.Context, string) (string, error)
	CreateIssueExport(context.Context, string, string, string, []any, bool) (json.RawMessage, string, string, []any, error)
	History(context.Context, string, string, djangoapi.Query) (json.RawMessage, error)
}
type workspaceExportsPrincipal struct{ userID, email, timezone string }
type workspaceExportsPrincipalKey struct{}
type workspaceExportsService struct {
	repo      workspaceExportsRepository
	sessions  workspaceExportsSessions
	publisher workspaceExportsPublisher
}

func NewWorkspaceExports(repo workspaceExportsRepository, sessions workspaceExportsSessions, publisher workspaceExportsPublisher) (api.Service, error) {
	if repo == nil || sessions == nil || publisher == nil {
		return nil, fmt.Errorf("workspace exports dependencies are required")
	}
	return &workspaceExportsService{repo: repo, sessions: sessions, publisher: publisher}, nil
}
func (s *workspaceExportsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, tz, sessionErr := serviceSessionIdentityFromRequest(ctx, key, s.sessions)
	email := ""
	if sessionErr != nil {
		userID, tz = "", ""
	}
	if _, err := uuid.Parse(userID); userID == "" || err != nil {
		return ctx, s.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	if err := djangosession.ValidateUserTimezone(tz); err != nil {
		return ctx, s.internal()
	}
	return context.WithValue(ctx, workspaceExportsPrincipalKey{}, workspaceExportsPrincipal{userID: userID, email: email, timezone: tz}), nil
}
func (s *workspaceExportsService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, nil)
	return workspaceExportsProtocolError(err)
}
func (s *workspaceExportsService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, _ SessionProtocolResource) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, nil)
	return workspaceExportsProtocolError(err)
}
func workspaceExportsProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.WorkspaceExportsErrorResponse
	if !errors.As(err, &failure) {
		return 500, map[string]any{"error": "Something went wrong please try again later"}
	}
	code, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	b, _ := json.Marshal(failure.Body)
	if json.Unmarshal(b, &body) != nil || body == nil {
		body = map[string]any{}
	}
	return code, body
}
func (s *workspaceExportsService) error(code int, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		raw = []byte(`{"error":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(raw))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.WorkspaceExportsErrorResponse{Name: "status_" + strconv.Itoa(code), Body: value}
}
func (s *workspaceExportsService) internal() error {
	return s.error(500, map[string]any{"error": "Something went wrong please try again later"})
}
func (s *workspaceExportsService) actor(ctx context.Context, slug string) (workspaceExportsPrincipal, error) {
	p, ok := ctx.Value(workspaceExportsPrincipalKey{}).(workspaceExportsPrincipal)
	if !ok {
		return p, s.error(401, map[string]any{"detail": unauthenticatedDetail})
	}
	allowed, err := s.repo.Authorized(ctx, p.userID, slug)
	if err != nil {
		return p, s.internal()
	}
	if !allowed {
		return p, s.error(403, map[string]any{"error": "You don't have the required permissions."})
	}
	return p, nil
}
func workspaceExportsRaw(data json.RawMessage) (*api.WorkspaceExportsRawJSON, io.ReadCloser, error) {
	if !json.Valid(data) {
		return nil, nil, fmt.Errorf("invalid workspace export JSON")
	}
	return &api.WorkspaceExportsRawJSON{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(data)), nil
}
func exportBody(ctx context.Context, body io.ReadCloser) (djangoinput.Input, error) {
	if body != nil {
		defer body.Close()
	}
	contentType := ""
	if m, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = m.Headers().Get("Content-Type")
	}
	return djangoinput.Parse(body, contentType)
}
func (s *workspaceExportsService) GetIssueExports(ctx context.Context, p *api.GetIssueExportsPayload) (*api.WorkspaceExportsRawJSON, io.ReadCloser, error) {
	principal, err := s.actor(ctx, p.Slug)
	if err != nil {
		return nil, nil, err
	}
	if p.PerPage == nil || *p.PerPage == "" || p.Cursor == nil || *p.Cursor == "" {
		return nil, nil, s.error(400, map[string]any{"error": "per_page and cursor are required"})
	}
	params := map[string]string{"per_page": *p.PerPage, "cursor": *p.Cursor}
	present := map[string]bool{"per_page": true, "cursor": true}
	order := "-created_at"
	if p.OrderBy != nil {
		order = *p.OrderBy
	}
	params["order_by"] = order
	present["order_by"] = p.OrderBy != nil
	data, err := s.repo.History(ctx, principal.userID, p.Slug, djangoapi.Query{Params: params, Present: present})
	if err != nil {
		var httpErr *djangoapi.HTTPError
		if errors.As(err, &httpErr) {
			return nil, nil, s.error(httpErr.Status, httpErr.Body)
		}
		return nil, nil, s.internal()
	}
	return workspaceExportsRaw(data)
}
func (s *workspaceExportsService) CreateIssueExport(ctx context.Context, p *api.CreateIssueExportPayload, body io.ReadCloser) (*api.WorkspaceExportsRawJSON, io.ReadCloser, error) {
	principal, err := s.actor(ctx, p.Slug)
	if err != nil {
		closeWorkspaceExportsBody(body)
		return nil, nil, err
	}
	input, err := exportBody(ctx, body)
	if err != nil {
		return nil, nil, workspaceExportsInputError(s, err)
	}
	if input.Kind != "" {
		return nil, nil, s.internal()
	}
	providerValue, ok := input.Fields["provider"]
	if !ok {
		providerValue = false
	}
	var sourceFields map[string]json.RawMessage
	_ = json.Unmarshal(input.Raw, &sourceFields)
	provider, err := pythoncompat.String(providerValue, sourceFields["provider"])
	if err != nil {
		return nil, nil, s.internal()
	}
	if provider != "csv" && provider != "xlsx" && provider != "json" {
		return nil, nil, s.error(400, map[string]any{"error": fmt.Sprintf("Provider '%s' not found.", provider)})
	}
	projects, err := workspaceExportProjectValues(input)
	if err != nil {
		return nil, nil, s.internal()
	}
	multiple, hasMultiple := input.Fields["multiple"]
	if !hasMultiple {
		multiple = false
	}
	includeCustom, err := workspaceExportCustomProperties(input)
	if err != nil {
		return nil, nil, s.error(400, map[string]any{"error": err.Error()})
	}
	_, workspaceID, token, chosen, err := s.repo.CreateIssueExport(ctx, principal.userID, p.Slug, provider, projects, includeCustom)
	if err != nil {
		if errors.Is(err, store.ErrWorkspaceMissing) {
			return nil, nil, s.error(404, map[string]any{"error": "The required object does not exist."})
		}
		return nil, nil, s.internal()
	}
	args := map[string]any{"provider": provider, "workspace_id": workspaceID, "project_ids": chosen, "token_id": token, "multiple": multiple, "slug": p.Slug, "include_custom_properties": includeCustom}
	if err := s.publisher.Task(ctx, "plane.bgtasks.export_task.issue_export_task", nil, args); err != nil {
		return nil, nil, s.internal()
	}
	return workspaceExportsRaw(json.RawMessage(`{"message":"Once the export is ready you will be able to download it"}`))
}
func workspaceExportCustomProperties(input djangoinput.Input) (bool, error) {
	value, exists := input.Fields["include_custom_properties"]
	if !exists {
		return false, nil
	}
	include, valid := value.(bool)
	if !valid {
		return false, errors.New("include_custom_properties must be a boolean")
	}
	return include, nil
}
func workspaceExportProjectValues(input djangoinput.Input) ([]any, error) {
	value, ok := input.Fields["project"]
	if !ok || !pythoncompat.Truthy(value) {
		return nil, nil
	}
	projects, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("project must be an array")
	}
	return append([]any(nil), projects...), nil
}
func (s *workspaceExportsService) ExportAnalytics(ctx context.Context, p *api.ExportAnalyticsPayload, body io.ReadCloser) (*api.WorkspaceExportsRawJSON, io.ReadCloser, error) {
	principal, err := s.actor(ctx, p.Slug)
	if err != nil {
		closeWorkspaceExportsBody(body)
		return nil, nil, err
	}
	input, err := exportBody(ctx, body)
	if err != nil {
		return nil, nil, workspaceExportsInputError(s, err)
	}
	if input.Kind != "" {
		return nil, nil, s.internal()
	}
	request := input.Fields
	xAxis, err := workspaceExportStringOrFalse(input, "x_axis")
	if err != nil {
		return nil, nil, s.internal()
	}
	yAxis, err := workspaceExportStringOrFalse(input, "y_axis")
	if err != nil {
		return nil, nil, s.internal()
	}
	segment, err := workspaceExportStringOrFalse(input, "segment")
	if err != nil {
		return nil, nil, s.internal()
	}
	if xAxis == "" || yAxis == "" || !validAnalyticsField(xAxis) || !validAnalyticsY(yAxis) {
		return nil, nil, s.error(400, map[string]any{"error": "x-axis and y-axis dimensions are required and the values should be valid"})
	}
	if segment != "" && (!validAnalyticsField(segment) || xAxis == segment) {
		return nil, nil, s.error(400, map[string]any{"error": "Both segment and x axis cannot be same and segment should be valid"})
	}
	if principal.email == "" {
		principal.email, err = s.repo.UserEmail(ctx, principal.userID)
		if err != nil {
			return nil, nil, s.internal()
		}
	}
	if err := s.publisher.Task(ctx, "plane.bgtasks.analytic_plot_export.analytic_export_task", nil, map[string]any{"email": principal.email, "data": request, "slug": p.Slug}); err != nil {
		return nil, nil, s.internal()
	}
	return workspaceExportsRaw(json.RawMessage(fmt.Sprintf(`{"message":%s}`, strconv.Quote("Once the export is ready it will be emailed to you at "+principal.email))))
}
func closeWorkspaceExportsBody(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}
func workspaceExportsInputError(s *workspaceExportsService, err error) error {
	var httpErr *djangoapi.HTTPError
	if errors.As(err, &httpErr) {
		return s.error(httpErr.Status, httpErr.Body)
	}
	return s.internal()
}
func workspaceExportStringOrFalse(input djangoinput.Input, name string) (string, error) {
	value, ok := input.Fields[name]
	if !ok || !pythoncompat.Truthy(value) {
		return "", nil
	}
	var sourceFields map[string]json.RawMessage
	_ = json.Unmarshal(input.Raw, &sourceFields)
	return pythoncompat.String(value, sourceFields[name])
}
func validAnalyticsField(value string) bool {
	switch value {
	case "state_id", "state__group", "labels__id", "assignees__id", "estimate_point__value", "issue_cycle__cycle_id", "issue_module__module_id", "priority", "start_date", "target_date", "created_at", "completed_at":
		return true
	}
	return false
}
func validAnalyticsY(value string) bool { return value == "issue_count" || value == "estimate" }

var _ api.Service = (*workspaceExportsService)(nil)
