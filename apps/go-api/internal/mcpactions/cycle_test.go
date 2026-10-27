package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/stretchr/testify/require"

	cycles "github.com/CaliLuke/go-plane/internal/cyclereads"
)

type cycleReadStub struct {
	value any
	err   error
}

func (s *cycleReadStub) ReadAPIV1(context.Context, string, cycles.APIV1Request) (any, error) {
	return s.value, s.err
}

func TestCycleFailuresAreTypedAndSanitized(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
		code    string
	}{
		{"denied", cycles.ErrForbidden, "forbidden"}, {"missing", cycles.ErrNotFound, "not_found"},
		{"invalid cursor", &djangoapi.HTTPError{Status: 400}, "invalid_params"},
		{"backend HTTP failure", &djangoapi.HTTPError{Status: 500, Body: map[string]any{"password": "secret"}}, "internal_error"},
		{"database failure", errors.New("secret database details"), "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.cycleReads.err = tc.failure
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", cycleCall(`{"action":"list","project_id":"30000000-0000-4000-8000-000000000001"}`))
			require.True(t, result.IsError)
			require.Contains(t, result.Content[0].Text, "["+tc.code+"]")
			require.NotContains(t, result.Content[0].Text, "secret")
			require.Empty(t, result.StructuredContent)
		})
	}
}

func TestCycleMalformedApplicationResultsCannotEscapeTypedContract(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		value        any
	}{
		{"non-document", "list", nil},
		{"non-page", "list", map[string]any{}},
		{"invalid pagination", "list", map[string]any{"results": []any{}, "count": "wrong"}},
		{"unencodable pagination", "list", map[string]any{"results": []any{}, "count": make(chan int)}},
		{"non-record", "list", map[string]any{"results": []any{"wrong"}}},
		{"missing name", "list", map[string]any{"results": []any{map[string]any{}}}},
		{"missing lite scope", "list", map[string]any{"results": []any{map[string]any{"name": "cycle"}}}},
		{"missing detail name", "retrieve", map[string]any{}},
		{"fractional metric", "retrieve", map[string]any{"name": "cycle", "total_estimates": json.Number("1.5")}},
		{"overflow metric", "retrieve", map[string]any{"name": "cycle", "total_issues": json.Number("9223372036854775808")}},
		{"unencodable metric", "retrieve", map[string]any{"name": "cycle", "total_issues": make(chan int)}},
		{"unencodable content", "retrieve", map[string]any{"name": "cycle", "view_props": make(chan int)}},
		{"wrong field type", "retrieve", map[string]any{"name": "cycle", "description": 12}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.cycleReads.value = tc.value
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", cycleCall(`{"action":"`+tc.action+`","project_id":"30000000-0000-4000-8000-000000000001"`+cycleDetailArgument(tc.action)+`}`))
			require.True(t, result.IsError)
			require.Contains(t, result.Content[0].Text, "[internal_error]")
			require.Empty(t, result.StructuredContent)
		})
	}
}

func cycleDetailArgument(action string) string {
	if action == "retrieve" {
		return `,"cycle_id":"50000000-0000-4000-8000-000000000001"`
	}
	return ""
}

func TestDirectCycleCallsValidateArgumentsBeforeReading(t *testing.T) {
	f := newFixture(t)
	negative, oversized := -1, 1001
	for _, input := range []*api.CycleRequest{
		nil, {},
		cycleRequest(api.CycleList{Action: "wrong"}),
		cycleRequest(api.CycleList{Action: "list", ProjectID: "invalid"}),
		cycleRequest(api.CycleList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001", PerPage: &negative}),
		cycleRequest(api.CycleList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001", PerPage: &oversized}),
		cycleRetrieveRequest(api.CycleRetrieve{Action: "wrong"}),
		cycleRetrieveRequest(api.CycleRetrieve{Action: "retrieve", ProjectID: "30000000-0000-4000-8000-000000000001", CycleID: "invalid"}),
	} {
		_, err := f.service.Cycle(t.Context(), input)
		require.ErrorContains(t, err, "invalid")
	}
	_, err := f.service.Cycle(t.Context(), cycleRequest(api.CycleList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001"}))
	require.Error(t, err)
}
func cycleRequest(input api.CycleList) *api.CycleRequest {
	result := api.NewCycleRequestCycleList(&input)
	return &result
}
func cycleRetrieveRequest(input api.CycleRetrieve) *api.CycleRequest {
	result := api.NewCycleRequestCycleRetrieve(&input)
	return &result
}

func cycleCall(input string) map[string]any {
	return map[string]any{"name": "cycle", "arguments": json.RawMessage(input)}
}
