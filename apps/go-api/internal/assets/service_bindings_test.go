package assets

import (
	"errors"
	"testing"
)

type bindingTestResult struct{ value string }

func TestCastBindingResultReturnsTypedValue(t *testing.T) {
	want := &bindingTestResult{value: "ok"}
	got, err := castBindingResult[*bindingTestResult]("ReadAsset", want)
	if err != nil {
		t.Fatalf("castBindingResult() error = %v", err)
	}
	if got != want {
		t.Fatalf("castBindingResult() = %p, want %p", got, want)
	}
}

func TestCastBindingResultAllowsNilResult(t *testing.T) {
	got, err := castBindingResult[*bindingTestResult]("ReadAsset", nil)
	if err != nil {
		t.Fatalf("castBindingResult() error = %v", err)
	}
	if got != nil {
		t.Fatalf("castBindingResult() = %v, want nil", got)
	}
}

func TestCastBindingResultRejectsNilForNonNilableType(t *testing.T) {
	_, err := castBindingResult[int]("ReadAsset", nil)
	var typeErr *BindingResultTypeError
	if !errors.As(err, &typeErr) {
		t.Fatalf("castBindingResult() error = %T, want *BindingResultTypeError", err)
	}
}

func TestCastBindingResultRejectsMismatchedType(t *testing.T) {
	_, err := castBindingResult[*bindingTestResult]("ReadAsset", "wrong result")
	var typeErr *BindingResultTypeError
	if !errors.As(err, &typeErr) {
		t.Fatalf("castBindingResult() error = %T, want *BindingResultTypeError", err)
	}
	if typeErr.Method != "ReadAsset" {
		t.Fatalf("error method = %q, want ReadAsset", typeErr.Method)
	}
}
