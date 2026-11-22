package main

import (
	"context"
	"net/http"

	"github.com/CaliLuke/loom/clue/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const browserActionDiagnosticsPath = "/api/diagnostics/browser-action/"

type browserActionDiagnostic struct {
	Action string `json:"action"`
}

func (event *browserActionDiagnostic) valid() bool {
	return event.Action == "webhook_secret_csv_download" || event.Action == "theme_config_download"
}

func newBrowserActionDiagnosticsHandler() http.Handler {
	return newBrowserDiagnosticHandler((*browserActionDiagnostic).valid, recordBrowserActionDiagnostic)
}

func recordBrowserActionDiagnostic(ctx context.Context, event browserActionDiagnostic) {
	const message = "Browser action failed"
	trace.SpanFromContext(ctx).AddEvent(message, trace.WithAttributes(attribute.String("browser.action.name", event.Action)))
	log.Error(ctx, nil, log.KV{K: "msg", V: message}, log.KV{K: "browser.action.name", V: event.Action})
}
