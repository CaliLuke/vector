package main

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type taskPublisherSpy struct {
	tasks []string
}

func (spy *taskPublisherSpy) Task(_ context.Context, name string, _ []any, _ map[string]any) error {
	spy.tasks = append(spy.tasks, name)
	return nil
}

func (*taskPublisherSpy) SoftDelete(context.Context, string, string, string) error { return nil }

type issueActivityAdmitterSpy struct {
	eventID string
	kwargs  map[string]any
}

func (spy *issueActivityAdmitterSpy) AdmitIssueActivityTask(_ context.Context, eventID string, kwargs map[string]any) error {
	spy.eventID = eventID
	spy.kwargs = kwargs
	return nil
}

func TestIssueActivityTaskPublisherFlagControlsNativeAdmission(t *testing.T) {
	ctx := context.Background()
	fallback := &taskPublisherSpy{}
	kwargs := map[string]any{"type": "issue.activity.updated"}

	t.Run("flag off preserves legacy publisher", func(t *testing.T) {
		publisher, err := issueActivityTaskPublisher(fallback, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := publisher.Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", nil, kwargs); err != nil {
			t.Fatal(err)
		}
		if len(fallback.tasks) != 1 {
			t.Fatalf("legacy publishes=%d, want 1", len(fallback.tasks))
		}
	})

	t.Run("flag on durably admits issue activity", func(t *testing.T) {
		fallback.tasks = nil
		admitter := &issueActivityAdmitterSpy{}
		publisher, err := issueActivityTaskPublisher(fallback, admitter)
		if err != nil {
			t.Fatal(err)
		}
		if err := publisher.Task(ctx, "plane.bgtasks.issue_activities_task.issue_activity", nil, kwargs); err != nil {
			t.Fatal(err)
		}
		if _, err := uuid.Parse(admitter.eventID); err != nil {
			t.Fatalf("native event identity %q is not a UUID: %v", admitter.eventID, err)
		}
		if admitter.kwargs["type"] != kwargs["type"] {
			t.Fatalf("native task payload = %#v, want %#v", admitter.kwargs, kwargs)
		}
		if len(fallback.tasks) != 0 {
			t.Fatalf("legacy publisher received %d tasks after native admission", len(fallback.tasks))
		}
	})

	t.Run("unrelated task remains on legacy publisher", func(t *testing.T) {
		admitter := &issueActivityAdmitterSpy{}
		publisher, err := issueActivityTaskPublisher(fallback, admitter)
		if err != nil {
			t.Fatal(err)
		}
		if err := publisher.Task(ctx, "plane.bgtasks.logger_task.process_logs", nil, nil); err != nil {
			t.Fatal(err)
		}
		if len(fallback.tasks) != 1 || fallback.tasks[0] != "plane.bgtasks.logger_task.process_logs" {
			t.Fatalf("legacy tasks = %v", fallback.tasks)
		}
		if admitter.eventID != "" {
			t.Fatalf("unrelated task reached native admission: %q", admitter.eventID)
		}
	})
}
