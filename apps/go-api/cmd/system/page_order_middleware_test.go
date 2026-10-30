package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	client "github.com/CaliLuke/go-plane/gen/http/page_reads/client"
	pagereads "github.com/CaliLuke/go-plane/gen/page_reads"
	"github.com/CaliLuke/go-plane/gen/system"
	store "github.com/CaliLuke/go-plane/internal/pagereads"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type pageOrderAuthorizationDatabase struct {
	t       *testing.T
	queries int
}

func (database *pageOrderAuthorizationDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	database.queries++
	require.Equal(database.t, 1, database.queries, "invalid ordering must stop after authorization")
	return database
}

func (database *pageOrderAuthorizationDatabase) Scan(destinations ...any) error {
	require.Len(database.t, destinations, 1)
	switch target := destinations[0].(type) {
	case *bool:
		*target = true
	case *string:
		*target = "allowed"
	default:
		database.t.Fatalf("unexpected page content read into %T", target)
	}
	return nil
}

type pageOrderVisits struct{ calls int }

func (visits *pageOrderVisits) Record(context.Context, uuid.UUID, string, string, uuid.UUID, uuid.UUID) error {
	visits.calls++
	return nil
}

func TestPageOrderValidationThroughProductionMiddleware(t *testing.T) {
	for _, detail := range []bool{false, true} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			for _, order := range []string{"", "not_a_page_field"} {
				t.Run(fmt.Sprintf("detail=%t/%s/order=%q", detail, method, order), func(t *testing.T) {
					handler, database, visits := newPageOrderHandler(t)
					path := "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000001/pages/"
					if detail {
						path += "00000000-0000-4000-8000-000000000002/"
					}
					request := httptest.NewRequest(method, path+"?order_by="+order, nil)
					request.AddCookie(&http.Cookie{Name: "session-id", Value: "valid-session"})
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)
					require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
					require.Contains(t, response.Header().Get("Content-Type"), "application/json")
					require.Equal(t, 1, database.queries)
					require.Zero(t, visits.calls)
					if method == http.MethodHead {
						require.Empty(t, response.Body.String())
					} else {
						var body map[string]string
						require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
						require.Equal(t, map[string]string{"error": "Invalid order_by parameter."}, body)
					}
				})
			}
		}
	}
}

func newPageOrderHandler(t *testing.T) (http.Handler, *pageOrderAuthorizationDatabase, *pageOrderVisits) {
	t.Helper()
	database := &pageOrderAuthorizationDatabase{t: t}
	repository, err := store.NewRepository(database)
	require.NoError(t, err)
	sessions := &cycleMiddlewareSessions{recorder: &cycleMiddlewareRecorder{}, userTimezone: "UTC"}
	visits := &pageOrderVisits{}
	service, err := systemapi.NewPageReads(repository, sessions, visits)
	require.NoError(t, err)
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			pageReads: pagereads.NewEndpoints(service), pageReadsMetadata: &pageLifecycleProtocolAuthorizerStub{},
			readServicesSession: sessions, readServicesCookie: djangoSessionCookieConfig{Name: "session-id", Configured: true},
		},
	)
	return handler, database, visits
}

func TestPageReadClientDecodesValidationErrors(t *testing.T) {
	for _, test := range []struct {
		name          string
		order, search *string
		want          error
	}{
		{name: "order", order: new("invalid"), want: &pagereads.PageReadsBadRequest{Message: "Invalid order_by parameter."}},
		{name: "search", search: new("\x00"), want: pagereads.InvalidSearch{"Null characters are not allowed."}},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, database, visits := newPageOrderHandler(t)
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			transport := client.NewClient("http", strings.TrimPrefix(server.URL, "http://"), server.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			result, err := transport.List()(t.Context(), &pagereads.ListPayload{
				Slug: "demo", ProjectID: "00000000-0000-4000-8000-000000000001", SessionID: new("valid-session"),
				OrderBy: test.order, Search: test.search,
			})
			require.Nil(t, result)
			require.Equal(t, test.want, err)
			require.Equal(t, 1, database.queries)
			require.Zero(t, visits.calls)
		})
	}
}
