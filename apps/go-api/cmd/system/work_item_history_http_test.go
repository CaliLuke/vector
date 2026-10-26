package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/internal/workitemhistory"
	workitemhistoryapi "github.com/CaliLuke/go-plane/workitemhistory/gen/work_item_history"
	"github.com/google/uuid"
)

const (
	historyHTTPProjectID = "11111111-1111-4111-8111-111111111111"
	historyHTTPIssueID   = "22222222-2222-4222-8222-222222222222"
	historyHTTPVersionID = "33333333-3333-4333-8333-333333333333"
)

type workItemHistoryHTTPRepository struct {
	allowed bool
	reads   []workitemhistory.Request
}

func (repository *workItemHistoryHTTPRepository) Read(_ context.Context, request workitemhistory.Request) (json.RawMessage, error) {
	repository.reads = append(repository.reads, request)
	return json.RawMessage(`{"ok":true}`), nil
}
func (repository *workItemHistoryHTTPRepository) ProjectEntityAllowed(context.Context, string, string, uuid.UUID) (bool, error) {
	return repository.allowed, nil
}

func TestWorkItemHistoryGeneratedHTTPMountCarriesAuthenticatedRequests(t *testing.T) {
	tests := []struct {
		name, path, operation string
		query                 url.Values
	}{
		{"history", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/issues/" + historyHTTPIssueID + "/history/", "history", url.Values{"created_at__gt": {"2026-10-01"}, "activity_type": {"comment"}}},
		{"metadata", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/issues/" + historyHTTPIssueID + "/meta/", "meta", nil},
		{"versions", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/issues/" + historyHTTPIssueID + "/versions/", "versions", url.Values{"cursor": {"25:2:0"}}},
		{"version detail", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/issues/" + historyHTTPIssueID + "/versions/" + historyHTTPVersionID + "/", "version_detail", nil},
		{"description versions", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/work-items/" + historyHTTPIssueID + "/description-versions/", "description_versions", url.Values{"cursor": {"10:1:0"}}},
		{"description version detail", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/work-items/" + historyHTTPIssueID + "/description-versions/" + historyHTTPVersionID + "/", "description_version_detail", nil},
		{"property history", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/work-items/" + historyHTTPIssueID + "/property-history/", "property_history", nil},
		{"property versions", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/work-items/" + historyHTTPIssueID + "/properties/" + historyHTTPVersionID + "/versions/", "property_versions", url.Values{"before": {historyHTTPVersionID}}},
		{"property version detail", "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/work-items/" + historyHTTPIssueID + "/properties/" + historyHTTPVersionID + "/versions/" + historyHTTPVersionID + "/", "property_version_detail", nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &workItemHistoryHTTPRepository{allowed: true}
			sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
			service, err := systemapi.NewWorkItemHistory(repository, sessions)
			if err != nil {
				t.Fatal(err)
			}
			handler := extensionHTTPHandler(pendingReplacementEndpoints{
				workItemHistory:         workitemhistoryapi.NewEndpoints(service),
				workItemHistoryMetadata: service.(sessionResourceProtocolAuthorizer),
				readServicesSession:     sessions,
				readServicesCookie:      djangoSessionCookieConfig{Name: "session-id", Configured: true},
			})
			requestURL := test.path
			if len(test.query) > 0 {
				requestURL += "?" + test.query.Encode()
			}
			request := httptest.NewRequest(http.MethodGet, requestURL, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || len(repository.reads) != 1 {
				t.Fatalf("status=%d reads=%d body=%s", response.Code, len(repository.reads), response.Body.String())
			}
			got := repository.reads[0]
			if got.Operation != test.operation || got.Slug != "demo" || got.ProjectID.String() != historyHTTPProjectID || got.ResourceID.String() != historyHTTPIssueID || got.UserID.String() != cycleMiddlewareActor || got.Timezone != "UTC" {
				t.Fatalf("repository request = %+v", got)
			}
			if got.Cursor != nil && test.query.Get("cursor") != *got.Cursor {
				t.Fatalf("cursor = %q, want %q", *got.Cursor, test.query.Get("cursor"))
			}
			if strings.HasPrefix(test.operation, "property_version") && got.PropertyID.String() != historyHTTPVersionID {
				t.Fatalf("wrong property: %+v", got)
			}
			if test.operation == "property_versions" && (got.BeforeID == nil || got.BeforeID.String() != historyHTTPVersionID) {
				t.Fatalf("missing revision cursor: %+v", got)
			}
			if test.operation == "history" && (got.CreatedAtGT == nil || *got.CreatedAtGT != "2026-10-01" || got.ActivityType == nil || *got.ActivityType != "comment") {
				t.Fatalf("history filters = created_at__gt %v activity_type %v", got.CreatedAtGT, got.ActivityType)
			}
		})
	}
}

func TestWorkItemHistoryProtocolAuthScopeHeadAndOptions(t *testing.T) {
	newHandler := func(allowed bool) (http.Handler, *workItemHistoryHTTPRepository, *cycleMiddlewareSessions) {
		repository := &workItemHistoryHTTPRepository{allowed: allowed}
		sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
		service, err := systemapi.NewWorkItemHistory(repository, sessions)
		if err != nil {
			t.Fatal(err)
		}
		handler := extensionHTTPHandler(pendingReplacementEndpoints{
			workItemHistory:         workitemhistoryapi.NewEndpoints(service),
			workItemHistoryMetadata: service.(sessionResourceProtocolAuthorizer),
			readServicesSession:     sessions,
			readServicesCookie:      djangoSessionCookieConfig{Name: "session-id", Configured: true},
			activityReadsBodyLimit:  1 << 20,
		})
		return handler, repository, sessions
	}
	historyPath := "/api/workspaces/demo/projects/" + historyHTTPProjectID + "/issues/" + historyHTTPIssueID + "/history/"

	t.Run("anonymous request is rejected", func(t *testing.T) {
		handler, repository, _ := newHandler(true)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, historyPath, nil))
		if response.Code != http.StatusUnauthorized || len(repository.reads) != 0 {
			t.Fatalf("status=%d reads=%d body=%s", response.Code, len(repository.reads), response.Body.String())
		}
	})

	t.Run("history scope is enforced", func(t *testing.T) {
		handler, repository, _ := newHandler(false)
		request := httptest.NewRequest(http.MethodGet, historyPath, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || len(repository.reads) != 0 {
			t.Fatalf("status=%d reads=%d body=%s", response.Code, len(repository.reads), response.Body.String())
		}
	})

	t.Run("head is generated from the authenticated GET handler", func(t *testing.T) {
		handler, repository, _ := newHandler(true)
		request := httptest.NewRequest(http.MethodHead, historyPath, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.Len() != 0 || len(repository.reads) != 1 || repository.reads[0].Operation != "history" {
			t.Fatalf("status=%d body=%q reads=%+v", response.Code, response.Body.String(), repository.reads)
		}
	})

	t.Run("options metadata uses the work item history authorizer", func(t *testing.T) {
		handler, repository, _ := newHandler(false)
		request := httptest.NewRequest(http.MethodOptions, historyPath, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || len(repository.reads) != 0 {
			t.Fatalf("status=%d reads=%d body=%s", response.Code, len(repository.reads), response.Body.String())
		}
	})

	t.Run("authorized options metadata is available", func(t *testing.T) {
		handler, repository, _ := newHandler(true)
		request := httptest.NewRequest(http.MethodOptions, historyPath, nil)
		request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || len(repository.reads) != 0 {
			t.Fatalf("status=%d reads=%d body=%s", response.Code, len(repository.reads), response.Body.String())
		}
		var metadata map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &metadata); err != nil || metadata["name"] != "Issue Activity Endpoint" {
			t.Fatalf("metadata=%v err=%v", metadata, err)
		}
	})
}

func TestWorkItemHistoryOptionsMetadataMatchesSourceViews(t *testing.T) {
	names := []string{"Issue Activity Endpoint", "Issue Meta Endpoint", "Issue Version Endpoint", "Issue Version Endpoint", "Work Item Description Version Endpoint", "Work Item Description Version Endpoint", "Work Item Property History Endpoint", "Work Item Property History Endpoint", "Work Item Property History Endpoint"}
	sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
	repository := &workItemHistoryHTTPRepository{allowed: true}
	service, err := systemapi.NewWorkItemHistory(repository, sessions)
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{workItemHistory: workitemhistoryapi.NewEndpoints(service), workItemHistoryMetadata: service.(sessionResourceProtocolAuthorizer), readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true}})
	for i, pattern := range workItemHistoryPaths {
		t.Run(names[i]+pattern, func(t *testing.T) {
			path := strings.NewReplacer("{slug}", "demo", "{project_id}", historyHTTPProjectID, "{issue_id}", historyHTTPIssueID, "{work_item_id}", historyHTTPIssueID, "{pk}", historyHTTPVersionID, "{property_id}", historyHTTPVersionID).Replace(pattern)
			request := httptest.NewRequest(http.MethodOptions, path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var got map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"name": names[i], "description": "", "renders": []any{"application/json"}, "parses": []any{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}}
			if response.Code != http.StatusOK || !reflect.DeepEqual(got, want) || len(repository.reads) != 0 {
				t.Fatalf("status=%d metadata=%v want=%v reads=%d", response.Code, got, want, len(repository.reads))
			}
		})
	}
}
