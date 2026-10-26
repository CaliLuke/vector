package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/platform"
	"github.com/CaliLuke/loom/clue/log"
)

func TestHTTPServerErrorLoggingClassifiesStatusWithoutQueryOrBody(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		cancel     bool
		wantClass  string
		wantRecord bool
	}{
		{name: "server error", status: http.StatusInternalServerError, wantClass: "http_server_error", wantRecord: true},
		{name: "canceled request", status: http.StatusInternalServerError, cancel: true, wantClass: "context_canceled", wantRecord: true},
		{name: "client error", status: http.StatusBadRequest},
		{name: "success", status: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := newLogContext(context.Background(), &output, false)
			requestContext := context.Background()
			if test.cancel {
				canceled, cancel := context.WithCancel(requestContext)
				cancel()
				requestContext = canceled
			}

			serverErrorLogger := logHTTPServerErrors(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(test.status)
				_, _ = response.Write([]byte("response-secret"))
			}))
			handler := log.HTTP(ctx, log.WithDisableRequestID(), log.WithDisableRequestLogging())(serverErrorLogger)
			wrapped := platform.RequestIDMiddleware(handler)
			request := httptest.NewRequest(http.MethodGet, "/api/example/?token=do-not-log", nil).WithContext(requestContext)
			response := httptest.NewRecorder()
			wrapped.ServeHTTP(response, request)

			logged := output.String()
			if !test.wantRecord {
				if logged != "" {
					t.Fatalf("unexpected log for status %d: %s", test.status, logged)
				}
				return
			}
			var entry map[string]any
			if err := json.Unmarshal([]byte(logged), &entry); err != nil {
				t.Fatalf("decode log record: %v; log=%s", err, logged)
			}
			for field, want := range map[string]any{
				"msg":         "HTTP server error response",
				"http.method": http.MethodGet,
				"http.path":   "/api/example/",
				"http.status": float64(test.status),
				"error.class": test.wantClass,
			} {
				if got := entry[field]; got != want {
					t.Errorf("%s=%v, want %v; log=%s", field, got, want, logged)
				}
			}
			if requestID, ok := entry["request_id"].(string); !ok || requestID == "" {
				t.Errorf("request_id missing from log: %s", logged)
			}
			if strings.Contains(logged, "do-not-log") {
				t.Fatalf("query string leaked into server error log: %s", logged)
			}
			if strings.Contains(logged, "response-secret") {
				t.Fatalf("response body leaked into server error log: %s", logged)
			}
			if !strings.Contains(response.Body.String(), "response-secret") {
				t.Fatalf("middleware changed response body: %q", response.Body.String())
			}
		})
	}
}
