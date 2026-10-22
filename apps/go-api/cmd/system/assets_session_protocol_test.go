package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
)

func TestAssetsSessionPrincipalReachesPublicAndRestoreHandlers(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	for _, path := range []string{
		"/api/assets/v2/static/" + id + "/",
		"/api/public/assets/v2/anchor/published/",
		"/api/public/assets/v2/anchor/published/" + id + "/",
		"/api/public/assets/v2/anchor/published/restore/" + id + "/",
		"/api/public/assets/v2/anchor/published/" + id + "/bulk/",
		"/api/workspaces/file-assets/" + id + "/image.png/restore/",
	} {
		t.Run(path, func(t *testing.T) {
			sessions := &publicProjectMetadataThrottleStub{userID: id, userTimezone: "UTC"}
			next := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				user, inspected := systemapi.ReadServiceSessionUserID(request.Context())
				if !inspected || user != id {
					t.Errorf("handler principal = %q, inspected = %t; want authenticated owner", user, inspected)
				}
				response.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "asset-session"})
			response := httptest.NewRecorder()
			readServicesSessionMiddleware(sessions, djangoSessionCookieConfig{Name: "session-id", Configured: true})(next).ServeHTTP(response, request)
			if sessions.calls != 1 || sessions.inspectedCookie != "asset-session" || response.Code != http.StatusNoContent {
				t.Fatalf("session calls=%d cookie=%q response=%d", sessions.calls, sessions.inspectedCookie, response.Code)
			}
		})
	}
}
