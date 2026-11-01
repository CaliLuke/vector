package systemapi

import (
	"net/http"
	"strconv"
	"strings"
)

// protocolErrorStatus converts generated error identities into HTTP failures.
// Zero means successful authorization at the protocol boundary, so malformed
// error identities must retain a failure status.
func protocolErrorStatus(name string) int {
	text, ok := strings.CutPrefix(name, "status_")
	if !ok || len(text) != 3 {
		return http.StatusInternalServerError
	}
	status, err := strconv.Atoi(text)
	if err != nil || status < 400 || status > 599 {
		return http.StatusInternalServerError
	}
	return status
}
