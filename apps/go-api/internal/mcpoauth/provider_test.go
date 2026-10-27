package mcpoauth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"
)

type issuerFixture struct {
	server    *httptest.Server
	key       *rsa.PrivateKey
	claims    map[string]any
	idClaims  map[string]any
	status    int
	raw       string
	calls     int
	challenge string
	grant     string
}

func newIssuer(t *testing.T) *issuerFixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &issuerFixture{key: key, status: 200}
	f.server = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	f.claims = map[string]any{"active": true, "iss": f.server.URL, "sub": "person-1", "client_id": "agent-1",
		"aud": "https://vector.example/mcp", "exp": time.Now().Add(time.Hour).Unix(),
		"scope": "vector:read vector:write", "token_type": "Bearer", "grant_type": "authorization_code"}
	f.idClaims = map[string]any{"iss": f.server.URL, "sub": "person-1", "aud": "link-client",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": "independent-nonce"}
	return f
}

func (f *issuerFixture) config() oauth.Config {
	return oauth.Config{Issuer: f.server.URL, Resource: "https://vector.example/mcp", IntrospectionURL: f.server.URL + "/introspect",
		IntrospectionClientID: "introspection-client", IntrospectionClientSecret: "introspection-secret",
		LinkClientID: "link-client", LinkClientSecret: "link-secret"}
}

func (f *issuerFixture) provider(t *testing.T) *oauth.Provider {
	t.Helper()
	p, err := oauth.NewProvider(context.Background(), f.config(), f.server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (f *issuerFixture) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		_ = json.NewEncoder(w).Encode(map[string]any{"issuer": f.server.URL, "authorization_endpoint": f.server.URL + "/authorize",
			"token_endpoint": f.server.URL + "/token", "jwks_uri": f.server.URL + "/keys",
			"response_types_supported": []string{"code"}, "response_modes_supported": []string{"form_post"},
			"code_challenge_methods_supported": []string{"S256"}, "scopes_supported": []string{"openid"},
			"token_endpoint_auth_methods_supported": []string{"client_secret_basic"}, "id_token_signing_alg_values_supported": []string{"RS256"}})
	case "/keys":
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &f.key.PublicKey, KeyID: "key-1", Algorithm: "RS256", Use: "sig"}}})
	case "/introspect":
		f.calls++
		client, secret, ok := r.BasicAuth()
		if !ok || client != "introspection-client" || secret != "introspection-secret" || r.Method != "POST" {
			http.Error(w, "bad issuer credentials", http.StatusUnauthorized)
			return
		}
		if r.ParseForm() != nil || r.Form.Get("token") != "access-user" || r.Form.Get("token_type_hint") != "access_token" {
			_, _ = io.WriteString(w, `{"active":false}`)
			return
		}
		w.WriteHeader(f.status)
		if f.raw != "" {
			_, _ = io.WriteString(w, f.raw)
			return
		}
		_ = json.NewEncoder(w).Encode(f.claims)
	case "/token":
		client, secret, ok := r.BasicAuth()
		if !ok || client != "link-client" || secret != "link-secret" || r.ParseForm() != nil ||
			r.Form.Get("code") != "one-time-code" || r.Form.Get("redirect_uri") != "https://vector.example/auth/mcp/callback/" ||
			oauth2.S256ChallengeFromVerifier(r.Form.Get("code_verifier")) != f.challenge {
			http.Error(w, "invalid code exchange", 400)
			return
		}
		f.grant = r.Form.Get("grant_type")
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: &jose.JSONWebKey{Key: f.key, KeyID: "key-1"}}, nil)
		if err != nil {
			http.Error(w, "signer failed", 500)
			return
		}
		claims, _ := json.Marshal(f.idClaims)
		signed, err := signer.Sign(claims)
		if err != nil {
			http.Error(w, "signing failed", 500)
			return
		}
		raw, err := signed.CompactSerialize()
		if err != nil {
			http.Error(w, "encoding failed", 500)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "link-only-access", "token_type": "Bearer", "id_token": raw})
	default:
		http.NotFound(w, r)
	}
}

func TestIntrospectionRechecksRevocationAndScopeReduction(t *testing.T) {
	f := newIssuer(t)
	p := f.provider(t)
	first, err := p.Introspect(context.Background(), "access-user")
	if err != nil || first.Subject != "person-1" || len(first.Scopes) != 2 {
		t.Fatalf("first identity = %+v, %v", first, err)
	}
	f.claims["scope"] = "vector:read"
	second, err := p.Introspect(context.Background(), "access-user")
	if err != nil || len(second.Scopes) != 1 || second.Scopes[0] != oauth.ReadScope {
		t.Fatalf("narrowed identity = %+v, %v", second, err)
	}
	f.claims["active"] = false
	_, err = p.Introspect(context.Background(), "access-user")
	if !errors.Is(err, oauth.ErrInvalidCredential) || f.calls != 3 {
		t.Fatalf("revocation = %v, calls = %d", err, f.calls)
	}
}

func TestIntrospectionRejectsInvalidCredentialClaims(t *testing.T) {
	for _, test := range []struct {
		name, key string
		value     any
	}{
		{"issuer", "iss", "https://other.example"}, {"audience", "aud", []string{"https://vector.example/mcp/api-key"}},
		{"missing expiry", "exp", nil}, {"expired", "exp", time.Now().Add(-time.Minute).Unix()},
		{"future not before", "nbf", time.Now().Add(time.Minute).Unix()},
		{"missing subject", "sub", ""}, {"missing client", "client_id", ""}, {"machine", "grant_type", "client_credentials"},
		{"missing grant", "grant_type", nil}, {"wrong type", "token_type", "ID"}, {"malformed scope", "scope", "vector:read\tvector:write"},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newIssuer(t)
			p := f.provider(t)
			f.claims[test.key] = test.value
			_, err := p.Introspect(context.Background(), "access-user")
			if !errors.Is(err, oauth.ErrInvalidCredential) {
				t.Fatalf("expected invalid credential, got %v", err)
			}
		})
	}
}

func TestIntrospectionFailsClosedWithoutDisclosingIssuerBodies(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"outage", `{"private":"credential-secret"}`, 503},
		{"malformed", `credential-secret`, 200}, {"missing active", `{"private":"credential-secret"}`, 200},
		{"oversized", strings.Repeat("credential-secret", 100000), 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newIssuer(t)
			p := f.provider(t)
			f.raw = test.body
			f.status = test.status
			_, err := p.Introspect(context.Background(), "access-user")
			if !errors.Is(err, oauth.ErrUnavailable) || strings.Contains(fmt.Sprint(err), "credential-secret") {
				t.Fatalf("unsafe outage error: %v", err)
			}
		})
	}
}

func TestLinkingUsesPKCEAndVerifiesSignedSubject(t *testing.T) {
	f := newIssuer(t)
	p := f.provider(t)
	verifier := oauth2.GenerateVerifier()
	authURL, err := url.Parse(p.AuthorizationURL("independent-state", "independent-nonce", verifier))
	if err != nil {
		t.Fatal(err)
	}
	q := authURL.Query()
	if q.Get("state") != "independent-state" || q.Get("nonce") != "independent-nonce" || q.Get("code_challenge_method") != "S256" ||
		q.Get("scope") != "openid" || q.Get("response_mode") != "form_post" || q.Get("client_secret") != "" || q.Get("code_verifier") != "" {
		t.Fatal("authorization request does not preserve the linking contract")
	}
	f.challenge = q.Get("code_challenge")
	subject, err := p.ExchangeSubject(context.Background(), "one-time-code", verifier, sha256.Sum256([]byte("independent-nonce")))
	if err != nil || subject != "person-1" || f.grant != "authorization_code" {
		t.Fatalf("subject=%q, error=%v, grant=%q", subject, err, f.grant)
	}
}

func TestLinkingRejectsMismatchedIdentityProof(t *testing.T) {
	for _, test := range []struct {
		name, key string
		value     any
	}{
		{"nonce", "nonce", "attacker-nonce"}, {"issuer", "iss", "https://wrong.example"}, {"audience", "aud", "other-client"},
		{"azp", "azp", "other-client"}, {"multi audience without azp", "aud", []string{"link-client", "other-client"}},
		{"expiry", "exp", time.Now().Add(-time.Minute).Unix()}, {"future nbf", "nbf", time.Now().Add(time.Minute).Unix()},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newIssuer(t)
			p := f.provider(t)
			verifier := oauth2.GenerateVerifier()
			f.challenge = oauth2.S256ChallengeFromVerifier(verifier)
			f.idClaims[test.key] = test.value
			_, err := p.ExchangeSubject(context.Background(), "one-time-code", verifier, sha256.Sum256([]byte("independent-nonce")))
			if !errors.Is(err, oauth.ErrInvalidCredential) {
				t.Fatalf("mismatched proof accepted: %v", err)
			}
		})
	}
}
