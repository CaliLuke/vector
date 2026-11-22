package main

import (
	"context"
	"net/http"
	"uuid"

	"github.com/CaliLuke/loom/clue/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const projectAccessDiagnosticsPath = "/api/diagnostics/project-access/"

type projectAccessDiagnostic struct {
	Stage     string `json:"stage"`
	Reason    string `json:"reason"`
	Status    int    `json:"http_status"`
	RequestID string `json:"request_id"`
	WillRetry bool   `json:"will_retry"`
}

func (event *projectAccessDiagnostic) valid() bool {
	switch event.Stage {
	case "identity", "workspace_membership", "project_roles", "project_catalogue", "project_detail", "project_probe", "project_preferences":
	default:
		return false
	}
	switch event.Reason {
	case "http":
		if event.Status < 400 || event.Status > 599 {
			return false
		}
		if event.RequestID != "" {
			id, err := uuid.Parse(event.RequestID)
			if err != nil {
				return false
			}
			event.RequestID = id.String()
		}
		return true
	case "network", "scope_changed", "response_mismatch", "unexpected":
		return event.Status == 0 && event.RequestID == ""
	default:
		return false
	}
}

func newProjectAccessDiagnosticsHandler() http.Handler {
	return newBrowserDiagnosticHandler((*projectAccessDiagnostic).valid, recordProjectAccessDiagnostic)
}

func recordProjectAccessDiagnostic(ctx context.Context, event projectAccessDiagnostic) {
	const message = "Browser project access read failed"
	attrs := []attribute.KeyValue{
		attribute.String("browser.access.stage", event.Stage),
		attribute.String("browser.access.reason", event.Reason),
		attribute.Bool("browser.access.will_retry", event.WillRetry),
	}
	fields := []log.Fielder{
		log.KV{K: "msg", V: message},
		log.KV{K: "browser.access.stage", V: event.Stage},
		log.KV{K: "browser.access.reason", V: event.Reason},
		log.KV{K: "browser.access.will_retry", V: event.WillRetry},
	}
	if event.Status != 0 {
		attrs = append(attrs, attribute.Int("browser.access.http_status", event.Status))
		fields = append(fields, log.KV{K: "browser.access.http_status", V: event.Status})
	}
	if event.RequestID != "" {
		attrs = append(attrs, attribute.String("browser.access.request_id", event.RequestID))
		fields = append(fields, log.KV{K: "browser.access.request_id", V: event.RequestID})
	}
	trace.SpanFromContext(ctx).AddEvent(message, trace.WithAttributes(attrs...))
	if event.WillRetry {
		log.Print(ctx, fields...)
	} else {
		log.Error(ctx, nil, fields...)
	}
}
