// Package clioutput reports command failures without treating failed diagnostic
// delivery as a successful command or a normal usage error.
package clioutput

import (
	"fmt"
	"io"
)

// Report writes one diagnostic and returns the requested nonzero exit code.
// A failed diagnostic write is an I/O failure (exit code 1).
func Report(stderr io.Writer, exitCode int, format string, args ...any) int {
	if _, err := fmt.Fprintf(stderr, format, args...); err != nil {
		return 1
	}
	return exitCode
}
