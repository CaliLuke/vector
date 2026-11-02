package systemapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	server "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/server"
	api "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type memberInputDatabase struct {
	t              *testing.T
	denied         bool
	writes, begins int
}

type memberInputRow func(...any) error

func (row memberInputRow) Scan(dest ...any) error { return row(dest...) }

func (database *memberInputDatabase) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	return memberInputRow(func(dest ...any) error {
		switch {
		case strings.Contains(query, "SELECT EXISTS"):
			*dest[0].(*bool) = !database.denied
		default:
			database.t.Fatalf("invalid input reached a data query: %s", query)
		}
		return nil
	})
}

func (database *memberInputDatabase) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	database.writes++
	return pgconn.CommandTag{}, errors.New("unexpected membership write")
}
func (database *memberInputDatabase) Begin(context.Context) (pgx.Tx, error) {
	database.begins++
	return nil, errors.New("invalid membership data must not start a transaction")
}

func TestMalformedMemberWritesReturn400WithoutEffects(t *testing.T) {
	for _, test := range []struct {
		name, method, body, contentType, want string
		denied                                bool
	}{
		{name: "object required", method: "PATCH", body: `true`, want: `{"error":"Membership data must be an object"}`},
		{name: "role choice", method: "PATCH", body: `{"role":true}`, want: `{"error":"Role must be 5, 15, or 20"}`},
		{name: "immutable identity", method: "PATCH", body: `{"member":"00000000-0000-4000-8000-00000000000b"}`, want: `{"error":"Unsupported membership field: member"}`},
		{name: "bulk list required", method: "POST", body: `{"members":true}`, want: `{"error":"At least one member is required"}`},
		{name: "permission precedes body", method: "PATCH", body: `{`, denied: true, want: `{"error":"Project administrator permission is required"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := &memberInputDatabase{t: t, denied: test.denied}
			repository, err := store.NewLifecycleRepository(database, "http://localhost:8000")
			require.NoError(t, err)
			sessions := &activitySessionStub{principal: browsersession.Principal{UserID: "00000000-0000-4000-8000-00000000000a", UserTimezone: "UTC"}}
			service, err := NewProjectMembershipLifecycle(repository, &activityAPIKeyStub{}, &activityThrottleStub{}, sessions)
			require.NoError(t, err)
			mux := loomhttp.NewMuxer()
			handler := server.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, loomhttp.RuntimeCORSPolicy{})
			metadata, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie", "Content-Type"}, nil)
			require.NoError(t, err)
			handler.Use(loomhttp.RequestMetadataMiddleware(metadata))
			server.Mount(mux, handler)
			path := "/api/workspaces/demo/projects/00000000-0000-4000-8000-000000000100/members/"
			if test.method == "PATCH" {
				path += "00000000-0000-4000-8000-000000000200/"
			}
			request := httptest.NewRequest(test.method, path, strings.NewReader(test.body))
			contentType := test.contentType
			if contentType == "" {
				contentType = "application/json"
			}
			request.Header.Set("Content-Type", contentType)
			request.AddCookie(&http.Cookie{Name: "session-id", Value: "session"})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			wantStatus := http.StatusBadRequest
			if test.denied {
				wantStatus = http.StatusForbidden
			}
			require.Equal(t, wantStatus, response.Code, response.Body.String())
			require.JSONEq(t, test.want, response.Body.String())
			require.Zero(t, database.writes)
			require.Zero(t, database.begins)
		})
	}
}
