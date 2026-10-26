package backgroundtasks

import (
	"context"
	"testing"
	"time"
)

type blockingEffectStore struct{ started chan struct{} }

func (s blockingEffectStore) Drain(ctx context.Context) (int, error) {
	close(s.started)
	<-ctx.Done()
	return 0, ctx.Err()
}

func TestEffectWorkerShutdownJoinsInFlightAdmission(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	worker := &workItemEffectWorker{store: blockingEffectStore{started: make(chan struct{})}}
	if err := worker.Start(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-worker.store.(blockingEffectStore).started:
	case <-ctx.Done():
		t.Fatal("worker did not resume committed effects on startup")
	}
	if err := worker.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-worker.done:
	default:
		t.Fatal("shutdown returned while admission was still running")
	}
	if err := worker.Start(ctx); err == nil {
		t.Fatal("duplicate lifecycle start admitted another worker")
	}
}
