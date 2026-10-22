package main

import (
	systemapi "github.com/CaliLuke/go-plane"
	stickystore "github.com/CaliLuke/go-plane/internal/stickies"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIV1StickyUnsupportedMethodsUseWorkspacePermission(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, key string
		permission              error
		allowed                 bool
		status, throttles       int
	}{
		{"collection put", "PUT", "/api/v1/workspaces/demo/stickies/", "sticky-key", nil, true, 405, 1},
		{"detail post", "POST", "/api/v1/workspaces/demo/stickies/not-a-uuid/", "sticky-key", nil, true, 405, 1},
		{"detail trace", "TRACE", "/api/v1/workspaces/demo/stickies/00000000-0000-4000-8000-000000000001/", "sticky-key", nil, true, 405, 1},
		{"format alias", "POST", "/api/v1/workspaces/demo/stickies/not-a-uuid.json", "sticky-key", nil, true, 405, 1},
		{"nonmember", "PUT", "/api/v1/workspaces/demo/stickies/", "sticky-key", stickystore.ErrWorkspaceAccessDenied, true, 403, 0},
		{"deleted workspace", "PUT", "/api/v1/workspaces/demo/stickies/", "sticky-key", stickystore.ErrStickyWorkspaceDeleted, true, 405, 1},
		{"anonymous", "PUT", "/api/v1/workspaces/demo/stickies/", "", nil, true, 401, 0},
		{"throttled", "PUT", "/api/v1/workspaces/demo/stickies/", "sticky-key", nil, false, 429, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &apiV1StickyMiddlewareRepository{resolveErr: tc.permission}
			keys := &apiV1StickyMiddlewareKeys{}
			throttle := &apiV1StickyMiddlewareThrottle{allowed: tc.allowed}
			svc, err := systemapi.NewAPIV1Stickies(repo, keys, throttle, apiV1StickyMiddlewareEffects{})
			if err != nil {
				t.Fatal(err)
			}
			dispatch := &activityHTTPService{}
			logger := &apiV1ActivityLoggerStub{}
			handler := newAPIV1StickyMiddlewareHandler(t, svc, dispatch, logger, defaultFileSizeLimit)
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("X-Api-Key", tc.key)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tc.status || throttle.calls != tc.throttles || dispatch.protocolCalls != 0 || repo.createCalls != 0 || repo.retrieveCalls != 0 {
				t.Fatalf("response=%d body=%s throttles=%d project_dispatch=%d creates=%d retrieves=%d", w.Code, w.Body.String(), throttle.calls, dispatch.protocolCalls, repo.createCalls, repo.retrieveCalls)
			}
			if tc.status == 405 && !strings.Contains(w.Body.String(), `Method \"`+tc.method+`\" not allowed.`) {
				t.Fatalf("405 body=%s", w.Body.String())
			}
		})
	}
}
