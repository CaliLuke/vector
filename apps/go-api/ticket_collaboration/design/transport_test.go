package design_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	server "github.com/CaliLuke/go-plane/ticket_collaboration/gen/http/ticket_collaboration/server"
	api "github.com/CaliLuke/go-plane/ticket_collaboration/gen/ticket_collaboration"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

// Only the optional public-read security shape is new. Shared work-item
// transport tests already cover raw parsers, path escaping and API-key security.
type publicTransportService struct {
	api.Service
	allow, called bool
	body, cookie  string
}

func (s *publicTransportService) APIKeyAuth(ctx context.Context, _ string, scheme *security.APIKeyScheme) (context.Context, error) {
	if scheme.Name != "PlaneSessionCookie" {
		panic("unexpected authentication scheme")
	}
	if !s.allow {
		value, _ := loom.JSONValueFrom(map[string]any{"detail": "Authentication credentials were not provided."})
		return ctx, &api.CollaborationError{Kind: "status_401", Body: value}
	}
	return ctx, nil
}

func (s *publicTransportService) GetAPIPublicAnchorByAnchorIssuesByIssueIDComments(ctx context.Context, _ *api.GetAPIPublicAnchorByAnchorIssuesByIssueIDCommentsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	s.called = true
	if metadata, ok := loomhttp.RequestMetadataFromContext(ctx); ok {
		s.cookie = metadata.Headers().Get("Cookie")
	}
	return &api.CollaborationResponse{}, io.NopCloser(strings.NewReader(`[]`)), nil
}

func (s *publicTransportService) GetAPIPublicAnchorByAnchorIssuesByIssueIDReactions(context.Context, *api.GetAPIPublicAnchorByAnchorIssuesByIssueIDReactionsPayload) (*api.CollaborationResponse, io.ReadCloser, error) {
	s.called = true
	return &api.CollaborationResponse{}, io.NopCloser(strings.NewReader(`[]`)), nil
}

func (s *publicTransportService) PostAPIPublicAnchorByAnchorIssuesByIssueIDComments(_ context.Context, _ *api.PostAPIPublicAnchorByAnchorIssuesByIssueIDCommentsPayload, body io.ReadCloser) (*api.CollaborationResponse, io.ReadCloser, error) {
	defer body.Close()
	data, err := io.ReadAll(body)
	s.body, s.called = string(data), true
	return &api.CollaborationResponse{}, io.NopCloser(strings.NewReader(`{"id":"created"}`)), err
}

func publicTransport(t *testing.T, service *publicTransportService) http.Handler {
	t.Helper()
	mux := loomhttp.NewMuxer()
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "http://localhost:3000", Methods: []string{"GET", "POST", "PATCH", "DELETE"}, Headers: []string{"Content-Type"}, Credentials: true}}})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	generated := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	generated.Use(loomhttp.RequestMetadataMiddleware(metadata))
	generated.Mount(mux)
	return mux
}

func TestPublicCommentOptionalSessionAndProtectedReactions(t *testing.T) {
	const prefix = "/api/public/anchor/board/issues/01234567-89ab-4def-8123-456789abcdef/"
	for _, test := range []struct {
		name, path, cookie string
		allow, called      bool
		status             int
	}{
		{"anonymous comments", "comments/", "", false, true, 200},
		{"session metadata on public comments", "comments/", "session-id=present", true, true, 200},
		{"anonymous reactions denied", "reactions/", "", false, false, 401},
		{"authenticated reactions", "reactions/", "session-id=present", true, true, 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &publicTransportService{allow: test.allow}
			request := httptest.NewRequest("GET", prefix+test.path, nil)
			request.Header.Set("Cookie", test.cookie)
			response := httptest.NewRecorder()
			publicTransport(t, service).ServeHTTP(response, request)
			if response.Code != test.status || service.called != test.called {
				t.Fatalf("status=%d called=%v body=%s", response.Code, service.called, response.Body.String())
			}
			if test.path == "comments/" && service.cookie != test.cookie {
				t.Fatalf("optional session metadata lost: %q", service.cookie)
			}
		})
	}
}

func TestPublicCommentWritesKeepRawBodyBehindAuthentication(t *testing.T) {
	const path = "/api/public/anchor/board/issues/01234567-89ab-4def-8123-456789abcdef/comments/"
	const body = `{"comment_html":null,"comment_html":"<p>last</p>"}`
	for _, allow := range []bool{false, true} {
		service := &publicTransportService{allow: allow}
		request := httptest.NewRequest("POST", path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		publicTransport(t, service).ServeHTTP(response, request)
		if allow {
			if response.Code != 201 || service.body != body {
				t.Fatalf("status=%d raw=%q body=%s", response.Code, service.body, response.Body.String())
			}
		} else if response.Code != 401 || service.called {
			t.Fatalf("anonymous write entered service: status=%d called=%v", response.Code, service.called)
		}
	}
}
