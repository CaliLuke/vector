// Package commandline owns exit-status reporting for the native commands.
package commandline

import (
	"fmt"
	"io"
)

// Diagnostic reports a failure or usage message. An unavailable output stream
// is itself a command failure, including when the requested status was success.
func Diagnostic(output io.Writer, status int, message string) int {
	if _, err := fmt.Fprintln(output, message); err != nil {
		return 1
	}
	return status
}
