package mcpactions_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	modules "github.com/CaliLuke/go-plane/internal/modulereads"
	labels "github.com/CaliLuke/go-plane/internal/projectlabelreads"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/stretchr/testify/require"
)

type moduleReadStub struct {
	value any
	err   error
}

func (s *moduleReadStub) ReadAPIV1(context.Context, string, modules.APIV1Request) (any, error) {
	return s.value, s.err
}

type labelReadStub struct {
	value any
	err   error
}

func (s *labelReadStub) ReadAPIV1(context.Context, string, labels.APIV1Request) (any, error) {
	return s.value, s.err
}

func resourceCall(tool, input string) map[string]any {
	return map[string]any{"name": tool, "arguments": json.RawMessage(input)}
}

func TestResourceReadFailuresAreSanitized(t *testing.T) {
	for _, tc := range []struct {
		tool string
		err  error
		code string
	}{
		{"module", modules.ErrForbidden, "forbidden"}, {"module", modules.ErrNotFound, "not_found"},
		{"label", labels.ErrForbidden, "forbidden"}, {"label", labels.ErrNotFound, "not_found"},
		{"module", &djangoapi.HTTPError{Status: 400}, "invalid_params"}, {"label", &djangoapi.HTTPError{Status: 400}, "invalid_params"},
		{"module", errors.New("secret database details"), "internal_error"}, {"label", errors.New("secret database details"), "internal_error"},
	} {
		t.Run(tc.tool+"/"+tc.code, func(t *testing.T) {
			f := newFixture(t)
			f.moduleReads.err = tc.err
			f.labelReads.err = tc.err
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", resourceCall(tc.tool, `{"action":"list","project_id":"30000000-0000-4000-8000-000000000001"}`))
			require.True(t, result.IsError)
			require.Contains(t, result.Content[0].Text, "["+tc.code+"]")
			require.NotContains(t, result.Content[0].Text, "secret")
			require.Empty(t, result.StructuredContent)
		})
	}
}

func TestResourceReadRejectsMalformedResults(t *testing.T) {
	for _, tc := range []struct {
		tool, action string
		value        any
	}{
		{"label", "retrieve", map[string]any{}},
		{"label", "retrieve", map[string]any{"name": "label", "description": 12}},
		{"label", "retrieve", map[string]any{"name": "label", "description": make(chan int)}},
		{"module", "retrieve", map[string]any{}},
		{"module", "retrieve", map[string]any{"name": "module", "status": "invalid"}},
		{"module", "retrieve", map[string]any{"name": "module", "description": 12}},
		{"module", "retrieve", map[string]any{"name": "module", "description": make(chan int)}},
		{"module", "retrieve", map[string]any{"name": "module", "total_issues": json.Number("1.5")}},
		{"module", "list", map[string]any{"results": []any{map[string]any{"name": "module"}}}},
	} {
		t.Run(tc.tool+"/"+tc.action, func(t *testing.T) {
			f := newFixture(t)
			f.moduleReads.value = tc.value
			f.labelReads.value = tc.value
			extra := ""
			if tc.action == "retrieve" {
				extra = `,"` + tc.tool + `_id":"50000000-0000-4000-8000-000000000001"`
			}
			result := rpc(t, f.handler(t, nil), "test-one", "tools/call", resourceCall(tc.tool, `{"action":"`+tc.action+`","project_id":"30000000-0000-4000-8000-000000000001"`+extra+`}`))
			require.True(t, result.IsError)
			require.Contains(t, result.Content[0].Text, "[internal_error]")
		})
	}
}

func TestDirectResourceCallsRejectInvalidArguments(t *testing.T) {
	f := newFixture(t)
	negative, large := -1, 1001
	for _, input := range []*api.ModuleRequest{nil, {}, moduleListRequest(api.ModuleList{Action: "wrong"}), moduleListRequest(api.ModuleList{Action: "list", ProjectID: "invalid"}), moduleListRequest(api.ModuleList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001", PerPage: &negative}), moduleListRequest(api.ModuleList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001", PerPage: &large}), moduleDetailRequest(api.ModuleRetrieve{Action: "wrong"}), moduleDetailRequest(api.ModuleRetrieve{Action: "retrieve", ProjectID: "30000000-0000-4000-8000-000000000001", ModuleID: "invalid"})} {
		_, err := f.service.Module(t.Context(), input)
		require.Error(t, err)
	}
	for _, input := range []*api.LabelRequest{nil, {}, labelListRequest(api.LabelList{Action: "wrong"}), labelListRequest(api.LabelList{Action: "list", ProjectID: "invalid"}), labelListRequest(api.LabelList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001", PerPage: &negative}), labelListRequest(api.LabelList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001", PerPage: &large}), labelDetailRequest(api.LabelRetrieve{Action: "wrong"}), labelDetailRequest(api.LabelRetrieve{Action: "retrieve", ProjectID: "30000000-0000-4000-8000-000000000001", LabelID: "invalid"})} {
		_, err := f.service.Label(t.Context(), input)
		require.Error(t, err)
	}
	_, err := f.service.Module(t.Context(), moduleListRequest(api.ModuleList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001"}))
	require.Error(t, err)
	_, err = f.service.Label(t.Context(), labelListRequest(api.LabelList{Action: "list", ProjectID: "30000000-0000-4000-8000-000000000001"}))
	require.Error(t, err)
}
func moduleListRequest(input api.ModuleList) *api.ModuleRequest {
	r := api.NewModuleRequestModuleList(&input)
	return &r
}
func moduleDetailRequest(input api.ModuleRetrieve) *api.ModuleRequest {
	r := api.NewModuleRequestModuleRetrieve(&input)
	return &r
}
func labelListRequest(input api.LabelList) *api.LabelRequest {
	r := api.NewLabelRequestLabelList(&input)
	return &r
}
func labelDetailRequest(input api.LabelRetrieve) *api.LabelRequest {
	r := api.NewLabelRequestLabelRetrieve(&input)
	return &r
}
