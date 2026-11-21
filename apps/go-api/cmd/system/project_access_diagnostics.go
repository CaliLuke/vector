package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
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

func (event projectAccessDiagnostic) valid() bool {
	switch event.Stage {
	case "workspace_membership", "project_roles", "project_detail", "project_probe":
	default:
		return false
	}
	switch event.Reason {
	case "http":
		return event.Status >= 400 && event.Status <= 599
	case "network", "scope_changed", "response_mismatch", "unexpected":
		return event.Status == 0 && event.RequestID == ""
	default:
		return false
	}
}

func newProjectAccessDiagnosticsHandler() http.Handler {
	var lock sync.Mutex
	var window time.Time
	var reports int
	allow := func() bool {
		lock.Lock()
		defer lock.Unlock()
		if now := time.Now(); !now.Before(window.Add(time.Second)) {
			window, reports = now, 0
		}
		if reports >= 20 {
			return false
		}
		reports++
		return true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Header.Get("Sec-Fetch-Site") != "same-origin" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512))
		decoder.DisallowUnknownFields()
		var event projectAccessDiagnostic
		if decoder.Decode(&event) != nil || decoder.Decode(new(any)) != io.EOF || !event.valid() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var requestID uuid.UUID
		if event.RequestID != "" {
			var err error
			requestID, err = uuid.Parse(event.RequestID)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		if !allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

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
			attrs = append(attrs, attribute.String("browser.access.request_id", requestID.String()))
			fields = append(fields, log.KV{K: "browser.access.request_id", V: requestID.String()})
		}
		trace.SpanFromContext(r.Context()).AddEvent(message, trace.WithAttributes(attrs...))
		if event.WillRetry {
			log.Print(r.Context(), fields...)
		} else {
			log.Error(r.Context(), nil, fields...)
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
