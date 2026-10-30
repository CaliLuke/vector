package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/require"
)

type metadataWriter func([]byte) (int, error)

func (write metadataWriter) Write(p []byte) (int, error) { return write(p) }

type metadataRunState struct {
	admissions, waits, closes       int
	admissionErr, waitErr, closeErr error
	result                          *selvedge.WorkflowResult[struct{}]
	waitedID                        string
}

func (s *metadataRunState) admit() (string, bool, error) {
	s.admissions++
	return "stable-run", true, s.admissionErr
}
func (s *metadataRunState) wait(id string) (*selvedge.WorkflowResult[struct{}], error) {
	s.waits++
	s.waitedID = id
	return s.result, s.waitErr
}
func (s *metadataRunState) close() error { s.closes++; return s.closeErr }

func TestMetadataAdmissionOutputFailureDoesNotWaitOrReadmit(t *testing.T) {
	cleanup := errors.New("shutdown failed")
	s := &metadataRunState{closeErr: cleanup, result: &selvedge.WorkflowResult[struct{}]{InstanceID: "stable-run", Status: "completed"}}
	writes := 0
	err := runMetadata(metadataWriter(func(p []byte) (int, error) {
		writes++
		if writes == 1 {
			return 0, io.ErrClosedPipe
		}
		return len(p), nil
	}), s.admit, s.wait, s.close)
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.ErrorIs(t, err, cleanup)
	require.Equal(t, 1, s.admissions)
	require.Zero(t, s.waits)
	require.Equal(t, 1, s.closes)
	require.Equal(t, 1, writes)
}

func TestMetadataPrimaryFailureRetainsCleanupFailure(t *testing.T) {
	for _, phase := range []string{"admission", "wait"} {
		t.Run(phase, func(t *testing.T) {
			primary, cleanup := errors.New("operation failed"), errors.New("shutdown failed")
			s := &metadataRunState{closeErr: cleanup}
			if phase == "admission" {
				s.admissionErr = primary
			} else {
				s.waitErr = primary
			}
			err := runMetadata(io.Discard, s.admit, s.wait, s.close)
			require.ErrorIs(t, err, primary)
			require.ErrorIs(t, err, cleanup)
			require.Equal(t, 1, s.admissions)
			require.Equal(t, 1, s.closes)
		})
	}
}

func TestMetadataResultOutputFailurePreservesWorkflowAndCleanupCauses(t *testing.T) {
	workflow, cleanup := errors.New("metadata workflow failed"), errors.New("shutdown failed")
	s := &metadataRunState{closeErr: cleanup, result: &selvedge.WorkflowResult[struct{}]{InstanceID: "stable-run", Status: "failed", Error: workflow}}
	writes := 0
	err := runMetadata(metadataWriter(func(p []byte) (int, error) {
		writes++
		if writes == 2 {
			return 0, io.ErrClosedPipe
		}
		return len(p), nil
	}), s.admit, s.wait, s.close)
	require.ErrorIs(t, err, io.ErrClosedPipe)
	require.ErrorIs(t, err, workflow)
	require.ErrorIs(t, err, cleanup)
	require.Equal(t, 1, s.admissions)
	require.Equal(t, 1, s.waits)
	require.Equal(t, 1, s.closes)
}

func TestMetadataCompletedRunKeepsStableIdentityAndReportsCleanupFailure(t *testing.T) {
	cleanup := errors.New("shutdown failed")
	s := &metadataRunState{closeErr: cleanup, result: &selvedge.WorkflowResult[struct{}]{InstanceID: "stable-run", Status: "completed"}}
	var output bytes.Buffer
	err := runMetadata(&output, s.admit, s.wait, s.close)
	require.ErrorIs(t, err, cleanup)
	require.Equal(t, "stable-run", s.waitedID)
	require.Contains(t, output.String(), `"status":"completed"`)
	require.Equal(t, 1, s.admissions)
	require.Equal(t, 1, s.waits)
	require.Equal(t, 1, s.closes)
}

func TestMetadataReadyOutputFailureReturnsBeforeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cleanup := errors.New("shutdown failed")
	closed := 0
	done := make(chan error, 1)
	go func() {
		done <- runMetadataWorker(ctx, metadataWriter(func([]byte) (int, error) { return 0, io.ErrClosedPipe }), func() error { closed++; return cleanup })
	}()
	select {
	case err := <-done:
		require.ErrorIs(t, err, io.ErrClosedPipe)
		require.ErrorIs(t, err, cleanup)
		require.NoError(t, ctx.Err())
		require.Equal(t, 1, closed)
	case <-time.After(2 * time.Second):
		t.Fatal("failed readiness output left the worker waiting for cancellation")
	}
}

func TestMetadataWorkerCancellationReportsCleanupFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cleanup := errors.New("shutdown failed")
	closed := 0
	err := runMetadataWorker(ctx, metadataWriter(func(p []byte) (int, error) { cancel(); return len(p), nil }), func() error { closed++; return cleanup })
	require.ErrorIs(t, err, cleanup)
	require.Equal(t, 1, closed)
}

func TestMetadataCleanupClosesPoolAfterShutdownFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cleanup := errors.New("shutdown failed")
	var order []string
	err := closeMetadataApp(ctx, func(ctx context.Context) error {
		require.NoError(t, ctx.Err())
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.Positive(t, time.Until(deadline))
		require.LessOrEqual(t, time.Until(deadline), 30*time.Second)
		order = append(order, "shutdown")
		return cleanup
	}, func() { order = append(order, "pool") })
	require.ErrorIs(t, err, cleanup)
	require.Equal(t, []string{"shutdown", "pool"}, order)
}

func TestMetadataCompletedRunReturnsSuccess(t *testing.T) {
	s := &metadataRunState{result: &selvedge.WorkflowResult[struct{}]{InstanceID: "stable-run", Status: "completed"}}
	var output bytes.Buffer
	require.NoError(t, runMetadata(&output, s.admit, s.wait, s.close))
	require.Equal(t, "stable-run", s.waitedID)
	require.Equal(t, 1, s.admissions)
	require.Equal(t, 1, s.waits)
	require.Equal(t, 1, s.closes)
	require.Contains(t, output.String(), `"status":"completed"`)
}

func TestMetadataUnsuccessfulStatusCannotBecomeSuccessfulExit(t *testing.T) {
	s := &metadataRunState{result: &selvedge.WorkflowResult[struct{}]{InstanceID: "stable-run", Status: "failed"}}
	err := runMetadata(io.Discard, s.admit, s.wait, s.close)
	require.ErrorContains(t, err, "failed")
	require.Equal(t, 1, s.admissions)
	require.Equal(t, 1, s.waits)
	require.Equal(t, 1, s.closes)
}
