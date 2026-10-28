package testsupport

import (
	"context"
	"io"
)

// errorReporter is the part of testing.TB needed to report cleanup failures.
type errorReporter interface {
	Helper()
	Errorf(string, ...any)
}

// Close closes a test-owned resource now and reports failure without interrupting
// other cleanup. It can also be deferred without capturing a reassigned variable.
func Close(t errorReporter, resource io.Closer) {
	t.Helper()
	if err := resource.Close(); err != nil {
		t.Errorf("close test resource: %v", err)
	}
}

// CloseContext preserves the caller's cleanup context for resources such as
// PostgreSQL connections. It closes now; the caller owns defer/cleanup ordering.
func CloseContext(t errorReporter, resource interface{ Close(context.Context) error }, ctx context.Context) {
	t.Helper()
	if err := resource.Close(ctx); err != nil {
		t.Errorf("close test resource: %v", err)
	}
}
