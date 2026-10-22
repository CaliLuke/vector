package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	store "github.com/CaliLuke/go-plane/internal/notifications"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	api "github.com/CaliLuke/go-plane/notificationactions/gen/notification_actions"
	reads "github.com/CaliLuke/go-plane/notificationreads/gen/notifications"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

type notificationActionRepository interface {
	notificationsRepository
	Set(context.Context, store.Principal, string, string, string, any) (map[string]any, error)
	MarkAllRead(context.Context, store.Principal, string, bool, bool, string) error
}

type notificationActionsService struct {
	*notificationsService
	actions notificationActionRepository
}

func NewNotificationActions(repository notificationActionRepository, sessions activityReadsSessions) (api.Service, error) {
	if repository == nil || sessions == nil {
		return nil, errors.New("notification action repository and sessions are required")
	}
	return &notificationActionsService{notificationsService: &notificationsService{repository: repository, counts: repository, sessions: sessions}, actions: repository}, nil
}

// Both notification transports share source authorization and error semantics.
// Each generated service still receives its own declared error type.
func notificationActionError(err error) error {
	if err == nil {
		return nil
	}
	var failure *reads.NotificationError
	if !errors.As(mapNotificationError(err), &failure) {
		panic("notification error mapping returned an undeclared type")
	}
	return &api.NotificationError{Name: failure.Name, Detail: failure.Detail, ErrorValue: failure.ErrorValue, SnoozedTill: failure.SnoozedTill}
}
func (s *notificationActionsService) APIKeyAuth(ctx context.Context, key string, scheme *security.APIKeyScheme) (context.Context, error) {
	ctx, err := s.notificationsService.APIKeyAuth(ctx, key, scheme)
	return ctx, notificationActionError(err)
}
func (s *notificationActionsService) target(ctx context.Context, slug, id string, inherited bool) (store.Principal, error) {
	p, err := s.principal(ctx, slug, !inherited, s.actions)
	if err == nil {
		_, err = s.actions.Detail(ctx, p, slug, id, inherited)
	}
	return p, notificationActionError(err)
}
func (s *notificationActionsService) Delete(ctx context.Context, payload *api.DeletePayload) error {
	p, err := s.target(ctx, payload.Slug, payload.Pk, true)
	if err != nil {
		return err
	}
	_, err = s.actions.Set(ctx, p, payload.Slug, payload.Pk, "deleted_at", time.Now().UTC())
	return notificationActionError(err)
}
func (s *notificationActionsService) update(ctx context.Context, slug, id, field string, value any) (*api.Notification, error) {
	p, err := s.target(ctx, slug, id, false)
	if err != nil {
		return nil, err
	}
	body, err := s.actions.Set(ctx, p, slug, id, field, value)
	result, err := notificationResult[api.Notification](body, err)
	return result, notificationActionError(err)
}
func (s *notificationActionsService) Archive(ctx context.Context, payload *api.ArchivePayload, body io.ReadCloser) (*api.Notification, error) {
	if body != nil {
		defer body.Close()
	}
	return s.update(ctx, payload.Slug, payload.Pk, "archived_at", time.Now().UTC())
}
func (s *notificationActionsService) Unarchive(ctx context.Context, payload *api.UnarchivePayload) (*api.Notification, error) {
	return s.update(ctx, payload.Slug, payload.Pk, "archived_at", nil)
}
func (s *notificationActionsService) MarkRead(ctx context.Context, payload *api.MarkReadPayload, body io.ReadCloser) (*api.Notification, error) {
	if body != nil {
		defer body.Close()
	}
	return s.update(ctx, payload.Slug, payload.Pk, "read_at", time.Now().UTC())
}
func (s *notificationActionsService) MarkUnread(ctx context.Context, payload *api.MarkUnreadPayload) (*api.Notification, error) {
	return s.update(ctx, payload.Slug, payload.Pk, "read_at", nil)
}

func (s *notificationActionsService) MarkAllRead(ctx context.Context, payload *api.MarkAllReadPayload, body io.ReadCloser) (*api.NotificationActionMessage, error) {
	p, err := s.principal(ctx, payload.Slug, true, s.actions)
	if err != nil {
		return nil, notificationActionError(err)
	}
	if body != nil {
		defer body.Close()
	}
	contentType, _ := ctx.Value(loomhttp.ContentTypeKey).(string)
	if contentType == "" {
		contentType = "application/json"
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, notificationActionParseError(err)
	}
	if input.Kind != "" {
		return nil, notificationActionBodyError(400, map[string]any{"detail": "Expected a JSON object."})
	}
	snoozed := pythoncompat.Truthy(input.Fields["snoozed"])
	archived := pythoncompat.Truthy(input.Fields["archived"])
	kind := "all"
	if value, ok := input.Fields["type"].(string); ok {
		kind = value
	}
	if err := s.actions.MarkAllRead(ctx, p, payload.Slug, snoozed, archived, kind); err != nil {
		return nil, notificationActionError(err)
	}
	return &api.NotificationActionMessage{Message: "Successful"}, nil
}

func notificationActionParseError(err error) error {
	var source *djangoapi.HTTPError
	if !errors.As(err, &source) {
		return notificationActionError(err)
	}
	return notificationActionBodyError(source.Status, source.Body)
}

func notificationActionBodyError(status int, body map[string]any) error {
	encoded, marshalErr := json.Marshal(body)
	if marshalErr != nil {
		return notificationActionError(errors.New("notification error body could not be encoded"))
	}
	failure := &api.NotificationError{Name: fmt.Sprintf("status_%d", status)}
	if json.Unmarshal(encoded, failure) != nil {
		return notificationActionError(errors.New("notification error body could not be decoded"))
	}
	return failure
}

func (s *notificationActionsService) Update(ctx context.Context, payload *api.UpdatePayload, body io.ReadCloser) (*api.Notification, error) {
	if body != nil {
		defer body.Close()
	}
	p, err := s.target(ctx, payload.Slug, payload.Pk, false)
	if err != nil {
		return nil, err
	}
	contentType, _ := ctx.Value(loomhttp.ContentTypeKey).(string)
	if contentType == "" {
		contentType = "application/json"
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, notificationActionParseError(err)
	}
	value := any(nil)
	if raw, present := input.FieldValue("snoozed_till", false, false, true); present && raw != nil {
		text := djangoinput.ScalarString(raw)
		location, locationErr := time.LoadLocation(p.Timezone)
		if locationErr != nil {
			return nil, notificationActionError(locationErr)
		}
		parsed, parseErr := pythoncompat.DjangoDateTime(text, location)
		if parseErr != nil {
			return nil, notificationActionBodyError(400, map[string]any{"snoozed_till": []string{"Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."}})
		}
		value = parsed
	}
	result, err := s.actions.Set(ctx, p, payload.Slug, payload.Pk, "snoozed_till", value)
	return notificationResult[api.Notification](result, notificationActionError(err))
}
