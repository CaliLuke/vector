package systemapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	workitemstore "github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestWorkItemErrorKind(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "canceled", err: fmt.Errorf("query interrupted: %w", context.Canceled), want: "context_canceled"},
		{name: "deadline", err: fmt.Errorf("query interrupted: %w", context.DeadlineExceeded), want: "deadline_exceeded"},
		{name: "postgres", err: &pgconn.PgError{Code: "23505"}, want: "postgres"},
		{name: "other", err: errors.New("sensitive database detail"), want: "other"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := workItemErrorKind(test.err); got != test.want {
				t.Fatalf("workItemErrorKind(%T)=%q, want %q", test.err, got, test.want)
			}
		})
	}
}

type canceledWorkItemRepository struct {
	executed          bool
	authorizeCanceled bool
}

func (repository *canceledWorkItemRepository) AuthorizeInitial(ctx context.Context, _ workitemstore.Request) error {
	if repository.authorizeCanceled {
		return fmt.Errorf("sensitive query detail: %w", ctx.Err())
	}
	return nil
}

func (repository *canceledWorkItemRepository) Execute(ctx context.Context, _ workitemstore.Request, _ io.Reader, _ string) (workitemstore.Result, error) {
	repository.executed = true
	if !errors.Is(ctx.Err(), context.Canceled) {
		return workitemstore.Result{}, fmt.Errorf("expected canceled request context, got %v", ctx.Err())
	}
	return workitemstore.Result{}, fmt.Errorf("sensitive query detail: %w", ctx.Err())
}

func TestWorkItemCanceledRequestLogsSafeErrorKind(t *testing.T) {
	testWorkItemCanceledRequestLogsSafeErrorKind(t, false)
}

func TestWorkItemCanceledInitialAuthorizationLogsSafeErrorKind(t *testing.T) {
	testWorkItemCanceledRequestLogsSafeErrorKind(t, true)
}

func testWorkItemCanceledRequestLogsSafeErrorKind(t *testing.T, authorizeCanceled bool) {
	t.Helper()
	const privateDetail = "sensitive query detail"
	repository := &canceledWorkItemRepository{authorizeCanceled: authorizeCanceled}
	// Exercise the real generated HTTP endpoint with cancellation observed at
	// the selected repository boundary.
	configured, err := NewWorkItemCore(repository, &activitySessionStub{}, &activityAPIKeyStub{
		principal: apikey.Principal{UserID: uuid.New(), Timezone: "UTC"},
	}, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	if err != nil {
		t.Fatal(err)
	}
	handler := workItemCoreHTTPHandler(t, configured)

	var logs bytes.Buffer
	logContext := log.Context(context.Background(),
		log.WithOutputs(log.Output{Writer: &logs, Format: log.FormatJSON}),
		log.WithDisableBuffering(func(context.Context) bool { return true }),
	)
	requestContext, cancel := context.WithCancel(logContext)
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/demo/projects/01234567-89ab-4def-8123-456789abcdef/issues/", nil).WithContext(requestContext)
	request.Header.Set("X-Api-Key", "valid")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if repository.executed == authorizeCanceled {
		t.Fatalf("executed=%t, initial authorization failure=%t", repository.executed, authorizeCanceled)
	}
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s, want unchanged generic 500", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), privateDetail) {
		t.Fatalf("response leaked raw error detail: %s", response.Body.String())
	}
	var responseBody map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response body %q: %v", response.Body.String(), err)
	}
	if responseBody["error"] != "Something went wrong please try again later" {
		t.Fatalf("response body=%#v, want existing generic error body", responseBody)
	}

	var entries []map[string]any
	decoder := json.NewDecoder(&logs)
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatalf("decode log entry: %v", err)
		}
		entries = append(entries, entry)
	}
	var operationFailure map[string]any
	for _, entry := range entries {
		if entry["msg"] == "Work-item operation failed" {
			operationFailure = entry
		}
	}
	if operationFailure == nil {
		t.Fatalf("missing operation failure log: %s", logs.String())
	}
	if operationFailure["error_kind"] != "context_canceled" {
		t.Fatalf("error_kind=%v, log=%#v", operationFailure["error_kind"], operationFailure)
	}
	wantStage := "execute"
	if authorizeCanceled {
		wantStage = "authorize_initial"
	}
	if operationFailure["stage"] != wantStage || operationFailure["error_type"] != "*fmt.wrapError" {
		t.Fatalf("stage/type=%v/%v, log=%#v", operationFailure["stage"], operationFailure["error_type"], operationFailure)
	}
	if strings.Contains(logs.String(), privateDetail) {
		t.Fatalf("log leaked raw error detail: %s", logs.String())
	}
}

func (*canceledWorkItemRepository) ProtocolPutAction(context.Context, workitemstore.Request) (bool, error) {
	return true, nil
}
