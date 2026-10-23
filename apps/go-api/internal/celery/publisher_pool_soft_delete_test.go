package celery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPublisherPoolSoftDeleteForwardsRequestAndReturnsWorkerAfterError(t *testing.T) {
	failure := errors.New("broker confirmation failed")
	worker := &taskPublisherWorker{err: failure}
	pool := newPublisherPool([]publisherWorker{worker})
	t.Cleanup(func() { _ = pool.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.SoftDelete(ctx, "assets", "asset", "9e92c429-48d6-4aa0-b285-fcf0e4f266de"); !errors.Is(err, failure) {
		t.Fatalf("SoftDelete() error = %v, want confirmation failure", err)
	}
	if worker.app != "assets" || worker.model != "asset" || worker.id != "9e92c429-48d6-4aa0-b285-fcf0e4f266de" {
		t.Fatalf("worker received (%q, %q, %q), want original delete request", worker.app, worker.model, worker.id)
	}
	worker.err = nil
	if err := pool.SoftDelete(ctx, "assets", "asset", "9e92c429-48d6-4aa0-b285-fcf0e4f266de"); err != nil {
		t.Fatalf("SoftDelete() after transient worker error: %v", err)
	}
	if worker.calls != 2 {
		t.Fatalf("worker calls = %d, want 2", worker.calls)
	}
}

func TestPublisherPoolSoftDeleteReturnsClosedError(t *testing.T) {
	worker := &taskPublisherWorker{}
	pool := newPublisherPool([]publisherWorker{worker})
	ctx := context.Background()
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	if err := pool.SoftDelete(ctx, "assets", "asset", "9e92c429-48d6-4aa0-b285-fcf0e4f266de"); !errors.Is(err, errPublisherClosed) {
		t.Fatalf("closed pool SoftDelete() error = %v", err)
	}
	if worker.calls != 0 {
		t.Fatalf("closed pool dispatched a delete: worker calls = %d", worker.calls)
	}
}

func TestPublisherSoftDeleteRejectsInvalidUUIDBeforeAMQP(t *testing.T) {
	publisher, err := NewPublisher("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := publisher.Close(); err != nil {
			t.Errorf("close publisher: %v", err)
		}
	})
	if err := publisher.SoftDelete(context.Background(), "assets", "asset", "not-a-uuid"); err == nil {
		t.Fatal("SoftDelete() accepted invalid UUID")
	} else if !uuid.IsInvalidLengthError(err) {
		t.Fatalf("SoftDelete() error = %v, want UUID validation error", err)
	}
}
