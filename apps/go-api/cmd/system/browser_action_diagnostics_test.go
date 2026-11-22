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

func TestBrowserActionDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name, site, contentType, body, action string
		status                                int
	}{
		{"webhook download", "same-origin", "application/json", `{"action":"webhook_secret_csv_download"}`, "webhook_secret_csv_download", http.StatusNoContent},
		{"theme download", "same-origin", "application/json", `{"action":"theme_config_download"}`, "theme_config_download", http.StatusNoContent},
		{"cross-origin page", "same-site", "application/json", `{"action":"theme_config_download"}`, "", http.StatusForbidden},
		{"untrusted body format", "same-origin", "text/plain", `{"action":"theme_config_download"}`, "", http.StatusUnsupportedMediaType},
		{"unrecognized action", "same-origin", "application/json", `{"action":"never-log"}`, "", http.StatusBadRequest},
		{"private extra field", "same-origin", "application/json", `{"action":"theme_config_download","value":"never-log"}`, "", http.StatusBadRequest},
		{"missing action", "same-origin", "application/json", `{}`, "", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			ctx := newLogContext(context.Background(), &output, false)
			handler := newHTTPHandler(ctx, system.NewEndpoints(systemapi.NewSystem()))
			request := httptest.NewRequest(http.MethodPost, browserActionDiagnosticsPath, strings.NewReader(tc.body))
			request.Header.Set("Sec-Fetch-Site", tc.site)
			request.Header.Set("Content-Type", tc.contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code)
			require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			require.NotContains(t, output.String(), "never-log")
			if tc.action == "" {
				require.NotContains(t, output.String(), "Browser action failed")
				return
			}
			var event map[string]any
			for _, line := range strings.Split(output.String(), "\n") {
				if strings.Contains(line, "Browser action failed") {
					require.NoError(t, json.Unmarshal([]byte(line), &event))
					break
				}
			}
			require.Equal(t, "Browser action failed", event["msg"])
			require.Equal(t, tc.action, event["browser.action.name"])
			require.Equal(t, "error", event["level"])
		})
	}
}
