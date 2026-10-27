//go:build integration

package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	store "github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	invitationserver "github.com/CaliLuke/go-plane/invitationsmembership/gen/http/invitations_membership/server"
	api "github.com/CaliLuke/go-plane/invitationsmembership/gen/invitations_membership"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type directoryReadOnlyTasks struct{}

func (directoryReadOnlyTasks) Task(context.Context, string, []any, map[string]any) error {
	return errors.New("directory reads must not publish tasks")
}

type directoryHTTPThrottle struct {
	db     *pgx.Conn
	revoke bool
	calls  int
}

func (t *directoryHTTPThrottle) Allow(ctx context.Context, _ string) (apikey.Rate, error) {
	t.calls++
	if t.revoke {
		if _, err := t.db.Exec(ctx, `UPDATE workspace_members SET is_active=false WHERE member_id='00000000-0000-4000-8000-000000000001'`); err != nil {
			return apikey.Rate{}, err
		}
	}
	return apikey.Rate{Allowed: true}, nil
}

func directoryHTTPFixture(t *testing.T) (http.Handler, *pgx.Conn, *directoryHTTPThrottle) {
	t.Helper()
	_, db := postgresfixture.OpenSQL(t, httpDirectorySchema)
	repository, err := store.NewInvitationsRepositoryWithReaderWriter(db, db, directoryReadOnlyTasks{}, "http://localhost:8000", "test-only")
	require.NoError(t, err)
	throttle := &directoryHTTPThrottle{db: db}
	service, err := newInvitationsMembershipService(repository, &invitationsMembershipKeyValidator{}, invitationsMembershipSessionValidator{}, throttle)
	require.NoError(t, err)
	mux := loomhttp.NewMuxer()
	server := invitationserver.New(api.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	server.Mount(mux)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(WithReadServiceQueryPresence(r.Context(), r.URL.Query())))
	})
	return handler, db, throttle
}

func directoryGET(t *testing.T, h http.Handler, path string, status int) []byte {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/current/"+path, nil)
	request.Header.Set("X-Api-Key", "test-key")
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	require.Equal(t, status, response.Code, response.Body.String())
	return response.Body.Bytes()
}

func TestHTTPMemberDirectoryUsesSharedFilteringAndRoleCatalogue(t *testing.T) {
	h, _, _ := directoryHTTPFixture(t)
	body := directoryGET(t, h, "members-lite/?first_name=BO&is_active=false&is_bot=false&role_slug=guest&per_page=1", 200)
	assert.JSONEq(t, `{"count":1,"extra_stats":null,"grouped_by":null,"next_cursor":"1:1:0","next_page_results":false,"prev_cursor":"1:-1:1","prev_page_results":false,"results":[{"id":"00000000-0000-4000-8000-000000000002","first_name":"Bob","last_name":"Member","email":"bob@example.test","avatar":"","avatar_url":null,"display_name":"bob","role":5,"role_slug":"guest","is_active":false,"is_bot":false}],"sub_grouped_by":null,"total_count":1,"total_pages":1,"total_results":1}`, string(body))
	all := directoryGET(t, h, "members-lite/", 200)
	var page struct {
		Total   int    `json:"total_count"`
		Cursor  string `json:"next_cursor"`
		Results []json.RawMessage
	}
	require.NoError(t, json.Unmarshal(all, &page))
	assert.Equal(t, 2, page.Total)
	assert.Equal(t, "1000:1:0", page.Cursor)
	var first map[string]any
	require.NoError(t, json.Unmarshal(page.Results[0], &first))
	assert.Equal(t, "", first["email"], "preserve the existing HTTP required-string projection")
	roles := directoryGET(t, h, "roles/?namespace=workspace&per_page=1", 200)
	require.NoError(t, json.Unmarshal(roles, &page))
	require.Len(t, page.Results, 1)
	assert.Equal(t, 4, page.Total)
	var role struct{ ID, Slug string }
	require.NoError(t, json.Unmarshal(page.Results[0], &role))
	assert.Equal(t, "owner", role.Slug)
	retrieved := directoryGET(t, h, "roles/"+role.ID+"/", 200)
	assert.JSONEq(t, string(page.Results[0]), string(retrieved))
	directoryGET(t, h, "roles/00000000-0000-4000-8000-000000000000/", 404)
	directoryGET(t, h, "roles/not-a-uuid/", 400)
	// Preserve the legacy cursor's decimal size and nonzero previous direction.
	directoryGET(t, h, "members-lite/?per_page=1&cursor=1.0:1:2", 200)
	directoryGET(t, h, "members-lite/?per_page=0", 500)
	directoryGET(t, h, "members-lite/?per_page=1&cursor=2:1:1", 500)
}

func TestHTTPMemberDirectoryRejectsBadFiltersAfterThrottling(t *testing.T) {
	h, _, throttle := directoryHTTPFixture(t)
	for _, path := range []string{
		"members-lite/?is_active=False", "members-lite/?is_bot=1", "members-lite/?is_bot=",
		"members-lite/?first_name=A&first_name=B", "members-lite/?order_by=email,-email",
		"roles/?namespace=other", "roles/?per_page=0", "roles/?per_page=1001", "roles/?per_page=abc",
		"roles/?namespace=project&namespace=workspace", "roles/?cursor=1:1:1",
	} {
		t.Run(path, func(t *testing.T) {
			before := throttle.calls
			directoryGET(t, h, path, 400)
			assert.Equal(t, before+1, throttle.calls)
		})
	}
}

func TestHTTPMemberDirectoryRechecksPrimaryAfterThrottleRevocation(t *testing.T) {
	h, db, throttle := directoryHTTPFixture(t)
	roles := directoryGET(t, h, "roles/", 200)
	var page struct{ Results []struct{ ID string } }
	require.NoError(t, json.Unmarshal(roles, &page))
	require.NotEmpty(t, page.Results)
	throttle.revoke = true
	for _, path := range []string{"members-lite/", "roles/", "roles/" + page.Results[0].ID + "/"} {
		_, err := db.Exec(t.Context(), `UPDATE workspace_members SET is_active=true WHERE member_id='00000000-0000-4000-8000-000000000001'`)
		require.NoError(t, err)
		body := directoryGET(t, h, path, 403)
		assert.NotContains(t, string(body), "bob@example.test")
	}
	throttle.revoke = false
	_, err := db.Exec(t.Context(), `UPDATE workspace_members SET is_active=true WHERE member_id='00000000-0000-4000-8000-000000000001'; UPDATE workspaces SET deleted_at=now()`)
	require.NoError(t, err)
	directoryGET(t, h, "members-lite/", 404)
	directoryGET(t, h, "roles/", 403)
}

const httpDirectorySchema = `
CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean NOT NULL,first_name text,last_name text,email text,avatar text,avatar_asset_id uuid,display_name text,is_bot boolean NOT NULL);
CREATE TABLE workspaces(id uuid PRIMARY KEY,slug text NOT NULL,owner_id uuid NOT NULL,deleted_at timestamptz);
CREATE TABLE workspace_members(id uuid PRIMARY KEY,workspace_id uuid NOT NULL,member_id uuid NOT NULL,role integer NOT NULL,is_active boolean NOT NULL,deleted_at timestamptz,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
INSERT INTO users VALUES
 ('00000000-0000-4000-8000-000000000001',true,'Alice','Owner',NULL,'',NULL,'alice',false),
 ('00000000-0000-4000-8000-000000000002',true,'Bob','Member','bob@example.test','',NULL,'bob',false);
INSERT INTO workspaces VALUES ('20000000-0000-4000-8000-000000000001','current','00000000-0000-4000-8000-000000000001',NULL);
INSERT INTO workspace_members(id,workspace_id,member_id,role,is_active) VALUES
 ('30000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000001',20,true),
 ('30000000-0000-4000-8000-000000000002','20000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002',5,false);
`
