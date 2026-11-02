//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"

	"github.com/CaliLuke/go-plane/internal/mcpauth"
	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type liveIssuer struct {
	mu     sync.Mutex
	server *httptest.Server
	active bool
	scope  string
	client string
	status int
	calls  int
}

func oauthFixture(t *testing.T) (fixture, *pgxpool.Pool, *mcpauth.OAuth, *liveIssuer) {
	t.Helper()
	f, pool := databaseFixture(t)
	auth, issuer := oauthForPool(t, pool, "10000000-0000-4000-8000-000000000001")
	return f, pool, auth, issuer
}

func oauthForPool(t *testing.T, pool *pgxpool.Pool, actor string) (*mcpauth.OAuth, *liveIssuer) {
	t.Helper()
	issuer := &liveIssuer{active: true, scope: "vector:read", client: "agent-client", status: 200}
	issuer.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/openid-configuration" {
			testsupport.WriteJSON(t, w, map[string]any{"issuer": issuer.server.URL, "authorization_endpoint": issuer.server.URL + "/authorize",
				"token_endpoint": issuer.server.URL + "/token", "jwks_uri": issuer.server.URL + "/keys",
				"response_types_supported": []string{"code"}, "response_modes_supported": []string{"form_post"},
				"code_challenge_methods_supported": []string{"S256"}, "scopes_supported": []string{"openid"},
				"token_endpoint_auth_methods_supported": []string{"client_secret_basic"}, "id_token_signing_alg_values_supported": []string{"RS256"}})
			return
		}
		issuer.mu.Lock()
		defer issuer.mu.Unlock()
		issuer.calls++
		if r.ParseForm() != nil || r.Form.Get("token") != "oauth-user" {
			testsupport.Write(t, w, []byte(`{"active":false}`))
			return
		}
		w.WriteHeader(issuer.status)
		testsupport.WriteJSON(t, w, map[string]any{"active": issuer.active, "iss": issuer.server.URL, "sub": "subject-one", "client_id": issuer.client,
			"aud": "https://vector.example/mcp", "exp": time.Now().Add(time.Hour).Unix(), "scope": issuer.scope, "token_type": "Bearer", "grant_type": "authorization_code"})
	}))
	t.Cleanup(issuer.server.Close)
	provider, err := oauth.NewProvider(t.Context(), oauth.Config{Issuer: issuer.server.URL, Resource: "https://vector.example/mcp", IntrospectionURL: issuer.server.URL + "/introspect",
		IntrospectionClientID: "resource-client", IntrospectionClientSecret: "resource-secret", LinkClientID: "link-client", LinkClientSecret: "link-secret"}, issuer.server.Client())
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `CREATE TABLE IF NOT EXISTS mcp_oauth_links(issuer text NOT NULL,subject text NOT NULL,user_id uuid NOT NULL REFERENCES users(id),PRIMARY KEY(issuer,subject),UNIQUE(issuer,user_id));`)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `INSERT INTO mcp_oauth_links(issuer,subject,user_id) VALUES($1,'subject-one',$2)`, issuer.server.URL, actor)
	require.NoError(t, err)
	links, err := oauth.NewLinks(pool, issuer.server.URL)
	require.NoError(t, err)
	members, err := mcpauth.NewWorkspaceReader(pool)
	require.NoError(t, err)
	auth, err := mcpauth.NewOAuth(provider, links, members, throttle{})
	require.NoError(t, err)
	return auth, issuer
}

func oauthHandler(t *testing.T, f fixture, authenticate mcptransport.Authenticate) http.Handler {
	t.Helper()
	cfg := mcptransport.DefaultConfig()
	cfg.Enabled, cfg.Mode, cfg.ResourceURL = true, "oauth", "https://vector.example/mcp"
	h, err := mcptransport.New(cfg, mcptransport.Options{Service: f.service, OAuthAuth: authenticate, OAuthIssuer: "https://issuer.example"}, http.NotFoundHandler())
	require.NoError(t, err)
	return h
}

func oauthCall(t *testing.T, h http.Handler, token, workspace, input string, status int) rpcResult {
	t.Helper()
	return toolCall(t, h, token, workspace, "member", input, status)
}

func TestOAuthGeneratedMembersUseOptionalWorkspaceAndCurrentIssuer(t *testing.T) {
	f, _, auth, issuer := oauthFixture(t)
	h := oauthHandler(t, f, auth.Authenticate)

	me := oauthCall(t, h, "oauth-user", "", `{"action":"me"}`, 200)
	require.False(t, me.IsError, me.Content)
	require.Contains(t, string(me.StructuredContent), "10000000-0000-4000-8000-000000000001")
	roles := oauthCall(t, h, "oauth-user", "", `{"action":"list_roles","workspace_slug":"workspace-one"}`, 200)
	require.False(t, roles.IsError, roles.Content)
	require.Contains(t, string(roles.StructuredContent), `"total_count":7`)
	var catalogue struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(roles.StructuredContent, &catalogue))
	require.Len(t, catalogue.Results, 7)
	role := oauthCall(t, h, "oauth-user", "workspace-one", fmt.Sprintf(`{"action":"retrieve_role","role_id":%q}`, catalogue.Results[0].ID), 200)
	require.False(t, role.IsError, role.Content)
	require.Contains(t, string(role.StructuredContent), catalogue.Results[0].ID)
	workspace := oauthCall(t, h, "oauth-user", "workspace-one", `{"action":"list_workspace","first_name":"zo"}`, 200)
	require.False(t, workspace.IsError, workspace.Content)
	require.Contains(t, string(workspace.StructuredContent), `"total_count":1`)
	project := oauthCall(t, h, "oauth-user", "", `{"action":"list_project","workspace_slug":"workspace-one","project_id":"30000000-0000-4000-8000-000000000001"}`, 200)
	require.False(t, project.IsError, project.Content)
	require.Len(t, project.Content, 1)
	require.Contains(t, project.Content[0].Text, "10000000-0000-4000-8000-000000000001")

	missing := oauthCall(t, h, "oauth-user", "", `{"action":"list_roles"}`, 200)
	require.True(t, missing.IsError)
	denied := oauthCall(t, h, "oauth-user", "", `{"action":"list_roles","workspace_slug":"workspace-two"}`, 200)
	require.True(t, denied.IsError)
	oauthCall(t, h, "oauth-user", "workspace-one", `{"action":"me","workspace_slug":"workspace-two"}`, 400)
	oauthCall(t, h, "test-one", "", `{"action":"me"}`, 401)

	issuer.mu.Lock()
	issuer.scope = "vector:write"
	issuer.mu.Unlock()
	oauthCall(t, h, "oauth-user", "", `{"action":"me"}`, 403)
	issuer.mu.Lock()
	issuer.status = 503
	issuer.mu.Unlock()
	oauthCall(t, h, "oauth-user", "", `{"action":"me"}`, 503)
}

func TestOAuthGeneratedReadsRevalidateBetweenAuthenticationAndCall(t *testing.T) {
	for _, change := range []string{"token", "scope", "issuer outage", "unlink", "account", "membership", "client", "link reassigned"} {
		for _, target := range []struct{ tool, workspace, input string }{
			{"member", "workspace-one", `{"action":"list_roles"}`},
			{"workspace", "", `{"action":"list"}`},
		} {
			t.Run(change+"/"+target.tool, func(t *testing.T) {
				f, pool, auth, issuer := oauthFixture(t)
				h := oauthHandler(t, f, func(ctx context.Context, r *http.Request) (context.Context, error) {
					verified, err := auth.Authenticate(ctx, r)
					if err != nil {
						return nil, err
					}
					switch change {
					case "client":
						issuer.mu.Lock()
						issuer.client = "other-client"
						issuer.mu.Unlock()
					case "link reassigned":
						_, err = pool.Exec(ctx, `UPDATE mcp_oauth_links SET user_id='10000000-0000-4000-8000-000000000002'`)
					case "token":
						issuer.mu.Lock()
						issuer.active = false
						issuer.mu.Unlock()
					case "scope":
						issuer.mu.Lock()
						issuer.scope = ""
						issuer.mu.Unlock()
					case "issuer outage":
						issuer.mu.Lock()
						issuer.status = 503
						issuer.mu.Unlock()
					case "unlink":
						_, err = pool.Exec(ctx, `DELETE FROM mcp_oauth_links`)
					case "account":
						_, err = pool.Exec(ctx, `UPDATE users SET is_active=false WHERE id='10000000-0000-4000-8000-000000000001'`)
					case "membership":
						_, err = pool.Exec(ctx, `UPDATE workspace_members SET is_active=false WHERE member_id='10000000-0000-4000-8000-000000000001'`)
					}
					require.NoError(t, err)
					return verified, nil
				})
				result := toolCall(t, h, "oauth-user", target.workspace, target.tool, target.input, 200)
				if change == "membership" && target.tool == "workspace" {
					require.False(t, result.IsError, result.Content)
					require.JSONEq(t, `{"results":[]}`, string(result.StructuredContent))
				} else {
					require.True(t, result.IsError, result.Content)
					require.Empty(t, result.StructuredContent)
				}
				issuer.mu.Lock()
				calls := issuer.calls
				issuer.mu.Unlock()
				require.Equal(t, 2, calls)
			})
		}
	}
}
