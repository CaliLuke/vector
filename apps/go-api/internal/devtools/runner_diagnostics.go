package devtools

import (
	"fmt"
	"io"
	"sync"
)

// runnerDiagnostics serializes diagnostics from process readers and the log
// exporter. A failed diagnostic stream signals the runner to stop its children.
type runnerDiagnostics struct {
	mu      sync.Mutex
	writer  io.Writer
	failure error
	failed  chan struct{}
}

func newRunnerDiagnostics(writer io.Writer) *runnerDiagnostics {
	return &runnerDiagnostics{writer: writer, failed: make(chan struct{})}
}

func (d *runnerDiagnostics) printf(format string, args ...any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.failure != nil {
		return
	}
	if _, err := fmt.Fprintf(d.writer, format, args...); err != nil {
		d.failure = err
		close(d.failed)
	}
}

func (d *runnerDiagnostics) err() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.failure
}
