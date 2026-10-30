package systemapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	server "github.com/CaliLuke/go-plane/gen/http/intake_inbox_reads/server"
	api "github.com/CaliLuke/go-plane/gen/intake_inbox_reads"
	store "github.com/CaliLuke/go-plane/internal/intakeinboxreads"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestIntakeValidationErrorsUseClientHTTPStatus(t *testing.T) {
	base := "/api/workspaces/demo/projects/" + intakeReadsProjectID
	for _, route := range []string{
		"/api/workspaces/demo/draft-issues/",
		base + "/intake-issues/",
		base + "/inbox-issues/",
		base + "/intake-work-items/" + intakeReadsObjectID + "/description-versions/",
	} {
		for _, test := range []struct {
			name        string
			err         error
			status      int
			key, detail string
		}{
			{"limit", store.ErrInvalidPerPage, 400, "detail", "Invalid per_page parameter."},
			{"cursor", store.ErrInvalidCursor, 400, "detail", "Invalid cursor parameter."},
			{"description cursor", store.ErrInvalidDescriptionCursor, 400, "detail", "Invalid cursor parameter."},
			{"filter", store.ErrBadRequest, 400, "detail", "Invalid pagination or filter parameter."},
			{"unexpected", errors.New("private database error"), 500, "error", intakeInboxReadsInternalErrorDetail},
		} {
			t.Run(route+test.name, func(t *testing.T) {
				repository := &intakeInboxReadsRepositoryStub{err: test.err}
				service, _ := authenticatedIntakeReadsService(t, repository)
				mux := loomhttp.NewMuxer()
				handler := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
				server.Mount(mux, handler)
				request := httptest.NewRequest(http.MethodGet, route, nil)
				request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				if response.Code != test.status {
					t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
				}
				var body map[string]string
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body[test.key] != test.detail || len(body) != 1 {
					t.Fatalf("body=%v", body)
				}
			})
		}
	}
}
