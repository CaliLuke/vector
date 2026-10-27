//go:build integration

package mcpoauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPeriodicCleanupRemovesExpiredProofsAndRetainsActiveFlow(t *testing.T) {
	db := linkDatabase(t)
	links, err := mcpoauth.NewLinks(db, "https://issuer.example")
	require.NoError(t, err)
	expired, active := uuid.New(), uuid.New()
	require.NoError(t, links.Start(t.Context(), mcpoauth.StartLink{ID: expired, UserID: uuid.MustParse("10000000-0000-4000-8000-000000000001"), VerifierCiphertext: "expired-encrypted-proof"}))
	require.NoError(t, links.Start(t.Context(), mcpoauth.StartLink{ID: active, UserID: uuid.MustParse("10000000-0000-4000-8000-000000000002"), StateHash: [32]byte{1}, VerifierCiphertext: "active-encrypted-proof"}))
	_, err = db.Exec(t.Context(), `UPDATE mcp_oauth_link_flows SET created_at=clock_timestamp()-INTERVAL '11 minutes',expires_at=clock_timestamp()-INTERVAL '2 minutes' WHERE id=$1`, expired)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); links.RunCleanup(ctx, nil) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("cleanup did not stop")
		}
	})
	require.Eventually(t, func() bool {
		var count int
		err := db.QueryRow(t.Context(), `SELECT count(*) FROM mcp_oauth_link_flows`).Scan(&count)
		return err == nil && count == 1
	}, 10*time.Second, 10*time.Millisecond)
	var retained uuid.UUID
	require.NoError(t, db.QueryRow(t.Context(), `SELECT id FROM mcp_oauth_link_flows`).Scan(&retained))
	require.Equal(t, active, retained)
}
