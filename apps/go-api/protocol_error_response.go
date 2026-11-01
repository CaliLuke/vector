package systemapi

import (
	"encoding/json"
	"net/http"
)

// protocolErrorResponse preserves generated JSON error objects at the shared
// authorization boundary. A broken envelope is an internal failure, rather
// than an empty or partially decoded response with the original status.
func protocolErrorResponse(name string, value any) (int, map[string]any) {
	encoded, err := json.Marshal(value)
	if err == nil {
		var body map[string]any
		if decodeErr := json.Unmarshal(encoded, &body); decodeErr == nil && body != nil {
			return protocolErrorStatus(name), body
		}
	}
	return http.StatusInternalServerError, map[string]any{"error": "Something went wrong please try again later"}
}
