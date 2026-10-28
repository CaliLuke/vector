package parity

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompareTransitionsEqual(t *testing.T) {
	before := testSnapshot()
	after := testSnapshot(json.RawMessage(`{"id":"one","name":"safe"}`))
	option := DeclaredNormalizationsOption(Normalizations{})
	source, err := NewRuntimeTransition(before, after, testResponse(`{"ok":true}`), []string{"users"})
	if err != nil {
		t.Fatalf("source NewRuntimeTransition() error = %v", err)
	}
	target, err := NewRuntimeTransition(before, after, testResponse(`{"ok":true}`), []string{"users"})
	if err != nil {
		t.Fatalf("target NewRuntimeTransition() error = %v", err)
	}
	diff, err := CompareTransitions(source, target, option)
	if err != nil || diff != "" {
		t.Fatalf("CompareTransitions() diff = %q, error = %v", diff, err)
	}
}

func TestCompareTransitionsUsesSafeGoCmpDiff(t *testing.T) {
	before := testSnapshot()
	option := DeclaredNormalizationsOption(Normalizations{})
	source, err := NewRuntimeTransition(
		before,
		testSnapshot(json.RawMessage(`{"id":"one","password":"source-secret"}`)),
		testResponse(`{"token":"source-response-secret"}`),
		[]string{"users"},
	)
	if err != nil {
		t.Fatalf("source NewRuntimeTransition() error = %v", err)
	}
	target, err := NewRuntimeTransition(
		before,
		testSnapshot(json.RawMessage(`{"id":"one","password":"target-secret"}`)),
		testResponse(`{"token":"target-response-secret"}`),
		[]string{"users"},
	)
	if err != nil {
		t.Fatalf("target NewRuntimeTransition() error = %v", err)
	}
	diff, err := CompareTransitions(source, target, option)
	if err == nil {
		t.Fatal("CompareTransitions() error = nil, want mismatch")
	}
	for _, secret := range []string{
		"source-secret",
		"target-secret",
		"source-response-secret",
		"target-response-secret",
	} {
		if strings.Contains(diff, secret) {
			t.Fatalf("CompareTransitions() leaked %q in %q", secret, diff)
		}
	}
	if !strings.Contains(diff, "Fingerprint") || !strings.Contains(diff, "users") {
		t.Fatalf("CompareTransitions() diff = %q, want go-cmp table and fingerprint locations", diff)
	}
}

func TestNewRuntimeTransitionRejectsUnexpectedWrite(t *testing.T) {
	_, err := NewRuntimeTransition(
		testSnapshot(),
		testSnapshot(json.RawMessage(`{"id":"one"}`)),
		testResponse(`null`),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "unexpected changed tables") {
		t.Fatalf("NewRuntimeTransition() error = %v, want unexpected changed tables", err)
	}
}

func TestNewRuntimeTransitionDetectsWriteInNormalizedField(t *testing.T) {
	_, err := NewRuntimeTransition(
		testSnapshot(json.RawMessage(`{"id":"one","updated_at":"before"}`)),
		testSnapshot(json.RawMessage(`{"id":"one","updated_at":"after"}`)),
		testResponse(`null`),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "users") {
		t.Fatalf("NewRuntimeTransition() error = %v, want exact write detection", err)
	}

	normalizations := Normalizations{VolatileFields: []VolatileFieldRule{{
		Table: "users", Column: "updated_at", Replacement: json.RawMessage(`"<timestamp>"`),
	}}}
	if validationErr := normalizations.Validate(); validationErr != nil {
		t.Fatalf("Validate() error = %v", validationErr)
	}
	option := DeclaredNormalizationsOption(normalizations)
	source, err := NewRuntimeTransition(
		testSnapshot(json.RawMessage(`{"id":"one","updated_at":"before"}`)),
		testSnapshot(json.RawMessage(`{"id":"one","updated_at":"after"}`)),
		testResponse(`null`),
		[]string{"users"},
	)
	if err != nil {
		t.Fatalf("source NewRuntimeTransition() error = %v", err)
	}
	target, err := NewRuntimeTransition(
		testSnapshot(json.RawMessage(`{"id":"one","updated_at":"different-before"}`)),
		testSnapshot(json.RawMessage(`{"id":"one","updated_at":"different-after"}`)),
		testResponse(`null`),
		[]string{"users"},
	)
	if err != nil {
		t.Fatalf("target NewRuntimeTransition() error = %v", err)
	}
	if diff, err := CompareTransitions(source, target, option); err != nil || diff != "" {
		t.Fatalf("CompareTransitions() diff = %q, error = %v", diff, err)
	}
}

func TestCompareTransitionsResponseDiffIsSafe(t *testing.T) {
	snapshot := testSnapshot()
	source, err := NewRuntimeTransition(
		snapshot,
		snapshot,
		CapturedResponse{
			Status:  201,
			Headers: map[string]json.RawMessage{"Authorization": json.RawMessage(`"source-header-secret"`)},
			Cookies: map[string]json.RawMessage{"session-id": json.RawMessage(`"source-cookie-secret"`)},
			Body:    json.RawMessage(`{"token":"source-body-secret"}`),
		},
		nil,
	)
	if err != nil {
		t.Fatalf("source NewRuntimeTransition() error = %v", err)
	}
	target, err := NewRuntimeTransition(
		snapshot,
		snapshot,
		CapturedResponse{
			Status:  200,
			Headers: map[string]json.RawMessage{"Authorization": json.RawMessage(`"target-header-secret"`)},
			Cookies: map[string]json.RawMessage{"session-id": json.RawMessage(`"target-cookie-secret"`)},
			Body:    json.RawMessage(`{"token":"target-body-secret"}`),
		},
		nil,
	)
	if err != nil {
		t.Fatalf("target NewRuntimeTransition() error = %v", err)
	}
	diff, err := CompareTransitions(source, target, DeclaredNormalizationsOption(Normalizations{}))
	if err == nil {
		t.Fatal("CompareTransitions() error = nil, want response mismatch")
	}
	for _, secret := range []string{"source-header-secret", "source-cookie-secret", "source-body-secret", "target-header-secret", "target-cookie-secret", "target-body-secret"} {
		if strings.Contains(diff, secret) {
			t.Fatalf("CompareTransitions() leaked %q in %q", secret, diff)
		}
	}
	for _, location := range []string{"Status", "Authorization", "session-id", "BodySHA256"} {
		if !strings.Contains(diff, location) {
			t.Fatalf("CompareTransitions() diff = %q, want location %q", diff, location)
		}
	}
}

func testSnapshot(rows ...json.RawMessage) Snapshot {
	return Snapshot{Tables: []TableDump{{
		Name:       "users",
		Columns:    []Column{{Name: "id", DataType: "uuid"}},
		PrimaryKey: []string{"id"},
		Rows:       rows,
	}}}
}

func testResponse(body string) CapturedResponse {
	return CapturedResponse{Status: 200, Headers: map[string]json.RawMessage{}, Cookies: map[string]json.RawMessage{}, Body: json.RawMessage(body)}
}
