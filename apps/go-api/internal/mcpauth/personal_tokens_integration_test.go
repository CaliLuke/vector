//go:build integration

package mcpauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	loom "github.com/CaliLuke/loom/pkg"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const personalTokenSchema = `
CREATE TABLE users (id uuid PRIMARY KEY, is_active boolean NOT NULL, user_timezone text);
CREATE TABLE workspaces (id uuid PRIMARY KEY, slug text NOT NULL, deleted_at timestamptz);
CREATE TABLE workspace_members (workspace_id uuid NOT NULL, member_id uuid NOT NULL, role int NOT NULL, is_active boolean NOT NULL, deleted_at timestamptz);
CREATE TABLE api_tokens (token text PRIMARY KEY, user_id uuid NOT NULL, workspace_id uuid, is_active boolean NOT NULL, expired_at timestamptz, deleted_at timestamptz, last_used timestamptz);
INSERT INTO users VALUES ('10000000-0000-4000-8000-000000000001',true,'America/Los_Angeles');
INSERT INTO workspaces VALUES ('20000000-0000-4000-8000-000000000001','workspace-one',NULL), ('20000000-0000-4000-8000-000000000002','workspace-two',NULL);
INSERT INTO workspace_members VALUES ('20000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001',15,true,NULL);
INSERT INTO api_tokens VALUES ('test-only-personal-token','10000000-0000-4000-8000-000000000001','20000000-0000-4000-8000-000000000002',true,NULL,NULL,NULL);
`

func personalTokenDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.Start(ctx, func(ctx context.Context, url string) error {
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			return err
		}
		defer func() { require.NoError(t, conn.Close(context.WithoutCancel(ctx))) }()
		_, err = conn.Exec(ctx, personalTokenSchema)
		return err
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	pool, err := pgxpool.New(ctx, pair.SourceURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

func TestPersonalTokenGeneratedHTTPUsesCurrentDatabaseAuthorization(t *testing.T) {
	pool := personalTokenDatabase(t)
	keys, err := apikey.NewWithReaderWriter(pool, pool)
	require.NoError(t, err)
	memberships, err := NewWorkspaceReader(pool)
	require.NoError(t, err)
	throttle := &throttleFake{rate: apikey.Rate{Allowed: true}}
	auth, err := NewPersonalTokens(keys, memberships, throttle)
	require.NoError(t, err)
	cfg := mcptransport.DefaultConfig()
	cfg.Enabled, cfg.Mode, cfg.ResourceURL = true, "header", "https://vector.example/mcp"
	h, err := mcptransport.New(cfg, mcptransport.Options{HeaderAuth: auth.Authenticate, Service: tokenTransportService{}}, http.NotFoundHandler())
	require.NoError(t, err)
	call := func(t *testing.T, token, workspace string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, cfg.ResourceURL+"/api-key", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json, text/event-stream")
		r.Header.Set("Mcp-Protocol-Version", "2025-11-25")
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Workspace-Slug", workspace)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		assert.Empty(t, w.Header().Get("WWW-Authenticate"))
		assert.NotContains(t, w.Body.String(), token)
		return w
	}

	// The token's workspace metadata points to workspace-two. Its owner belongs
	// only to workspace-one; explicit current membership is the authority.
	require.Equal(t, http.StatusOK, call(t, "test-only-personal-token", "workspace-one").Code)
	require.Equal(t, http.StatusForbidden, call(t, "test-only-personal-token", "workspace-two").Code)
	require.Equal(t, http.StatusUnauthorized, call(t, "test-only-wrong-token", "workspace-one").Code)
	require.Equal(t, http.StatusBadRequest, call(t, "test-only-personal-token", "").Code)
	var used bool
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT last_used IS NOT NULL FROM api_tokens`).Scan(&used))
	assert.True(t, used)

	for _, tc := range []struct {
		name, mutation string
		status         int
	}{
		{"revoked key", `UPDATE api_tokens SET deleted_at=CURRENT_TIMESTAMP`, http.StatusUnauthorized},
		{"disabled key", `UPDATE api_tokens SET is_active=false`, http.StatusUnauthorized},
		{"expired key", `UPDATE api_tokens SET expired_at=CURRENT_TIMESTAMP-INTERVAL '1 second'`, http.StatusUnauthorized},
		{"inactive owner", `UPDATE users SET is_active=false`, http.StatusUnauthorized},
		{"inactive member", `UPDATE workspace_members SET is_active=false`, http.StatusForbidden},
		{"deleted member", `UPDATE workspace_members SET deleted_at=CURRENT_TIMESTAMP`, http.StatusForbidden},
		{"deleted workspace", `UPDATE workspaces SET deleted_at=CURRENT_TIMESTAMP`, http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, mutationErr := pool.Exec(t.Context(), tc.mutation)
			require.NoError(t, mutationErr)
			assert.Equal(t, tc.status, call(t, "test-only-personal-token", "workspace-one").Code)
			_, restoreErr := pool.Exec(t.Context(), `
UPDATE api_tokens SET is_active=true, expired_at=NULL, deleted_at=NULL;
UPDATE users SET is_active=true;
UPDATE workspace_members SET is_active=true, deleted_at=NULL;
UPDATE workspaces SET deleted_at=NULL;`)
			require.NoError(t, restoreErr)
		})
	}

	ctx, err := auth.Authenticate(t.Context(), tokenRequest())
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `DELETE FROM workspace_members`)
	require.NoError(t, err)
	_, err = Revalidate(ctx)
	require.ErrorIs(t, err, mcptransport.ErrAccessDenied)
	pool.Close()
	w := call(t, "test-only-personal-token", "workspace-one")
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "authentication unavailable\n", w.Body.String())
}

type tokenTransportService struct{}

func (tokenTransportService) Member(context.Context, *api.MemberRequest) (*api.MemberResult, error) {
	return nil, errors.New("authentication tests do not execute identity reads")
}

func (tokenTransportService) Project(context.Context, *api.ProjectRequest) (loom.Nullable[api.ProjectResult], error) {
	return loom.Nullable[api.ProjectResult]{}, errors.New("this fixture does not execute project reads")
}

func (tokenTransportService) DiscoverActions(context.Context) (mcptransport.ActionVisibility, error) {
	return mcptransport.ActionVisibility{"member": {"me": true, "list_workspace": true, "list_project": true, "list_roles": true, "retrieve_role": true}, "project": {"list": true, "retrieve": true}}, nil
}

func (tokenTransportService) Cycle(context.Context, *api.CycleRequest) (*api.CycleResult, error) {
	return nil, errors.New("this fixture does not execute cycle reads")
}

func (tokenTransportService) Module(context.Context, *api.ModuleRequest) (*api.ModuleResult, error) {
	return nil, nil
}

func (tokenTransportService) Label(context.Context, *api.LabelRequest) (*api.LabelResult, error) {
	return nil, nil
}

func (tokenTransportService) WorkitemComment(context.Context, *api.WorkitemCommentRequest) (*api.WorkitemCommentResult, error) {
	return nil, nil
}

func (tokenTransportService) WorkitemLink(context.Context, *api.WorkitemLinkRequest) (*api.WorkitemLinkResult, error) {
	return nil, nil
}

func (tokenTransportService) WorkitemActivity(context.Context, *api.WorkitemActivityRequest) (*api.WorkitemActivityResult, error) {
	return nil, nil
}

func (tokenTransportService) Milestone(context.Context, *api.MilestoneRequest) (loom.Nullable[api.MilestoneResult], error) {
	return loom.NullValue[api.MilestoneResult](), nil
}
