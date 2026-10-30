package systemapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/CaliLuke/go-plane/gen/cycle_reads"
	server "github.com/CaliLuke/go-plane/gen/http/cycle_reads/server"
	store "github.com/CaliLuke/go-plane/internal/cyclereads"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestCyclePaginationErrorsUseClientHTTPStatus(t *testing.T) {
	for _, test := range []struct {
		name        string
		failure     error
		status      int
		key, detail string
	}{
		{"zero page size", store.ErrZeroPageSize, 400, "detail", "Invalid per_page parameter."},
		{"negative page size", store.ErrNegativePageSize, 400, "detail", "Invalid per_page parameter."},
		{"previous slice", store.ErrPreviousPageSlice, 400, "detail", "Invalid cursor parameter."},
		{"overflow", store.ErrPaginationOverflow, 400, "detail", "Invalid cursor parameter."},
		{"malformed cursor", store.ErrInvalidPagination, 400, "detail", "Invalid cursor parameter."},
		{"parse error", store.ErrPaginationParsing, 400, "detail", "Error in parsing"},
		{"unexpected", errors.New("private database error"), 500, "error", cycleReadsInternalErrorDetail},
	} {
		for _, wrapped := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/wrapped=%t", test.name, wrapped), func(t *testing.T) {
				failure := test.failure
				if wrapped {
					failure = fmt.Errorf("private repository context: %w", failure)
				}
				repository := &cycleReadsRepositoryStub{err: failure}
				service, err := NewCycleReads(repository, cycleReadsSessionsStub{}, &cycleRecentVisitsStub{})
				require.NoError(t, err)
				mux := loomhttp.NewMuxer()
				handler := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
				server.Mount(mux, handler)
				request := httptest.NewRequest(http.MethodGet, "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000003/cycles/00000000-0000-4000-8000-000000000001/cycle-issues/", nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
				request = request.WithContext(WithReadServiceDjangoHTML500(request.Context()))
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				require.Equal(t, test.status, response.Code, response.Body.String())
				require.False(t, ReadServiceDjangoHTML500(request.Context()), "validation must not request an HTML 500 rewrite")
				var body map[string]string
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
				require.Equal(t, map[string]string{test.key: test.detail}, body)
			})
		}
	}
}
