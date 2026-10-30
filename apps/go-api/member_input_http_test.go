package systemapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	server "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/http/project_membership_lifecycle/server"
	api "github.com/CaliLuke/go-plane/projectmembershiplifecycle/gen/project_membership_lifecycle"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type memberInputDatabase struct {
	t               *testing.T
	denied          bool
	writes, effects int
}

type memberInputRow func(...any) error

func (row memberInputRow) Scan(dest ...any) error { return row(dest...) }

func (database *memberInputDatabase) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	return memberInputRow(func(dest ...any) error {
		switch {
		case strings.Contains(query, "SELECT EXISTS"):
			*dest[0].(*bool) = !database.denied
		case strings.Contains(query, "SELECT pm.member_id::text,pm.role"):
			*dest[0].(**string) = new("00000000-0000-4000-8000-00000000000b")
			*dest[1].(*int) = 15
		case strings.Contains(query, "SELECT wm.role"), strings.Contains(query, "SELECT role FROM project_members"):
			*dest[0].(*int) = 20
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
func (database *memberInputDatabase) Task(context.Context, string, []any, map[string]any) error {
	database.effects++
	return errors.New("unexpected membership notification")
}

func TestMalformedMemberWritesReturn400WithoutEffects(t *testing.T) {
	for _, test := range []struct {
		name, method, body, contentType, want string
		denied                                bool
	}{
		{name: "null patch", method: "PATCH", body: `null`, want: `{"non_field_errors":["No data provided"]}`},
		{name: "boolean patch", method: "PATCH", body: `true`, want: `{"non_field_errors":["Invalid data. Expected a dictionary, but got bool."]}`},
		{name: "numeric patch", method: "PATCH", body: `5`, want: `{"non_field_errors":["Invalid data. Expected a dictionary, but got int."]}`},
		{name: "role list", method: "PATCH", body: `["role"]`, want: `{"non_field_errors":["Invalid data. Expected a dictionary, but got list."]}`},
		{name: "invalid role", method: "PATCH", body: `{"role":"invalid","comment":"must not persist"}`, want: `{"role":["A valid integer is required."]}`},
		{name: "null role", method: "PATCH", body: `{"role":null}`, want: `{"role":["This field may not be null."]}`},
		{name: "empty form role", method: "PATCH", body: `role=`, contentType: "application/x-www-form-urlencoded", want: `{"role":["A valid integer is required."]}`},
		{name: "null bulk", method: "POST", body: `null`, want: `{"non_field_errors":["No data provided"]}`},
		{name: "array bulk", method: "POST", body: `[]`, want: `{"non_field_errors":["Invalid data. Expected a dictionary, but got list."]}`},
		{name: "members field", method: "POST", body: `{"members":true}`, want: `{"error":"Members must be a list."}`},
		{name: "member entry", method: "POST", body: `{"members":[true]}`, want: `{"error":"Each member must be an object."}`},
		{name: "member identity", method: "POST", body: `{"members":[{"member_id":[]}]}`, want: `{"error":"Please provide valid detail"}`},
		{name: "denied malformed patch", method: "PATCH", body: `{`, denied: true, want: `{"error":"You don't have the required permissions."}`},
		{name: "denied malformed bulk", method: "POST", body: `{`, denied: true, want: `{"error":"You don't have the required permissions."}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := &memberInputDatabase{t: t, denied: test.denied}
			repository, err := store.NewLifecycleRepository(database, database, "http://localhost:8000")
			require.NoError(t, err)
			sessions := &activitySessionStub{principal: djangosession.Principal{UserID: "00000000-0000-4000-8000-00000000000a", UserTimezone: "UTC"}}
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
			require.Zero(t, database.effects)
		})
	}
}
