package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type lifecycleWorker struct {
	startErr, shutdownErr error
	starts, shutdowns     int
	cleanupErr            error
	cleanupDeadline       time.Time
	cleanupBounded        bool
}

func (worker *lifecycleWorker) Start(context.Context) error {
	worker.starts++
	return worker.startErr
}
func (worker *lifecycleWorker) Shutdown(ctx context.Context) error {
	worker.shutdowns++
	worker.cleanupErr = ctx.Err()
	worker.cleanupDeadline, worker.cleanupBounded = ctx.Deadline()
	return worker.shutdownErr
}

type readyWriter func([]byte) (int, error)

func (write readyWriter) Write(value []byte) (int, error) { return write(value) }

func TestWorkerStartupFailureRetainsCleanupFailure(t *testing.T) {
	startFailure := errors.New("schedule registration failed")
	cleanupFailure := errors.New("worker drain failed")
	worker := &lifecycleWorker{startErr: startFailure, shutdownErr: cleanupFailure}
	var output bytes.Buffer
	err := runWorker(t.Context(), worker, &output)
	if !errors.Is(err, startFailure) || !errors.Is(err, cleanupFailure) {
		t.Fatalf("error lost a cause: %v", err)
	}
	if worker.starts != 1 || worker.shutdowns != 1 || output.Len() != 0 {
		t.Fatalf("worker=%+v output=%q", worker, output.String())
	}
	if !worker.cleanupBounded || worker.cleanupErr != nil {
		t.Fatalf("cleanup context not bounded and usable: %+v", worker)
	}
}

func TestWorkerReadyWriteFailureStopsWithoutWaitingForCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	cleanupFailure := errors.New("worker drain failed")
	worker := &lifecycleWorker{shutdownErr: cleanupFailure}
	result := make(chan error, 1)
	go func() {
		result <- runWorker(ctx, worker, readyWriter(func([]byte) (int, error) { return 0, io.ErrClosedPipe }))
	}()
	select {
	case err := <-result:
		if !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, cleanupFailure) {
			t.Fatalf("error lost a cause: %v", err)
		}
		if worker.starts != 1 || worker.shutdowns != 1 {
			t.Fatalf("worker lifecycle repeated: %+v", worker)
		}
		if ctx.Err() != nil {
			t.Fatal("worker only returned after caller cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker waited for caller cancellation after failed readiness output")
	}
}

func TestWorkerCancellationUsesUsableBoundedCleanupContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	worker := &lifecycleWorker{}
	var output bytes.Buffer
	err := runWorker(ctx, worker, readyWriter(func(value []byte) (int, error) {
		written, writeErr := output.Write(value)
		cancel()
		return written, writeErr
	}))
	if err != nil {
		t.Fatal(err)
	}
	if worker.starts != 1 || worker.shutdowns != 1 || output.Len() == 0 {
		t.Fatalf("worker=%+v output=%q", worker, output.String())
	}
	if worker.cleanupErr != nil || !worker.cleanupBounded || !worker.cleanupDeadline.After(time.Now()) {
		t.Fatalf("cleanup context inherited cancellation or lost deadline: %+v", worker)
	}
}

func TestWorkerFailedStartupCleansUpAfterParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	worker := &lifecycleWorker{startErr: context.Canceled}
	var output bytes.Buffer
	err := runWorker(ctx, worker, &output)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("startup error lost cancellation: %v", err)
	}
	if worker.shutdowns != 1 || worker.cleanupErr != nil || !worker.cleanupBounded {
		t.Fatalf("cleanup inherited cancelled startup: %+v", worker)
	}
	if output.Len() != 0 {
		t.Fatal("failed startup announced readiness")
	}
}

func TestWorkerCancellationRetainsShutdownFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	failure := errors.New("worker drain failed")
	worker := &lifecycleWorker{shutdownErr: failure}
	err := runWorker(ctx, worker, readyWriter(func(value []byte) (int, error) {
		cancel()
		return len(value), nil
	}))
	if !errors.Is(err, failure) {
		t.Fatalf("shutdown failure lost: %v", err)
	}
	if worker.starts != 1 || worker.shutdowns != 1 || worker.cleanupErr != nil || !worker.cleanupBounded {
		t.Fatalf("invalid shutdown: %+v", worker)
	}
}
