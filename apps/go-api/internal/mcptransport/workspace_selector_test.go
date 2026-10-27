package mcptransport_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceSelectorConflictsFailAtTheTransportBoundary(t *testing.T) {
	for _, path := range []string{"/mcp", "/mcp/api-key"} {
		for _, selector := range []string{`"other"`, `""`} {
			h := handler(t, config("both"), options())
			response := call(h, path, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":{"action":"me","workspace_slug":`+selector+`}}}`, http.Header{
				"X-Workspace-Slug": []string{"current"}, "Mcp-Protocol-Version": []string{"2025-11-25"},
			})
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Equal(t, "invalid_request\n", response.Body.String())
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		}
	}
}

func TestCurrentProtocolWorkspaceConflictReturnsInvalidRequest(t *testing.T) {
	h := handler(t, config("both"), options())
	response := call(h, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":{"action":"me","workspace_slug":"other"},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}`, http.Header{
		"X-Workspace-Slug": []string{"current"}, "Mcp-Protocol-Version": []string{"2026-07-28"},
		"Mcp-Method": []string{"tools/call"}, "Mcp-Name": []string{"member"},
	})
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "invalid_request\n", response.Body.String())
}

func TestWorkspaceInspectionLeavesProtocolAndActionValidationToGeneratedServer(t *testing.T) {
	opts := options()
	opts.Service = memberResponseService{body: identityResult}
	h, err := mcptransport.New(config("both"), opts, http.NotFoundHandler())
	require.NoError(t, err)
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"matching selector", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":{"action":"me","workspace_slug":"current"}}}`, 200},
		{"omitted selector", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":{"action":"me"}}}`, 200},
		{"null selector", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":{"action":"me","workspace_slug":null}}}`, 200},
		{"number selector", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":{"action":"me","workspace_slug":123}}}`, 200},
		{"ping", `{"jsonrpc":"2.0","id":1,"method":"ping"}`, 200},
		{"invalid JSON", `{`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := call(h, "/mcp/api-key", tc.body, http.Header{"X-Workspace-Slug": []string{"current"}, "Mcp-Protocol-Version": []string{"2025-11-25"}})
			assert.Equal(t, tc.status, response.Code, response.Body.String())
			assert.NotEqual(t, "invalid_request\n", response.Body.String())
			if tc.name == "null selector" || tc.name == "number selector" {
				assert.Contains(t, response.Body.String(), "[invalid_params]")
			}
		})
	}
}

type unreadableBody struct{}

func (unreadableBody) Read([]byte) (int, error) { return 0, errors.New("private reader error") }
func (unreadableBody) Close() error             { return nil }

func TestWorkspaceInspectionKeepsBodyReadsBoundedAndSafe(t *testing.T) {
	cfg := config("header")
	cfg.MaxRequestBytes = 16
	h := handler(t, cfg, options())
	r := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", strings.NewReader(strings.Repeat("a", 17)))
	r.ContentLength = -1
	r.Header.Set("X-Workspace-Slug", "current")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	r = httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", nil)
	r.Body = unreadableBody{}
	r.Header.Set("X-Workspace-Slug", "current")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.NotContains(t, w.Body.String(), "private reader")
}
