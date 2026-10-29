package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/platform"
	"github.com/CaliLuke/loom/clue/log"
)

func TestNewLogContextReportsRequestCancellationSafely(t *testing.T) {
	for _, test := range []struct {
		name, want string
		withID     bool
	}{
		{name: "normal", withID: true},
		{name: "canceled", want: "context_canceled", withID: true},
		{name: "deadline", want: "deadline_exceeded", withID: true},
		{name: "canceled-without-id", want: "context_canceled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := newLogContext(context.Background(), &output, false)
			ctx = log.Context(ctx, log.WithDisableBuffering(func(context.Context) bool { return true }))
			handler := http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				requestContext := request.Context()
				switch test.want {
				case "context_canceled":
					var cancel context.CancelFunc
					requestContext, cancel = context.WithCancel(requestContext)
					cancel()
				case "deadline_exceeded":
					var cancel context.CancelFunc
					requestContext, cancel = context.WithDeadline(requestContext, time.Unix(1, 0))
					defer cancel()
				}
				log.Print(requestContext, log.KV{K: "msg", V: "request complete"})
			})
			var serve http.Handler = handler
			if test.withID {
				serve = platform.RequestIDMiddleware(handler)
			}
			request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			const requestID = "00000000-0000-4000-8000-000000000001"
			request.Header.Set(platform.RequestIDHeader, requestID)
			serve.ServeHTTP(httptest.NewRecorder(), request)
			var entry map[string]any
			if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
				t.Fatalf("decode log: %v, output=%s", err, &output)
			}
			if value, present := entry["request_context_error"]; (test.want == "" && present) || (test.want != "" && value != test.want) {
				t.Fatalf("request_context_error=%v present=%t, log=%v", value, present, entry)
			}
			if test.withID && entry[log.RequestIDKey] != requestID {
				t.Fatalf("request ID was lost: %v", entry)
			}
		})
	}
}

func TestRunLogsStartupFailureBeforeReturning(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("SECRET_KEY", "test-only-secret-key")
	var output bytes.Buffer
	err := run(context.Background(), []string{"--go-background-tasks"}, &output)
	if err == nil || err.Error() != "database-url is required" {
		t.Fatalf("run() error = %v, want missing database URL", err)
	}
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode startup log: %v, output=%s", err, &output)
	}
	if entry[log.MessageKey] != "Go API exited with an error" || entry[log.ErrorMessageKey] != "database-url is required" {
		t.Fatalf("startup failure log = %v", entry)
	}
}
