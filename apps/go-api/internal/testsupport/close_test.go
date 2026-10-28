package testsupport_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

type cleanupReporter struct{ messages []string }

func (*cleanupReporter) Helper() {}
func (reporter *cleanupReporter) Errorf(format string, args ...any) {
	reporter.messages = append(reporter.messages, fmt.Sprintf(format, args...))
}

type cleanupResource struct {
	err   error
	calls int
}

func (resource *cleanupResource) Close() error {
	resource.calls++
	return resource.err
}

type contextualCleanupResource struct {
	cleanupResource
	ctx context.Context
}

func (resource *contextualCleanupResource) Close(ctx context.Context) error {
	resource.ctx = ctx
	return resource.cleanupResource.Close()
}

func TestCloseClosesExactlyOnce(t *testing.T) {
	reporter := &cleanupReporter{}
	resource := &cleanupResource{}
	testsupport.Close(reporter, resource)
	require.Equal(t, 1, resource.calls)
	require.Empty(t, reporter.messages)
}

func TestCloseReportsFailure(t *testing.T) {
	reporter := &cleanupReporter{}
	closeErr := errors.New("cleanup connection lost")
	resource := &cleanupResource{err: closeErr}
	testsupport.Close(reporter, resource)
	require.Equal(t, 1, resource.calls)
	require.Len(t, reporter.messages, 1)
	require.Contains(t, reporter.messages[0], closeErr.Error())
}

func TestCloseContextPreservesCallerContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reporter := &cleanupReporter{}
	resource := &contextualCleanupResource{}
	testsupport.CloseContext(reporter, resource, ctx)
	require.Equal(t, 1, resource.calls)
	require.Same(t, ctx, resource.ctx)
	require.Empty(t, reporter.messages)
}

func TestCloseContextReportsFailure(t *testing.T) {
	reporter := &cleanupReporter{}
	closeErr := errors.New("cleanup connection lost")
	resource := &contextualCleanupResource{cleanupResource: cleanupResource{err: closeErr}}
	testsupport.CloseContext(reporter, resource, t.Context())
	require.Equal(t, 1, resource.calls)
	require.Len(t, reporter.messages, 1)
	require.Contains(t, reporter.messages[0], closeErr.Error())
}
