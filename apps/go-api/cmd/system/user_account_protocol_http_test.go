package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	useraccountwritesapi "github.com/CaliLuke/go-plane/user_account_writes/gen/user_account_writes"
)

func TestUserAccountProtocolMetadataAndAppendSlash(t *testing.T) {
	repository := &accountMountRepository{}
	cache := &accountMountCache{values: map[string][]byte{}}
	tasks := &accountMountTasks{}
	sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
	service, err := systemapi.NewUserAccountWrites(repository, sessions, cache, tasks, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	handler := extensionHTTPHandler(pendingReplacementEndpoints{
		userAccountWrites:         useraccountwritesapi.NewEndpoints(service),
		userAccountWritesMetadata: service.(sessionResourceProtocolAuthorizer),
		readServicesSession:       sessions,
		readServicesCookie:        djangoSessionCookieConfig{Name: "session-id", Configured: true},
		activityReadsBodyLimit:    1 << 20,
	})

	ordinary := httptest.NewRequest(http.MethodOptions, "/api/users/me/", nil)
	ordinary.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	ordinaryResponse := httptest.NewRecorder()
	handler.ServeHTTP(ordinaryResponse, ordinary)
	if ordinaryResponse.Code != http.StatusOK || ordinaryResponse.Header().Get("Allow") != "GET, PATCH, DELETE, HEAD, OPTIONS" {
		t.Fatalf("ordinary OPTIONS = %d headers=%v body=%q", ordinaryResponse.Code, ordinaryResponse.Header(), ordinaryResponse.Body.String())
	}
	if strings.Contains(ordinaryResponse.Body.String(), `"actions"`) || !strings.Contains(ordinaryResponse.Body.String(), `"name":"User Endpoint"`) {
		t.Fatalf("user metadata OPTIONS body = %q", ordinaryResponse.Body.String())
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/api/users/me/", nil)
	preflight.Header.Set("Origin", "https://app.example.test")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusOK || preflightResponse.Body.Len() != 0 {
		t.Fatalf("preflight OPTIONS = %d %q headers=%v", preflightResponse.Code, preflightResponse.Body.String(), preflightResponse.Header())
	}

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run("slashless "+method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/api/users/me?next=1", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != "/api/users/me/?next=1" {
				t.Fatalf("slashless response = %d location=%q body=%q", response.Code, response.Header().Get("Location"), response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Fatalf("slashless content type = %q", got)
			}
		})
	}

	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		request := httptest.NewRequest(http.MethodOptions, "/api/users/me", nil)
		request.Header.Set("Origin", "https://app.example.test")
		request.Header.Set("Access-Control-Request-Method", method)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.Len() != 0 || response.Header().Get("Location") != "" {
			t.Fatalf("slashless preflight %s = %d headers=%v body=%q", method, response.Code, response.Header(), response.Body.String())
		}
	}

	codeOptions := httptest.NewRequest(http.MethodOptions, "/api/users/me/email/generate-code/", nil)
	codeOptions.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	codeResponse := httptest.NewRecorder()
	handler.ServeHTTP(codeResponse, codeOptions)
	wantCodeMetadata, err := os.ReadFile("user_email_code_options.json")
	if err != nil {
		t.Fatal(err)
	}
	if codeResponse.Code != http.StatusOK || !bytes.Equal(codeResponse.Body.Bytes(), wantCodeMetadata) {
		t.Fatalf("email-code OPTIONS status=%d body bytes=%d, want status 200 and captured metadata bytes=%d", codeResponse.Code, codeResponse.Body.Len(), len(wantCodeMetadata))
	}
}

func TestWorkspaceCollectionOptionsUsesRecordedSourceMetadata(t *testing.T) {
	policy, err := djangoRuntimeCORSPolicy([]string{"https://app.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	options := apiV1CompatibilityOptions{bodySizeLimit: defaultFileSizeLimit}
	cookie := djangoSessionCookieConfig{Name: "session-id", Configured: true}
	stack := sessionServiceMiddlewareStack(options, nil, cookie)
	handler := newSessionServiceProtocolMiddleware(
		notificationProtocolMetadata,
		sessionProtocolAuthorizerStub{allowed: true},
		policy,
		stack,
		cookie,
		false,
		options,
	)(http.NotFoundHandler())
	handler = readServicesCompatibilityMiddleware(handler)

	request := httptest.NewRequest(http.MethodOptions, "/api/workspaces/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	want, err := os.ReadFile("workspace_collection_options.json")
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), want) {
		t.Fatalf("workspace collection OPTIONS status=%d body bytes=%d, want status 200 and captured metadata bytes=%d", response.Code, response.Body.Len(), len(want))
	}
}
