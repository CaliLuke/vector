package mcptransport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

type discoveryPolicy struct {
	visible ActionVisibility
	err     error
}

func (p *discoveryPolicy) DiscoverActions(context.Context) (ActionVisibility, error) {
	return p.visible, p.err
}

func TestDiscoveryFiltersCopiesOfDeclarations(t *testing.T) {
	source := json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"action":{"enum":["read","write","delete"]},"secret_write_field":{"type":"string"}},"oneOf":[{"type":"object","required":["action"],"properties":{"action":{"type":"string","enum":["read"]},"id":{"type":"string"}}},{"type":"object","required":["action","secret_write_field"],"properties":{"action":{"type":"string","enum":["write","delete"]},"id":{"type":"string"},"secret_write_field":{"type":"string"}}}]}`)
	original := &mcpsdk.ListToolsResult{Tools: []*mcpsdk.Tool{{Name: "resource", InputSchema: source}, {Name: "reference", InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)}}}
	policy := &discoveryPolicy{}
	invoke := discoveryMiddleware(policy)(func(context.Context, string, mcpsdk.Request) (mcpsdk.Result, error) { return original, nil })
	for _, actions := range []map[string]bool{{"read": true}, {"read": true, "write": true}, {"delete": true}, {"read": true}, {}} {
		policy.visible = ActionVisibility{"resource": actions}
		value, err := invoke(t.Context(), "tools/list", nil)
		require.NoError(t, err)
		result := value.(*mcpsdk.ListToolsResult)
		require.Equal(t, "private", result.CacheScope)
		require.Zero(t, result.TTLMs)
		if len(actions) == 0 {
			require.Empty(t, result.Tools)
			continue
		}
		require.Len(t, result.Tools, 1)
		var schema struct {
			Properties map[string]json.RawMessage
			OneOf      []json.RawMessage
		}
		raw, err := json.Marshal(result.Tools[0].InputSchema)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &schema))
		for _, action := range []string{"read", "write", "delete"} {
			if actions[action] {
				require.Contains(t, string(schema.Properties["action"]), `"`+action+`"`)
			} else {
				require.NotContains(t, string(schema.Properties["action"]), `"`+action+`"`)
			}
		}
		if !actions["write"] && !actions["delete"] {
			require.NotContains(t, schema.Properties, "secret_write_field")
		}
		require.JSONEq(t, `{"type":"string"}`, string(schema.Properties["id"]))
		require.Equal(t, source, original.Tools[0].InputSchema)
	}
	policy.visible = ActionVisibility{"reference": {"": true}}
	value, err := invoke(t.Context(), "tools/list", nil)
	require.NoError(t, err)
	require.Len(t, value.(*mcpsdk.ListToolsResult).Tools, 1)
	require.Equal(t, "reference", value.(*mcpsdk.ListToolsResult).Tools[0].Name)
	original.Tools[0].InputSchema = json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["read","write"]}}}`)
	policy.visible = ActionVisibility{"resource": {"read": true}}
	value, err = invoke(t.Context(), "tools/list", nil)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"object","properties":{"action":{"type":"string","enum":["read"]}}}`, string(value.(*mcpsdk.ListToolsResult).Tools[0].InputSchema.(json.RawMessage)))
}

func TestDiscoveryFailsClosedOnInvalidDeclarations(t *testing.T) {
	for _, input := range []any{
		func() {}, nil, 1,
		json.RawMessage(`{"oneOf":{}}`),
		json.RawMessage(`{"oneOf":[{}]}`),
		json.RawMessage(`{"oneOf":[{"properties":[]} ]}`),
		json.RawMessage(`{"properties":{}}`),
		json.RawMessage(`{"properties":{"action":[]}}`),
		json.RawMessage(`{"properties":{"action":{}}}`),
		json.RawMessage(`{"properties":{"action":{"enum":[1]}}}`),
		json.RawMessage(`{"properties":{"action":{"enum":[]}}}`),
	} {
		policy := &discoveryPolicy{visible: ActionVisibility{"resource": {"read": true}}}
		invoke := discoveryMiddleware(policy)(func(context.Context, string, mcpsdk.Request) (mcpsdk.Result, error) {
			return &mcpsdk.ListToolsResult{Tools: []*mcpsdk.Tool{{Name: "resource", InputSchema: input}}}, nil
		})
		result, err := invoke(t.Context(), "tools/list", nil)
		if raw, ok := input.(json.RawMessage); ok && string(raw) == `{"properties":{}}` {
			require.NoError(t, err)
			require.Empty(t, result.(*mcpsdk.ListToolsResult).Tools)
		} else {
			require.Error(t, err)
			require.Nil(t, result)
			require.EqualError(t, err, "Tool discovery is unavailable.")
		}
	}
}

func TestDiscoveryAuthorizationFailureStopsCatalogueRead(t *testing.T) {
	policy := &discoveryPolicy{err: errors.New("private authorization failure")}
	called := false
	invoke := discoveryMiddleware(policy)(func(context.Context, string, mcpsdk.Request) (mcpsdk.Result, error) { called = true; return nil, nil })
	result, err := invoke(t.Context(), "tools/list", nil)
	require.Error(t, err)
	require.Nil(t, result)
	require.False(t, called)
	require.NotContains(t, err.Error(), "private")
}

func TestDiscoveryHandlesCatalogueFailureWithoutPartialResults(t *testing.T) {
	for _, value := range []mcpsdk.Result{nil, (*mcpsdk.ListToolsResult)(nil), &mcpsdk.CallToolResult{}, &mcpsdk.ListToolsResult{Tools: []*mcpsdk.Tool{nil}}} {
		invoke := discoveryMiddleware(&discoveryPolicy{})(func(context.Context, string, mcpsdk.Request) (mcpsdk.Result, error) { return value, nil })
		result, err := invoke(t.Context(), "tools/list", nil)
		require.Error(t, err)
		require.Nil(t, result)
	}
	expected := errors.New("catalogue error")
	invoke := discoveryMiddleware(&discoveryPolicy{})(func(context.Context, string, mcpsdk.Request) (mcpsdk.Result, error) { return nil, expected })
	_, err := invoke(t.Context(), "tools/list", nil)
	require.ErrorIs(t, err, expected)
}
