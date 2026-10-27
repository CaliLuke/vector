//go:build integration

package mcpoauth_test

import (
	"net/url"
	"testing"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFailedCallbacksConsumeProofAndRemoveEncryptedVerifier(t *testing.T) {
	for _, scenario := range []struct {
		name, code, denial string
		wantError          error
	}{
		{name: "provider denial", denial: "access_denied"},
		{name: "invalid signed nonce", code: "one-time-code", wantError: oauth.ErrInvalidCredential},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := linkDatabase(t)
			issuer := newIssuer(t)
			links, err := oauth.NewLinks(db, issuer.server.URL)
			require.NoError(t, err)
			cipher, err := instancecipher.New("isolated-test-instance-key")
			require.NoError(t, err)
			linker, err := oauth.NewLinking(links, issuer.provider(t), cipher)
			require.NoError(t, err)
			browser := oauth.BrowserSession{UserID: uuid.MustParse("10000000-0000-4000-8000-000000000001"), Key: "signed-in-session"}
			authorization, err := linker.Start(t.Context(), browser)
			require.NoError(t, err)
			parsed, err := url.Parse(authorization)
			require.NoError(t, err)
			issuer.challenge = parsed.Query().Get("code_challenge")
			state := parsed.Query().Get("state")

			proof, err := linker.Complete(t.Context(), scenario.code, state, scenario.denial)

			require.ErrorIs(t, err, scenario.wantError)
			require.Empty(t, proof.Secret)
			var remaining int
			require.NoError(t, db.QueryRow(t.Context(), `SELECT count(*) FROM mcp_oauth_link_flows`).Scan(&remaining))
			require.Zero(t, remaining)
			_, err = links.Resolve(t.Context(), "person-1")
			require.ErrorIs(t, err, oauth.ErrInvalidCredential)
			_, err = linker.Complete(t.Context(), scenario.code, state, scenario.denial)
			require.ErrorIs(t, err, oauth.ErrInvalidFlow)
		})
	}
}
