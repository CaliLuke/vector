package main

import (
	"context"
	"net/http"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/CaliLuke/loom/observability/transport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func logHTTPServerErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		capturedResponse, capture := transport.CaptureResponse(response)
		next.ServeHTTP(capturedResponse, request)

		status := capture.StatusCode()
		if status < http.StatusInternalServerError {
			return
		}

		errorClass := httpServerErrorClass(request.Context())
		span := trace.SpanFromContext(request.Context())
		if span.IsRecording() {
			span.SetAttributes(attribute.String("app.http.error_class", errorClass))
		}
		fields := []log.Fielder{
			log.KV{K: "msg", V: "HTTP server error response"},
			log.KV{K: "http.method", V: request.Method},
			log.KV{K: "http.path", V: request.URL.Path},
			log.KV{K: "http.status", V: status},
			log.KV{K: "error.class", V: errorClass},
		}
		log.Error(request.Context(), nil, fields...)
	})
}

func httpServerErrorClass(ctx context.Context) string {
	switch ctx.Err() {
	case context.Canceled:
		return "context_canceled"
	case context.DeadlineExceeded:
		return "deadline_exceeded"
	default:
		return "http_server_error"
	}
}
