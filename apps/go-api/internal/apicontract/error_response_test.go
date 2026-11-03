package apicontract

import (
	"encoding/json"
	"errors"
	"testing"
)

type stringDetailError struct {
	Detail string `json:"detail"`
}

func (e *stringDetailError) Error() string { return e.Detail }

func TestProjectErrorRejectsIncompatibleTransportFields(t *testing.T) {
	response := &stringDetailError{}
	err := ProjectError(map[string]any{"detail": []string{"invalid"}}, response)
	var mismatch *json.UnmarshalTypeError
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected conversion failure, got %v", err)
	}
}
