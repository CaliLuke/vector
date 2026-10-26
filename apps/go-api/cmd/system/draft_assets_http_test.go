package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/google/uuid"
)

func TestDraftAssetAssociationHTTPAuthorizationAndBody(t *testing.T) {
	draft := uuid.New()
	asset := uuid.MustParse(assetsTransportAssetID)
	for _, tc := range []struct {
		name, body, origin  string
		session, csrf, deny bool
		status              int
		called              bool
	}{
		{name: "anonymous", status: 401},
		{name: "missing csrf", session: true, status: 403},
		{name: "foreign origin", session: true, csrf: true, origin: "https://foreign.example", status: 403},
		{name: "malformed body", session: true, csrf: true, body: "{", status: 400},
		{name: "invalid id", session: true, csrf: true, body: `{"asset_ids":["wrong"]}`, status: 400},
		{name: "empty set", session: true, csrf: true, body: `{"asset_ids":[]}`, status: 400},
		{name: "owner denied", session: true, csrf: true, deny: true, status: 404, called: true},
		{name: "associated", session: true, csrf: true, status: 204, called: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			repo := assetsTransportRepo{draftAssociation: func(_ context.Context, workspace, actor, target uuid.UUID, ids []uuid.UUID) error {
				called = true
				if workspace.String() != assetsTransportWorkspaceID || actor.String() != assetsTransportUserID || target != draft || len(ids) != 1 || ids[0] != asset {
					t.Fatalf("incorrect association scope: %v %v %v %v", workspace, actor, target, ids)
				}
				if tc.deny {
					return &djangoapi.HTTPError{Status: 404, Body: map[string]any{"detail": "Not found."}}
				}
				return nil
			}}
			body := tc.body
			if body == "" {
				body = `{"asset_ids":["` + asset.String() + `"]}`
			}
			request := httptest.NewRequest(http.MethodPost, "http://api.example.test/api/assets/v2/workspaces/drafts/"+draft.String()+"/bulk/", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			if tc.session {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			}
			if tc.csrf {
				token := strings.Repeat("a", 32)
				request.AddCookie(&http.Cookie{Name: "csrftoken", Value: token})
				request.Header.Set("X-CSRFToken", token)
			}
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}
			response := httptest.NewRecorder()
			assetsTransportHandlerWith(t, repo, &assetsTransportThrottleCache{}).ServeHTTP(response, request)
			if response.Code != tc.status || called != tc.called {
				t.Fatalf("status %d want %d; called %v want %v: %s", response.Code, tc.status, called, tc.called, response.Body.String())
			}
		})
	}
}
