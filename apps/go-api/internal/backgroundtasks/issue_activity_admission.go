package backgroundtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const issueActivityCeleryTaskName = "plane.bgtasks.issue_activities_task.issue_activity"

type IssueActivityAdmission struct{ dispatcher *IssueActivityDispatcher }

func NewIssueActivityAdmission(dispatcher *IssueActivityDispatcher) (*IssueActivityAdmission, error) {
	if dispatcher == nil {
		return nil, errors.New("issue activity dispatcher is required")
	}
	return &IssueActivityAdmission{dispatcher: dispatcher}, nil
}

func (admission *IssueActivityAdmission) AdmitIssueActivity(ctx context.Context, eventID string, input IssueActivityInput) error {
	if admission == nil || admission.dispatcher == nil {
		return errors.New("native issue activity admission is not configured")
	}
	if _, err := admission.dispatcher.Admit(ctx, eventID, input); err != nil {
		return fmt.Errorf("admit issue activity workflow: %w", err)
	}
	return nil
}

func (admission *IssueActivityAdmission) AdmitIssueActivityTask(ctx context.Context, eventID string, kwargs map[string]any) error {
	input, err := issueActivityInputFromTask(kwargs)
	if err != nil {
		return err
	}
	return admission.AdmitIssueActivity(ctx, eventID, input)
}

// TaskFallback is the existing publisher contract used by Plane repositories.
// The adapter routes only the issue activity task to durable execution and
// preserves the existing publisher for every other task and soft deletion.
type TaskFallback interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
}

type IssueActivityTaskAdmitter interface {
	AdmitIssueActivityTask(context.Context, string, map[string]any) error
}

type IssueActivityTaskPublisher struct {
	fallback  TaskFallback
	admission IssueActivityTaskAdmitter
}

func NewIssueActivityTaskPublisher(fallback TaskFallback, admission IssueActivityTaskAdmitter) (*IssueActivityTaskPublisher, error) {
	if fallback == nil || admission == nil {
		return nil, errors.New("task fallback and issue activity admission are required")
	}
	return &IssueActivityTaskPublisher{fallback: fallback, admission: admission}, nil
}

func (publisher *IssueActivityTaskPublisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if publisher == nil || publisher.fallback == nil || publisher.admission == nil {
		return errors.New("issue activity task publisher is not configured")
	}
	if name != issueActivityCeleryTaskName {
		return publisher.fallback.Task(ctx, name, args, kwargs)
	}
	return publisher.admission.AdmitIssueActivityTask(ctx, uuid.NewString(), kwargs)
}

func (publisher *IssueActivityTaskPublisher) SoftDelete(ctx context.Context, table, id, field string) error {
	if publisher == nil || publisher.fallback == nil {
		return errors.New("issue activity task publisher is not configured")
	}
	return publisher.fallback.SoftDelete(ctx, table, id, field)
}

func issueActivityInputFromTask(kwargs map[string]any) (IssueActivityInput, error) {
	if kwargs == nil {
		return IssueActivityInput{}, fmt.Errorf("%w: issue activity task payload is required", ErrPermanentIssueActivity)
	}
	input := IssueActivityInput{Type: stringFromTask(kwargs["type"]), ProjectID: stringFromTask(kwargs["project_id"]), WorkspaceID: stringFromTask(kwargs["workspace_id"]), IssueID: nullableStringFromTask(kwargs["issue_id"]), ActorID: nullableStringFromTask(kwargs["actor_id"]), Notification: boolFromTask(kwargs["notification"])}
	if value, ok := kwargs["subscriber"]; ok {
		subscriber := boolFromTask(value)
		input.Subscriber = &subscriber
	}
	if value, ok := kwargs["epoch"]; ok {
		var epoch float64
		raw, err := json.Marshal(value)
		if err != nil {
			return IssueActivityInput{}, fmt.Errorf("encode activity epoch: %w", err)
		}
		if err := json.Unmarshal(raw, &epoch); err != nil {
			return IssueActivityInput{}, fmt.Errorf("decode activity epoch: %w", err)
		}
		input.Epoch = epoch
	}
	input.Origin = stringFromTask(kwargs["origin"])
	var err error
	input.RequestedData, err = taskJSON(kwargs["requested_data"])
	if err != nil {
		return IssueActivityInput{}, fmt.Errorf("encode requested activity data: %w", err)
	}
	input.CurrentInstance, err = taskJSON(kwargs["current_instance"])
	if err != nil {
		return IssueActivityInput{}, fmt.Errorf("encode current activity data: %w", err)
	}
	return input, nil
}

func taskJSON(value any) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage("null"), nil
	}
	if text, ok := value.(string); ok {
		trimmed := json.RawMessage(text)
		if json.Valid(trimmed) {
			return trimmed, nil
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

func stringFromTask(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}
func nullableStringFromTask(value any) *string {
	if value == nil {
		return nil
	}
	text := stringFromTask(value)
	if text == "" || text == "None" {
		return nil
	}
	return &text
}
func boolFromTask(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "1"
	case float64:
		return typed != 0
	case int:
		return typed != 0
	default:
		return false
	}
}
