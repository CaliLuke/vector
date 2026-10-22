package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	sticky "github.com/CaliLuke/go-plane/gen/stickies"
	"github.com/CaliLuke/go-plane/gen/system"
	workspaceapi "github.com/CaliLuke/go-plane/gen/workspace_slug_check"
)

func TestStickyHTTPAllowsMalformedUnusedQueryValues(t *testing.T) {
	stub := stickyHTTPStub{success: true}
	handler := newHTTPHandlerWithStickies(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, sticky.NewEndpoints(stub), stub.APIKeyAuth)
	for _, rawQuery := range []string{"per_page=20&ignored=%ZZ", "ignored=%&cursor=20:0:0", "ignored=%2", "per_page=%ZZ&per_page=20", "cursor=%ZZ&cursor=20:0:0"} {
		t.Run(rawQuery, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/workspaces/test/stickies/?"+rawQuery, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("response = %d %s, want 200", response.Code, response.Body.String())
			}
			var body struct {
				Results []struct{ ID string }
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Results) != 1 || body.Results[0].ID != "0198a28d-d227-7752-82ac-bc4c1c290703" {
				t.Fatalf("list response = %s, want the existing Sticky", response.Body.String())
			}
		})
	}
}

func TestStickyHEADAllowsMalformedUnusedQueryValues(t *testing.T) {
	stub := stickyHTTPStub{success: true}
	handler := newHTTPHandlerWithStickies(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, sticky.NewEndpoints(stub), stub.APIKeyAuth)
	for _, rawQuery := range []string{"per_page=20&ignored=%ZZ", "ignored=%&cursor=20:0:0", "ignored=%2", "per_page=%ZZ&per_page=20", "cursor=%ZZ&cursor=20:0:0"} {
		t.Run(rawQuery, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodHead, "/api/workspaces/test/stickies/?"+rawQuery, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Body.Len() != 0 {
				t.Fatalf("response = %d %q, want 200 with empty body", response.Code, response.Body.String())
			}
			if response.Header().Get("Allow") != "GET, POST, HEAD, OPTIONS" || response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("HEAD lost list protocol headers: %v", response.Header())
			}
		})
	}
}

func TestStickyHTTPMalformedSupportedQueryValuesKeepDjangoErrors(t *testing.T) {
	stub := stickyHTTPStub{success: true}
	handler := newHTTPHandlerWithStickies(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, sticky.NewEndpoints(stub), stub.APIKeyAuth)
	for _, test := range []struct {
		query, body, contentLength string
	}{
		{"cursor=%ZZ", `{"detail":"Invalid cursor parameter."}`, "39"},
		{"cursor=20:0:0&cursor=%ZZ", `{"detail":"Invalid cursor parameter."}`, "39"},
		{"per_page=%ZZ", `{"detail":"Invalid per_page parameter."}`, "41"},
		{"per_page=20&per_page=%ZZ", `{"detail":"Invalid per_page parameter."}`, "41"},
		{"cursor=bad&ignored=%ZZ", `{"detail":"Invalid cursor parameter."}`, "39"},
		{"per_page=bad&ignored=%ZZ", `{"detail":"Invalid per_page parameter."}`, "41"},
	} {
		t.Run(test.query, func(t *testing.T) {
			t.Run("GET", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, "/api/workspaces/test/stickies/?"+test.query, nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusBadRequest || strings.TrimSpace(response.Body.String()) != test.body {
					t.Fatalf("response = %d %q, want 400 %s", response.Code, response.Body.String(), test.body)
				}
			})
			t.Run("HEAD", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodHead, "/api/workspaces/test/stickies/?"+test.query, nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusBadRequest || response.Body.Len() != 0 || response.Header().Get("Content-Length") != test.contentLength || response.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("response = %d %q %v, want empty 400 with Django error length %s", response.Code, response.Body.String(), response.Header(), test.contentLength)
				}
			})
		})
	}
}

func TestWorkspaceSlugHTTPPreservesMalformedQueryValueAsLiteral(t *testing.T) {
	// Django preserves malformed escapes as text. Dropping the slug would turn
	// this otherwise valid nonempty slug into the missing-slug error.
	handler := newHTTPHandlerWithStickiesAndWorkspaceSlugCheck(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, nil, nil, workspaceapi.NewEndpoints(workspaceSlugHTTPStub{}))
	request := httptest.NewRequest(http.MethodGet, "/api/workspace-slug-check/?slug=%ZZ&ignored=%", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"status":true}` {
		t.Fatalf("response = %d %q, want 200 with available slug", response.Code, response.Body.String())
	}
}

func TestReadServiceMalformedQueryPrecedesScalarSelection(t *testing.T) {
	var got string
	handler := readServicesCompatibilityMiddleware(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		got = request.URL.Query().Get("cursor")
		response.Header().Set("Content-Type", "application/json")
		response.Write([]byte(`[]`))
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces/test/projects/0198a28d-d227-7752-82ac-bc4c1c290703/cycles/?cursor=20:0:0&cursor=20:1:0&cursor=%ZZ", nil)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if got != "%ZZ" {
		t.Fatalf("last cursor = %q, want literal malformed value", got)
	}
}

func TestTolerantQueryPreservesRepeatedArrayValues(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?items=a&items=%ZZ&items=c&ignored=%", nil)
	values := normalizeDjangoTolerantQuery(request).URL.Query()["items"]
	if len(values) != 3 || values[0] != "a" || values[1] != "%ZZ" || values[2] != "c" {
		t.Fatalf("repeated array values = %q", values)
	}
}

// Django Sticky.list reads query_params.get("query"), selecting the last
// decoded occurrence before applying its substring filter.
type stickyQuerySelectionStub struct {
	stickyHTTPStub
	selected string
}

func (stub *stickyQuerySelectionStub) List(ctx context.Context, payload *sticky.ListPayload) (*sticky.StickyPage, error) {
	if payload.Query != nil {
		stub.selected = *payload.Query
	}
	page, err := stub.stickyHTTPStub.List(ctx, payload)
	if len(page.Results) > 0 {
		page.Results[0].Name = payload.Query
	}
	return page, err
}

func TestStickySearchUsesLastDecodedScalar(t *testing.T) {
	for _, test := range []struct{ query, selected string }{
		{"query=%ZZ&query=needle", "needle"},
		{"query=needle&query=%ZZ", "%ZZ"},
		{"query=first&query=last", "last"},
		{"query=first&query=", ""},
		{"query=first&qu%65ry=last", "last"},
		{"query=%FF%FF&ignored=%", "\ufffd\ufffd"},
		{"query=%E2%82", "\ufffd"},
	} {
		t.Run(test.query, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				t.Run(method, func(t *testing.T) {
					stub := &stickyQuerySelectionStub{stickyHTTPStub: stickyHTTPStub{success: true}}
					handler := newHTTPHandlerWithStickies(newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, sticky.NewEndpoints(stub), stub.APIKeyAuth)
					request := httptest.NewRequest(method, "/api/workspaces/test/stickies/?"+test.query, nil)
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					if response.Code != http.StatusOK || stub.selected != test.selected {
						t.Fatalf("status=%d selected=%q, want 200 and %q", response.Code, stub.selected, test.selected)
					}
					if method == http.MethodHead {
						if response.Body.Len() != 0 {
							t.Fatal("HEAD returned a body")
						}
						return
					}
					var body struct{ Results []struct{ Name string } }
					if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
						t.Fatal(err)
					}
					if len(body.Results) != 1 || body.Results[0].Name != test.selected {
						t.Fatalf("query-aware result = %s", response.Body.String())
					}
				})
			}
		})
	}
}
