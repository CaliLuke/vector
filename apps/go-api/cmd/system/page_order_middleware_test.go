package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	queries atomic.Int64
}

func (database *pageOrderAuthorizationDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	database.queries.Add(1)
	return database
}

func (database *pageOrderAuthorizationDatabase) Scan(destinations ...any) error {
	if database.queries.Load() != 1 {
		return errors.New("invalid ordering must stop after authorization")
	}
	if len(destinations) != 1 {
		return fmt.Errorf("unexpected page query with %d scan destinations", len(destinations))
	}
	switch target := destinations[0].(type) {
	case *bool:
		*target = true
	case *string:
		*target = "allowed"
	default:
		return fmt.Errorf("unexpected page content read into %T", target)
	}
	return nil
}

type pageOrderVisits struct{ calls atomic.Int64 }

func (visits *pageOrderVisits) Record(context.Context, uuid.UUID, string, string, uuid.UUID, uuid.UUID) error {
	visits.calls.Add(1)
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
					require.Equal(t, int64(1), database.queries.Load())
					require.Zero(t, visits.calls.Load())
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
	database := &pageOrderAuthorizationDatabase{}
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
			require.Equal(t, int64(1), database.queries.Load())
			require.Zero(t, visits.calls.Load())
		})
	}
}
