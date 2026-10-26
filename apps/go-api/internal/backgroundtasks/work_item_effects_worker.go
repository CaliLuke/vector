package backgroundtasks

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type workItemEffectStore interface {
	Drain(context.Context) (int, error)
}

// The database outbox owns pending work across process loss. This lifecycle-owned
// poller only admits it to existing durable workflows; no request owns its lifetime.
type workItemEffectWorker struct {
	store  workItemEffectStore
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func (w *workItemEffectWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil || w.store == nil {
		return errors.New("work item effect worker is unconfigured or already started")
	}
	ctx, w.cancel = context.WithCancel(ctx)
	w.done = make(chan struct{})
	go w.run(ctx)
	return nil
}

func (w *workItemEffectWorker) run(ctx context.Context) {
	defer close(w.done)
	for ctx.Err() == nil {
		count, err := w.store.Drain(ctx)
		delay := 5 * time.Second
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.ErrorContext(ctx, "Work item effect admission polling failed", "error", err)
			delay = time.Minute
		} else if count == 50 {
			continue
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (w *workItemEffectWorker) Shutdown(ctx context.Context) error {
	w.mu.Lock()
	cancel, done := w.cancel, w.done
	w.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
