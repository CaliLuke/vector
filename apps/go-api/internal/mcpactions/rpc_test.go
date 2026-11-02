package mcpactions_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func rpc(t *testing.T, handler http.Handler, token, method string, params map[string]any) rpcResult {
	t.Helper()
	return mcpRPC(t, handler, token, "workspace-one", method, params, http.StatusOK)
}

func toolCall(t *testing.T, h http.Handler, token, workspace, tool, input string, status int) rpcResult {
	t.Helper()
	return mcpRPC(t, h, token, workspace, "tools/call", map[string]any{"name": tool, "arguments": json.RawMessage(input)}, status)
}

func mcpRPC(t *testing.T, h http.Handler, token, workspace, method string, params map[string]any, status int) rpcResult {
	t.Helper()
	r := mcpRequest(t, token, workspace, method, params)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, status, w.Code, w.Body.String())
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
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

func mcpRequest(t *testing.T, token, workspace, method string, params map[string]any) *http.Request {
	t.Helper()

	params["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": "2026-07-28", "io.modelcontextprotocol/clientInfo": map[string]string{"name": "test", "version": "1"}, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}

	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "https://vector.example/mcp", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	r.Header.Set("Mcp-Protocol-Version", "2026-07-28")

	r.Header.Set("Mcp-Method", method)
	if method == "tools/call" {
		r.Header.Set("Mcp-Name", params["name"].(string))
	}

	r.Header.Set("Authorization", "Bearer "+token)
	if workspace != "" {
		r.Header.Set("X-Workspace-Slug", workspace)
	}
	return r
}
