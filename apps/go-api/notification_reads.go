package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/notifications"
	api "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

type notificationsRepository interface {
	Authorize(context.Context, store.Principal, string) error
	List(context.Context, store.Principal, string, url.Values) (any, error)
	Counts(context.Context, store.Principal, string) (map[string]int64, error)
	Detail(context.Context, store.Principal, string, string, bool) (map[string]any, error)
}
type SessionProtocolAuthorizer interface {
	AuthorizeSessionProtocol(context.Context, string) (int, map[string]any)
}

// SessionProtocolResource supplies URL facts for permission checks which run
// before unsupported-method dispatch. It never substitutes for authorized queries.
type SessionProtocolResource struct {
	Method, Pattern, Slug, ProjectID, ID string
	PathParameters                       map[string]string
}

type SessionResourceProtocolAuthorizer interface {
	AuthorizeSessionResourceProtocol(context.Context, string, SessionProtocolResource) (int, map[string]any)
}
type notificationsService struct {
	repository, counts notificationsRepository
	sessions           activityReadsSessions
}
type notificationPrincipalKey struct{}

func NewNotificationReads(repository, counts notificationsRepository, sessions activityReadsSessions) (api.Service, error) {
	if repository == nil || counts == nil || sessions == nil {
		return nil, errors.New("notification repositories and sessions are required")
	}
	return &notificationsService{repository, counts, sessions}, nil
}
func notificationError(status int, body map[string]any) error {
	encoded, _ := json.Marshal(body)
	result := &api.NotificationError{Name: fmt.Sprintf("status_%d", status)}
	if err := json.Unmarshal(encoded, result); err != nil {
		panic(err)
	}
	return result
}
func mapNotificationError(err error) error {
	if err == nil {
		return nil
	}
	var generated *api.NotificationError
	if errors.As(err, &generated) {
		return err
	}
	var failure *store.HTTPError
	if errors.As(err, &failure) {
		return notificationError(failure.Status, failure.Body)
	}
	return notificationError(500, map[string]any{"error": "Something went wrong please try again later"})
}
func (s *notificationsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	userID, timezone, inspected := readServiceSessionPrincipal(ctx)
	if !inspected {
		if key == "" {
			if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
				request := &http.Request{Header: metadata.Headers()}
				if cookie, err := request.Cookie("session-id"); err == nil {
					key = cookie.Value
				}
			}
		}
		principal, err := s.sessions.Validate(ctx, key)
		if err == nil {
			userID, timezone = principal.UserID, principal.UserTimezone
		}
	}
	if userID == "" {
		return ctx, notificationError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	return context.WithValue(ctx, notificationPrincipalKey{}, store.Principal{UserID: userID, Timezone: timezone}), nil
}
func (s *notificationsService) AuthorizeSessionProtocol(ctx context.Context, key string) (int, map[string]any) {
	_, err := s.APIKeyAuth(ctx, key, nil)
	if err == nil {
		return 0, nil
	}
	return 401, map[string]any{"detail": "Authentication credentials were not provided."}
}
func (s *notificationsService) principal(ctx context.Context, slug string, authorize bool, repository notificationsRepository) (store.Principal, error) {
	p, _ := ctx.Value(notificationPrincipalKey{}).(store.Principal)
	if p.UserID == "" {
		if id, zone, inspected := readServiceSessionPrincipal(ctx); inspected {
			p = store.Principal{UserID: id, Timezone: zone}
		}
	}
	if p.UserID == "" {
		return p, notificationError(401, map[string]any{"detail": "Authentication credentials were not provided."})
	}
	if err := djangosession.ValidateUserTimezone(p.Timezone); err != nil {
		if errors.Is(err, djangosession.ErrTimezoneNotFound) {
			return p, notificationError(400, map[string]any{"error": "The required key does not exist."})
		}
		return p, mapNotificationError(err)
	}
	if authorize {
		if err := repository.Authorize(ctx, p, slug); err != nil {
			return p, mapNotificationError(err)
		}
	}
	return p, nil
}
func notificationResult[T any](body any, err error) (*T, error) {
	if err != nil {
		return nil, mapNotificationError(err)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, mapNotificationError(err)
	}
	var result T
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, mapNotificationError(err)
	}
	return &result, nil
}
func (s *notificationsService) List(ctx context.Context, payload *api.ListPayload) (*api.NotificationArrayOrPaginatedNotifications, error) {
	p, err := s.principal(ctx, payload.Slug, true, s.repository)
	if err != nil {
		return nil, err
	}
	query := ReadServiceQueryValues(ctx)
	if query == nil {
		query = url.Values{}
		for key, value := range map[string]*string{"snoozed": payload.Snoozed, "archived": payload.Archived, "read": payload.Read, "type": payload.Type, "mentioned": payload.Mentioned, "per_page": payload.PerPage, "cursor": payload.Cursor, "order_by": payload.OrderBy} {
			if value != nil {
				query.Set(key, *value)
			}
		}
	}
	body, err := s.repository.List(ctx, p, payload.Slug, query)
	// Select the known source branch directly without decoding the response twice.
	if err != nil {
		return nil, mapNotificationError(err)
	}
	var result api.NotificationArrayOrPaginatedNotifications
	if _, array := body.([]any); array {
		value, err := notificationResult[api.NotificationArray](body, nil)
		if err != nil {
			return nil, err
		}
		result.SetNotificationArray(*value)
	} else {
		value, err := notificationResult[api.PaginatedNotifications](body, nil)
		if err != nil {
			return nil, err
		}
		result.SetPaginatedNotifications(value)
	}
	return &result, nil
}
func (s *notificationsService) Counts(ctx context.Context, payload *api.CountsPayload) (*api.UnreadCounts, error) {
	p, err := s.principal(ctx, payload.Slug, true, s.counts)
	if err != nil {
		return nil, err
	}
	body, err := s.counts.Counts(ctx, p, payload.Slug)
	return notificationResult[api.UnreadCounts](body, err)
}
func (s *notificationsService) Get(ctx context.Context, payload *api.GetPayload) (*api.Notification, error) {
	p, err := s.principal(ctx, payload.Slug, false, s.repository)
	if err != nil {
		return nil, err
	}
	body, err := s.repository.Detail(ctx, p, payload.Slug, payload.Pk, true)
	return notificationResult[api.Notification](body, err)
}
