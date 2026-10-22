package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	store "github.com/CaliLuke/go-plane/internal/workitems"
	api "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	"github.com/CaliLuke/loom/clue/log"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/jackc/pgx/v5/pgconn"
)

type workItemRepository interface {
	AuthorizeInitial(context.Context, store.Request) error
	ProtocolPutAction(context.Context, store.Request) (bool, error)
	Execute(context.Context, store.Request, io.Reader, string) (store.Result, error)
}

type workItemCoreService struct {
	repository workItemRepository
	sessions   activityReadsSessions
	keys       activityReadsAPIKeys
	throttle   activityReadsThrottle
}
type workItemPrincipal struct{ user, timezone, key string }
type workItemPrincipalKey struct{}

type WorkItemProtocolResource struct {
	Method, Pattern, APIKey, SessionID string
	Params                             map[string]string
}
type WorkItemProtocolMetadata struct {
	OmitPutAction bool
}

type WorkItemProtocolAuthorizer interface {
	AuthorizeWorkItemProtocol(context.Context, WorkItemProtocolResource) (WorkItemProtocolMetadata, error)
}

func (s *workItemCoreService) AuthorizeWorkItemProtocol(ctx context.Context, resource WorkItemProtocolResource) (WorkItemProtocolMetadata, error) {
	isAPI := strings.HasPrefix(resource.Pattern, "/api/v1/")
	key, scheme := resource.SessionID, &security.APIKeyScheme{Name: "SessionAuth"}
	if isAPI {
		key, scheme = resource.APIKey, &security.APIKeyScheme{Name: "ApiKeyAuth"}
	}
	ctx, err := s.APIKeyAuth(ctx, key, scheme)
	if err != nil {
		return WorkItemProtocolMetadata{}, err
	}
	p := ctx.Value(workItemPrincipalKey{}).(workItemPrincipal)
	operation := strings.ToLower(resource.Method) + "_" + strings.NewReplacer("/", "_", "-", "_", "{", "by_", "}", "").Replace(strings.Trim(resource.Pattern, "/"))
	q := store.Request{Operation: operation, Method: resource.Method, Slug: resource.Params["slug"], ProjectID: resource.Params["project_id"], IssueID: resource.Params["issue_id"], Params: resource.Params, ActorID: p.user, Timezone: p.timezone, API: isAPI}
	if q.IssueID == "" {
		q.IssueID = resource.Params["pk"]
	}
	if err = s.repository.AuthorizeInitial(ctx, q); err != nil {
		workItemLogError(ctx, operation, "authorize_initial", err)
		return WorkItemProtocolMetadata{}, workItemError(err)
	}
	if isAPI {
		rate, e := s.throttle.Allow(ctx, p.key)
		if e != nil {
			workItemLogError(ctx, operation, "throttle", e)
			return WorkItemProtocolMetadata{}, workItemError(e)
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		if !rate.Allowed {
			return WorkItemProtocolMetadata{}, workItemError(&djangoapi.HTTPError{Status: 429, Body: map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))}})
		}
	}
	if err = djangosession.ValidateUserTimezone(p.timezone); errors.Is(err, djangosession.ErrTimezoneNotFound) {
		err = &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "The required key does not exist."}}
	}
	if err != nil {
		workItemLogError(ctx, operation, "timezone", err)
		return WorkItemProtocolMetadata{}, workItemError(err)
	}
	if resource.Method == http.MethodOptions && resource.Pattern == "/api/workspaces/{slug}/projects/{project_id}/issues/{pk}/" {
		available, err := s.repository.ProtocolPutAction(ctx, q)
		if err != nil {
			workItemLogError(ctx, operation, "protocol_put_action", err)
			return WorkItemProtocolMetadata{}, workItemError(err)
		}
		return WorkItemProtocolMetadata{OmitPutAction: !available}, nil
	}
	return WorkItemProtocolMetadata{}, nil
}

func NewWorkItemCore(repository workItemRepository, sessions activityReadsSessions, keys activityReadsAPIKeys, throttle activityReadsThrottle) (api.Service, error) {
	if repository == nil || sessions == nil || keys == nil || throttle == nil {
		return nil, errors.New("work-item repository, session and API-key validators, and throttle are required")
	}
	return &workItemCoreService{repository: repository, sessions: sessions, keys: keys, throttle: throttle}, nil
}

func workItemError(err error) error {
	var already *api.WorkItemError
	if errors.As(err, &already) {
		return err
	}
	status, body := 500, map[string]any{"error": "Something went wrong please try again later"}
	var failure *djangoapi.HTTPError
	if errors.As(err, &failure) {
		status, body = failure.Status, failure.Body
	}
	var filter *issuequery.FilterError
	if errors.As(err, &filter) {
		status = 400
		body = map[string]any{"message": filter.Message, "code": filter.Code}
		if filter.Errors != nil {
			body["errors"] = filter.Errors
		}
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		if strings.HasPrefix(postgres.Code, "23") {
			status = 400
			body = map[string]any{"error": "The payload is not valid"}
		}
	}
	encoded, marshalErr := loom.JSONValueFrom(body)
	if marshalErr != nil {
		panic(marshalErr)
	}
	return &api.WorkItemError{Kind: fmt.Sprintf("status_%d", status), Body: encoded}
}

func workItemLogError(ctx context.Context, operation, stage string, err error) {
	var httpError *djangoapi.HTTPError
	if errors.As(err, &httpError) && httpError.Status < 500 {
		return
	}
	fields := []log.Fielder{log.KV{K: "msg", V: "Work-item operation failed"}, log.KV{K: "operation", V: operation}, log.KV{K: "stage", V: stage}, log.KV{K: "error_type", V: fmt.Sprintf("%T", err)}, log.KV{K: "error_kind", V: workItemErrorKind(err)}}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		fields = append(fields, log.KV{K: "sqlstate", V: pg.Code}, log.KV{K: "table", V: pg.TableName}, log.KV{K: "column", V: pg.ColumnName}, log.KV{K: "constraint", V: pg.ConstraintName})
	}
	log.Print(ctx, fields...)
}

func workItemErrorKind(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		return "postgres"
	}
	return "other"
}

func (s *workItemCoreService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	p := workItemPrincipal{}
	if scheme != nil && scheme.Name == "ApiKeyAuth" {
		principal, err := s.keys.Validate(ctx, key)
		if err != nil {
			status, message := 500, "Something went wrong please try again later"
			if key == "" {
				status, message = 401, unauthenticatedDetail
			} else if errors.Is(err, apikey.ErrInvalid) {
				status, message = 403, "Given API token is not valid"
			}
			if status == 500 {
				workItemLogError(ctx, "authentication", "api_key_validation", err)
			}
			return ctx, workItemError(&djangoapi.HTTPError{Status: status, Body: map[string]any{"detail": message}})
		}
		p = workItemPrincipal{principal.UserID.String(), principal.Timezone, key}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Authenticated = true
		}
	} else {
		id, zone, inspected := readServiceSessionPrincipal(ctx)
		if !inspected {
			if key == "" {
				if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
					if cookie, err := (&http.Request{Header: metadata.Headers()}).Cookie("session-id"); err == nil {
						key = cookie.Value
					}
				}
			}
			if principal, err := s.sessions.Validate(ctx, key); err == nil {
				id, zone = principal.UserID, principal.UserTimezone
			} else if errors.Is(err, djangosession.ErrStore) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				workItemLogError(ctx, "authentication", "session_validation", err)
			}
		}
		if id == "" {
			return ctx, workItemError(&djangoapi.HTTPError{Status: 401, Body: map[string]any{"detail": unauthenticatedDetail}})
		}
		p = workItemPrincipal{user: id, timezone: zone}
	}
	return context.WithValue(ctx, workItemPrincipalKey{}, p), nil
}

func workItemRequest(ctx context.Context, operation string, payload any) store.Request {
	raw := activityPayloadRequest(ctx, operation, payload)
	// Resource-like names can also be legitimate query parameters (for example
	// issue_id in search). Restore all queries, then protect only actual paths.
	for name, values := range ReadServiceQueryValues(ctx) {
		raw.Present[name] = true
		if len(values) > 0 {
			raw.Params[name] = values[len(values)-1]
		}
	}
	// Query parameters may not replace any part of the authenticated resource path.
	value := reflect.Indirect(reflect.ValueOf(payload))
	for i := 0; i < value.NumField(); i++ {
		name := strings.Split(value.Type().Field(i).Tag.Get("json"), ",")[0]
		switch name {
		case "slug", "project_id", "pk", "issue_id", "cycle_id", "module_id", "user_id", "draft_id", "identifier":
			raw.Params[name] = fmt.Sprint(value.Field(i).Interface())
		}
	}
	method, _, _ := strings.Cut(operation, "_")
	q := store.Request{Operation: operation, Method: strings.ToUpper(method), Slug: raw.Params["slug"], ProjectID: raw.Params["project_id"], IssueID: raw.Params["issue_id"], Params: raw.Params, Present: raw.Present, API: strings.HasPrefix(operation, "get_api_v1_") || strings.Contains(operation, "_api_v1_")}
	if q.IssueID == "" {
		q.IssueID = q.Params["pk"]
	}
	return q
}

func (s *workItemCoreService) execute(ctx context.Context, operation string, payload any, body io.ReadCloser) (*api.WorkItemResponse, io.ReadCloser, error) {
	if body != nil {
		defer body.Close()
	}
	p, ok := ctx.Value(workItemPrincipalKey{}).(workItemPrincipal)
	if !ok {
		return nil, nil, workItemError(&djangoapi.HTTPError{Status: 401, Body: map[string]any{"detail": unauthenticatedDetail}})
	}
	q := workItemRequest(ctx, operation, payload)
	q.ActorID, q.Timezone = p.user, p.timezone
	if err := s.repository.AuthorizeInitial(ctx, q); err != nil {
		workItemLogError(ctx, operation, "authorize_initial", err)
		return nil, nil, workItemError(err)
	}
	if q.API {
		rate, err := s.throttle.Allow(ctx, p.key)
		if err != nil {
			workItemLogError(ctx, operation, "throttle", err)
			return nil, nil, workItemError(err)
		}
		if effects := apikey.EffectsFromContext(ctx); effects != nil {
			effects.Rate = &rate
		}
		if !rate.Allowed {
			return nil, nil, workItemError(&djangoapi.HTTPError{Status: 429, Body: map[string]any{"detail": fmt.Sprintf("Request was throttled. Expected available in %d seconds.", int(rate.Wait/time.Second))}})
		}
	}
	if err := djangosession.ValidateUserTimezone(p.timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			err = &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "The required key does not exist."}}
		}
		workItemLogError(ctx, operation, "timezone", err)
		return nil, nil, workItemError(err)
	}
	contentType := ""
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		contentType = metadata.Headers().Get("Content-Type")
	}
	result, err := s.repository.Execute(ctx, q, body, contentType)
	if err != nil {
		workItemLogError(ctx, operation, "execute", err)
		return nil, nil, workItemError(err)
	}
	if result.Status == 204 {
		return &api.WorkItemResponse{}, io.NopCloser(bytes.NewReader(nil)), nil
	}
	data, err := json.Marshal(activityTimestamps(result.Body, p.timezone))
	if err != nil {
		workItemLogError(ctx, operation, "encode_response", err)
		MarkReadServiceDjangoHTML500(ctx)
		return nil, nil, workItemError(err)
	}
	return &api.WorkItemResponse{}, io.NopCloser(bytes.NewReader(data)), nil
}
