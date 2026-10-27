//go:build integration

package mcpactions_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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
	issuer := &liveIssuer{active: true, scope: "vector:read", client: "agent-client", status: 200}
	issuer.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/openid-configuration" {
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": issuer.server.URL, "authorization_endpoint": issuer.server.URL + "/authorize",
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
			_, _ = w.Write([]byte(`{"active":false}`))
			return
		}
		w.WriteHeader(issuer.status)
		_ = json.NewEncoder(w).Encode(map[string]any{"active": issuer.active, "iss": issuer.server.URL, "sub": "subject-one", "client_id": issuer.client,
			"aud": "https://vector.example/mcp", "exp": time.Now().Add(time.Hour).Unix(), "scope": issuer.scope, "token_type": "Bearer", "grant_type": "authorization_code"})
	}))
	t.Cleanup(issuer.server.Close)
	provider, err := oauth.NewProvider(t.Context(), oauth.Config{Issuer: issuer.server.URL, Resource: "https://vector.example/mcp", IntrospectionURL: issuer.server.URL + "/introspect",
		IntrospectionClientID: "resource-client", IntrospectionClientSecret: "resource-secret", LinkClientID: "link-client", LinkClientSecret: "link-secret"}, issuer.server.Client())
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `CREATE TABLE mcp_oauth_links(issuer text NOT NULL,subject text NOT NULL,user_id uuid NOT NULL REFERENCES users(id),PRIMARY KEY(issuer,subject),UNIQUE(issuer,user_id));`)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `INSERT INTO mcp_oauth_links VALUES($1,'subject-one','10000000-0000-4000-8000-000000000001')`, issuer.server.URL)
	require.NoError(t, err)
	links, err := oauth.NewLinks(pool, issuer.server.URL)
	require.NoError(t, err)
	members, err := mcpauth.NewWorkspaceReader(pool)
	require.NoError(t, err)
	auth, err := mcpauth.NewOAuth(provider, links, members, throttle{})
	require.NoError(t, err)
	return f, pool, auth, issuer
}

func oauthHandler(t *testing.T, f fixture, authenticate mcptransport.Authenticate) http.Handler {
	t.Helper()
	cfg := mcptransport.DefaultConfig()
	cfg.Enabled, cfg.Mode, cfg.ResourceURL = true, "oauth", "https://vector.example/mcp"
	h, err := mcptransport.New(cfg, mcptransport.Options{Service: f.service, OAuthAuth: authenticate, OAuthIssuer: "https://issuer.example"}, http.NotFoundHandler())
	require.NoError(t, err)
	return h
}

func oauthCall(t *testing.T, h http.Handler, version, token, workspace, input string, status int) rpcResult {
	t.Helper()
	return oauthToolCall(t, h, version, token, workspace, "member", input, status)
}

func oauthToolCall(t *testing.T, h http.Handler, version, token, workspace, tool, input string, status int) rpcResult {
	t.Helper()
	return oauthRPC(t, h, version, token, workspace, "tools/call", map[string]any{"name": tool, "arguments": json.RawMessage(input)}, status)
}

func oauthRPC(t *testing.T, h http.Handler, version, token, workspace, method string, params map[string]any, status int) rpcResult {
	t.Helper()
	if version == "2026-07-28" {
		params["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": version, "io.modelcontextprotocol/clientInfo": map[string]string{"name": "test", "version": "1"}, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}
	}
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "https://vector.example/mcp", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	r.Header.Set("Mcp-Protocol-Version", version)
	if version == "2026-07-28" {
		r.Header.Set("Mcp-Method", method)
		if method == "tools/call" {
			r.Header.Set("Mcp-Name", params["name"].(string))
		}
	}
	r.Header.Set("Authorization", "Bearer "+token)
	if workspace != "" {
		r.Header.Set("X-Workspace-Slug", workspace)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, status, w.Code, w.Body.String())
	if status != http.StatusOK {
		return rpcResult{}
	}
	var response struct {
		Result rpcResult
		Error  json.RawMessage
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Empty(t, response.Error, w.Body.String())
	return response.Result
}

func TestOAuthGeneratedMembersUseOptionalWorkspaceAndCurrentIssuer(t *testing.T) {
	f, _, auth, issuer := oauthFixture(t)
	h := oauthHandler(t, f, auth.Authenticate)
	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			me := oauthCall(t, h, version, "oauth-user", "", `{"action":"me"}`, 200)
			require.False(t, me.IsError, me.Content)
			require.Contains(t, string(me.StructuredContent), "10000000-0000-4000-8000-000000000001")
			roles := oauthCall(t, h, version, "oauth-user", "", `{"action":"list_roles","workspace_slug":"workspace-one"}`, 200)
			require.False(t, roles.IsError, roles.Content)
			require.Contains(t, string(roles.StructuredContent), `"total_count":7`)
			var catalogue struct {
				Results []struct {
					ID string `json:"id"`
				} `json:"results"`
			}
			require.NoError(t, json.Unmarshal(roles.StructuredContent, &catalogue))
			require.Len(t, catalogue.Results, 7)
			role := oauthCall(t, h, version, "oauth-user", "workspace-one", fmt.Sprintf(`{"action":"retrieve_role","role_id":%q}`, catalogue.Results[0].ID), 200)
			require.False(t, role.IsError, role.Content)
			require.Contains(t, string(role.StructuredContent), catalogue.Results[0].ID)
			workspace := oauthCall(t, h, version, "oauth-user", "workspace-one", `{"action":"list_workspace","first_name":"zo"}`, 200)
			require.False(t, workspace.IsError, workspace.Content)
			require.Contains(t, string(workspace.StructuredContent), `"total_count":1`)
			project := oauthCall(t, h, version, "oauth-user", "", `{"action":"list_project","workspace_slug":"workspace-one","project_id":"30000000-0000-4000-8000-000000000001"}`, 200)
			require.False(t, project.IsError, project.Content)
			require.Len(t, project.Content, 1)
			require.Contains(t, project.Content[0].Text, "10000000-0000-4000-8000-000000000001")

			missing := oauthCall(t, h, version, "oauth-user", "", `{"action":"list_roles"}`, 200)
			require.True(t, missing.IsError)
			denied := oauthCall(t, h, version, "oauth-user", "", `{"action":"list_roles","workspace_slug":"workspace-two"}`, 200)
			require.True(t, denied.IsError)
			oauthCall(t, h, version, "oauth-user", "workspace-one", `{"action":"me","workspace_slug":"workspace-two"}`, 400)
			oauthCall(t, h, version, "test-one", "", `{"action":"me"}`, 401)
		})
	}
	issuer.mu.Lock()
	issuer.scope = "vector:write"
	issuer.mu.Unlock()
	oauthCall(t, h, "2025-11-25", "oauth-user", "", `{"action":"me"}`, 403)
	issuer.mu.Lock()
	issuer.status = 503
	issuer.mu.Unlock()
	oauthCall(t, h, "2025-11-25", "oauth-user", "", `{"action":"me"}`, 503)
}

func TestOAuthGeneratedMemberRevalidatesBetweenAuthenticationAndCall(t *testing.T) {
	for _, change := range []string{"token", "scope", "issuer outage", "unlink", "account", "membership", "client", "link reassigned"} {
		t.Run(change, func(t *testing.T) {
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
			result := oauthCall(t, h, "2025-11-25", "oauth-user", "workspace-one", `{"action":"list_roles"}`, 200)
			require.True(t, result.IsError, result.Content)
			require.Empty(t, result.StructuredContent)
			issuer.mu.Lock()
			calls := issuer.calls
			issuer.mu.Unlock()
			require.Equal(t, 2, calls)
		})
	}
}
