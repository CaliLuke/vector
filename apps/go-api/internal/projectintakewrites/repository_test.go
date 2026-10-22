package projectintakewrites

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreatedContainerResponseOmitsReadOnlyPendingIssueCount(t *testing.T) {
	got, err := createdContainerResponse(json.RawMessage(`{"id":"container-1","pending_issue_count":0,"name":"Inbox"}`))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(got, &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["pending_issue_count"]; ok {
		t.Fatalf("create response includes read-only pending_issue_count: %s", got)
	}
	if string(body["name"]) != `"Inbox"` {
		t.Fatalf("create response lost writable fields: %s", got)
	}
}

type missingIntakeDatabase struct{}

func (missingIntakeDatabase) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	if strings.HasPrefix(query, "SELECT EXISTS") {
		return missingIntakeRow{}
	}
	return missingIntakeRow{err: pgx.ErrNoRows}
}

func (missingIntakeDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

type missingIntakeRow struct{ err error }

func (row missingIntakeRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	value, ok := dest[0].(*bool)
	if !ok {
		return errors.New("unexpected scan destination")
	}
	*value = false
	return nil
}

func TestMissingIntakeWriteResponsesMatchDRFContracts(t *testing.T) {
	repository, err := NewRepository(missingIntakeDatabase{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = repository.Patch(context.Background(), "user", "workspace", "project", "missing", nil)
	assertHTTPError(t, err, 404, "detail", "No Intake matches the given query.")

	err = repository.Delete(context.Background(), "user", "workspace", "project", "missing")
	assertHTTPError(t, err, 404, "detail", "No Intake matches the given query.")
}

func assertHTTPError(t *testing.T, err error, status int, key, message string) {
	t.Helper()
	var response *HTTPError
	if !errors.As(err, &response) {
		t.Fatalf("error=%T %v, want HTTPError", err, err)
	}
	if response.Status != status || response.Body[key] != message {
		t.Fatalf("error status/body=%d/%v, want %d/%s=%q", response.Status, response.Body, status, key, message)
	}
}
