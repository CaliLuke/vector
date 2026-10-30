package systemapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	server "github.com/CaliLuke/go-plane/gen/http/page_reads/server"
	api "github.com/CaliLuke/go-plane/gen/page_reads"
	store "github.com/CaliLuke/go-plane/internal/pagereads"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestPageOrderErrorsReturnJSON400(t *testing.T) {
	for _, detail := range []bool{false, true} {
		for _, wrapped := range []bool{false, true} {
			t.Run(fmt.Sprintf("detail=%t/wrapped=%t", detail, wrapped), func(t *testing.T) {
				failure := store.ErrInvalidOrder
				if wrapped {
					failure = fmt.Errorf("private repository context: %w", failure)
				}
				checkPageErrorResponse(t, detail, failure, http.StatusBadRequest, "Invalid order_by parameter.")
			})
		}
	}
}

func TestPageUnexpectedErrorsRemainSanitized(t *testing.T) {
	for _, detail := range []bool{false, true} {
		t.Run(fmt.Sprintf("detail=%t", detail), func(t *testing.T) {
			checkPageErrorResponse(t, detail, errors.New("private database details"), http.StatusInternalServerError, pageReadsInternalErrorDetail)
		})
	}
}

func checkPageErrorResponse(t *testing.T, detail bool, failure error, status int, message string) {
	t.Helper()
	service, err := NewPageReads(&pageReadsRepositoryStub{err: failure}, pageReadsSessionsStub{}, &pageRecentVisitsStub{})
	require.NoError(t, err)
	mux := loomhttp.NewMuxer()
	handler := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
	server.Mount(mux, handler)
	path := "/api/workspaces/demo/projects/" + pageReadsProjectID + "/pages/"
	if detail {
		path += pageReadsPageID + "/"
	}
	request := httptest.NewRequest(http.MethodGet, path+"?order_by=invalid", nil)
	request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
	request = request.WithContext(WithReadServiceDjangoHTML500(request.Context()))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, status, response.Code, response.Body.String())
	require.Contains(t, response.Header().Get("Content-Type"), "application/json")
	require.False(t, ReadServiceDjangoHTML500(request.Context()))
	var body map[string]string
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, map[string]string{"error": message}, body)
}
