package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	store "github.com/CaliLuke/go-plane/internal/workitems"
	server "github.com/CaliLuke/go-plane/ticket_collaboration/gen/http/ticket_collaboration/server"
	api "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	workapi "github.com/CaliLuke/go-plane/work_item_core/gen/work_item_core"
	loomhttp "github.com/CaliLuke/loom/http"
)

type collaborationRepositorySpy struct {
	requests     []store.Request
	authorizeErr error
	executeErr   error
}

func (r *collaborationRepositorySpy) AuthorizeCollaborationInitial(_ context.Context, q store.Request) error {
	return r.authorizeErr
}
func (r *collaborationRepositorySpy) ProtocolCollaborationPutAction(context.Context, store.Request) (bool, error) {
	return true, nil
}
func (r *collaborationRepositorySpy) ExecuteCollaboration(_ context.Context, q store.Request, _ io.Reader, _ string) (store.Result, error) {
	r.requests = append(r.requests, q)
	return store.Result{Status: 200, Body: []any{}}, r.executeErr
}
func collaborationHandler(t *testing.T, r *collaborationRepositorySpy, sessions *activitySessionStub) http.Handler {
	t.Helper()
	s, err := NewTicketCollaboration(r, sessions, &activityAPIKeyStub{}, &activityThrottleStub{})
	if err != nil {
		t.Fatal(err)
	}
	mux := loomhttp.NewMuxer()
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "http://localhost:3000", Methods: []string{"GET", "POST", "PATCH", "PUT", "DELETE"}, Headers: []string{"Content-Type", "X-Api-Key"}, Credentials: true}}})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	generated := server.New(api.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	generated.Use(loomhttp.RequestMetadataMiddleware(metadata))
	generated.Mount(mux)
	return mux
}
func TestCollaborationPublicCommentReadInitializesOptionalSession(t *testing.T) {
	for _, tc := range []struct {
		name, cookie, user, zone string
		status                   int
	}{
		{name: "anonymous", status: 200},
		{name: "authenticated", cookie: "session", user: "reader", zone: "UTC", status: 200},
		{name: "invalid timezone", cookie: "session", user: "reader", zone: "Not/AZone", status: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := &collaborationRepositorySpy{}
			sessions := &activitySessionStub{principal: djangosession.Principal{UserID: tc.user, UserTimezone: tc.zone}}
			request := httptest.NewRequest(http.MethodGet, "/api/public/anchor/board/issues/00000000-0000-4000-8000-000000000001/comments/", nil)
			if tc.cookie != "" {
				request.AddCookie(&http.Cookie{Name: "session-id", Value: tc.cookie})
			}
			response := httptest.NewRecorder()
			collaborationHandler(t, repository, sessions).ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status%d body%s", response.Code, response.Body.String())
			}
			if tc.status == 200 {
				if len(repository.requests) != 1 || repository.requests[0].ActorID != tc.user {
					t.Fatalf("requests%#v", repository.requests)
				}
			} else if len(repository.requests) != 0 {
				t.Fatalf("business entered with invalidtimezone")
			}
			if tc.cookie != "" && sessions.key != tc.cookie {
				t.Fatalf("session cookie not retained: %#v", sessions)
			}
		})
	}
}
func TestCollaborationQueriesCannotReplaceResourcePaths(t *testing.T) {
	ctx := WithReadServiceQueryPresence(context.Background(), url.Values{"anchor": {"foreign"}, "comment_id": {"foreign"}, "reaction_code": {"foreign"}, "search": {"query"}})
	q := collaborationRequest(ctx, "delete_api_public_anchor_by_anchor_comments_by_comment_id_reactions_by_reaction_code", &api.DeleteAPIPublicAnchorByAnchorCommentsByCommentIDReactionsByReactionCodePayload{Anchor: "board", CommentID: "local", ReactionCode: "heart"})
	if q.Params["anchor"] != "board" || q.Params["comment_id"] != "local" || q.Params["reaction_code"] != "heart" || q.Params["search"] != "query" {
		t.Fatalf("request%#v", q)
	}
}
func TestCollaborationNativeAuthErrorUsesSharedProtocolBody(t *testing.T) {
	err := collaborationProtocolError(collaborationError(&djangoapi.HTTPError{Status: 401, Body: map[string]any{"detail": "credentials required"}}))
	var protocol *workapi.WorkItemError
	if !errors.As(err, &protocol) || protocol.Kind != "status_401" {
		t.Fatalf("protocol error%#v", err)
	}
	body, e := json.Marshal(protocol.Body)
	if e != nil || !strings.Contains(string(body), "credentials required") {
		t.Fatalf("body%s error%v", body, e)
	}
}

func TestCollaborationMalformedFiltersReachAuthorizedDomainValidation(t *testing.T) {
	for _, field := range []string{"workspace__id", "issue__id"} {
		for _, denied := range []bool{false, true} {
			repository := &collaborationRepositorySpy{executeErr: &djangoapi.HTTPError{Status: 400, Body: map[string]any{field: []string{"Enter a valid UUID."}}}}
			if denied {
				repository.authorizeErr = &djangoapi.HTTPError{Status: 403, Body: map[string]any{"detail": "Not permitted."}}
			}
			sessions := &activitySessionStub{principal: djangosession.Principal{UserID: "reader", UserTimezone: "UTC"}}
			path := "/api/workspaces/test/projects/00000000-0000-4000-8000-000000000001/issues/00000000-0000-4000-8000-000000000002/comments/?" + field + "=not-a-uuid"
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			response := httptest.NewRecorder()
			collaborationHandler(t, repository, sessions).ServeHTTP(response, request)
			if denied {
				if response.Code != 403 || len(repository.requests) != 0 {
					t.Fatalf("denied filter entered domain: status=%d requests=%d body=%s", response.Code, len(repository.requests), response.Body.String())
				}
			} else if response.Code != 400 || len(repository.requests) != 1 || !strings.Contains(response.Body.String(), "Enter a valid UUID.") || !strings.Contains(response.Body.String(), field) {
				t.Fatalf("filter %s rejected outside domain: status=%d requests=%d body=%s", field, response.Code, len(repository.requests), response.Body.String())
			}
		}
	}
}

func TestCollaborationGeneratedTransportAcceptsDjangoUUIDVariants(t *testing.T) {
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "00000000-0000-4000-c000-000000000001"} {
		for _, prefix := range []string{
			"/api/public/anchor/board/issues/",
			"/api/workspaces/test/projects/00000000-0000-4000-8000-000000000001/issues/",
		} {
			t.Run(prefix+id, func(t *testing.T) {
				repository := &collaborationRepositorySpy{}
				sessions := &activitySessionStub{principal: djangosession.Principal{UserID: "reader", UserTimezone: "UTC"}}
				request := httptest.NewRequest(http.MethodGet, prefix+id+"/comments/", nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
				response := httptest.NewRecorder()
				collaborationHandler(t, repository, sessions).ServeHTTP(response, request)
				if response.Code != 200 || len(repository.requests) != 1 || repository.requests[0].Params["issue_id"] != id {
					t.Fatalf("Django UUID path rejected: status=%d requests=%#v body=%s", response.Code, repository.requests, response.Body.String())
				}
			})
		}
	}
}
