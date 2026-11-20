package telemetry

import (
	"context"
	"errors"

	"github.com/CaliLuke/loom/clue/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// PasswordSignInOutcome distinguishes a normal credential rejection from session creation without logging user input.
func PasswordSignInOutcome(ctx context.Context, rejectionCode int, reason string, sessionCreated bool) {
	span := trace.SpanFromContext(ctx)
	if rejectionCode != 0 {
		span.SetAttributes(
			attribute.String("auth.outcome", "rejected"),
			attribute.Int("auth.rejection_code", rejectionCode),
			attribute.String("auth.rejection_reason", reason),
		)
		log.Print(ctx, log.KV{K: "msg", V: "Password sign-in rejected"},
			log.KV{K: "auth.outcome", V: "rejected"}, log.KV{K: "auth.rejection_code", V: rejectionCode},
			log.KV{K: "auth.rejection_reason", V: reason})
	} else if sessionCreated {
		span.SetAttributes(attribute.String("auth.outcome", "session_created"))
		log.Print(ctx, log.KV{K: "msg", V: "Password sign-in session created"},
			log.KV{K: "auth.outcome", V: "session_created"})
	}
}

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
