// Package mcpoauth verifies the configured issuer's end-user OAuth credentials.
package mcpoauth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

var (
	ErrInvalidCredential    = errors.New("invalid OAuth credential")
	ErrUnavailable          = errors.New("OAuth verification unavailable")
	ErrInvalidConfiguration = errors.New("invalid MCP OAuth configuration")
)

const (
	ReadScope         = "vector:read"
	WriteScope        = "vector:write"
	maxIssuerResponse = 1 << 20
)

// Config is operator-owned. Never include it in logs: client secrets are private.
type Config struct {
	Issuer                    string
	Resource                  string
	IntrospectionURL          string
	IntrospectionClientID     string
	IntrospectionClientSecret string
	LinkClientID              string
	LinkClientSecret          string
}

func (c Config) Validate() error {
	if !validEndpoint(c.Issuer) || !validEndpoint(c.IntrospectionURL) || !validEndpoint(c.Resource) {
		return ErrInvalidConfiguration
	}
	resource, _ := url.Parse(c.Resource)
	if resource.Path != "/mcp" || resource.RawPath != "" || c.IntrospectionClientID == c.LinkClientID {
		return ErrInvalidConfiguration
	}
	for _, value := range []string{c.IntrospectionClientID, c.IntrospectionClientSecret, c.LinkClientID, c.LinkClientSecret} {
		if strings.TrimSpace(value) == "" {
			return ErrInvalidConfiguration
		}
	}
	return nil
}

func validEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil &&
		u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.Opaque == ""
}

type Provider struct {
	config   Config
	client   *http.Client
	link     oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewProvider validates trusted discovery once. Access-token state is never cached.
// A custom client may supply a trusted test TLS root; timeouts and redirect policy
// are always owned here and cannot be disabled through that client.
func NewProvider(ctx context.Context, config Config, client *http.Client) (*Provider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	bounded := issuerClient(client)
	ctx = oidc.ClientContext(ctx, bounded)
	discovered, err := oidc.NewProvider(ctx, config.Issuer)
	if err != nil {
		return nil, ErrUnavailable
	}
	var metadata struct {
		Issuer           string   `json:"issuer"`
		AuthURL          string   `json:"authorization_endpoint"`
		TokenURL         string   `json:"token_endpoint"`
		JWKSURL          string   `json:"jwks_uri"`
		ResponseTypes    []string `json:"response_types_supported"`
		ResponseModes    []string `json:"response_modes_supported"`
		ChallengeMethods []string `json:"code_challenge_methods_supported"`
		Scopes           []string `json:"scopes_supported"`
		AuthMethods      []string `json:"token_endpoint_auth_methods_supported"`
		Algorithms       []string `json:"id_token_signing_alg_values_supported"`
	}
	if discovered.Claims(&metadata) != nil || metadata.Issuer != config.Issuer ||
		!validEndpoint(metadata.AuthURL) || !validEndpoint(metadata.TokenURL) || !validEndpoint(metadata.JWKSURL) ||
		!slices.Contains(metadata.ResponseTypes, "code") || !slices.Contains(metadata.ResponseModes, "form_post") ||
		!slices.Contains(metadata.ChallengeMethods, "S256") || !slices.Contains(metadata.Scopes, "openid") ||
		!slices.Contains(metadata.AuthMethods, "client_secret_basic") || !slices.Contains(metadata.Algorithms, "RS256") {
		return nil, ErrInvalidConfiguration
	}
	resource, _ := url.Parse(config.Resource)
	return &Provider{config: config, client: bounded,
		link: oauth2.Config{ClientID: config.LinkClientID, ClientSecret: config.LinkClientSecret,
			RedirectURL: resource.Scheme + "://" + resource.Host + "/auth/mcp/callback/", Scopes: []string{oidc.ScopeOpenID},
			Endpoint: oauth2.Endpoint{AuthURL: metadata.AuthURL, TokenURL: metadata.TokenURL, AuthStyle: oauth2.AuthStyleInHeader}},
		verifier: discovered.Verifier(&oidc.Config{ClientID: config.LinkClientID, SupportedSigningAlgs: []string{oidc.RS256}}),
	}, nil
}

// Identity is a fresh, issuer-validated access-token assertion, without the token.
type Identity struct {
	Issuer    string
	Subject   string
	ClientID  string
	Scopes    []string
	ExpiresAt time.Time
}

type audience []string

func (a *audience) UnmarshalJSON(raw []byte) error {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		*a = []string{single}
		return nil
	}
	var multiple []string
	if err := json.Unmarshal(raw, &multiple); err != nil {
		return err
	}
	*a = multiple
	return nil
}

func (p *Provider) Introspect(ctx context.Context, token string) (Identity, error) {
	if token == "" {
		return Identity{}, ErrInvalidCredential
	}
	form := url.Values{"token": {token}, "token_type_hint": {"access_token"}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.IntrospectionURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Identity{}, ErrUnavailable
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.SetBasicAuth(url.QueryEscape(p.config.IntrospectionClientID), url.QueryEscape(p.config.IntrospectionClientSecret))
	response, err := p.client.Do(request)
	if err != nil {
		return Identity{}, ErrUnavailable
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return Identity{}, ErrUnavailable
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Identity{}, ErrUnavailable
	}
	var activity struct {
		Active *bool `json:"active"`
	}
	if json.Unmarshal(body, &activity) != nil || activity.Active == nil {
		return Identity{}, ErrUnavailable
	}
	if !*activity.Active {
		return Identity{}, ErrInvalidCredential
	}
	var claims struct {
		Active     *bool    `json:"active"`
		Issuer     string   `json:"iss"`
		Subject    string   `json:"sub"`
		ClientID   string   `json:"client_id"`
		Audience   audience `json:"aud"`
		Expiration int64    `json:"exp"`
		NotBefore  *int64   `json:"nbf"`
		TokenType  string   `json:"token_type"`
		GrantType  string   `json:"grant_type"`
		Scope      string   `json:"scope"`
	}
	if json.Unmarshal(body, &claims) != nil {
		return Identity{}, ErrInvalidCredential
	}
	now := time.Now().Unix()
	if !*claims.Active || claims.Issuer != p.config.Issuer || claims.Subject == "" || claims.ClientID == "" ||
		!slices.Contains(claims.Audience, p.config.Resource) || claims.Expiration <= now ||
		(claims.NotBefore != nil && *claims.NotBefore > now) || !strings.EqualFold(claims.TokenType, "Bearer") ||
		(claims.GrantType != "authorization_code" && claims.GrantType != "refresh_token") {
		return Identity{}, ErrInvalidCredential
	}
	scopes, ok := parseScope(claims.Scope)
	if !ok {
		return Identity{}, ErrInvalidCredential
	}
	return Identity{Issuer: claims.Issuer, Subject: claims.Subject, ClientID: claims.ClientID,
		Scopes: scopes, ExpiresAt: time.Unix(claims.Expiration, 0)}, nil
}

func parseScope(raw string) ([]string, bool) {
	if raw == "" {
		return nil, true
	}
	for _, r := range raw {
		if r != ' ' && (r < 0x21 || r > 0x7e || r == '"' || r == '\\') {
			return nil, false
		}
	}
	scopes := strings.Split(raw, " ")
	if slices.Contains(scopes, "") {
		return nil, false
	}
	return scopes, true
}

func (p *Provider) AuthorizationURL(state, nonce, verifier string) string {
	return p.link.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("response_mode", "form_post"), oauth2.SetAuthURLParam("prompt", "select_account"))
}

// ExchangeSubject returns only a verified subject. No upstream token is retained.
// nonceHash comes from the independently generated, session-bound pending flow.
func (p *Provider) ExchangeSubject(ctx context.Context, code, verifier string, nonceHash [32]byte) (string, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	token, err := p.link.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return "", ErrUnavailable
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return "", ErrInvalidCredential
	}
	id, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return "", ErrInvalidCredential
	}
	var claims struct {
		AuthorizedParty string `json:"azp"`
		NotBefore       *int64 `json:"nbf"`
	}
	actualNonce := sha256.Sum256([]byte(id.Nonce))
	if id.Claims(&claims) != nil || id.Subject == "" || id.Issuer != p.config.Issuer ||
		!id.Expiry.After(time.Now()) || subtle.ConstantTimeCompare(actualNonce[:], nonceHash[:]) != 1 ||
		(claims.NotBefore != nil && *claims.NotBefore > time.Now().Unix()) ||
		(len(id.Audience) > 1 && claims.AuthorizedParty != p.config.LinkClientID) ||
		(claims.AuthorizedParty != "" && claims.AuthorizedParty != p.config.LinkClientID) {
		return "", ErrInvalidCredential
	}
	return id.Subject, nil
}

type boundedIssuerTransport struct{ base http.RoundTripper }

func issuerClient(client *http.Client) *http.Client {
	base := http.DefaultTransport
	if client != nil && client.Transport != nil {
		base = client.Transport
	}
	return &http.Client{Transport: boundedIssuerTransport{base: base}, Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return ErrUnavailable },
	}
}

func (t boundedIssuerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if err != nil {
		return nil, ErrUnavailable
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxIssuerResponse+1))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || len(body) > maxIssuerResponse {
		return nil, ErrUnavailable
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}
