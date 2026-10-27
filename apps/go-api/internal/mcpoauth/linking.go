package mcpoauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type linkCipher interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}

type linkingIssuer interface {
	AuthorizationURL(string, string, string) string
	ExchangeSubject(context.Context, string, string, [32]byte) (string, error)
}

// Linking owns session-bound OIDC proof and consent. The HTTP adapter supplies
// only a freshly validated session; a callback can verify a subject but cannot
// establish the account link without the separate same-session confirmation.
type Linking struct {
	links  *Links
	issuer linkingIssuer
	cipher linkCipher
}

func NewLinking(links *Links, issuer linkingIssuer, cipher linkCipher) (*Linking, error) {
	if links == nil || issuer == nil || cipher == nil {
		return nil, ErrInvalidConfiguration
	}
	return &Linking{links: links, issuer: issuer, cipher: cipher}, nil
}

type BrowserSession struct {
	UserID uuid.UUID
	Key    string
}

func (s *Linking) Start(ctx context.Context, browser BrowserSession) (string, error) {
	if browser.UserID == uuid.Nil || browser.Key == "" {
		return "", ErrInvalidCredential
	}
	if err := s.links.PruneExpired(ctx); err != nil {
		return "", ErrUnavailable
	}
	state, nonce := randomProof(), randomProof()
	verifier := oauth2.GenerateVerifier()
	encrypted, err := s.cipher.Encrypt(verifier)
	if err != nil {
		return "", ErrUnavailable
	}
	err = s.links.Start(ctx, StartLink{ID: uuid.New(), UserID: browser.UserID, SessionHash: sha256.Sum256([]byte(browser.Key)),
		StateHash: sha256.Sum256([]byte(state)), NonceHash: sha256.Sum256([]byte(nonce)), VerifierCiphertext: encrypted})
	if err != nil {
		return "", safeLinkError(err)
	}
	return s.issuer.AuthorizationURL(state, nonce, verifier), nil
}

type CallbackProof struct {
	Secret    string
	ExpiresAt time.Time
}

// Complete handles a single generated/validated callback payload. A denial
// consumes the flow but produces no proof; no external error text is returned.
func (s *Linking) Complete(ctx context.Context, code, state, denial string) (CallbackProof, error) {
	rawState, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil || len(rawState) != 32 || (code == "") == (denial == "") {
		return CallbackProof{}, ErrInvalidFlow
	}
	exchange, err := s.links.Consume(ctx, sha256.Sum256([]byte(state)))
	if err != nil {
		return CallbackProof{}, safeLinkError(err)
	}
	complete := false
	defer func() {
		if !complete {
			// The admission is already durable. Cleanup cannot authorize a replay,
			// and expires_at still bounds the encrypted verifier if cleanup fails.
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = s.links.Fail(cleanup, exchange.ID)
		}
	}()
	if denial != "" {
		return CallbackProof{}, nil
	}
	verifier, err := s.cipher.Decrypt(exchange.VerifierCiphertext)
	if err != nil || verifier == "" {
		return CallbackProof{}, ErrUnavailable
	}
	subject, err := s.issuer.ExchangeSubject(ctx, code, verifier, exchange.NonceHash)
	if err != nil {
		return CallbackProof{}, safeLinkError(err)
	}
	proof := randomProof()
	expires, err := s.links.Verified(ctx, exchange.ID, subject, sha256.Sum256([]byte(proof)))
	if err != nil {
		return CallbackProof{}, safeLinkError(err)
	}
	complete = true
	return CallbackProof{Secret: proof, ExpiresAt: expires}, nil
}

func (s *Linking) Status(ctx context.Context, browser BrowserSession, proof string) (LinkStatus, error) {
	if browser.UserID == uuid.Nil || browser.Key == "" {
		return LinkStatus{}, ErrInvalidCredential
	}
	result, err := s.links.Status(ctx, browser.UserID, sha256.Sum256([]byte(browser.Key)), sha256.Sum256([]byte(proof)))
	return result, safeLinkError(err)
}

func (s *Linking) Confirm(ctx context.Context, browser BrowserSession, id uuid.UUID, proof string) error {
	if browser.UserID == uuid.Nil || browser.Key == "" {
		return ErrInvalidCredential
	}
	if proof == "" || id == uuid.Nil {
		return ErrLinkConflict
	}
	return safeLinkError(s.links.Confirm(ctx, browser.UserID, id, sha256.Sum256([]byte(browser.Key)), sha256.Sum256([]byte(proof))))
}

func (s *Linking) Disconnect(ctx context.Context, browser BrowserSession) error {
	if browser.UserID == uuid.Nil || browser.Key == "" {
		return ErrInvalidCredential
	}
	return safeLinkError(s.links.Disconnect(ctx, browser.UserID))
}

func safeLinkError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrInvalidCredential):
		return ErrInvalidCredential
	case errors.Is(err, ErrInvalidFlow):
		return ErrInvalidFlow
	case errors.Is(err, ErrLinkConflict):
		return ErrLinkConflict
	default:
		return ErrUnavailable
	}
}

func randomProof() string {
	var value [32]byte
	_, _ = rand.Read(value[:])
	return base64.RawURLEncoding.EncodeToString(value[:])
}
