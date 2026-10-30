package main

import (
	"errors"
	"fmt"
	"io"
)

// closeResource retains cleanup failures alongside the startup or serving error.
// Defer it immediately after acquiring a resource to preserve cleanup ordering.
func closeResource(resultErr *error, name string, resource io.Closer) {
	if closeErr := resource.Close(); closeErr != nil {
		*resultErr = errors.Join(*resultErr, fmt.Errorf("close %s: %w", name, closeErr))
	}
}
