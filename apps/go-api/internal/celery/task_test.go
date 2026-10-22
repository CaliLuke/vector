package celery

import (
	"context"
	"strings"
	"testing"
)

func TestTaskRejectsInvalidNameBeforePublishing(t *testing.T) {
	var publisher *Publisher
	err := publisher.Task(context.Background(), "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "task name is required") {
		t.Fatalf("Task() error = %v, want required-name error", err)
	}
}

func TestTaskRejectsUnencodableArgumentsBeforePublishing(t *testing.T) {
	var publisher *Publisher
	err := publisher.Task(context.Background(), "plane.example.task", []any{make(chan int)}, nil)
	if err == nil || !strings.Contains(err.Error(), "encode Celery task") {
		t.Fatalf("Task() error = %v, want encoding error", err)
	}
}
