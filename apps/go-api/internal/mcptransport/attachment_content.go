package mcptransport

import (
	"context"
	"encoding/json"
	"errors"

	generated "github.com/CaliLuke/go-plane/mcp/gen/mcp_vector_mcp"
)

// Preserve the native MCP image channel. The generated typed result carries the
// application result; this transport projection uses its supported interceptor.
func attachmentContent(ctx context.Context, _ generated.ToolCallInterceptorInfo, payload *generated.ToolsCallPayload, next generated.ToolCallHandler) (*generated.ToolsCallResult, error) {
	result, err := next(ctx, payload)
	if err != nil || result == nil || payload == nil || payload.Name != "workitem_attachment" {
		return result, err
	}
	var action struct {
		Action string `json:"action"`
	}
	if json.Unmarshal(payload.Arguments, &action) != nil {
		return nil, errors.New("invalid attachment content action")
	}
	if action.Action != "read" || len(result.StructuredContent) == 0 || result.IsError != nil && *result.IsError {
		return result, nil
	}
	var content struct {
		Kind     string `json:"kind"`
		MIMEType string `json:"mime_type"`
		Data     string `json:"data"`
		Text     string `json:"text"`
	}
	if json.Unmarshal(result.StructuredContent, &content) != nil {
		return nil, errors.New("invalid attachment content result")
	}
	switch content.Kind {
	case "image":
		result.Content = []*generated.ContentItem{{Type: "image", Data: &content.Data, MimeType: &content.MIMEType}}
	case "text":
		result.Content = []*generated.ContentItem{{Type: "text", Text: &content.Text}}
	default:
		return result, nil // Generated error result remains unchanged.
	}
	result.StructuredContent = nil
	return result, nil
}
