package celery

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	activitylog "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
)

type taskPublisherWorker struct {
	calls  int
	ctx    context.Context
	name   string
	args   []any
	kwargs map[string]any
	err    error
}

func (worker *taskPublisherWorker) Log(context.Context, activitylog.APIActivityLog) error { return nil }
func (worker *taskPublisherWorker) Close() error                                          { return nil }
func (worker *taskPublisherWorker) SoftDelete(context.Context, string, string, string) error {
	return nil
}
func (worker *taskPublisherWorker) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	worker.calls++
	worker.ctx, worker.name, worker.args, worker.kwargs = ctx, name, args, kwargs
	return worker.err
}

func TestPublisherPoolTaskForwardsRequestAndReturnsWorkerAfterError(t *testing.T) {
	failure := errors.New("broker confirmation failed")
	worker := &taskPublisherWorker{err: failure}
	pool := newPublisherPool([]publisherWorker{worker})
	t.Cleanup(func() { _ = pool.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	args := []any{"asset-id"}
	kwargs := map[string]any{"workspace_id": "workspace-id"}
	if err := pool.Task(ctx, "plane.assets.delete", args, kwargs); !errors.Is(err, failure) {
		t.Fatalf("Task error = %v, want confirmation failure", err)
	}
	if worker.ctx != ctx || worker.name != "plane.assets.delete" || !reflect.DeepEqual(worker.args, args) || !reflect.DeepEqual(worker.kwargs, kwargs) {
		t.Fatal("pool changed task context, name, or payload")
	}
	worker.err = nil
	if err := pool.Task(ctx, "plane.assets.delete", args, kwargs); err != nil {
		t.Fatal(err)
	}
	if worker.calls != 2 {
		t.Fatalf("worker calls = %d, want 2", worker.calls)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	if err := pool.Task(ctx, "plane.assets.delete", args, kwargs); !errors.Is(err, errPublisherClosed) {
		t.Fatalf("closed Task error = %v", err)
	}
	if worker.calls != 2 {
		t.Fatal("closed pool dispatched a task")
	}
}

func (worker *blockingPublisherWorker) Task(ctx context.Context, _ string, _ []any, _ map[string]any) error {
	return worker.Log(ctx, activitylog.APIActivityLog{})
}

func TestPublisherPoolTaskSharesBoundedLanesWithLogs(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	pool := newPublisherPool([]publisherWorker{&blockingPublisherWorker{entered: entered, release: release}})
	t.Cleanup(func() { _ = pool.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	completed := make(chan error, 1)
	go func() { completed <- pool.Log(ctx, activitylog.APIActivityLog{}) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	waiting, stop := context.WithCancel(ctx)
	stop()
	if err := pool.Task(waiting, "plane.assets.delete", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting Task error = %v", err)
	}
	close(release)
	if err := <-completed; err != nil {
		t.Fatal(err)
	}
	if err := pool.Task(ctx, "plane.assets.delete", nil, nil); err != nil {
		t.Fatal(err)
	}
}
