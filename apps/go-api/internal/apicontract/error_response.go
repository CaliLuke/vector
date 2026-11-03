package apicontract

import (
	"encoding/json"
	"fmt"
)

// ProjectError converts a domain error body to a generated transport error.
// The caller supplies the status classification. Failed conversion returns an
// operational error instead of exposing a partial response as that status.
func ProjectError(body map[string]any, response error) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode error response: %w", err)
	}
	if err := json.Unmarshal(data, response); err != nil {
		return fmt.Errorf("project error response: %w", err)
	}
	return response
}
