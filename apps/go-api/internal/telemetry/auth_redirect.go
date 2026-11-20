package telemetry

import (
	"context"
	"errors"

	"github.com/CaliLuke/loom/clue/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// AuthRedirectFailure emits a fixed stage and error class without exposing form input or dependency errors.
func AuthRedirectFailure(ctx context.Context, stage string, err error) {
	class := "internal"
	switch {
	case errors.Is(err, context.Canceled):
		class = "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		class = "deadline_exceeded"
	}

	trace.SpanFromContext(ctx).SetAttributes(
		attribute.String("auth.stage", stage),
		attribute.String("auth.error_class", class),
	)
	fields := []log.Fielder{
		log.KV{K: "msg", V: "Authentication redirect failed"},
		log.KV{K: "auth.stage", V: stage},
		log.KV{K: "auth.error_class", V: class},
	}
	if class == "context_canceled" {
		log.Print(ctx, fields...)
		return
	}
	log.Error(ctx, nil, fields...)
}
