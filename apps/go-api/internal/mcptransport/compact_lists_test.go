package mcptransport

import (
	"context"
	"testing"

	generated "github.com/CaliLuke/go-plane/mcp/gen/mcp_vector_mcp"
	"github.com/stretchr/testify/require"
)

func TestCompactListContentPreservesWritableReferences(t *testing.T) {
	payload := &generated.ToolsCallPayload{Name: "workitem", Arguments: []byte(`{"action":"list","format":"compact"}`)}
	result, err := compactListContent(t.Context(), nil, payload, func(context.Context, *generated.ToolsCallPayload) (*generated.ToolsCallResult, error) {
		return &generated.ToolsCallResult{StructuredContent: []byte(`{
			"results": [
				{"id":"draft-id","name":"Draft title","is_draft":true,"sequence_id":7,"project":{"identifier":"VEC"},"priority":"high"},
				{"id":"fallback-id","name":"Unassigned project","sequence_id":8}
			]
		}`)}, nil
	})
	require.NoError(t, err)
	require.Empty(t, result.StructuredContent)
	require.Len(t, result.Content, 1)
	require.Equal(t, "2 work items\ndraft-id  Draft title  high  draft\nfallback-id  Unassigned project", *result.Content[0].Text)
}

func TestCompactListContentRejectsInvalidPage(t *testing.T) {
	payload := &generated.ToolsCallPayload{Name: "project", Arguments: []byte(`{"action":"list","format":"compact"}`)}
	for _, body := range []string{`{"results":{}}`, `{"results":null}`} {
		result, err := compactListContent(t.Context(), nil, payload, func(context.Context, *generated.ToolsCallPayload) (*generated.ToolsCallResult, error) {
			return &generated.ToolsCallResult{StructuredContent: []byte(body)}, nil
		})
		require.ErrorContains(t, err, "invalid compact list result")
		require.Nil(t, result)
	}
}
