package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	workitemapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestWorkItemProtocolHEADPreservesAuditBody(t *testing.T) {
	for _, path := range []string{
		"/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/bulk-archive-issues/",
		"/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/cycles/00000000-0000-4000-8000-000000000002/cycle-issues/",
		"/api/v1/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/modules/00000000-0000-4000-8000-000000000002/module-issues/",
	} {
		for _, denied := range []bool{false, true} {
			t.Run(path+map[bool]string{false: "method", true: "permission"}[denied], func(t *testing.T) {
				authorizer := &workItemProtocolAuthorizerStub{}
				status := http.StatusMethodNotAllowed
				body := `{"detail":"Method \"HEAD\" not allowed."}`
				if denied {
					status = http.StatusForbidden
					body = `{"detail":"You do not have permission to perform this action."}`
					authorizer.err = &workitemapi.WorkItemError{Kind: "status_403", Body: loom.MustJSONValueFrom(map[string]string{"detail": "You do not have permission to perform this action."})}
				}
				logger := &apiV1ActivityLoggerStub{}
				handler := buildWorkItemProductionProtocolHandler(newLogContext(context.Background(), io.Discard, false), authorizer, &activityHTTPService{}, logger)
				request := httptest.NewRequest(http.MethodHead, path, nil)
				request.Header.Set("X-Api-Key", "test-token")
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != status || response.Body.Len() != 0 || authorizer.calls != 1 {
					t.Fatalf("status=%d body=%q auth=%d", response.Code, response.Body.String(), authorizer.calls)
				}
				if len(logger.records) != 1 || logger.records[0].ResponseCode != status || logger.records[0].ResponseBody != body {
					t.Fatalf("audit=%#v, want one status %d body %q", logger.records, status, body)
				}
			})
		}
	}
}
