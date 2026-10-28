package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	systemapi "github.com/CaliLuke/go-plane"
	"github.com/CaliLuke/go-plane/gen/system"
	workspacefavoritesapi "github.com/CaliLuke/go-plane/gen/workspace_favorites"
)

type favoriteCollectionQueryStub struct {
	workspaceFavoritesHTTPStub
	all *bool
}

func (stub *favoriteCollectionQueryStub) List(_ context.Context, payload *workspacefavoritesapi.ListPayload) ([]*workspacefavoritesapi.WorkspaceFavorite, error) {
	stub.all = payload.All
	return []*workspacefavoritesapi.WorkspaceFavorite{}, nil
}

func TestWorkspaceFavoritesAllQueryUsesLastScalarValue(t *testing.T) {
	for _, scenario := range []struct {
		query string
		want  bool
	}{
		{"all=false&all=true", true},
		{"all=true&all=false", false},
		{"all=%ZZ&all=true", true},
	} {
		t.Run(scenario.query, func(t *testing.T) {
			stub := &favoriteCollectionQueryStub{}
			handler := buildHTTPHandlerWithReplacementServices(
				newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				nil, nil, workspacefavoritesapi.NewEndpoints(stub),
			)
			request := httptest.NewRequest(http.MethodGet, "/api/workspaces/acme/user-favorites/?"+scenario.query, nil)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session-cookie"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
			}
			if stub.all == nil {
				t.Fatal("all query parameter was not decoded")
			}
			if *stub.all != scenario.want {
				t.Fatalf("all = %v, want %v", *stub.all, scenario.want)
			}
		})
	}
}
