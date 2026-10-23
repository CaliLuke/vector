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

	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	store "github.com/CaliLuke/go-plane/internal/workspaceanalytics"
	api "github.com/CaliLuke/go-plane/workspaceanalytics/gen/workspace_analytics"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type workspaceAnalyticsSessions interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type workspaceAnalyticsRepository interface {
	Authorized(context.Context, uuid.UUID, string, string) (bool, error)
	AuthorizedProjectChart(context.Context, uuid.UUID, string, string) (bool, error)
	AuthorizedDefault(context.Context, uuid.UUID, string) (bool, error)
	Overview(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error)
	WorkItemStats(context.Context, uuid.UUID, string, string, string, string, string, string) (json.RawMessage, error)
	ProjectStats(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error)
	ProjectAssigneeStats(context.Context, uuid.UUID, string, string, string, string, string, string) (json.RawMessage, error)
	ProjectChart(context.Context, uuid.UUID, string, string, string) (json.RawMessage, error)
	SavedView(context.Context, string, uuid.UUID) (json.RawMessage, error)
	Chart(context.Context, uuid.UUID, string, string, string, string, string, string, string, string) (json.RawMessage, error)
	JSON(context.Context, string, ...any) (json.RawMessage, error)
}
type workspaceAnalyticsActor struct {
	id       uuid.UUID
	timezone string
}
type workspaceAnalyticsActorKey struct{}
type workspaceAnalyticsService struct {
	repository workspaceAnalyticsRepository
	sessions   workspaceAnalyticsSessions
}

func NewWorkspaceAnalytics(repository workspaceAnalyticsRepository, sessions workspaceAnalyticsSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, fmt.Errorf("workspace analytics repository and sessions are required")
	}
	return &workspaceAnalyticsService{repository: repository, sessions: sessions}, nil
}

func (s *workspaceAnalyticsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, sessionErr := serviceSessionIdentityFromRequest(ctx, key, s.sessions)
	if sessionErr != nil {
		userID, timezone = "", ""
	}
	id, err := uuid.Parse(userID)
	if userID == "" || err != nil {
		return ctx, s.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	if err := djangosession.ValidateUserTimezone(timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return ctx, s.error(http.StatusBadRequest, map[string]any{"error": "The required key does not exist."})
		}
		return ctx, s.error(http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"})
	}
	return context.WithValue(ctx, workspaceAnalyticsActorKey{}, workspaceAnalyticsActor{id: id, timezone: timezone}), nil
}

func (s *workspaceAnalyticsService) AuthorizeSessionProtocol(ctx context.Context, credential string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, nil)
	return workspaceAnalyticsProtocolError(err)
}
func (s *workspaceAnalyticsService) AuthorizeSessionResourceProtocol(ctx context.Context, credential string, _ SessionProtocolResource) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, credential, nil)
	return workspaceAnalyticsProtocolError(err)
}

func workspaceAnalyticsProtocolError(err error) (int, map[string]any) {
	if err == nil {
		return 0, nil
	}
	var failure *api.WorkspaceAnalyticsErrorResponse
	if !errors.As(err, &failure) {
		return http.StatusInternalServerError, map[string]any{"detail": "Something went wrong please try again later"}
	}
	status, _ := strconv.Atoi(strings.TrimPrefix(failure.Name, "status_"))
	var body map[string]any
	encoded, _ := json.Marshal(failure.Body)
	if json.Unmarshal(encoded, &body) != nil || body == nil {
		body = map[string]any{}
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	return status, body
}

func (s *workspaceAnalyticsService) actor(ctx context.Context, slug, project string) (workspaceAnalyticsActor, error) {
	actor, ok := ctx.Value(workspaceAnalyticsActorKey{}).(workspaceAnalyticsActor)
	if !ok {
		return actor, s.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	allowed, err := s.repository.Authorized(ctx, actor.id, slug, project)
	if err != nil {
		return actor, s.internal()
	}
	if !allowed {
		return actor, s.error(http.StatusForbidden, map[string]any{"error": forbiddenDetail})
	}
	return actor, nil
}

func (s *workspaceAnalyticsService) projectChartActor(ctx context.Context, slug, project string) (workspaceAnalyticsActor, error) {
	actor, ok := ctx.Value(workspaceAnalyticsActorKey{}).(workspaceAnalyticsActor)
	if !ok {
		return actor, s.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	allowed, err := s.repository.AuthorizedProjectChart(ctx, actor.id, slug, project)
	if err != nil {
		return actor, s.internal()
	}
	if !allowed {
		return actor, s.error(http.StatusForbidden, map[string]any{"error": forbiddenDetail})
	}
	return actor, nil
}

func workspaceAnalyticsRaw(data json.RawMessage) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	if !json.Valid(data) {
		return nil, nil, fmt.Errorf("invalid analytics JSON")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return nil, nil, fmt.Errorf("invalid analytics JSON: %w", err)
	}
	return &api.WorkspaceAnalyticsRawJSON{ContentType: "application/json"}, io.NopCloser(bytes.NewReader(compact.Bytes())), nil
}
func (s *workspaceAnalyticsService) respond(ctx context.Context, slug, project string, query string, args ...any) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	if _, err := s.actor(ctx, slug, project); err != nil {
		return nil, nil, err
	}
	body, err := s.repository.JSON(ctx, query, args...)
	if err != nil {
		return nil, nil, s.internal()
	}
	return workspaceAnalyticsRaw(body)
}
func (s *workspaceAnalyticsService) error(status int, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		encoded = []byte(`{"detail":"Something went wrong please try again later"}`)
	}
	value, err := loom.JSONValueFrom(json.RawMessage(encoded))
	if err != nil {
		value = loom.JSONValue(`null`)
	}
	return &api.WorkspaceAnalyticsErrorResponse{Name: "status_" + strconv.Itoa(status), Body: value}
}
func (s *workspaceAnalyticsService) internal() error {
	return s.error(500, map[string]any{"error": "Something went wrong please try again later"})
}

func (s *workspaceAnalyticsService) AdvanceAnalytics(ctx context.Context, p *api.AdvanceAnalyticsPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	actor, err := s.actor(ctx, p.Slug, "")
	if err != nil {
		return nil, nil, err
	}
	tab := analyticsStringValue(p.Tab, "overview")
	if tab != "overview" && tab != "work-items" {
		return nil, nil, s.error(400, map[string]any{"message": "Invalid tab"})
	}
	if tab == "work-items" {
		b, e := s.repository.WorkItemStats(ctx, actor.id, p.Slug, "", "", "", analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""))
		if e != nil {
			return nil, nil, s.internal()
		}
		return workspaceAnalyticsRaw(b)
	}
	b, e := s.repository.Overview(ctx, actor.id, p.Slug, analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""))
	if e != nil {
		return nil, nil, s.internal()
	}
	return workspaceAnalyticsRaw(b)
}
func (s *workspaceAnalyticsService) AdvanceAnalyticsStats(ctx context.Context, p *api.AdvanceAnalyticsStatsPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	actor, e := s.actor(ctx, p.Slug, "")
	if e != nil {
		return nil, nil, e
	}
	if typ := analyticsStringValue(p.Type, "work-items"); typ != "work-items" {
		return nil, nil, s.error(400, map[string]any{"message": "Invalid type"})
	}
	b, e := s.repository.ProjectStats(ctx, actor.id, p.Slug, analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""))
	if e != nil {
		return nil, nil, s.internal()
	}
	return workspaceAnalyticsRaw(b)
}
func (s *workspaceAnalyticsService) AdvanceAnalyticsCharts(ctx context.Context, p *api.AdvanceAnalyticsChartsPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	actor, e := s.actor(ctx, p.Slug, "")
	if e != nil {
		return nil, nil, e
	}
	typ := analyticsStringValue(p.Type, "projects")
	switch typ {
	case "projects":
		b, e := s.repository.ProjectChart(ctx, actor.id, p.Slug, analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""))
		if e != nil {
			return nil, nil, s.internal()
		}
		return workspaceAnalyticsRaw(b)
	case "work-items":
		b, e := s.repository.JSON(ctx, `WITH ws AS (SELECT id,created_at FROM workspaces WHERE slug=$1 AND deleted_at IS NULL), months AS (SELECT generate_series(GREATEST(date_trunc('month',ws.created_at)::date,COALESCE(date_trunc('month',filter_window.start_date)::date,date_trunc('month',ws.created_at)::date)),date_trunc('month',CURRENT_DATE::timestamp)::date,'1 month')::date AS month FROM ws CROSS JOIN (SELECT CASE $4 WHEN 'yesterday' THEN CURRENT_DATE-1 WHEN 'last_7_days' THEN CURRENT_DATE-7 WHEN 'last_30_days' THEN CURRENT_DATE-30 WHEN 'last_3_months' THEN CURRENT_DATE-90 ELSE NULL END AS start_date) filter_window), counts AS (SELECT date_trunc('month',i.created_at)::date AS month,count(*)::int created_count,count(*) FILTER(WHERE st."group"='completed')::int completed_count FROM issues i JOIN ws ON ws.id=i.workspace_id JOIN projects p ON p.id=i.project_id JOIN project_members pm ON pm.project_id=p.id AND pm.member_id=$2 AND pm.is_active=true AND pm.deleted_at IS NULL LEFT JOIN states st ON st.id=i.state_id WHERE p.deleted_at IS NULL AND p.archived_at IS NULL AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE AND NOT EXISTS(SELECT 1 FROM states hidden_state WHERE hidden_state.id=i.state_id AND hidden_state."group"='triage') AND ($3='' OR p.id=ANY(string_to_array($3,',')::uuid[])) AND ($4='' OR CASE $4 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE TRUE END) GROUP BY 1) SELECT json_build_object('data',COALESCE(json_agg(json_build_object('key',to_char(months.month,'YYYY-MM-DD'),'name',to_char(months.month,'YYYY-MM-DD'),'count',COALESCE(counts.created_count,0),'created_issues',COALESCE(counts.created_count,0),'completed_issues',COALESCE(counts.completed_count,0)) ORDER BY months.month),'[]'::json),'schema',json_build_object('completed_issues','completed_issues','created_issues','created_issues')) FROM months LEFT JOIN counts USING(month)`, p.Slug, actor.id, analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""))
		if e != nil {
			return nil, nil, s.internal()
		}
		return workspaceAnalyticsRaw(b)
	case "custom-work-items":
		b, e := s.repository.Chart(ctx, actor.id, p.Slug, "", analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""), analyticsStringValue(p.XAxis, "PRIORITY"), analyticsStringValue(p.GroupBy, ""), "", "")
		if e != nil {
			return nil, nil, workspaceAnalyticsChartError(s, e)
		}
		return workspaceAnalyticsRaw(b)
	default:
		return nil, nil, s.error(400, map[string]any{"message": "Invalid type"})
	}
}
func (s *workspaceAnalyticsService) ProjectAdvanceAnalytics(ctx context.Context, p *api.ProjectAdvanceAnalyticsPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	if _, e := s.actor(ctx, p.Slug, p.ProjectID); e != nil {
		return nil, nil, e
	}
	actor, _ := ctx.Value(workspaceAnalyticsActorKey{}).(workspaceAnalyticsActor)
	b, e := s.repository.WorkItemStats(ctx, actor.id, p.Slug, p.ProjectID, analyticsStringValue(p.CycleID, ""), analyticsStringValue(p.ModuleID, ""), analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""))
	if e != nil {
		return nil, nil, s.internal()
	}
	return workspaceAnalyticsRaw(b)
}
func (s *workspaceAnalyticsService) ProjectAdvanceAnalyticsStats(ctx context.Context, p *api.ProjectAdvanceAnalyticsStatsPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	if _, e := s.actor(ctx, p.Slug, p.ProjectID); e != nil {
		return nil, nil, e
	}
	if typ := analyticsStringValue(p.Type, "work-items"); typ != "work-items" {
		return nil, nil, s.error(400, map[string]any{"message": "Invalid type"})
	}
	actor, _ := ctx.Value(workspaceAnalyticsActorKey{}).(workspaceAnalyticsActor)
	b, e := s.repository.ProjectAssigneeStats(ctx, actor.id, p.Slug, p.ProjectID, analyticsStringValue(p.CycleID, ""), analyticsStringValue(p.ModuleID, ""), analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""))
	if e != nil {
		return nil, nil, s.internal()
	}
	return workspaceAnalyticsRaw(b)
}
func (s *workspaceAnalyticsService) ProjectAdvanceAnalyticsCharts(ctx context.Context, p *api.ProjectAdvanceAnalyticsChartsPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	actor, e := s.projectChartActor(ctx, p.Slug, p.ProjectID)
	if e != nil {
		return nil, nil, e
	}
	typ := analyticsStringValue(p.Type, "projects")
	if typ == "custom-work-items" {
		b, e := s.repository.Chart(ctx, actor.id, p.Slug, p.ProjectID, analyticsStringValue(p.ProjectIds, ""), analyticsStringValue(p.DateFilter, ""), analyticsStringValue(p.XAxis, "PRIORITY"), analyticsStringValue(p.GroupBy, ""), analyticsStringValue(p.CycleID, ""), analyticsStringValue(p.ModuleID, ""))
		if e != nil {
			return nil, nil, workspaceAnalyticsChartError(s, e)
		}
		return workspaceAnalyticsRaw(b)
	}
	if typ == "work-items" {
		cycleID := nullableAnalytics(p.CycleID)
		moduleID := nullableAnalytics(p.ModuleID)
		if cycleID != nil || moduleID != nil {
			b, e := s.repository.JSON(ctx, `WITH bounds AS (
 SELECT CASE WHEN $3::uuid IS NOT NULL THEN cycle.start_date::date ELSE module.start_date END AS start_day,
        CASE WHEN $3::uuid IS NOT NULL THEN cycle.end_date::date ELSE module.target_date END AS end_day
 FROM (SELECT $3::uuid AS cycle_id,$4::uuid AS module_id) request
 LEFT JOIN cycles cycle ON cycle.id=request.cycle_id AND cycle.project_id=$2::uuid AND cycle.deleted_at IS NULL
 LEFT JOIN modules module ON module.id=request.module_id AND module.project_id=$2::uuid AND module.deleted_at IS NULL
), days AS (
 SELECT generate_series(bounds.start_day,bounds.end_day,interval '1 day')::date AS day
 FROM bounds WHERE bounds.start_day IS NOT NULL AND bounds.end_day IS NOT NULL
), counts AS (
 SELECT cycle_issue.created_at::date AS day,count(*)::int AS created_count,
        count(*) FILTER(WHERE state."group"='completed')::int AS completed_count
 FROM cycle_issues cycle_issue
 JOIN issues issue ON issue.id=cycle_issue.issue_id
 JOIN projects project ON project.id=issue.project_id
 JOIN workspaces workspace ON workspace.id=project.workspace_id
 JOIN project_members member ON member.project_id=project.id AND member.member_id=$5::uuid AND member.is_active=true AND member.deleted_at IS NULL
 LEFT JOIN states state ON state.id=issue.state_id
 WHERE $3::uuid IS NOT NULL AND cycle_issue.cycle_id=$3::uuid AND cycle_issue.deleted_at IS NULL
 AND workspace.slug=$1 AND project.id=$2::uuid AND project.deleted_at IS NULL AND project.archived_at IS NULL
 AND ($6='' OR project.id=ANY(string_to_array($6,',')::uuid[]))
 GROUP BY 1
 UNION ALL
 SELECT module_issue.created_at::date AS day,count(*)::int AS created_count,
        count(*) FILTER(WHERE state."group"='completed')::int AS completed_count
 FROM module_issues module_issue
 JOIN issues issue ON issue.id=module_issue.issue_id
 JOIN projects project ON project.id=issue.project_id
 JOIN workspaces workspace ON workspace.id=project.workspace_id
 JOIN project_members member ON member.project_id=project.id AND member.member_id=$5::uuid AND member.is_active=true AND member.deleted_at IS NULL
 LEFT JOIN states state ON state.id=issue.state_id
 WHERE $3::uuid IS NULL AND $4::uuid IS NOT NULL AND module_issue.module_id=$4::uuid AND module_issue.deleted_at IS NULL
 AND workspace.slug=$1 AND project.id=$2::uuid AND project.deleted_at IS NULL AND project.archived_at IS NULL
 AND ($6='' OR project.id=ANY(string_to_array($6,',')::uuid[]))
 GROUP BY 1
)
SELECT json_build_object(
 'data',COALESCE(json_agg(json_build_object('key',to_char(days.day,'YYYY-MM-DD'),'name',to_char(days.day,'YYYY-MM-DD'),'count',COALESCE(counts.created_count,0)+COALESCE(counts.completed_count,0),'created_issues',COALESCE(counts.created_count,0),'completed_issues',COALESCE(counts.completed_count,0)) ORDER BY days.day),'[]'::json),
 'schema',CASE WHEN EXISTS(SELECT 1 FROM bounds WHERE start_day IS NOT NULL AND end_day IS NOT NULL) THEN json_build_object('completed_issues','completed_issues','created_issues','created_issues') ELSE '{}'::json END
)
FROM days LEFT JOIN counts USING(day)`, p.Slug, p.ProjectID, cycleID, moduleID, actor.id, analyticsStringValue(p.ProjectIds, ""))
			if e != nil {
				return nil, nil, s.internal()
			}
			return workspaceAnalyticsRaw(b)
		}
		b, e := s.repository.JSON(ctx, `WITH project AS (
 SELECT p.created_at
 FROM projects p JOIN workspaces w ON w.id=p.workspace_id
 WHERE w.slug=$1 AND p.id=$2::uuid AND p.deleted_at IS NULL
), months AS (
 SELECT generate_series(
   GREATEST(date_trunc('month',project.created_at)::date,COALESCE(date_trunc('month',filter_window.start_date)::date,date_trunc('month',project.created_at)::date)),
   date_trunc('month',CURRENT_DATE::timestamp)::date,'1 month'
 )::date AS month
 FROM project CROSS JOIN (SELECT CASE $5 WHEN 'yesterday' THEN CURRENT_DATE-1 WHEN 'last_7_days' THEN CURRENT_DATE-7 WHEN 'last_30_days' THEN CURRENT_DATE-30 WHEN 'last_3_months' THEN CURRENT_DATE-90 ELSE NULL END AS start_date) filter_window
), counts AS (
 SELECT date_trunc('month',i.created_at)::date AS month,count(*)::int AS created_count,
 count(*) FILTER(WHERE st."group"='completed')::int AS completed_count
 FROM issues i JOIN workspaces w ON w.id=i.workspace_id JOIN projects p ON p.id=i.project_id LEFT JOIN states st ON st.id=i.state_id
 WHERE w.slug=$1 AND i.project_id=$2::uuid AND p.deleted_at IS NULL AND p.archived_at IS NULL
 AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$6::uuid AND pm.is_active=true AND pm.deleted_at IS NULL)
 AND ($7='' OR p.id=ANY(string_to_array($7,',')::uuid[]))
 AND i.deleted_at IS NULL AND i.archived_at IS NULL AND i.is_draft IS NOT TRUE
 AND NOT EXISTS(SELECT 1 FROM states hidden_state WHERE hidden_state.id=i.state_id AND hidden_state."group"='triage')
 AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM cycle_issues ci WHERE ci.issue_id=i.id AND ci.cycle_id=$3::uuid AND ci.deleted_at IS NULL))
 AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM module_issues mi WHERE mi.issue_id=i.id AND mi.module_id=$4::uuid AND mi.deleted_at IS NULL))
 AND ($5='' OR CASE $5 WHEN 'yesterday' THEN i.created_at::date=CURRENT_DATE-1 WHEN 'last_7_days' THEN i.created_at::date BETWEEN CURRENT_DATE-7 AND CURRENT_DATE WHEN 'last_30_days' THEN i.created_at::date BETWEEN CURRENT_DATE-30 AND CURRENT_DATE WHEN 'last_3_months' THEN i.created_at::date BETWEEN CURRENT_DATE-90 AND CURRENT_DATE ELSE TRUE END)
 GROUP BY 1
)
SELECT json_build_object('data',COALESCE(json_agg(json_build_object('key',to_char(months.month,'YYYY-MM-DD'),'name',to_char(months.month,'YYYY-MM-DD'),'count',COALESCE(counts.created_count,0),'created_issues',COALESCE(counts.created_count,0),'completed_issues',COALESCE(counts.completed_count,0)) ORDER BY months.month),'[]'::json),'schema',json_build_object('completed_issues','completed_issues','created_issues','created_issues'))
FROM months LEFT JOIN counts USING(month)`, p.Slug, p.ProjectID, cycleID, moduleID, analyticsStringValue(p.DateFilter, ""), actor.id, analyticsStringValue(p.ProjectIds, ""))
		if e != nil {
			return nil, nil, s.internal()
		}
		return workspaceAnalyticsRaw(b)
	}
	return nil, nil, s.error(400, map[string]any{"message": "Invalid type"})
}

func workspaceAnalyticsChartError(service *workspaceAnalyticsService, err error) error {
	switch {
	case errors.Is(err, store.ErrInvalidChartXAxis):
		return service.error(http.StatusBadRequest, map[string]any{"message": "Invalid x_axis field"})
	case errors.Is(err, store.ErrInvalidChartGroupBy):
		return service.error(http.StatusBadRequest, map[string]any{"message": "Invalid group_by field"})
	default:
		return service.internal()
	}
}

func nullableAnalytics(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

func workspaceAnalyticsFilters(values map[string]*string) (map[string]string, map[string]bool) {
	filters := make(map[string]string, len(values))
	present := make(map[string]bool, len(values))
	for key, value := range values {
		if value == nil {
			continue
		}
		filters[key] = *value
		present[key] = true
	}
	return filters, present
}

func analyticsFilterError(service *workspaceAnalyticsService, err error) error {
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) {
		return service.error(failure.Status, failure.Body)
	}
	return service.internal()
}

func (s *workspaceAnalyticsService) Analytics(ctx context.Context, p *api.AnalyticsPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	filters, _ := workspaceAnalyticsFilters(map[string]*string{
		"state": p.State, "state_group": p.StateGroup, "estimate_point": p.EstimatePoint, "priority": p.Priority, "parent": p.Parent,
		"created_by": p.CreatedBy, "logged_by": p.LoggedBy, "name": p.Name, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		"start_date": p.StartDate, "target_date": p.TargetDate, "completed_at": p.CompletedAt, "type": p.Type, "project": p.Project,
		"labels": p.Labels, "assignees": p.Assignees, "mentions": p.Mentions, "intake_status": p.IntakeStatus,
		"inbox_status": p.InboxStatus, "cycle": p.Cycle, "module": p.Module, "subscriber": p.Subscriber,
		"sub_issue": p.SubIssue, "start_target_date": p.StartTargetDate,
	})
	return s.analytics(ctx, p.Slug, analyticsStringValue(p.XAxis, ""), analyticsStringValue(p.YAxis, ""), analyticsStringValue(p.Segment, ""), filters)
}
func (s *workspaceAnalyticsService) DefaultAnalytics(ctx context.Context, p *api.DefaultAnalyticsPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	actor, ok := ctx.Value(workspaceAnalyticsActorKey{}).(workspaceAnalyticsActor)
	if !ok {
		return nil, nil, s.error(http.StatusUnauthorized, map[string]any{"detail": unauthenticatedDetail})
	}
	allowed, e := s.repository.AuthorizedDefault(ctx, actor.id, p.Slug)
	if e != nil {
		return nil, nil, s.internal()
	}
	if !allowed {
		return nil, nil, s.error(403, map[string]any{"error": forbiddenDetail})
	}
	filters, present := workspaceAnalyticsFilters(map[string]*string{
		"state": p.State, "state_group": p.StateGroup, "estimate_point": p.EstimatePoint, "priority": p.Priority, "parent": p.Parent,
		"labels": p.Labels, "assignees": p.Assignees, "mentions": p.Mentions, "created_by": p.CreatedBy,
		"logged_by": p.LoggedBy, "name": p.Name, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		"start_date": p.StartDate, "target_date": p.TargetDate, "completed_at": p.CompletedAt, "type": p.Type,
		"project": p.Project, "cycle": p.Cycle, "module": p.Module, "intake_status": p.IntakeStatus,
		"inbox_status": p.InboxStatus, "sub_issue": p.SubIssue, "subscriber": p.Subscriber,
		"start_target_date": p.StartTargetDate,
	})
	baseWhere := []string{
		"workspace.slug=$1", "issue.deleted_at IS NULL", "issue.archived_at IS NULL",
		"issue.is_draft IS NOT TRUE", "project.deleted_at IS NULL", "project.archived_at IS NULL",
		`NOT EXISTS (SELECT 1 FROM states manager_state WHERE manager_state.id=issue.state_id AND manager_state."group"='triage')`,
	}
	where, args, e := workitems.BuildAppIssueFilters(filters, present, baseWhere, []any{p.Slug})
	if e != nil {
		return nil, nil, analyticsFilterError(s, e)
	}
	query := `WITH base AS (
	SELECT issue.id,issue.state_id,issue.created_by_id,issue.completed_at,issue.point
 FROM issues issue JOIN workspaces workspace ON workspace.id=issue.workspace_id JOIN projects project ON project.id=issue.project_id
 WHERE ` + strings.Join(where, " AND ") + `
), classified AS (SELECT s."group" state_group,count(*)::int state_count FROM base b LEFT JOIN states s ON s.id=b.state_id GROUP BY 1 ORDER BY 1),
 open_issues AS (SELECT b.*,s."group" state_group FROM base b LEFT JOIN states s ON s.id=b.state_id WHERE s."group" IN ('backlog','unstarted','started')),
 open_classified AS (SELECT state_group,count(*)::int state_count FROM open_issues GROUP BY 1 ORDER BY 1),
 monthly AS (SELECT extract(month FROM completed_at)::int month_num,count(*)::int item_count FROM base WHERE completed_at >= date_trunc('year',now()) AND completed_at < date_trunc('year',now())+interval '1 year' GROUP BY 1 ORDER BY 1),
 creators AS (SELECT json_build_object('created_by__first_name',u.first_name,'created_by__last_name',u.last_name,'created_by__display_name',u.display_name,'created_by__id',u.id,'created_by__avatar_url',COALESCE(CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/' END,u.avatar),'count',count(*)) row FROM base b JOIN users u ON u.id=b.created_by_id GROUP BY u.id ORDER BY count(*) DESC LIMIT 5),
 closed_users AS (SELECT json_build_object('assignees__first_name',u.first_name,'assignees__last_name',u.last_name,'assignees__display_name',u.display_name,'assignees__id',u.id,'assignees__avatar_url',COALESCE(CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/' END,u.avatar),'count',count(DISTINCT b.id)) row FROM base b JOIN issue_assignees ia ON ia.issue_id=b.id AND ia.deleted_at IS NULL JOIN users u ON u.id=ia.assignee_id WHERE b.completed_at IS NOT NULL GROUP BY u.id ORDER BY count(DISTINCT b.id) DESC LIMIT 5),
 pending_users AS (SELECT json_build_object('assignees__first_name',u.first_name,'assignees__last_name',u.last_name,'assignees__display_name',u.display_name,'assignees__id',u.id,'assignees__avatar_url',COALESCE(CASE WHEN u.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||u.avatar_asset_id::text||'/' END,u.avatar),'count',count(DISTINCT b.id)) row FROM base b LEFT JOIN issue_assignees ia ON ia.issue_id=b.id AND ia.deleted_at IS NULL LEFT JOIN users u ON u.id=ia.assignee_id WHERE b.completed_at IS NULL GROUP BY u.id ORDER BY count(DISTINCT b.id) DESC)

 SELECT json_build_object('total_issues',(SELECT count(*) FROM base),'total_issues_classified',COALESCE((SELECT json_agg(json_build_object('state_group',state_group,'state_count',state_count) ORDER BY state_group) FROM classified),'[]'::json),'open_issues',(SELECT count(*) FROM open_issues),'open_issues_classified',COALESCE((SELECT json_agg(json_build_object('state_group',state_group,'state_count',state_count) ORDER BY state_group) FROM open_classified),'[]'::json),'issue_completed_month_wise',COALESCE((SELECT json_agg(json_build_object('month',month_num,'count',item_count) ORDER BY month_num) FROM monthly),'[]'::json),'most_issue_created_user',COALESCE((SELECT json_agg(row) FROM creators),'[]'::json),'most_issue_closed_user',COALESCE((SELECT json_agg(row) FROM closed_users),'[]'::json),'pending_issue_user',COALESCE((SELECT json_agg(row) FROM pending_users),'[]'::json),'open_estimate_sum',(SELECT sum(point) FROM open_issues),'total_estimate_sum',(SELECT sum(point) FROM base))`
	b, e := s.repository.JSON(ctx, query, args...)
	if e != nil {
		return nil, nil, s.internal()
	}
	return workspaceAnalyticsRaw(b)
}
func (s *workspaceAnalyticsService) SavedAnalyticView(ctx context.Context, p *api.SavedAnalyticViewPayload) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	if _, e := s.actor(ctx, p.Slug, ""); e != nil {
		return nil, nil, e
	}
	id, err := uuid.Parse(p.AnalyticID)
	if err != nil {
		return nil, nil, s.error(404, map[string]any{"detail": "Not found."})
	}
	saved, err := s.repository.SavedView(ctx, p.Slug, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, s.error(404, map[string]any{"detail": "Not found."})
		}
		return nil, nil, s.internal()
	}
	var view struct {
		Query     map[string]any `json:"query"`
		QueryDict struct {
			XAxis string `json:"x_axis"`
			YAxis string `json:"y_axis"`
		} `json:"query_dict"`
	}
	if err := json.Unmarshal(saved, &view); err != nil {
		return nil, nil, s.internal()
	}
	filters := map[string]string{}
	for key, value := range view.Query {
		switch typed := value.(type) {
		case string:
			filters[key] = typed
		case json.Number:
			filters[key] = typed.String()
		}
	}
	return s.analyticsWithProjection(ctx, p.Slug, view.QueryDict.XAxis, view.QueryDict.YAxis, analyticsStringValue(p.Segment, ""), filters, false)
}
func (s *workspaceAnalyticsService) analytics(ctx context.Context, slug, x, y, segment string, filters map[string]string) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	return s.analyticsWithProjection(ctx, slug, x, y, segment, filters, true)
}
func (s *workspaceAnalyticsService) analyticsWithProjection(ctx context.Context, slug, x, y, segment string, filters map[string]string, includeExtras bool) (*api.WorkspaceAnalyticsRawJSON, io.ReadCloser, error) {
	if _, e := s.actor(ctx, slug, ""); e != nil {
		return nil, nil, e
	}
	valid := map[string]string{
		"state_id": "issue.state_id::text", "state__group": `state."group"`,
		"labels__id": "label_link.label_id::text", "assignees__id": "assignee_link.assignee_id::text",
		"estimate_point__value": "estimate_point.value::text", "issue_cycle__cycle_id": "cycle_link.cycle_id::text",
		"issue_module__module_id": "module_link.module_id::text", "priority": "issue.priority",
		"start_date":   "extract(year FROM issue.start_date)::int::text||'-'||extract(month FROM issue.start_date)::int::text",
		"target_date":  "extract(year FROM issue.target_date)::int::text||'-'||extract(month FROM issue.target_date)::int::text",
		"created_at":   "extract(year FROM issue.created_at)::int::text||'-'||extract(month FROM issue.created_at)::int::text",
		"completed_at": "extract(year FROM issue.completed_at)::int::text||'-'||extract(month FROM issue.completed_at)::int::text",
	}
	if x == "" || y != "issue_count" && y != "estimate" || valid[x] == "" {
		return nil, nil, s.error(400, map[string]any{"error": "x-axis and y-axis dimensions are required and the values should be valid"})
	}
	if segment != "" && (valid[segment] == "" || segment == x) {
		return nil, nil, s.error(400, map[string]any{"error": "Both segment and x axis cannot be same and segment should be valid"})
	}
	metric := "count(*)::double precision"
	if y == "estimate" {
		metric = "sum(estimate_point.value::double precision)"
	}
	segexpr := "NULL::text"
	if segment != "" {
		segexpr = valid[segment]
	}
	joins := ` LEFT JOIN states state ON state.id=issue.state_id LEFT JOIN estimate_points estimate_point ON estimate_point.id=issue.estimate_point_id`
	if x == "labels__id" || segment == "labels__id" {
		joins += ` LEFT JOIN issue_labels label_link ON label_link.issue_id=issue.id AND label_link.deleted_at IS NULL LEFT JOIN labels label ON label.id=label_link.label_id`
	}
	if x == "assignees__id" || segment == "assignees__id" {
		joins += ` LEFT JOIN issue_assignees assignee_link ON assignee_link.issue_id=issue.id AND assignee_link.deleted_at IS NULL LEFT JOIN users assignee ON assignee.id=assignee_link.assignee_id`
	}
	if x == "issue_cycle__cycle_id" || segment == "issue_cycle__cycle_id" {
		joins += ` LEFT JOIN cycle_issues cycle_link ON cycle_link.issue_id=issue.id AND cycle_link.deleted_at IS NULL LEFT JOIN cycles cycle ON cycle.id=cycle_link.cycle_id`
	}
	if x == "issue_module__module_id" || segment == "issue_module__module_id" {
		joins += ` LEFT JOIN module_issues module_link ON module_link.issue_id=issue.id AND module_link.deleted_at IS NULL LEFT JOIN modules module ON module.id=module_link.module_id`
	}
	baseWhere := []string{
		"workspace.slug=$1", "issue.deleted_at IS NULL", "issue.archived_at IS NULL",
		"issue.is_draft IS NOT TRUE", "project.deleted_at IS NULL", "project.archived_at IS NULL",
		`NOT EXISTS (SELECT 1 FROM states manager_state WHERE manager_state.id=issue.state_id AND manager_state."group"='triage')`,
	}
	present := make(map[string]bool, len(filters))
	for key := range filters {
		present[key] = true
	}
	where, args, err := workitems.BuildAppIssueFilters(filters, present, baseWhere, []any{slug})
	if err != nil {
		return nil, nil, analyticsFilterError(s, err)
	}
	labelBaseWhere := []string{"workspace.slug=$1", "issue.deleted_at IS NULL"}
	labelWhere, _, err := workitems.BuildAppIssueFilters(filters, present, labelBaseWhere, []any{slug})
	if err != nil {
		return nil, nil, analyticsFilterError(s, err)
	}
	from := ` FROM issues issue JOIN workspaces workspace ON workspace.id=issue.workspace_id JOIN projects project ON project.id=issue.project_id`
	query := `WITH filtered_issues AS (SELECT issue.*` + from + ` WHERE ` + strings.Join(where, " AND ") + `),
label_filtered_issues AS (SELECT issue.*` + from + ` WHERE ` + strings.Join(labelWhere, " AND ") + `),
plot AS (SELECT ` + valid[x] + ` dimension,` + segexpr + ` segment,` + metric + ` metric FROM filtered_issues issue` + joins + ` GROUP BY 1,2),
filtered AS (SELECT count(*)::int total FROM filtered_issues),
distribution AS (
 SELECT COALESCE(json_object_agg(dimension::text,items),'{}'::json) value FROM (
   SELECT dimension,json_agg(json_build_object('dimension',dimension`
	if segment != "" {
		query += `,'segment',segment`
	}
	metricName := "count"
	if y == "estimate" {
		metricName = "estimate"
	}
	query += `,'` + metricName + `',metric) ORDER BY segment NULLS FIRST) items FROM plot WHERE dimension IS NOT NULL GROUP BY dimension
 ) grouped
),
extras AS (
 SELECT
  CASE WHEN %t THEN COALESCE((SELECT json_agg(json_build_object('state_id',details.id,'state__name',details.name,'state__color',details.color) ORDER BY details.id) FROM (SELECT DISTINCT state.id,state.name,state.color FROM filtered_issues issue JOIN states state ON state.id=issue.state_id) details),'[]'::json) ELSE '{}'::json END state_details,
  CASE WHEN %t THEN COALESCE((SELECT json_agg(json_build_object('labels__id',details.id,'labels__color',details.color,'labels__name',details.name) ORDER BY details.id) FROM (SELECT DISTINCT label.id,label.color,label.name FROM label_filtered_issues issue JOIN issue_labels link ON link.issue_id=issue.id AND link.deleted_at IS NULL JOIN labels label ON label.id=link.label_id) details),'[]'::json) ELSE '{}'::json END label_details,
  CASE WHEN %t THEN COALESCE((SELECT json_agg(json_build_object('assignees__avatar_url',details.avatar_url,'assignees__display_name',details.display_name,'assignees__first_name',details.first_name,'assignees__last_name',details.last_name,'assignees__id',details.id) ORDER BY details.id) FROM (SELECT DISTINCT assignee.id,assignee.display_name,assignee.first_name,assignee.last_name,COALESCE(CASE WHEN assignee.avatar_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||assignee.avatar_asset_id::text||'/' END,assignee.avatar) avatar_url FROM filtered_issues issue JOIN issue_assignees link ON link.issue_id=issue.id AND link.deleted_at IS NULL JOIN users assignee ON assignee.id=link.assignee_id WHERE assignee.avatar IS NOT NULL OR assignee.avatar_asset_id IS NOT NULL) details),'[]'::json) ELSE '{}'::json END assignee_details,
  CASE WHEN %t THEN COALESCE((SELECT json_agg(json_build_object('issue_cycle__cycle_id',details.id,'issue_cycle__cycle__name',details.name) ORDER BY details.id) FROM (SELECT DISTINCT cycle.id,cycle.name FROM filtered_issues issue JOIN cycle_issues link ON link.issue_id=issue.id AND link.deleted_at IS NULL JOIN cycles cycle ON cycle.id=link.cycle_id) details),'[]'::json) ELSE '{}'::json END cycle_details,
  CASE WHEN %t THEN COALESCE((SELECT json_agg(json_build_object('issue_module__module_id',details.id,'issue_module__module__name',details.name) ORDER BY details.id) FROM (SELECT DISTINCT module.id,module.name FROM filtered_issues issue JOIN module_issues link ON link.issue_id=issue.id AND link.deleted_at IS NULL JOIN modules module ON module.id=link.module_id) details),'[]'::json) ELSE '{}'::json END module_details

)`
	query = fmt.Sprintf(query,
		x == "state_id" || segment == "state_id",
		x == "labels__id" || segment == "labels__id",
		x == "assignees__id" || segment == "assignees__id",
		x == "issue_cycle__cycle_id" || segment == "issue_cycle__cycle_id",
		x == "issue_module__module_id" || segment == "issue_module__module_id",
	)
	query += `SELECT json_build_object('total',(SELECT total FROM filtered),'distribution',(SELECT value FROM distribution)`
	if includeExtras {
		query += `,'extras',(SELECT row_to_json(extras) FROM extras)`
	}
	query += `)`
	b, e := s.repository.JSON(ctx, query, args...)
	if e != nil {
		return nil, nil, s.internal()
	}
	return workspaceAnalyticsRaw(b)
}
func analyticsStringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return strings.TrimSpace(*value)
}
