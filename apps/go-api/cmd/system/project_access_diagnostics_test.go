package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	system "github.com/CaliLuke/go-plane/gen/system"
	"github.com/stretchr/testify/require"
)

func TestProjectAccessDiagnostics(t *testing.T) {
	for _, test := range []struct {
		name, site, body string
		status           int
	}{
		{"network retry", "same-origin", `{"stage":"project_detail","reason":"network","will_retry":true}`, 204},
		{"HTTP failure", "same-origin", `{"stage":"project_roles","reason":"http","http_status":503,"request_id":"f4382efe-a999-4396-8664-5cba5aad55ac"}`, 204},
		{"untrusted page", "cross-site", `{"stage":"project_detail","reason":"network"}`, 403},
		{"extra field", "same-origin", `{"stage":"project_detail","reason":"network","secret":"never-log"}`, 400},
		{"unrecognized stage", "same-origin", `{"stage":"never-log","reason":"network"}`, 400},
		{"invalid correlation ID", "same-origin", `{"stage":"project_detail","reason":"http","http_status":503,"request_id":"never-log"}`, 400},
		{"trailing body", "same-origin", `{"stage":"project_detail","reason":"network"}{}`, 400},
		{"oversized body", "same-origin", `{"stage":"project_detail","reason":"network"}` + strings.Repeat(" ", 513), 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := newLogContext(context.Background(), &output, false)
			handler := newHTTPHandler(ctx, system.NewEndpoints(systemapi.NewSystem()))
			request := httptest.NewRequest(http.MethodPost, projectAccessDiagnosticsPath, strings.NewReader(test.body))
			request.Header.Set("Sec-Fetch-Site", test.site)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code)
			require.NotContains(t, output.String(), "never-log")
			if test.status != http.StatusNoContent {
				require.NotContains(t, output.String(), "Browser project access read failed")
				return
			}
			var event map[string]any
			for _, line := range strings.Split(output.String(), "\n") {
				if strings.Contains(line, "Browser project access read failed") {
					require.NoError(t, json.Unmarshal([]byte(line), &event))
					break
				}
			}
			require.Equal(t, "Browser project access read failed", event["msg"])
			if test.name == "HTTP failure" {
				require.Equal(t, "error", event["level"])
				require.Equal(t, "project_roles", event["browser.access.stage"])
				require.Equal(t, float64(503), event["browser.access.http_status"])
				require.Equal(t, "f4382efe-a999-4396-8664-5cba5aad55ac", event["browser.access.request_id"])
			} else {
				require.Equal(t, "network", event["browser.access.reason"])
				require.Equal(t, true, event["browser.access.will_retry"])
			}
		})
	}
}
