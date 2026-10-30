package platform

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/CaliLuke/loom/clue/log"
)

// CloseRequestBody releases a request input stream at its existing cleanup
// boundary. A cleanup failure cannot change a previously accepted write into a
// failed operation. Report its classification without logging body or error text.
func CloseRequestBody(ctx context.Context, body io.Closer) {
	if body == nil {
		return
	}
	if err := body.Close(); err != nil {
		log.Error(ctx, nil,
			log.KV{K: log.MessageKey, V: "Request body cleanup failed"},
			log.KV{K: "error_type", V: fmt.Sprintf("%T", err)},
			log.KV{K: "context_canceled", V: errors.Is(err, context.Canceled)},
			log.KV{K: "deadline_exceeded", V: errors.Is(err, context.DeadlineExceeded)},
		)
	}
}
