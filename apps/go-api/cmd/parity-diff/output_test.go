package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/parity"
)

func TestParityMatchOutputFailureIsNotSuccess(t *testing.T) {
	var stderr bytes.Buffer
	exitCode := reportComparison(&rejectedPromptWriter{}, &stderr, "", nil)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; want I/O failure", exitCode)
	}
	if stderr.Len() == 0 {
		t.Fatal("output failure did not produce a diagnostic")
	}
}

func TestParityMatchWritesSuccess(t *testing.T) {
	var stdout bytes.Buffer
	if exitCode := reportComparison(&stdout, io.Discard, "", nil); exitCode != 0 {
		t.Fatalf("exit code = %d; want success", exitCode)
	}
	if stdout.Len() == 0 {
		t.Fatal("successful comparison did not produce output")
	}
}

func TestParityComparisonFailureDoesNotPrintSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failure := errors.New("snapshot unavailable")
	if code := reportComparison(&stdout, &stderr, "", failure); code != 1 {
		t.Fatalf("exit code = %d; want failure", code)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), failure.Error()) {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestParityMismatchAndOutputFailureAreBothReported(t *testing.T) {
	var stderr bytes.Buffer
	mismatch := &parity.MismatchError{Diff: "different transitions"}
	if code := reportComparison(&rejectedPromptWriter{}, &stderr, mismatch.Diff, mismatch); code != 1 {
		t.Fatalf("exit code = %d; want failure", code)
	}
	if !strings.Contains(stderr.String(), mismatch.Error()) || !strings.Contains(stderr.String(), io.ErrClosedPipe.Error()) {
		t.Fatalf("diagnostic lost a failure: %q", stderr.String())
	}
}

func TestParityDifferenceStillExitsNonzeroWithoutTypedMismatch(t *testing.T) {
	var stdout bytes.Buffer
	if code := reportComparison(&stdout, io.Discard, "different transitions", nil); code != 1 {
		t.Fatalf("exit code = %d; want difference failure", code)
	}
	if !strings.Contains(stdout.String(), "different transitions") {
		t.Fatal("difference output was lost")
	}
}

func TestParityUsageDiagnosticFailureExitsAsIOFailure(t *testing.T) {
	if code := run(nil, strings.NewReader(""), io.Discard, &rejectedPromptWriter{}); code != 1 {
		t.Fatalf("exit code = %d; want I/O failure", code)
	}
}
