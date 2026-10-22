package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"strconv"
	"strings"
	"time"

	api "github.com/CaliLuke/go-plane/gen/analytic_views"
	store "github.com/CaliLuke/go-plane/internal/analyticviews"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/loom/security"
	"uuid"
)

type analyticViewsRepository interface {
	Authorized(context.Context, uuid.UUID, string) (bool, error)
	List(context.Context, uuid.UUID, string) (json.RawMessage, error)
	Get(context.Context, uuid.UUID, string, uuid.UUID) (json.RawMessage, error)
	Create(context.Context, uuid.UUID, string, string, string, json.RawMessage, json.RawMessage) (json.RawMessage, error)
	Update(context.Context, uuid.UUID, string, uuid.UUID, *string, *string, json.RawMessage, bool) (json.RawMessage, error)
	Delete(context.Context, uuid.UUID, string, uuid.UUID) error
}
type analyticViewsSession interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}
type analyticViewsService struct {
	repository analyticViewsRepository
	sessions   analyticViewsSession
}
type analyticViewsPrincipal struct{}

func NewAnalyticViews(r analyticViewsRepository, s analyticViewsSession) (api.Service, error) {
	if r == nil || s == nil {
		return nil, fmt.Errorf("analytic views dependencies are required")
	}
	return &analyticViewsService{r, s}, nil
}
func (s *analyticViewsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	p, e := s.sessions.Validate(ctx, key)
	if e != nil {
		return ctx, &api.AnalyticViewUnauthenticated{Detail: unauthenticatedDetail}
	}
	id, e := uuid.Parse(p.UserID)
	if e != nil {
		return ctx, &api.AnalyticViewUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, analyticViewsPrincipal{}, id), nil
}
func (s *analyticViewsService) auth(ctx context.Context, slug string) (uuid.UUID, error) {
	id, ok := ctx.Value(analyticViewsPrincipal{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &api.AnalyticViewUnauthenticated{Detail: unauthenticatedDetail}
	}
	yes, e := s.repository.Authorized(ctx, id, slug)
	if e != nil {
		return uuid.Nil(), e
	}
	if !yes {
		return uuid.Nil(), &api.AnalyticViewForbidden{Detail: forbiddenDetail}
	}
	return id, nil
}
func decodeAnalytic(b json.RawMessage) (*api.AnalyticView, error) {
	var v api.AnalyticView
	if e := json.Unmarshal(b, &v); e != nil {
		return nil, e
	}
	return &v, nil
}

func analyticQuery(value any) (json.RawMessage, error) {
	m, ok := value.(map[string]any)
	if !ok {
		return json.Marshal(value)
	}
	query := make(map[string]any)
	for source, destination := range map[string]string{
		"state":          "state__in",
		"state_group":    "state__group__in",
		"estimate_point": "estimate_point__in",
		"priority":       "priority__in",
		"parent":         "parent__in",
		"labels":         "labels__in",
		"assignees":      "assignees__in",
		"mentions":       "issue_mention__mention__id__in",
		"created_by":     "created_by__in",
		"logged_by":      "logged_by__in",
		"project":        "project__in",
		"cycle":          "issue_cycle__cycle_id__in",
		"module":         "issue_module__module_id__in",
		"inbox_status":   "issue_intake__status__in",
		"subscriber":     "issue_subscribers__subscriber_id__in",
	} {
		if candidate, exists := m[source]; exists && analyticTruthy(candidate) && candidate != "null" {
			query[destination] = candidate
		}
	}
	if candidate, exists := m["intake_status"]; exists && analyticTruthy(candidate) && candidate != "null" {
		// Preserve the source implementation's intake_status/inbox_status mix-up.
		query["issue_intake__status__in"] = m["inbox_status"]
	}
	if candidate, exists := m["name"]; exists && candidate != "" {
		query["name__icontains"] = candidate
	}
	if _, exists := m["labels"]; exists {
		query["label_issue__deleted_at__isnull"] = true
	}
	if _, exists := m["assignees"]; exists {
		query["issue_assignee__deleted_at__isnull"] = true
	}
	if _, exists := m["cycle"]; exists {
		query["issue_cycle__deleted_at__isnull"] = true
	}
	if _, exists := m["module"]; exists {
		query["issue_module__deleted_at__isnull"] = true
	}
	if _, exists := m["subscriber"]; exists {
		query["issue_subscribers__deleted_at__isnull"] = true
	}
	if candidate, exists := m["start_date"]; exists && analyticTruthy(candidate) {
		query["start_date"] = candidate
	}
	if candidate, exists := m["target_date"]; exists && analyticTruthy(candidate) {
		query["target_date"] = candidate
	}
	for source, destination := range map[string]string{
		"created_at":   "created_at__date",
		"updated_at":   "updated_at__date",
		"completed_at": "completed_at__date",
	} {
		if candidate, exists := m[source]; exists && analyticTruthy(candidate) {
			analyticDateFilters(query, destination, candidate)
		}
	}
	if candidate, exists := m["type"]; exists {
		groups := []string{"backlog", "unstarted", "started", "completed", "cancelled"}
		if candidate == "backlog" {
			groups = []string{"backlog"}
		} else if candidate == "active" {
			groups = []string{"unstarted", "started"}
		}
		query["state__group__in"] = groups
	}
	if candidate, exists := m["sub_issue"]; exists && candidate == "false" {
		query["parent__isnull"] = true
	}
	if candidate, exists := m["start_target_date"]; exists && candidate == "true" {
		query["target_date__isnull"] = false
		query["start_date__isnull"] = false
	}
	return json.Marshal(query)
}

func analyticTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	case float64:
		return typed != 0
	case json.Number:
		if strings.ContainsAny(string(typed), ".eE") {
			// Python's JSON decoder uses binary64 for decimals and exponents.
			// Underflow therefore becomes false even while the raw value survives.
			number, _ := strconv.ParseFloat(string(typed), 64)
			return number != 0
		}
		return strings.Trim(string(typed), "-0") != ""
	default:
		return true
	}
}

func analyticDateFilters(query map[string]any, field string, value any) {
	var candidates []string
	switch typed := value.(type) {
	case []any:
		for _, candidate := range typed {
			if text, ok := candidate.(string); ok {
				candidates = append(candidates, text)
			}
		}
	case string:
		candidates = []string{typed}
	}
	for _, candidate := range candidates {
		parts := strings.Split(candidate, ";")
		if len(parts) < 2 {
			query[field+"__contains"] = candidate
			continue
		}
		if len(parts) == 3 {
			durationParts := strings.SplitN(parts[0], "_", 2)
			if len(durationParts) == 2 && (durationParts[1] == "weeks" || durationParts[1] == "months") {
				duration, err := strconv.Atoi(durationParts[0])
				if err == nil {
					days := duration * 7
					if durationParts[1] == "months" {
						days = duration * 30
					}
					if parts[2] != "fromnow" {
						days = -days
					}
					date := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, days).Format("2006-01-02")
					if parts[1] == "after" {
						query[field+"__gte"] = date
					} else {
						query[field+"__lte"] = date
					}
					continue
				}
			}
		}
		if parts[1] == "after" {
			query[field+"__gte"] = parts[0]
		} else {
			query[field+"__lte"] = parts[0]
		}
	}
}
func (s *analyticViewsService) List(ctx context.Context, p *api.ListPayload) ([]*api.AnalyticView, error) {
	id, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	b, e := s.repository.List(ctx, id, p.Slug)
	if e != nil {
		return nil, e
	}
	var out []*api.AnalyticView
	if e = json.Unmarshal(b, &out); e != nil {
		return nil, e
	}
	return out, nil
}
func (s *analyticViewsService) Create(ctx context.Context, p *api.CreatePayload) (*api.AnalyticView, error) {
	id, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	qd := json.RawMessage(`{}`)
	query := json.RawMessage(`{}`)
	if len(p.QueryDict) > 0 && p.QueryDict.Kind() != 'n' {
		qd, e = pythoncompat.NormalizeJSONNumbers(p.QueryDict)
		if e != nil {
			return nil, api.JSONFieldValidation{"query_dict": []string{"Value must be valid JSON."}}
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(qd))
		decoder.UseNumber()
		if e = decoder.Decode(&value); e != nil {
			return nil, e
		}
		query, e = analyticQuery(value)
		if e != nil {
			return nil, e
		}
	}

	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	b, e := s.repository.Create(ctx, id, p.Slug, p.Name, desc, query, qd)
	if e != nil {
		return nil, e
	}
	return decodeAnalytic(b)
}
func (s *analyticViewsService) Retrieve(ctx context.Context, p *api.RetrievePayload) (*api.AnalyticView, error) {
	id, e := uuid.Parse(p.Pk)
	if e != nil {
		return nil, &api.AnalyticViewNotFound{Detail: "Not found."}
	}
	user, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	b, e := s.repository.Get(ctx, user, p.Slug, id)
	if errors.Is(e, store.ErrNotFound) {
		return nil, &api.AnalyticViewNotFound{Detail: "Not found."}
	}
	if e != nil {
		return nil, e
	}
	return decodeAnalytic(b)
}
func (s *analyticViewsService) Patch(ctx context.Context, p *api.PatchPayload) (*api.AnalyticView, error) {
	id, e := uuid.Parse(p.Pk)
	if e != nil {
		return nil, &api.AnalyticViewNotFound{Detail: "Not found."}
	}
	user, e := s.auth(ctx, p.Slug)
	if e != nil {
		return nil, e
	}
	// query_data is not a declared serializer field in Django. It is ignored
	// during validation, and the custom update hook consequently resets query
	// to {} on every PATCH while leaving query_dict unchanged.
	q := json.RawMessage(`{}`)
	present := true
	b, e := s.repository.Update(ctx, user, p.Slug, id, p.Name, p.Description, q, present)
	if errors.Is(e, store.ErrNotFound) {
		return nil, &api.AnalyticViewNotFound{Detail: "Not found."}
	}
	if e != nil {
		return nil, e
	}
	return decodeAnalytic(b)
}
func (s *analyticViewsService) Delete(ctx context.Context, p *api.DeletePayload) error {
	id, e := uuid.Parse(p.Pk)
	if e != nil {
		return &api.AnalyticViewNotFound{Detail: "Not found."}
	}
	user, e := s.auth(ctx, p.Slug)
	if e != nil {
		return e
	}
	e = s.repository.Delete(ctx, user, p.Slug, id)
	if errors.Is(e, store.ErrNotFound) {
		return &api.AnalyticViewNotFound{Detail: "Not found."}
	}
	return e
}
