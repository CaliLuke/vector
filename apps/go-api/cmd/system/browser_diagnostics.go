package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

func newBrowserDiagnosticHandler[T any](validate func(*T) bool, record func(context.Context, T)) http.Handler {
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
		var event T
		if decoder.Decode(&event) != nil || decoder.Decode(new(any)) != io.EOF || !validate(&event) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		record(r.Context(), event)
		w.WriteHeader(http.StatusNoContent)
	})
}
