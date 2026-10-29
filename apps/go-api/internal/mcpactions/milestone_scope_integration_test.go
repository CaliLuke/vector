//go:build integration

package mcpactions_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mcpactions"
	"github.com/CaliLuke/go-plane/internal/milestones"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMilestoneMCPDiscoveryTracksProjectAndOAuthPermissions(t *testing.T) {
	f := newMilestoneFixture(t)
	_, err := f.pool.Exec(t.Context(), `INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at)
 VALUES('00000000-0000-4000-8000-000000000099','Other','00000000-0000-4000-8000-000000000003','other','UTC','',now(),now())`)
	require.NoError(t, err)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		for _, mode := range []string{"personal token", "OAuth"} {
			t.Run(version+"/"+mode, func(t *testing.T) {
				call := f.call(t, mode, version)
				pat, oauth := f.handler(t, nil), oauthHandler(t, f.fixture, f.oauth.Authenticate)
				discover := func() []string {
					var result rpcResult
					if mode == "OAuth" {
						result = oauthRPC(t, oauth, version, "oauth-user", "workspace-one", "tools/list", map[string]any{}, 200)
					} else {
						result = rpcVersion(t, pat, "test-one", "tools/list", map[string]any{}, version)
					}
					return discoveredMilestoneActions(t, result)
				}
				for _, tc := range []struct {
					name, sql   string
					read, write bool
				}{
					{"member", `SELECT 1`, true, true},
					{"restricted guest", `UPDATE project_members SET role=5`, false, false},
					{"feature guest", `UPDATE project_members SET role=5; UPDATE projects SET guest_view_all_features=true`, true, false},
					{"workspace guest project member", `UPDATE workspace_members SET role=5`, true, true},
					{"workspace admin without project membership", `UPDATE workspace_members SET role=20; UPDATE project_members SET is_active=false`, false, false},
					{"archived", `UPDATE projects SET archived_at=now()`, false, false},
					{"deleted", `UPDATE projects SET deleted_at=now()`, false, false},
					{"foreign membership", `UPDATE project_members SET workspace_id='00000000-0000-4000-8000-000000000099'`, false, false},
				} {
					t.Run(tc.name, func(t *testing.T) {
						_, err := f.pool.Exec(t.Context(), tc.sql)
						require.NoError(t, err)
						t.Cleanup(func() {
							cleanup, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
							defer cancel()
							_, err := f.pool.Exec(cleanup, `UPDATE project_members SET role=15,is_active=true,workspace_id='00000000-0000-4000-8000-000000000001'; UPDATE workspace_members SET role=15; UPDATE projects SET archived_at=NULL,deleted_at=NULL,guest_view_all_features=false`)
							require.NoError(t, err)
						})
						actions := discover()
						if tc.read {
							require.Contains(t, actions, "list")
							require.Contains(t, actions, "list_workitems")
						} else {
							require.Empty(t, actions)
							milestoneFailure(t, call(map[string]any{"action": "list"}), "forbidden")
						}
						if tc.write {
							require.Len(t, actions, 7)
						} else {
							require.NotContains(t, actions, "create")
							milestoneFailure(t, call(map[string]any{"action": "create", "title": uuid.NewString(), "request_id": uuid.NewString()}), "forbidden")
						}
					})
				}
				if mode == "OAuth" {
					f.issuer.mu.Lock()
					f.issuer.scope = "vector:read"
					f.issuer.mu.Unlock()
					require.ElementsMatch(t, []string{"list", "retrieve", "list_workitems"}, discover())
					milestoneFailure(t, call(map[string]any{"action": "create", "title": "read only"}), "forbidden")
					f.issuer.mu.Lock()
					f.issuer.scope = "vector:read vector:write"
					f.issuer.mu.Unlock()
				}
			})
		}
	}
	var records, receipts int
	require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM milestones),(SELECT count(*) FROM mutation_receipts)`).Scan(&records, &receipts))
	require.Zero(t, records)
	require.Zero(t, receipts)
}

type changingMilestoneApplication struct {
	*milestones.Repository
	change func()
}

func (r changingMilestoneApplication) Create(ctx context.Context, o milestones.WriteOptions, fields milestones.Fields) (milestones.MutationResult, error) {
	r.change()
	return r.Repository.Create(ctx, o, fields)
}
func (r changingMilestoneApplication) List(ctx context.Context, scope milestones.Scope, q djangoapi.Query) (milestones.Page[milestones.Record], error) {
	r.change()
	return r.Repository.List(ctx, scope, q)
}

func TestMilestoneMCPRevalidatesCredentialInsideWrite(t *testing.T) {
	for _, mode := range []string{"personal token", "OAuth"} {
		t.Run(mode, func(t *testing.T) {
			f := newMilestoneFixture(t)
			apps := databaseApplications(t, f.pool)
			apps.Milestones = changingMilestoneApplication{apps.Milestones.(*milestones.Repository), func() {
				if mode == "OAuth" {
					f.issuer.mu.Lock()
					f.issuer.active = false
					f.issuer.mu.Unlock()
				} else {
					_, err := f.pool.Exec(t.Context(), `UPDATE api_tokens SET is_active=false`)
					require.NoError(t, err)
				}
			}}
			service, err := mcpactions.New(apps)
			require.NoError(t, err)
			f.service = service
			result := f.call(t, mode, "2026-07-28")(map[string]any{"action": "create", "title": "Must roll back", "request_id": uuid.NewString()})
			milestoneFailure(t, result, "unauthorized")
			var records, receipts, effects int
			require.NoError(t, f.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM milestones),(SELECT count(*) FROM mutation_receipts),(SELECT count(*) FROM work_item_effects)`).Scan(&records, &receipts, &effects))
			require.Zero(t, records)
			require.Zero(t, receipts)
			require.Zero(t, effects)
		})
	}
}

func TestMilestoneMCPReadRetainsAuthenticatedWorkspaceBinding(t *testing.T) {
	f := newMilestoneFixture(t)
	apps := databaseApplications(t, f.pool)
	apps.Milestones = changingMilestoneApplication{apps.Milestones.(*milestones.Repository), func() {
		_, err := f.pool.Exec(t.Context(), `UPDATE workspaces SET slug='renamed' WHERE id=$1`, testsupport.LabelWorkspace)
		require.NoError(t, err)
	}}
	service, err := mcpactions.New(apps)
	require.NoError(t, err)
	f.service = service
	milestoneFailure(t, f.call(t, "OAuth", "2026-07-28")(map[string]any{"action": "list"}), "forbidden")
	// Renaming retains membership: unscoped discovery uses the current workspace
	// name, while the in-flight read above remains bound to its admitted scope.
	h := oauthHandler(t, f.fixture, f.oauth.Authenticate)
	result := oauthRPC(t, h, "2026-07-28", "oauth-user", "", "tools/list", map[string]any{}, http.StatusOK)
	require.Contains(t, discoveredMilestoneActions(t, result), "list")
	_, err = f.pool.Exec(t.Context(), `UPDATE workspace_members SET is_active=false WHERE workspace_id=$1`, testsupport.LabelWorkspace)
	require.NoError(t, err)
	result = oauthRPC(t, h, "2026-07-28", "oauth-user", "", "tools/list", map[string]any{}, http.StatusOK)
	require.Empty(t, discoveredMilestoneActions(t, result))
}
