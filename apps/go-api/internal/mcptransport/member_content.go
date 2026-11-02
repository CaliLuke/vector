package mcptransport

import (
	"bytes"
	"context"

	generated "github.com/CaliLuke/go-plane/mcp/gen/mcp_vector_mcp"
)

// memberContent preserves the pinned FastMCP unannotated-list response for all tools: JSON
// text for a nonempty list, no content for an empty list, and no structuredContent.
// https://github.com/CaliLuke/loom-mcp/issues/383 tracks upstream output shaping.
// Remove this interceptor once the minimum upstream version can produce this
// content-only list contract, including empty lists.
func memberContent(ctx context.Context, _ generated.ToolCallInterceptorInfo, payload *generated.ToolsCallPayload, next generated.ToolCallHandler) (*generated.ToolsCallResult, error) {
	result, err := next(ctx, payload)
	if err != nil || result == nil || payload == nil {
		return result, err
	}
	// FastMCP maps None to an empty content list, not a JSON null text block.
	if bytes.Equal(result.StructuredContent, []byte("null")) {
		result.StructuredContent = nil
		result.Content = []*generated.ContentItem{}
		return result, nil
	}
	if result.StructuredContent.Kind() != '[' {
		return result, err
	}
	// The generated encoder supplies valid, compact JSON from the typed result.
	empty := bytes.Equal(result.StructuredContent, []byte("[]"))
	result.StructuredContent = nil
	if empty {
		result.Content = []*generated.ContentItem{}
	}
	return result, nil
}
