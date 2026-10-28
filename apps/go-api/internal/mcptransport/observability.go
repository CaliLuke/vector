package mcptransport

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/CaliLuke/go-plane/internal/platform"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/CaliLuke/loom/observability/transport"
)

// LogObserver projects framework lifecycle events into bounded application
// fields. It never forwards request IDs from JSON-RPC, arguments, result bodies,
// arbitrary event attributes, or provider error messages.
func LogObserver(logContext context.Context) transport.Observer {
	return transport.ObserverFunc(func(ctx context.Context, event transport.Event) {
		if event.Kind != transport.EventKindRequestFailure && event.Kind != transport.EventKindStreamFailure {
			return
		}
		method := "unknown"
		if slices.Contains(api.MethodNames[:], event.Method) || event.Method == "HTTP" {
			method = event.Method
		}
		fields := []log.Fielder{
			log.KV{K: log.MessageKey, V: "MCP request failed"},
			log.KV{K: "mcp_method", V: method},
			log.KV{K: "reason", V: string(event.Reason)},
			log.KV{K: "http_status", V: event.StatusCode},
		}
		if requestID, ok := platform.RequestID(ctx); ok {
			fields = append(fields, log.KV{K: log.RequestIDKey, V: requestID})
		}
		logger := log.WithContext(ctx, logContext)
		if event.StatusCode >= http.StatusInternalServerError {
			log.Error(logger, errors.New("MCP request failed"), fields...)
		} else {
			log.Warn(logger, fields...)
		}
	})
}

func observeHTTPRejection(observer transport.Observer, next http.Handler) http.Handler {
	if observer == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer, capture := transport.CaptureResponse(w)
		started := time.Now()
		next.ServeHTTP(writer, r)
		if capture.StatusCode() >= 400 {
			observer.ObserveEvent(r.Context(), transport.Event{
				Kind: transport.EventKindRequestFailure, Transport: transport.TransportMCP,
				Method: "HTTP", Reason: "request_rejected", StatusCode: capture.StatusCode(), Duration: time.Since(started),
			})
		}
	})
}
