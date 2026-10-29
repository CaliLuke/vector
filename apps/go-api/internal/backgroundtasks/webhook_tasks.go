package backgroundtasks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	webhookModelActivityTask  = "plane.bgtasks.webhook_task.model_activity"
	webhookActivityCeleryTask = "plane.bgtasks.webhook_task.webhook_activity"
	webhookSendCeleryTask     = "plane.bgtasks.webhook_task.webhook_send_task"
	webhookActivityTask       = "plane.bgtasks.webhook_task.webhook_activity"
	webhookSendTask           = "plane.bgtasks.webhook_task.webhook_send_task"
	webhookModelWorkflow      = "plane.webhooks.model_activity"
	webhookActivityWorkflow   = "plane.webhooks.webhook_activity"
	webhookSendWorkflow       = "plane.webhooks.send"
	webhookModelActivityName  = "plane.webhooks.expand_model_activity"
	webhookFanoutActivity     = "plane.webhooks.fanout_activity"
	webhookSendActivity       = "plane.webhooks.deliver"
	webhookQueue              = "plane-background"
	webhookMaxAttempts        = 6 // Celery's max_retries=5 includes the initial request.
)

var ErrPermanentWebhookTask = errors.New("permanent webhook task failure")

// WebhookTaskInput is the persisted task envelope accepted from existing Plane
// producers. A stable event ID is attached before durable admission.
type WebhookTaskInput struct {
	EventID  string                     `json:"event_id"`
	TaskName string                     `json:"task_name"`
	Args     []json.RawMessage          `json:"args,omitempty"`
	Kwargs   map[string]json.RawMessage `json:"kwargs,omitempty"`
}

type webhookActivityInput struct {
	WorkflowEventID string          `json:"workflow_event_id,omitempty"`
	Event           string          `json:"event"`
	Verb            string          `json:"verb"`
	Field           json.RawMessage `json:"field"`
	OldValue        json.RawMessage `json:"old_value"`
	NewValue        json.RawMessage `json:"new_value"`
	ActorID         string          `json:"actor_id"`
	Slug            string          `json:"slug"`
	WorkspaceID     string          `json:"workspace_id,omitempty"`
	CurrentSite     string          `json:"current_site"`
	EventID         string          `json:"event_id"`
	OldIdentifier   json.RawMessage `json:"old_identifier"`
	NewIdentifier   json.RawMessage `json:"new_identifier"`
}

type webhookSendInput struct {
	EventID     string          `json:"event_id"`
	WebhookID   string          `json:"webhook_id"`
	Slug        string          `json:"slug"`
	WorkspaceID string          `json:"workspace_id,omitempty"`
	Event       string          `json:"event"`
	EventData   json.RawMessage `json:"event_data"`
	Action      string          `json:"action"`
	CurrentSite string          `json:"current_site"`
	Activity    json.RawMessage `json:"activity"`
}

type WebhookTaskDispatcher struct {
	app      *selvedge.App
	model    *selvedge.WorkflowFunc[WebhookTaskInput, struct{}]
	activity *selvedge.WorkflowFunc[webhookActivityInput, struct{}]
	send     *selvedge.WorkflowFunc[webhookSendInput, struct{}]
	store    webhookTaskStore
}

type webhookTaskStore interface {
	ModelActivities(context.Context, WebhookTaskInput) ([]webhookActivityInput, error)
	Fanout(context.Context, webhookActivityInput) ([]webhookSendInput, error)
	Deliver(context.Context, webhookSendInput, int) error
}

// RegisterWebhookTaskWorkflows registers the source webhook task chain. The
// send retry policy follows Celery max_retries=5 (six requests total),
// exponential base 600s, and jitter; Celery's one-sided jitter is represented
// by Selvedge's symmetric jitter with the same expected delay.
func RegisterWebhookTaskWorkflows(app *selvedge.App, store webhookTaskStore, policy *retry.Policy) (*WebhookTaskDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and webhook task store are required")
	}
	if policy == nil {
		policy = &retry.Policy{MaxAttempts: webhookMaxAttempts, InitialInterval: 10 * time.Minute, MaxInterval: 24 * time.Hour, Multiplier: 2, RandomizationFactor: 0.5}
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	modelActivity := execution.Activity[WebhookTaskInput, []webhookActivityInput]{Name: webhookModelActivityName}
	if err := executionSelvedge.Register(activities, modelActivity, func(ctx context.Context, input WebhookTaskInput) ([]webhookActivityInput, error) {
		return store.ModelActivities(ctx, input)
	}, selvedge.ActivityOptions{Queue: webhookQueue, StartToCloseTimeout: 30 * time.Second}); err != nil {
		return nil, fmt.Errorf("register webhook model activity: %w", err)
	}
	fanout := execution.Activity[webhookActivityInput, []webhookSendInput]{Name: webhookFanoutActivity}
	if err := executionSelvedge.Register(activities, fanout, func(ctx context.Context, input webhookActivityInput) ([]webhookSendInput, error) {
		return store.Fanout(ctx, input)
	}, selvedge.ActivityOptions{Queue: webhookQueue, StartToCloseTimeout: 30 * time.Second}); err != nil {
		return nil, fmt.Errorf("register webhook fanout activity: %w", err)
	}
	deliver := execution.Activity[webhookSendInput, struct{}]{Name: webhookSendActivity}
	if err := executionSelvedge.Register(activities, deliver, func(ctx context.Context, input webhookSendInput) (struct{}, error) {
		info := selvedge.GetActivityInfo(ctx)
		attempt := 1
		if info != nil {
			attempt = info.Attempt
		}
		return struct{}{}, store.Deliver(ctx, input, attempt)
	}, selvedge.ActivityOptions{Queue: webhookQueue, StartToCloseTimeout: 35 * time.Second, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register webhook delivery activity: %w", err)
	}
	sendWorkflow := selvedge.DefineWorkflow(webhookSendWorkflow, func(ctx *selvedge.WorkflowContext, input webhookSendInput) (struct{}, error) {
		if err := validateWebhookSendInput(input); err != nil {
			return struct{}{}, err
		}
		return deliver.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	activityWorkflow := selvedge.DefineWorkflow(webhookActivityWorkflow, func(ctx *selvedge.WorkflowContext, input webhookActivityInput) (struct{}, error) {
		sends, err := fanout.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err != nil {
			return struct{}{}, err
		}
		for i, send := range sends {
			child, err := selvedge.StartChild[struct{}](ctx, webhookSendWorkflow, send, selvedge.WithChildID("plane-webhook-send:"+send.EventID))
			if err != nil {
				return struct{}{}, fmt.Errorf("start webhook delivery child %d: %w", i, err)
			}
			if _, err := child.Get(ctx); err != nil {
				return struct{}{}, fmt.Errorf("wait for webhook delivery child %d: %w", i, err)
			}
		}
		return struct{}{}, nil
	})
	modelWorkflow := selvedge.DefineWorkflow(webhookModelWorkflow, func(ctx *selvedge.WorkflowContext, input WebhookTaskInput) (struct{}, error) {
		if err := validateWebhookModelTask(input); err != nil {
			return struct{}{}, err
		}
		activitiesToSend, err := modelActivity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err != nil {
			return struct{}{}, err
		}
		for i, activity := range activitiesToSend {
			childID := webhookActivityChildID(input.EventID, i, activity)
			child, err := selvedge.StartChild[struct{}](ctx, webhookActivityWorkflow, activity, selvedge.WithChildID(childID))
			if err != nil {
				return struct{}{}, fmt.Errorf("start webhook activity child %d: %w", i, err)
			}
			if _, err := child.Get(ctx); err != nil {
				return struct{}{}, fmt.Errorf("wait for webhook activity child %d: %w", i, err)
			}
		}
		return struct{}{}, nil
	})
	selvedge.RegisterWorkflow(app, sendWorkflow)
	selvedge.RegisterWorkflow(app, activityWorkflow)
	selvedge.RegisterWorkflow(app, modelWorkflow)
	return &WebhookTaskDispatcher{app: app, model: modelWorkflow, activity: activityWorkflow, send: sendWorkflow, store: store}, nil
}

func (dispatcher *WebhookTaskDispatcher) Admit(ctx context.Context, input WebhookTaskInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.model == nil {
		return "", errors.New("durable webhook task dispatcher is not configured")
	}
	switch input.TaskName {
	case webhookModelActivityTask:
		if err := validateWebhookModelTask(input); err != nil {
			return "", err
		}
		return admitOrReconcile(ctx, dispatcher.app, dispatcher.model, "plane-webhook-model:"+input.EventID, input)
	case webhookActivityCeleryTask:
		activity, err := webhookActivityInputFromTask(input)
		if err != nil {
			return "", err
		}
		activity.WorkflowEventID = input.EventID
		return admitOrReconcile(ctx, dispatcher.app, dispatcher.activity, "plane-webhook-activity:"+input.EventID, activity)
	case webhookSendCeleryTask:
		send, err := webhookSendInputFromTask(input)
		if err != nil {
			return "", err
		}
		return admitOrReconcile(ctx, dispatcher.app, dispatcher.send, "plane-webhook-send:"+input.EventID, send)
	default:
		return "", fmt.Errorf("%w: unsupported webhook task %q", ErrPermanentWebhookTask, input.TaskName)
	}
}

func webhookActivityChildID(eventID string, index int, input webhookActivityInput) string {
	return "plane-webhook-activity:" + uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s:%d:%s:%s:%s", eventID, index, input.Event, input.Verb, string(input.Field)))).String()
}

func validateWebhookModelTask(input WebhookTaskInput) error {
	if _, err := uuid.Parse(input.EventID); err != nil {
		return fmt.Errorf("%w: event ID must be a UUID", ErrPermanentWebhookTask)
	}
	if input.TaskName != webhookModelActivityTask {
		return fmt.Errorf("%w: unsupported webhook producer task %q", ErrPermanentWebhookTask, input.TaskName)
	}
	for _, key := range []string{"model_name", "model_id", "slug", "actor_id"} {
		var value string
		if err := json.Unmarshal(input.Kwargs[key], &value); err != nil || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%w: %s is required", ErrPermanentWebhookTask, key)
		}
	}
	_, err := webhookWorkspaceID(input.Kwargs, stringValue(input.Kwargs["model_name"]), nil)
	return err
}

func webhookActivityInputFromTask(input WebhookTaskInput) (webhookActivityInput, error) {
	var result webhookActivityInput
	for key, target := range map[string]*string{"event": &result.Event, "verb": &result.Verb, "actor_id": &result.ActorID, "slug": &result.Slug, "current_site": &result.CurrentSite} {
		if err := json.Unmarshal(input.Kwargs[key], target); err != nil {
			return result, fmt.Errorf("%w: invalid webhook activity %s", ErrPermanentWebhookTask, key)
		}
	}
	for key, target := range map[string]*json.RawMessage{"field": &result.Field, "old_value": &result.OldValue, "new_value": &result.NewValue, "old_identifier": &result.OldIdentifier, "new_identifier": &result.NewIdentifier} {
		if value, ok := input.Kwargs[key]; ok {
			*target = value
		}
	}
	if err := json.Unmarshal(input.Kwargs["event_id"], &result.EventID); err != nil {
		return result, fmt.Errorf("%w: invalid webhook activity event_id", ErrPermanentWebhookTask)
	}
	if result.Event == "" || result.Verb == "" || result.ActorID == "" || result.Slug == "" || result.EventID == "" {
		return result, fmt.Errorf("%w: webhook activity requires event, verb, actor, slug and event_id", ErrPermanentWebhookTask)
	}
	var err error
	result.WorkspaceID, err = webhookWorkspaceID(input.Kwargs, result.Event, result.Field)
	return result, err
}

func webhookSendInputFromTask(input WebhookTaskInput) (webhookSendInput, error) {
	var result webhookSendInput
	for key, target := range map[string]*string{"webhook_id": &result.WebhookID, "slug": &result.Slug, "event": &result.Event, "action": &result.Action, "current_site": &result.CurrentSite} {
		if err := json.Unmarshal(input.Kwargs[key], target); err != nil {
			return result, fmt.Errorf("%w: invalid webhook send %s", ErrPermanentWebhookTask, key)
		}
	}
	for key, target := range map[string]*json.RawMessage{"event_data": &result.EventData, "activity": &result.Activity} {
		if value, ok := input.Kwargs[key]; ok {
			*target = value
		}
	}
	result.EventID = input.EventID
	var err error
	result.WorkspaceID, err = webhookWorkspaceID(input.Kwargs, result.Event, nil)
	if err != nil {
		return result, err
	}
	if err := validateWebhookSendInput(result); err != nil {
		return result, err
	}
	return result, nil
}

func validateWebhookSendInput(input webhookSendInput) error {
	if err := validateWebhookWorkspace(input.WorkspaceID, input.Event, nil); err != nil {
		return err
	}
	for name, value := range map[string]string{"event ID": input.EventID, "webhook ID": input.WebhookID} {
		if _, err := uuid.Parse(value); err != nil {
			return fmt.Errorf("%w: invalid %s", ErrPermanentWebhookTask, name)
		}
	}
	if input.Slug == "" || input.Event == "" || input.Action == "" {
		return fmt.Errorf("%w: slug, event and action are required", ErrPermanentWebhookTask)
	}
	return nil
}

// PostgresWebhookTaskStore implements webhook model lookup, subscription
// fanout, safe delivery and attempt logging against Plane's application DB.
type PostgresWebhookTaskStore struct {
	pool          *pgxpool.Pool
	mailer        TemplatedEmailTaskAdmitter
	requestClient func(context.Context, string, http.Header, []byte) (*http.Response, error)
}

func NewPostgresWebhookTaskStore(pool *pgxpool.Pool, mailer TemplatedEmailTaskAdmitter) (*PostgresWebhookTaskStore, error) {
	if pool == nil {
		return nil, errors.New("webhook task PostgreSQL pool is required")
	}
	return &PostgresWebhookTaskStore{pool: pool, mailer: mailer}, nil
}

func (store *PostgresWebhookTaskStore) ModelActivities(ctx context.Context, input WebhookTaskInput) ([]webhookActivityInput, error) {
	kwargs, err := decodeTaskKwargs(input.Kwargs)
	if err != nil {
		return nil, fmt.Errorf("decode webhook model task: %w", err)
	}
	model := stringValue(kwargs["model_name"])
	workspaceID, err := webhookWorkspaceID(kwargs, model, nil)
	if err != nil {
		return nil, err
	}
	modelID := stringValue(kwargs["model_id"])
	requested, err := orderedJSONObject(kwargs["requested_data"])
	if err != nil {
		return nil, fmt.Errorf("decode requested webhook fields: %w", err)
	}
	current, err := optionalJSONObject(kwargs["current_instance"])
	if err != nil {
		return nil, fmt.Errorf("decode current webhook instance: %w", err)
	}
	actor := stringValue(kwargs["actor_id"])
	slug := stringValue(kwargs["slug"])
	origin := stringValue(kwargs["origin"])
	if current == nil {
		return []webhookActivityInput{{WorkflowEventID: input.EventID, Event: model, Verb: "created", Field: json.RawMessage("null"), OldValue: json.RawMessage("null"), NewValue: json.RawMessage("null"), ActorID: actor, Slug: slug, WorkspaceID: workspaceID, CurrentSite: origin, EventID: modelID, OldIdentifier: json.RawMessage("null"), NewIdentifier: json.RawMessage("null")}}, nil
	}
	activities := make([]webhookActivityInput, 0, len(requested))
	for _, item := range requested {
		old, exists := current[item.key]
		if !exists || jsonValuesEqual(old, item.value) {
			continue
		}
		field, _ := json.Marshal(item.key)
		activities = append(activities, webhookActivityInput{WorkflowEventID: input.EventID, Event: model, Verb: "updated", Field: field, OldValue: nonemptyRaw(old), NewValue: nonemptyRaw(item.value), ActorID: actor, Slug: slug, WorkspaceID: workspaceID, CurrentSite: origin, EventID: modelID, OldIdentifier: json.RawMessage("null"), NewIdentifier: json.RawMessage("null")})
	}
	return activities, nil
}

func (store *PostgresWebhookTaskStore) Fanout(ctx context.Context, input webhookActivityInput) ([]webhookSendInput, error) {
	if err := validateWebhookWorkspace(input.WorkspaceID, input.Event, input.Field); err != nil {
		return nil, err
	}
	if input.WorkspaceID != "" {
		// The saved slug is display context, never the identity of a queued event.
		err := store.pool.QueryRow(ctx, `SELECT slug FROM workspaces WHERE id=$1 AND deleted_at IS NULL`, input.WorkspaceID).Scan(&input.Slug)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}
	var eventData json.RawMessage
	if input.Verb == "deleted" {
		if input.WorkspaceID != "" {
			owned, err := store.webhookModelOwned(ctx, input.Event, input.EventID, input.WorkspaceID)
			if err != nil {
				return nil, err
			}
			if !owned {
				return nil, nil
			}
		}
		id, _ := json.Marshal(input.EventID)
		eventData = json.RawMessage(`{"id":` + string(id) + `}`)
	} else {
		var err error
		eventData, err = store.serializedScopedModel(ctx, input.Event, input.EventID, input.WorkspaceID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil // source ignores the model-deleted race
			}
			return nil, err
		}
	}
	actor, err := store.serializedModel(ctx, "user", input.ActorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	activityJSON, err := json.Marshal(struct {
		Field         json.RawMessage `json:"field"`
		NewValue      json.RawMessage `json:"new_value"`
		OldValue      json.RawMessage `json:"old_value"`
		Actor         json.RawMessage `json:"actor"`
		OldIdentifier json.RawMessage `json:"old_identifier"`
		NewIdentifier json.RawMessage `json:"new_identifier"`
	}{nonemptyRaw(input.Field), nonemptyRaw(input.NewValue), nonemptyRaw(input.OldValue), actor, nonemptyRaw(input.OldIdentifier), nonemptyRaw(input.NewIdentifier)})
	if err != nil {
		return nil, err
	}
	query := `SELECT h.id FROM webhooks h JOIN workspaces w ON w.id=h.workspace_id WHERE w.slug=$1 AND w.deleted_at IS NULL AND h.deleted_at IS NULL AND h.is_active=true`
	args := []any{input.Slug}
	if input.WorkspaceID != "" {
		query = `SELECT h.id FROM webhooks h JOIN workspaces w ON w.id=h.workspace_id WHERE w.id=$1 AND w.deleted_at IS NULL AND h.deleted_at IS NULL AND h.is_active=true`
		args = []any{input.WorkspaceID}
	}
	query += webhookSubscriptionSQL(input.Event)
	query += ` ORDER BY h.created_at DESC`
	rows, err := store.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	activity := json.RawMessage(activityJSON)
	var result []webhookSendInput
	for rows.Next() {
		var webhookID uuid.UUID
		if err := rows.Scan(&webhookID); err != nil {
			return nil, err
		}
		id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(input.WorkflowEventID+":"+webhookID.String()+":"+input.Event+":"+input.Verb+":"+string(input.Field))).String()
		result = append(result, webhookSendInput{EventID: id, WebhookID: webhookID.String(), Slug: input.Slug, WorkspaceID: input.WorkspaceID, Event: input.Event, EventData: eventData, Action: input.Verb, CurrentSite: input.CurrentSite, Activity: activity})
	}
	return result, rows.Err()
}

type webhookRecord struct {
	ID, WorkspaceID, Slug, URL, SecretKey string
	CreatorID                             *string
}

func (store *PostgresWebhookTaskStore) Deliver(ctx context.Context, input webhookSendInput, attempt int) error {
	if err := validateWebhookWorkspace(input.WorkspaceID, input.Event, nil); err != nil {
		return err
	}
	var webhook webhookRecord
	query := `SELECT h.id,h.workspace_id,w.slug,h.url,h.secret_key,h.created_by_id::text FROM webhooks h JOIN workspaces w ON w.id=h.workspace_id WHERE h.id=$1 AND w.slug=$2 AND w.deleted_at IS NULL AND h.deleted_at IS NULL`
	owner := input.Slug
	if input.WorkspaceID != "" {
		query = `SELECT h.id,h.workspace_id,w.slug,h.url,h.secret_key,h.created_by_id::text FROM webhooks h JOIN workspaces w ON w.id=h.workspace_id WHERE h.id=$1 AND w.id=$2 AND w.deleted_at IS NULL AND h.deleted_at IS NULL AND h.is_active=true`
		query += webhookSubscriptionSQL(input.Event)
		owner = input.WorkspaceID
	}
	err := store.pool.QueryRow(ctx, query, input.WebhookID, owner).Scan(&webhook.ID, &webhook.WorkspaceID, &webhook.Slug, &webhook.URL, &webhook.SecretKey, &webhook.CreatorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // source catches missing webhook and drops the stale send task
	}
	if err != nil {
		return fmt.Errorf("load webhook for delivery: %w", err)
	}
	payload := webhookPayload{Event: input.Event, Action: mapWebhookAction(input.Action), WebhookID: webhook.ID, WorkspaceID: webhook.WorkspaceID, WorkspaceSlug: webhook.Slug, Data: nonemptyRaw(input.EventData), Activity: nonemptyRaw(input.Activity)}
	body, err := marshalWebhookJSON(payload, false)
	if err != nil {
		return err
	}
	deliveryID := uuid.NewString()
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", "Autopilot")
	headers.Set("X-Plane-Delivery", deliveryID)
	headers.Set("X-Plane-Event", input.Event)
	if webhook.SecretKey != "" {
		signature := hmac.New(sha256.New, []byte(webhook.SecretKey))
		signingBody, marshalErr := marshalWebhookJSON(payload, true)
		if marshalErr != nil {
			return marshalErr
		}
		_, _ = signature.Write(signingBody)
		headers.Set("X-Plane-Signature", hex.EncodeToString(signature.Sum(nil)))
	}
	response, requestErr := store.post(ctx, webhook.URL, headers, body)
	if requestErr != nil {
		var policyErr *webhookURLPolicyError
		if errors.As(requestErr, &policyErr) {
			if err := store.saveLog(ctx, webhook, input, payload, headers, "400", "", "Webhook URL rejected: "+policyErr.Error(), attempt); err != nil {
				return err
			}
			return nil
		}
		if err := store.saveLog(ctx, webhook, input, payload, headers, "500", "", requestErr.Error(), attempt); err != nil {
			return err
		}
		if attempt >= webhookMaxAttempts {
			if _, err := store.pool.Exec(ctx, `UPDATE webhooks SET is_active=false,updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, webhook.ID); err != nil {
				return fmt.Errorf("deactivate webhook after retries: %w", err)
			}
			if store.mailer != nil && webhook.CreatorID != nil {
				_ = store.mailer.AdmitTemplatedEmailTask(ctx, uuid.NewString(), webhookDeactivationEmailTask, []any{webhook.ID, *webhook.CreatorID, input.CurrentSite, requestErr.Error()})
			}
			return nil
		}
		return requestErr
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if readErr != nil {
		return readErr
	}
	if len(responseBody) > 1<<20 {
		return errors.New("webhook response body exceeds 1 MiB")
	}
	if err := store.saveLog(ctx, webhook, input, payload, headers, strconv.Itoa(response.StatusCode), headerString(response.Header), string(responseBody), attempt); err != nil {
		return err
	}
	return nil // source records every HTTP status and does not retry 4xx/5xx
}

type webhookPayload struct {
	Event         string          `json:"event"`
	Action        string          `json:"action"`
	WebhookID     string          `json:"webhook_id"`
	WorkspaceID   string          `json:"workspace_id"`
	WorkspaceSlug string          `json:"workspace_slug"`
	Data          json.RawMessage `json:"data"`
	Activity      json.RawMessage `json:"activity"`
}

func (store *PostgresWebhookTaskStore) saveLog(ctx context.Context, webhook webhookRecord, input webhookSendInput, payload webhookPayload, headers http.Header, status, responseHeaders, responseBody string, attempt int) error {
	body, err := marshalWebhookJSON(payload, false)
	if err != nil {
		return err
	}
	_, err = store.pool.Exec(ctx, `INSERT INTO webhook_logs(created_at,updated_at,id,event_type,request_method,request_headers,request_body,response_status,response_headers,response_body,retry_count,webhook,workspace_id) VALUES(NOW(),NOW(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, uuid.New(), input.Event, payload.Action, headerString(headers), string(body), status, responseHeaders, responseBody, max(0, attempt-1), webhook.ID, webhook.WorkspaceID)
	if err != nil {
		return fmt.Errorf("save webhook delivery log: %w", err)
	}
	return nil
}

func (store *PostgresWebhookTaskStore) post(ctx context.Context, rawURL string, headers http.Header, body []byte) (*http.Response, error) {
	if store.requestClient != nil {
		return store.requestClient(ctx, rawURL, headers, body)
	}
	return pinnedTaskRequest(ctx, http.MethodPost, rawURL, headers, body, 30*time.Second, false, nil, nil)
}

type webhookURLPolicyError struct{ message string }

func (err *webhookURLPolicyError) Error() string { return err.message }

func (store *PostgresWebhookTaskStore) serializedModel(ctx context.Context, event, id string) (json.RawMessage, error) {
	return store.serializedScopedModel(ctx, event, id, "")
}

func (store *PostgresWebhookTaskStore) serializedScopedModel(ctx context.Context, event, id, workspaceID string) (json.RawMessage, error) {
	table, ok := webhookModelTables[event]
	if !ok {
		return nil, fmt.Errorf("unsupported webhook model %q", event)
	}
	var raw []byte
	query := `SELECT to_jsonb(model) FROM ` + table + ` model WHERE model.id=$1 AND model.deleted_at IS NULL`
	if event == "user" {
		query = `SELECT to_jsonb(model) FROM users model WHERE model.id=$1`
	}
	if event == "milestone" {
		query = milestoneWebhookProjectionSQL
	}
	args := []any{id}
	if workspaceID != "" {
		if event == "user" {
			return nil, fmt.Errorf("%w: users are not workspace models", ErrPermanentWebhookTask)
		}
		query += ` AND model.workspace_id=$2`
		args = append(args, workspaceID)
	}
	err := store.pool.QueryRow(ctx, query, args...).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(raw, &fields); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	for key, value := range fields {
		if strings.HasSuffix(key, "_id") {
			base := strings.TrimSuffix(key, "_id")
			if webhookSerializerRelation(event, base) {
				fields[base] = value
				delete(fields, key)
			}
		}
	}
	if event == "user" {
		fields = pickJSONFields(fields, "id", "first_name", "last_name", "email", "avatar", "display_name")
		avatar := fields["avatar"]
		if len(avatar) == 0 {
			fields["avatar"] = json.RawMessage("null")
		}
		fields["avatar_url"] = json.RawMessage("null")
		if email, ok := fields["email"]; ok {
			fields["email"] = email
		}
	}
	if event == "module" {
		members, moduleMembersErr := store.moduleMembers(ctx, id)
		if moduleMembersErr != nil {
			return nil, moduleMembersErr
		}
		encoded, _ := json.Marshal(members)
		fields["members"] = encoded
	}
	encoded, err := json.Marshal(fields)
	return json.RawMessage(encoded), err
}

func (store *PostgresWebhookTaskStore) moduleMembers(ctx context.Context, moduleID string) ([]string, error) {
	rows, err := store.pool.Query(ctx, `SELECT DISTINCT mm.member_id::text FROM module_members mm WHERE mm.module_id=$1 AND mm.deleted_at IS NULL ORDER BY mm.member_id::text`, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		members = append(members, id)
	}
	return members, rows.Err()
}

var webhookModelTables = map[string]string{
	"project": "projects", "issue": "issues", "cycle": "cycles", "module": "modules",
	"cycle_issue": "cycle_issues", "module_issue": "module_issues", "issue_comment": "issue_comments",
	"user": "users", "intake_issue": "intake_issues", "milestone": "milestones",
}

func webhookSerializerRelation(event, field string) bool {
	common := map[string]map[string]bool{
		"project":       {"workspace": true, "created_by": true, "updated_by": true, "default_assignee": true, "project_lead": true, "estimate": true, "default_state": true},
		"issue":         {"workspace": true, "project": true, "created_by": true, "updated_by": true, "parent": true, "state": true, "estimate_point": true, "type": true},
		"cycle":         {"workspace": true, "project": true, "created_by": true, "updated_by": true, "owned_by": true},
		"module":        {"workspace": true, "project": true, "created_by": true, "updated_by": true},
		"cycle_issue":   {"workspace": true, "project": true, "cycle": true, "issue": true, "created_by": true, "updated_by": true},
		"module_issue":  {"workspace": true, "project": true, "module": true, "issue": true, "created_by": true, "updated_by": true},
		"issue_comment": {"workspace": true, "project": true, "issue": true, "created_by": true, "updated_by": true},
		"intake_issue":  {"workspace": true, "project": true, "issue": true, "created_by": true, "updated_by": true, "intake": true},
	}
	return common[event][field]
}

func pickJSONFields(source map[string]json.RawMessage, names ...string) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(names))
	for _, name := range names {
		if value, ok := source[name]; ok {
			result[name] = value
		}
	}
	return result
}

func decodeTaskKwargs(raw map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	return raw, nil
}

type orderedJSONField struct {
	key   string
	value json.RawMessage
}

func orderedJSONObject(raw json.RawMessage) ([]orderedJSONField, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	if raw[0] == '"' {
		var text string
		if err := json.Unmarshal(raw, &text); err == nil && json.Valid([]byte(text)) {
			raw = []byte(text)
		}
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("value must be a JSON object")
	}
	var result []orderedJSONField
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, errors.New("object key is not a string")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		result = append(result, orderedJSONField{key: key, value: value})
	}
	return result, nil
}

func optionalJSONObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	if len(raw) > 1 && raw[0] == '"' {
		var text string
		if err := json.Unmarshal(raw, &text); err == nil && json.Valid([]byte(text)) {
			raw = []byte(text)
		}
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(raw, &result); err != nil || result == nil {
		return nil, errors.New("value must be a JSON object or null")
	}
	return result, nil
}

func jsonValuesEqual(left, right json.RawMessage) bool {
	var a, b any
	if json.Unmarshal(left, &a) != nil || json.Unmarshal(right, &b) != nil {
		return string(left) == string(right)
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func nonemptyRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	return raw
}

func stringValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	return string(raw)
}

func mapWebhookAction(action string) string {
	switch strings.ToUpper(action) {
	case "POST":
		return "create"
	case "PATCH", "PUT":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return action
	}
}

func headerString(headers http.Header) string {
	encoded, _ := marshalWebhookJSON(headers, true)
	return string(encoded)
}

func marshalWebhookJSON(value any, spaced bool) ([]byte, error) {
	var buffer strings.Builder
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	compact := []byte(strings.TrimSuffix(buffer.String(), "\n"))
	compact = escapeNonASCIIJSON(compact)
	if !spaced {
		return compact, nil
	}
	var output strings.Builder
	quoted, escaped := false, false
	for _, b := range compact {
		if quoted {
			output.WriteByte(b)
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				quoted = false
			}
			continue
		}
		switch b {
		case '"':
			quoted = true
			output.WriteByte(b)
		case ',', ':':
			output.WriteByte(b)
			output.WriteByte(' ')
		default:
			output.WriteByte(b)
		}
	}
	return []byte(output.String()), nil
}

func escapeNonASCIIJSON(input []byte) []byte {
	var output strings.Builder
	quoted, escaped := false, false
	for i := 0; i < len(input); {
		b := input[i]
		if quoted && b >= 0x80 {
			r, size := utf8.DecodeRune(input[i:])
			if size == 1 {
				output.WriteByte(b)
				i++
				continue
			}
			if r <= 0xffff {
				fmt.Fprintf(&output, "\\u%04x", r)
			} else {
				r -= 0x10000
				fmt.Fprintf(&output, "\\u%04x\\u%04x", 0xd800+(r>>10), 0xdc00+(r&0x3ff))
			}
			i += size
			continue
		}
		output.WriteByte(b)
		if quoted {
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				quoted = false
			}
		} else if b == '"' {
			quoted = true
		}
		i++
	}
	return []byte(output.String())
}
