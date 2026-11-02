package mcptransport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type workspaceCleanupBody struct {
	*strings.Reader
	closes int
}

func (body *workspaceCleanupBody) Close() error {
	body.closes++
	return errors.New("private workspace request")
}

func TestWorkspaceInspectionReportsCleanupWithoutChangingBoundedRequest(t *testing.T) {
	for _, scenario := range []struct {
		name, selector string
		status         int
	}{
		{"matching", "current", http.StatusOK},
		{"conflicting", "other", http.StatusBadRequest},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			h := handler(t, config("both"), options())
			body := &workspaceCleanupBody{Reader: strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":{"action":"me","workspace_slug":"` + scenario.selector + `"},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}}}`)}
			r := httptest.NewRequest(http.MethodPost, "https://vector.example/mcp/api-key", nil)
			r.Body = body
			r.Header.Set("X-Workspace-Slug", "current")
			r.Header.Set("Mcp-Protocol-Version", "2026-07-28")
			r.Header.Set("Mcp-Method", "tools/call")
			r.Header.Set("Mcp-Name", "member")
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Accept", "application/json, text/event-stream")
			var output bytes.Buffer
			ctx := log.Context(t.Context(), log.WithOutputs(log.Output{Writer: &output, Format: log.FormatJSON}), log.WithDisableBuffering(func(context.Context) bool { return true }))
			response := httptest.NewRecorder()
			h.ServeHTTP(response, r.WithContext(ctx))
			require.Equal(t, scenario.status, response.Code, response.Body.String())
			require.Equal(t, 1, body.closes)
			var event map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &event))
			require.NotEmpty(t, event["error_type"])
			require.NotContains(t, output.String(), "private")
		})
	}
}

func TestWorkspaceSelectorConflictsFailAtTheTransportBoundary(t *testing.T) {
	for _, path := range []string{"/mcp", "/mcp/api-key"} {
		for _, selector := range []string{`"other"`, `""`} {
			h := handler(t, config("both"), options())
			response := call(h, path, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"member","arguments":{"action":"me","workspace_slug":`+selector+`}}}`, http.Header{
				"X-Workspace-Slug": []string{"current"}, "Mcp-Protocol-Version": []string{"2026-07-28"},
			})
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Equal(t, "invalid_request\n", response.Body.String())
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		}
	}
}

func TestCurrentProtocolWorkspaceConflictReturnsInvalidRequest(t *testing.T) {
	h := handler(t, config("both"), options())
	response := call(h, "/mcp", `{"id":1,"jsonrpc":"2.0","method":"tools/call","params":{"_meta":{"io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/protocolVersion":"2026-07-28"},"arguments":{"action":"me","workspace_slug":"other"},"name":"member"}}`, http.Header{
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
		{"matching selector", `{"id":1,"jsonrpc":"2.0","method":"tools/call","params":{"_meta":{"io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/protocolVersion":"2026-07-28"},"arguments":{"action":"me","workspace_slug":"current"},"name":"member"}}`, 200},
		{"omitted selector", `{"id":1,"jsonrpc":"2.0","method":"tools/call","params":{"_meta":{"io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/protocolVersion":"2026-07-28"},"arguments":{"action":"me"},"name":"member"}}`, 200},
		{"null selector", `{"id":1,"jsonrpc":"2.0","method":"tools/call","params":{"_meta":{"io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/protocolVersion":"2026-07-28"},"arguments":{"action":"me","workspace_slug":null},"name":"member"}}`, 200},
		{"number selector", `{"id":1,"jsonrpc":"2.0","method":"tools/call","params":{"_meta":{"io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/protocolVersion":"2026-07-28"},"arguments":{"action":"me","workspace_slug":123},"name":"member"}}`, 200},
		{"discovery", `{"id":1,"jsonrpc":"2.0","method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, 200},
		{"invalid JSON", `{`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := call(h, "/mcp/api-key", tc.body, http.Header{"X-Workspace-Slug": []string{"current"}, "Mcp-Protocol-Version": []string{"2026-07-28"}})
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
