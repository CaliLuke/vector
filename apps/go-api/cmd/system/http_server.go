package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/CaliLuke/loom/clue/log"
)

func listenAndServeHTTP(
	ctx context.Context,
	address string,
	handler http.Handler,
	shutdownTimeout time.Duration,
) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}
	return serveHTTP(ctx, listener, handler, shutdownTimeout)
}

func serveHTTP(
	ctx context.Context,
	listener net.Listener,
	handler http.Handler,
	shutdownTimeout time.Duration,
) error {
	server := &http.Server{
		Handler:             handler,
		ReadHeaderTimeout:   5 * time.Second,
		MaxHeaderValueCount: http.DefaultMaxHeaderValueCount,
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Print(ctx,
			log.KV{K: "msg", V: "HTTP server listening"},
			log.KV{K: "address", V: listener.Addr().String()},
		)
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		log.Print(ctx, log.KV{K: "msg", V: "HTTP server shutting down"})
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP during shutdown: %w", err)
		}
		log.Print(ctx, log.KV{K: "msg", V: "HTTP server stopped"})
		return nil
	}
}

func errorHandler(ctx context.Context, _ http.ResponseWriter, err error) {
	log.Error(ctx, err, log.KV{K: "msg", V: "encode HTTP response"})
}

func logHTTPRouteMount(ctx context.Context, service, method, verb, pattern string) {
	log.Print(ctx,
		log.KV{K: "msg", V: "HTTP route mounted"},
		log.KV{K: "service", V: service},
		log.KV{K: "method", V: method},
		log.KV{K: "http_method", V: verb},
		log.KV{K: "http_path", V: pattern},
	)
}
