package platform

import (
	"context"
	"io"
)

// CloseRequestBody releases a request input stream at its existing cleanup
// boundary. A cleanup failure cannot change a previously accepted write into a
// failed operation. Report its classification without logging body or error text.
func CloseRequestBody(ctx context.Context, body io.Closer) {
	if body == nil {
		return
	}
	logHTTPBodyCloseError(ctx, "Request body cleanup failed", body.Close())
}
