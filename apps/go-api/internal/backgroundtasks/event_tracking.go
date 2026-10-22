package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/posthog/posthog-go"
)

const (
	EventTrackingWorkflowName = "plane.analytics.track_event"
	eventTrackingActivityName = "plane.analytics.capture_event"
	eventTrackingCeleryTask   = "plane.bgtasks.event_tracking_task.track_event"
)

type EventTrackingInput struct {
	UserID          string         `json:"user_id"`
	EventName       string         `json:"event_name"`
	Slug            string         `json:"slug"`
	EventProperties map[string]any `json:"event_properties"`
}

type instanceConfigCipher interface {
	Decrypt(string) (string, error)
}

type eventTrackingClient interface {
	Enqueue(posthog.Message) error
	FlushWithContext(context.Context) error
	Close() error
}

type eventTrackingClientFactory func(string, string) (eventTrackingClient, error)

type eventTrackingStore struct {
	pool      *pgxpool.Pool
	cipher    instanceConfigCipher
	newClient eventTrackingClientFactory
}

type EventTrackingDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[EventTrackingInput, struct{}]
}

type EventTrackingTaskAdmitter interface {
	AdmitEventTracking(context.Context, string, EventTrackingInput) error
}

type EventTrackingTaskPublisher struct {
	fallback  TaskFallback
	admission EventTrackingTaskAdmitter
}

func NewPostgresEventTrackingStore(pool *pgxpool.Pool, cipher instanceConfigCipher) (*eventTrackingStore, error) {
	if pool == nil || cipher == nil {
		return nil, errors.New("event tracking PostgreSQL pool and configuration cipher are required")
	}
	return &eventTrackingStore{pool: pool, cipher: cipher, newClient: func(apiKey, host string) (eventTrackingClient, error) {
		return posthog.NewWithConfig(apiKey, posthog.Config{Endpoint: host})
	}}, nil
}

func RegisterEventTrackingWorkflow(app *selvedge.App, store *eventTrackingStore, policy *retry.Policy) (*EventTrackingDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and event tracking store are required")
	}
	if policy == nil {
		policy = retry.Exponential(5, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[EventTrackingInput, struct{}]{Name: eventTrackingActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input EventTrackingInput) (struct{}, error) {
		if err := store.Track(ctx, input); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: issueActivityQueue, StartToCloseTimeout: 30 * time.Second, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register event tracking activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(EventTrackingWorkflowName, func(ctx *selvedge.WorkflowContext, input EventTrackingInput) (struct{}, error) {
		if _, err := uuid.Parse(input.UserID); err != nil {
			return struct{}{}, fmt.Errorf("invalid tracking user id: %w", err)
		}
		_, err := activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
		return struct{}{}, err
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &EventTrackingDispatcher{app: app, workflow: workflow}, nil
}

func (dispatcher *EventTrackingDispatcher) Admit(ctx context.Context, eventID string, input EventTrackingInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable event tracking dispatcher is not configured")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return "", fmt.Errorf("event tracking admission id must be a UUID: %w", err)
	}
	if _, err := uuid.Parse(input.UserID); err != nil {
		return "", fmt.Errorf("event tracking user id must be a UUID: %w", err)
	}
	if strings.TrimSpace(input.EventName) == "" || input.Slug == "" {
		return "", errors.New("event tracking event name and workspace slug are required")
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "plane-track-event:"+eventID, input)
}

func (dispatcher *EventTrackingDispatcher) AdmitEventTracking(ctx context.Context, eventID string, input EventTrackingInput) error {
	_, err := dispatcher.Admit(ctx, eventID, input)
	return err
}

func NewEventTrackingTaskPublisher(fallback TaskFallback, admission EventTrackingTaskAdmitter) (*EventTrackingTaskPublisher, error) {
	if fallback == nil || admission == nil {
		return nil, errors.New("task fallback and event tracking admission are required")
	}
	return &EventTrackingTaskPublisher{fallback: fallback, admission: admission}, nil
}

func (publisher *EventTrackingTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if publisher == nil || publisher.fallback == nil || publisher.admission == nil {
		return errors.New("event tracking task publisher is not configured")
	}
	if name != eventTrackingCeleryTask {
		return publisher.fallback.Task(ctx, name, args, kwargs)
	}
	if len(args) != 0 {
		return errors.New("event tracking task requires keyword arguments")
	}
	data, err := json.Marshal(kwargs)
	if err != nil {
		return fmt.Errorf("encode event tracking task arguments: %w", err)
	}
	var input EventTrackingInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("decode event tracking task arguments: %w", err)
	}
	return publisher.admission.AdmitEventTracking(ctx, uuid.NewString(), input)
}

func (publisher *EventTrackingTaskPublisher) SoftDelete(ctx context.Context, table, id, field string) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("event tracking task publisher is not configured")
	}
	return publisher.fallback.SoftDelete(ctx, table, id, field)
}

func (store *eventTrackingStore) Track(ctx context.Context, input EventTrackingInput) error {
	apiKey, host, err := store.configuration(ctx)
	if err != nil {
		return fmt.Errorf("read PostHog configuration: %w", err)
	}
	if apiKey == "" || host == "" {
		slog.WarnContext(ctx, "event tracking is not configured")
		return nil
	}
	if input.EventProperties == nil {
		input.EventProperties = map[string]any{}
	}
	if input.EventName == "user_invited_to_workspace" || input.EventName == "workspace_deleted" {
		var ownerID string
		err := store.pool.QueryRow(ctx, `SELECT owner_id::text FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, input.Slug).Scan(&ownerID)
		if errors.Is(err, pgx.ErrNoRows) {
			slog.WarnContext(ctx, "workspace is missing while preparing event properties", "slug", input.Slug, "user_id", input.UserID, "event", input.EventName)
			input.EventProperties["role"] = "unknown"
		} else if err != nil {
			return fmt.Errorf("read workspace owner for event properties: %w", err)
		} else if ownerID == input.UserID {
			input.EventProperties["role"] = "owner"
		} else {
			input.EventProperties["role"] = "admin"
		}
	}
	client, err := store.newClient(apiKey, host)
	if err != nil {
		return fmt.Errorf("create PostHog client: %w", err)
	}
	if err := client.Enqueue(posthog.Capture{DistinctId: input.UserID, Event: input.EventName, Properties: posthog.Properties(input.EventProperties), Groups: posthog.NewGroups().Set("workspace", input.Slug)}); err != nil {
		return errors.Join(fmt.Errorf("enqueue PostHog event: %w", err), client.Close())
	}
	if err := client.FlushWithContext(ctx); err != nil {
		return errors.Join(fmt.Errorf("flush PostHog event: %w", err), client.Close())
	}
	if err := client.Close(); err != nil {
		return fmt.Errorf("close PostHog client after capture: %w", err)
	}
	return nil
}

func (store *eventTrackingStore) configuration(ctx context.Context) (string, string, error) {
	apiKey, host := os.Getenv("POSTHOG_API_KEY"), os.Getenv("POSTHOG_HOST")
	skipEnv := os.Getenv("SKIP_ENV_VAR")
	if skipEnv == "0" || strings.EqualFold(skipEnv, "false") {
		return apiKey, host, nil
	}
	rows, err := store.pool.Query(ctx, `SELECT key,value,is_encrypted FROM instance_configurations WHERE deleted_at IS NULL AND key=ANY($1::text[])`, []string{"POSTHOG_API_KEY", "POSTHOG_HOST"})
	if err != nil {
		return "", "", err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		var encrypted bool
		if err := rows.Scan(&key, &value, &encrypted); err != nil {
			return "", "", err
		}
		if encrypted {
			value, err = store.cipher.Decrypt(value)
			if err != nil {
				return "", "", fmt.Errorf("decrypt %s: %w", key, err)
			}
		}
		if key == "POSTHOG_API_KEY" {
			apiKey = value
		} else if key == "POSTHOG_HOST" {
			host = value
		}
	}
	return apiKey, host, rows.Err()
}
