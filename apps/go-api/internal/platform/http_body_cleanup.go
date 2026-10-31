package platform

import (
	"context"
	"errors"
	"fmt"

	"github.com/CaliLuke/loom/clue/log"
)

// LogResponseBodyCloseError reports cleanup failure without changing the result
// of a completed HTTP exchange. Retrying a remote write because its response
// failed to close could repeat an accepted effect. Callers retain the explicit
// Close call and report only its classification, never remote response contents.
func LogResponseBodyCloseError(ctx context.Context, err error) {
	logHTTPBodyCloseError(ctx, "Response body cleanup failed", err)
}

func logHTTPBodyCloseError(ctx context.Context, message string, err error) {
	if err == nil {
		return
	}
	log.Error(ctx, nil,
		log.KV{K: log.MessageKey, V: message},
		log.KV{K: "error_type", V: fmt.Sprintf("%T", err)},
		log.KV{K: "context_canceled", V: errors.Is(err, context.Canceled)},
		log.KV{K: "deadline_exceeded", V: errors.Is(err, context.DeadlineExceeded)},
	)
}
