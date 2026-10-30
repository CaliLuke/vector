package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCycleNonfiniteOrderReturns400BeforePersistence(t *testing.T) {
	for _, route := range []struct {
		name, method, path string
		api                bool
	}{
		{"app create", "POST", "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/cycles/", false},
		{"app patch", "PATCH", "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/cycles/22222222-2222-4222-8222-222222222222/", false},
		{"API patch", "PATCH", "/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/cycles/22222222-2222-4222-8222-222222222222/", true},
	} {
		for _, contentType := range []string{"application/json", "application/x-www-form-urlencoded"} {
			for _, value := range []string{"NaN", "Infinity", "-Infinity", "1e999"} {
				t.Run(route.name+"/"+contentType+"/"+value, func(t *testing.T) {
					recorder := &cycleMiddlewareRecorder{}
					sessions := &cycleMiddlewareSessions{recorder: recorder, userTimezone: "UTC"}
					activity := &cycleMiddlewareActivity{recorder: recorder}
					handler := newCycleMiddlewareTestHandler(t, recorder, sessions, cycleMiddlewareAPIKeys{recorder: recorder, timezone: "UTC"}, cycleMiddlewareThrottle{recorder: recorder}, activity, 1<<20)
					body := `{"name":"Rejected","sort_order":"` + value + `"}`
					if contentType == "application/x-www-form-urlencoded" {
						body = url.Values{"name": {"Rejected"}, "sort_order": {value}}.Encode()
					}
					request := httptest.NewRequest(route.method, route.path, strings.NewReader(body))
					request.Header.Set("Content-Type", contentType)
					if route.api {
						request.Header.Set("X-Api-Key", "valid-api-key")
					} else {
						request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
					}
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					require.Equal(t, 400, response.Code, response.Body.String())
					require.JSONEq(t, `{"sort_order":["A valid number is required."]}`, response.Body.String())
					require.NotContains(t, recorder.events, "dispatch")
				})
			}
		}
	}
}
