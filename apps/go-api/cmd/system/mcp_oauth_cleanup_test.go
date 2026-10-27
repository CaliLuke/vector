package main

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type mcpCleanupLog chan string

func (output mcpCleanupLog) Write(value []byte) (int, error) {
	select {
	case output <- string(value):
	default:
	}
	return len(value), nil
}

func TestMCPLinkCleanupReportsFailureAndJoinsBeforeShutdown(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgresql://localhost/unused")
	require.NoError(t, err)
	pool.Close()
	links, err := mcpoauth.NewLinks(pool, "https://issuer.example")
	require.NoError(t, err)
	output := make(mcpCleanupLog, 1)
	ctx := newLogContext(t.Context(), output, false)
	ctx = log.Context(ctx, log.WithDisableBuffering(func(context.Context) bool { return true }))
	stop := startMCPLinkCleanup(ctx, links)
	t.Cleanup(stop)
	select {
	case message := <-output:
		require.Contains(t, message, mcpoauth.ErrUnavailable.Error())
		require.NotContains(t, message, "pool closed")
	case <-time.After(10 * time.Second):
		t.Fatal("cleanup did not report the closed primary database")
	}
	// The returned stop function waits for the worker, so closing application
	// dependencies after it returns cannot race a later cleanup attempt.
	stop()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	startMCPLinkCleanup(canceled, links)()
	require.Empty(t, output)
}
