package clioutput

import (
	"bytes"
	"io"
	"testing"
)

type rejectedDiagnostic struct{ attempts int }

func (output *rejectedDiagnostic) Write([]byte) (int, error) {
	output.attempts++
	return 0, io.ErrClosedPipe
}

func TestReportPreservesUsageExitWhenDiagnosticIsDelivered(t *testing.T) {
	var output bytes.Buffer
	code := Report(&output, 2, "invalid option %q\n", "example")
	if code != 2 {
		t.Fatalf("exit code = %d; want usage failure", code)
	}
	if !bytes.Contains(output.Bytes(), []byte("example")) {
		t.Fatal("diagnostic lost the supplied option")
	}
}

func TestReportFailedDeliveryIsAnIOFailureWithoutRetry(t *testing.T) {
	output := &rejectedDiagnostic{}
	if code := Report(output, 2, "invalid option\n"); code != 1 {
		t.Fatalf("exit code = %d; want I/O failure", code)
	}
	if output.attempts != 1 {
		t.Fatalf("write attempts = %d; want one", output.attempts)
	}
}
