package mcptransport

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// workspaceSelectorStatus rejects conflicting authentication scope before tool
// execution. It inspects only the shared workspace selector and restores the
// bounded body verbatim; the generated server owns protocol and action validation.
func workspaceSelectorStatus(request *http.Request) int {
	header := request.Header.Get("X-Workspace-Slug")
	if request.Method != http.MethodPost || header == "" {
		return 0
	}
	body, err := io.ReadAll(request.Body)
	_ = request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			return http.StatusRequestEntityTooLarge
		}
		return http.StatusBadRequest
	}
	var call struct {
		Method string
		Params struct {
			Arguments map[string]json.RawMessage
		}
	}
	if json.Unmarshal(body, &call) != nil || call.Method != "tools/call" {
		return 0
	}
	selector, present := call.Params.Arguments["workspace_slug"]
	if !present {
		return 0
	}
	var slug *string
	if json.Unmarshal(selector, &slug) == nil && slug != nil && *slug != header {
		return http.StatusBadRequest
	}
	return 0
}
