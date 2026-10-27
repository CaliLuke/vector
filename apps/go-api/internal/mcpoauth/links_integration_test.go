//go:build integration

package mcpoauth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/instancecipher"
	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func linkDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	linkSchema, err := os.ReadFile("../bootstrap/schema_migrations/000015_mcp_oauth_links.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	pair, err := postgres.StartUncloned(ctx, func(seedContext context.Context, connection string) error {
		db, connectErr := pgx.Connect(seedContext, connection)
		if connectErr != nil {
			return connectErr
		}
		defer func() { _ = db.Close(context.WithoutCancel(seedContext)) }()
		_, seedErr := db.Exec(seedContext, `CREATE TABLE users(id uuid PRIMARY KEY,is_active boolean NOT NULL,is_bot boolean NOT NULL,user_timezone text NOT NULL);
INSERT INTO users VALUES ('10000000-0000-4000-8000-000000000001',true,false,'UTC'),('10000000-0000-4000-8000-000000000002',true,false,'UTC');`+string(linkSchema))
		return seedErr
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := pair.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	connection := pair.SeedURL()
	config, err := pgxpool.ParseConfig(connection)
	if err != nil {
		t.Fatal(err)
	}
	if err = database.ConfigurePool(config); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestLinksRequireBothBrowserProofsAndRecheckDisconnect(t *testing.T) {
	db := linkDatabase(t)
	links, err := oauth.NewLinks(db, "https://issuer.example")
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	other := uuid.MustParse("10000000-0000-4000-8000-000000000002")
	session := sha256.Sum256([]byte("original-vector-session"))
	proof := sha256.Sum256([]byte("callback-browser-secret"))
	state := sha256.Sum256([]byte("authorization-state"))
	id := uuid.New()
	err = links.Start(t.Context(), oauth.StartLink{ID: id, UserID: user, SessionHash: session, StateHash: state,
		NonceHash: sha256.Sum256([]byte("nonce")), VerifierCiphertext: "encrypted-test-verifier"})
	if err != nil {
		t.Fatal(err)
	}
	exchange, err := links.Consume(t.Context(), state)
	if err != nil || exchange.ID != id {
		t.Fatalf("exchange admission: %+v %v", exchange, err)
	}
	_, err = links.Consume(t.Context(), state)
	if !errors.Is(err, oauth.ErrInvalidFlow) {
		t.Fatalf("reused state: %v", err)
	}
	_, err = links.Verified(t.Context(), id, "verified-person", proof)
	if err != nil {
		t.Fatal(err)
	}
	withoutProof, err := links.Status(t.Context(), user, session, [32]byte{})
	if err != nil || withoutProof.Pending != nil {
		t.Fatalf("pending identity leaked without callback proof: %+v %v", withoutProof, err)
	}
	wrongSession := sha256.Sum256([]byte("other-browser-session"))
	withoutSession, err := links.Status(t.Context(), user, wrongSession, proof)
	if err != nil || withoutSession.Pending != nil {
		t.Fatalf("pending identity leaked without original session: %+v %v", withoutSession, err)
	}
	err = links.Confirm(t.Context(), user, id, session, [32]byte{})
	if !errors.Is(err, oauth.ErrLinkConflict) {
		t.Fatalf("attacker confirmed without callback proof: %v", err)
	}
	err = links.Confirm(t.Context(), user, id, wrongSession, proof)
	if !errors.Is(err, oauth.ErrLinkConflict) {
		t.Fatalf("callback browser confirmed without original session: %v", err)
	}
	pending, err := links.Status(t.Context(), user, session, proof)
	if err != nil || pending.Pending == nil || pending.Pending.Subject != "verified-person" {
		t.Fatalf("pending status: %+v %v", pending, err)
	}
	err = links.Confirm(t.Context(), user, id, session, proof)
	if err != nil {
		t.Fatal(err)
	}
	linked, err := links.Resolve(t.Context(), "verified-person")
	if err != nil || linked.ID != user {
		t.Fatalf("linked identity: %+v %v", linked, err)
	}

	// Another current account cannot take an already-linked issuer subject.
	otherID := uuid.New()
	otherState := sha256.Sum256([]byte("other-state"))
	err = links.Start(t.Context(), oauth.StartLink{ID: otherID, UserID: other, SessionHash: wrongSession, StateHash: otherState,
		NonceHash: sha256.Sum256([]byte("other-nonce")), VerifierCiphertext: "other-encrypted-verifier"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = links.Consume(t.Context(), otherState); err != nil {
		t.Fatal(err)
	}
	if _, err = links.Verified(t.Context(), otherID, "verified-person", proof); err != nil {
		t.Fatal(err)
	}
	err = links.Confirm(t.Context(), other, otherID, wrongSession, proof)
	if !errors.Is(err, oauth.ErrLinkConflict) {
		t.Fatalf("subject ownership collision: %v", err)
	}

	if err = links.Disconnect(t.Context(), user); err != nil {
		t.Fatal(err)
	}
	_, err = links.Resolve(t.Context(), "verified-person")
	if !errors.Is(err, oauth.ErrInvalidCredential) {
		t.Fatalf("disconnected subject still resolves: %v", err)
	}
	err = links.Confirm(t.Context(), user, id, session, proof)
	if !errors.Is(err, oauth.ErrLinkConflict) {
		t.Fatalf("consumed confirmation resurrected link: %v", err)
	}
}

func TestDisconnectCancelsPendingIssuerExchange(t *testing.T) {
	db := linkDatabase(t)
	links, err := oauth.NewLinks(db, "https://issuer.example")
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	id := uuid.New()
	state := sha256.Sum256([]byte("state"))
	err = links.Start(t.Context(), oauth.StartLink{ID: id, UserID: user, SessionHash: sha256.Sum256([]byte("session")),
		StateHash: state, NonceHash: sha256.Sum256([]byte("nonce")), VerifierCiphertext: "encrypted-test-verifier"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = links.Consume(t.Context(), state); err != nil {
		t.Fatal(err)
	}
	if err = links.Disconnect(t.Context(), user); err != nil {
		t.Fatal(err)
	}
	_, err = links.Verified(t.Context(), id, "verified-person", sha256.Sum256([]byte("callback-proof")))
	if !errors.Is(err, oauth.ErrInvalidFlow) {
		t.Fatalf("late issuer exchange recreated pending link: %v", err)
	}
}

func TestSignedIssuerExchangeNeedsExplicitSameSessionConsent(t *testing.T) {
	db := linkDatabase(t)
	issuer := newIssuer(t)
	links, err := oauth.NewLinks(db, issuer.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := instancecipher.New("isolated-test-instance-key")
	if err != nil {
		t.Fatal(err)
	}
	linker, err := oauth.NewLinking(links, issuer.provider(t), cipher)
	if err != nil {
		t.Fatal(err)
	}
	browser := oauth.BrowserSession{UserID: uuid.MustParse("10000000-0000-4000-8000-000000000001"), Key: "signed-in-vector-session"}
	authorization, err := linker.Start(t.Context(), browser)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(authorization)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	issuer.challenge = query.Get("code_challenge")
	issuer.idClaims["nonce"] = query.Get("nonce")
	callback, err := linker.Complete(t.Context(), "one-time-code", query.Get("state"), "")
	if err != nil || callback.Secret == "" || !callback.ExpiresAt.After(time.Now()) {
		t.Fatalf("callback proof missing: %v", err)
	}
	_, err = links.Resolve(t.Context(), "person-1")
	if !errors.Is(err, oauth.ErrInvalidCredential) {
		t.Fatalf("callback established a link before consent: %v", err)
	}
	status, err := linker.Status(t.Context(), browser, callback.Secret)
	if err != nil || status.Pending == nil || status.Pending.Subject != "person-1" {
		t.Fatalf("verified status missing: %v", err)
	}
	otherSession := browser
	otherSession.Key = "different-signed-in-session"
	if err = linker.Confirm(t.Context(), otherSession, status.Pending.ID, callback.Secret); !errors.Is(err, oauth.ErrLinkConflict) {
		t.Fatalf("wrong session confirmed: %v", err)
	}
	if err = linker.Confirm(t.Context(), browser, status.Pending.ID, callback.Secret); err != nil {
		t.Fatal(err)
	}
	linked, err := links.Resolve(t.Context(), "person-1")
	if err != nil || linked.ID != browser.UserID {
		t.Fatalf("confirmed subject did not resolve: %v", err)
	}
	if err = linker.Disconnect(t.Context(), browser); err != nil {
		t.Fatal(err)
	}
	_, err = links.Resolve(t.Context(), "person-1")
	if !errors.Is(err, oauth.ErrInvalidCredential) {
		t.Fatalf("unlinked subject still resolved: %v", err)
	}
}

func TestLinkedSubjectRejectsCurrentInactiveOrBotOwner(t *testing.T) {
	db := linkDatabase(t)
	links, err := oauth.NewLinks(db, "https://issuer.example")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(t.Context(), `INSERT INTO mcp_oauth_links(issuer,subject,user_id) VALUES('https://issuer.example','person','10000000-0000-4000-8000-000000000001')`)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{
		`UPDATE users SET is_active=false WHERE id='10000000-0000-4000-8000-000000000001'`,
		`UPDATE users SET is_active=true,is_bot=true WHERE id='10000000-0000-4000-8000-000000000001'`,
	} {
		if _, err = db.Exec(t.Context(), mutation); err != nil {
			t.Fatal(err)
		}
		_, err = links.Resolve(t.Context(), "person")
		if !errors.Is(err, oauth.ErrInvalidCredential) {
			t.Fatalf("non-human or inactive owner accepted: %v", err)
		}
	}
}

func TestExpiredFlowCannotExchangeOrConfirm(t *testing.T) {
	db := linkDatabase(t)
	links, err := oauth.NewLinks(db, "https://issuer.example")
	if err != nil {
		t.Fatal(err)
	}
	user := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	state := sha256.Sum256([]byte("expiring-state"))
	session := sha256.Sum256([]byte("expiring-session"))
	proof := sha256.Sum256([]byte("expiring-proof"))
	id := uuid.New()
	input := oauth.StartLink{ID: id, UserID: user, SessionHash: session, StateHash: state,
		NonceHash: sha256.Sum256([]byte("nonce")), VerifierCiphertext: "encrypted-verifier"}
	if err = links.Start(t.Context(), input); err != nil {
		t.Fatal(err)
	}
	expire := func() {
		t.Helper()
		_, expiryErr := db.Exec(t.Context(), `UPDATE mcp_oauth_link_flows SET created_at=clock_timestamp()-INTERVAL '11 minutes',expires_at=clock_timestamp()-INTERVAL '2 minutes' WHERE id=$1`, id)
		if expiryErr != nil {
			t.Fatal(expiryErr)
		}
	}
	expire()
	_, err = links.Consume(t.Context(), state)
	if !errors.Is(err, oauth.ErrInvalidFlow) {
		t.Fatalf("expired callback exchanged: %v", err)
	}
	if err = links.Start(t.Context(), input); err != nil {
		t.Fatal(err)
	}
	if _, err = links.Consume(t.Context(), state); err != nil {
		t.Fatal(err)
	}
	if _, err = links.Verified(t.Context(), id, "person", proof); err != nil {
		t.Fatal(err)
	}
	expire()
	err = links.Confirm(t.Context(), user, id, session, proof)
	if !errors.Is(err, oauth.ErrLinkConflict) {
		t.Fatalf("expired confirmation accepted: %v", err)
	}
	if err = links.PruneExpired(t.Context()); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err = db.QueryRow(t.Context(), `SELECT count(*) FROM mcp_oauth_link_flows`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("expired verifier retained: count=%d, error=%v", remaining, err)
	}
}
