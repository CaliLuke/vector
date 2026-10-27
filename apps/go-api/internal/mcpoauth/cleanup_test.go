package mcpoauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestCleanupReportsSafeDatabaseFailureAndStopsOnCancellation(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgresql://localhost/unused")
	require.NoError(t, err)
	pool.Close()
	links, err := mcpoauth.NewLinks(pool, "https://issuer.example")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var reported error
	links.RunCleanup(ctx, func(err error) { reported = err; cancel() })
	require.ErrorIs(t, reported, mcpoauth.ErrUnavailable)
	links.RunCleanup(ctx, func(error) { t.Error("cancelled cleanup reported another attempt") })
}
