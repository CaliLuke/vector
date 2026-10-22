package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type WebhookTaskAdmitter interface {
	AdmitWebhookTask(context.Context, string, string, []any, map[string]any) error
}

type WebhookTaskAdmission struct{ dispatcher *WebhookTaskDispatcher }

func NewWebhookTaskAdmission(dispatcher *WebhookTaskDispatcher) (*WebhookTaskAdmission, error) {
	if dispatcher == nil {
		return nil, errors.New("webhook task dispatcher is required")
	}
	return &WebhookTaskAdmission{dispatcher: dispatcher}, nil
}

func (admission *WebhookTaskAdmission) AdmitWebhookTask(ctx context.Context, eventID, taskName string, args []any, kwargs map[string]any) error {
	if admission == nil || admission.dispatcher == nil {
		return errors.New("native webhook task admission is not configured")
	}
	switch taskName {
	case webhookModelActivityTask, webhookActivityCeleryTask, webhookSendCeleryTask:
	default:
		return fmt.Errorf("unsupported webhook producer task %q", taskName)
	}
	encoded, err := json.Marshal(kwargs)
	if err != nil {
		return fmt.Errorf("encode webhook task kwargs: %w", err)
	}
	var rawKwargs map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &rawKwargs); err != nil {
		return fmt.Errorf("decode webhook task kwargs: %w", err)
	}
	encodedArgs, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("encode webhook task args: %w", err)
	}
	var rawArgs []json.RawMessage
	if err := json.Unmarshal(encodedArgs, &rawArgs); err != nil {
		return fmt.Errorf("decode webhook task args: %w", err)
	}
	return admission.admit(ctx, WebhookTaskInput{EventID: eventID, TaskName: taskName, Args: rawArgs, Kwargs: rawKwargs})
}

func (admission *WebhookTaskAdmission) admit(ctx context.Context, input WebhookTaskInput) error {
	if _, err := uuid.Parse(input.EventID); err != nil {
		return fmt.Errorf("webhook event ID must be a UUID: %w", err)
	}
	if _, err := admission.dispatcher.Admit(ctx, input); err != nil {
		return fmt.Errorf("admit webhook workflow: %w", err)
	}
	return nil
}

// WebhookTaskPublisher intercepts the model_activity producer task and
// delegates all other Celery tasks to the configured fallback publisher.
type WebhookTaskPublisher struct {
	fallback  TaskFallback
	admission WebhookTaskAdmitter
}

func NewWebhookTaskPublisher(fallback TaskFallback, admission WebhookTaskAdmitter) (*WebhookTaskPublisher, error) {
	if fallback == nil || admission == nil {
		return nil, errors.New("task fallback and webhook task admission are required")
	}
	return &WebhookTaskPublisher{fallback: fallback, admission: admission}, nil
}

func (publisher *WebhookTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if publisher == nil || publisher.fallback == nil || publisher.admission == nil {
		return errors.New("webhook task publisher is not configured")
	}
	if name != webhookModelActivityTask && name != webhookActivityCeleryTask && name != webhookSendCeleryTask {
		return publisher.fallback.Task(ctx, name, args, kwargs)
	}
	return publisher.admission.AdmitWebhookTask(ctx, uuid.NewString(), name, args, kwargs)
}

func (publisher *WebhookTaskPublisher) SoftDelete(ctx context.Context, table, id, field string) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("webhook task publisher is not configured")
	}
	return publisher.fallback.SoftDelete(ctx, table, id, field)
}
