package systemapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	oauth "github.com/CaliLuke/go-plane/internal/mcpoauth"
	api "github.com/CaliLuke/go-plane/mcpoauthlinks/gen/mcp_oauth_links"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
	"github.com/google/uuid"
)

type mcpLinkSessions interface {
	ValidateRequest(context.Context, *http.Request) (djangosession.Principal, error)
}

type mcpOAuthLinksService struct {
	linking   *oauth.Linking
	sessions  mcpLinkSessions
	issuer    string
	returnURL string
}

type mcpLinkBrowserContext struct{}

func NewMCPOAuthLinks(linking *oauth.Linking, sessions mcpLinkSessions, issuer, resource string) (api.Service, error) {
	parsed, err := url.Parse(resource)
	if linking == nil || sessions == nil || err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "/mcp" {
		return nil, oauth.ErrInvalidConfiguration
	}
	return &mcpOAuthLinksService{linking: linking, sessions: sessions, issuer: issuer,
		returnURL: parsed.Scheme + "://" + parsed.Host + "/settings/profile/api-tokens/"}, nil
}

func (s *mcpOAuthLinksService) APIKeyAuth(ctx context.Context, _ string, _ *security.APIKeyScheme) (context.Context, error) {
	metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
	if !ok {
		return ctx, mcpLinkError(oauth.ErrInvalidCredential)
	}
	principal, err := s.sessions.ValidateRequest(ctx, &http.Request{Header: metadata.Headers()})
	if errors.Is(err, djangosession.ErrStore) {
		return ctx, mcpLinkError(oauth.ErrUnavailable)
	}
	user, parseErr := uuid.Parse(principal.UserID)
	if err != nil || parseErr != nil || user == uuid.Nil || principal.SessionKey == "" {
		return ctx, mcpLinkError(oauth.ErrInvalidCredential)
	}
	return context.WithValue(ctx, mcpLinkBrowserContext{}, oauth.BrowserSession{UserID: user, Key: principal.SessionKey}), nil
}

func (s *mcpOAuthLinksService) Status(ctx context.Context, input *api.StatusPayload) (*api.LinkStatus, error) {
	browser, _ := ctx.Value(mcpLinkBrowserContext{}).(oauth.BrowserSession)
	status, err := s.linking.Status(ctx, browser, mcpLinkOptional(input.Proof))
	if err != nil {
		return nil, mcpLinkError(err)
	}
	result := &api.LinkStatus{Issuer: status.Issuer, Connected: status.Connected, Subject: loom.NullValue[string](), Pending: loom.NullValue[api.PendingLink]()}
	if status.Connected {
		result.Subject = loom.NullableValue(status.Subject)
	}
	if status.Pending != nil {
		result.Pending = loom.NullableValue(api.PendingLink{ID: status.Pending.ID.String(), Subject: status.Pending.Subject, ExpiresAt: status.Pending.ExpiresAt.UTC().Format(time.RFC3339Nano)})
	}
	return result, nil
}

func (s *mcpOAuthLinksService) Start(ctx context.Context) (*api.StartResult, error) {
	browser, _ := ctx.Value(mcpLinkBrowserContext{}).(oauth.BrowserSession)
	authorization, err := s.linking.Start(ctx, browser)
	if err != nil {
		return nil, mcpLinkError(err)
	}
	return &api.StartResult{AuthorizationURL: authorization, SetCookie: mcpLinkProofCookie("", time.Time{})}, nil
}

func (s *mcpOAuthLinksService) Confirm(ctx context.Context, input *api.ConfirmPayload) (*api.ConfirmResult, error) {
	browser, _ := ctx.Value(mcpLinkBrowserContext{}).(oauth.BrowserSession)
	id, err := uuid.Parse(input.ID)
	if err != nil {
		return nil, mcpLinkError(oauth.ErrInvalidFlow)
	}
	if err = s.linking.Confirm(ctx, browser, id, mcpLinkOptional(input.Proof)); err != nil {
		return nil, mcpLinkError(err)
	}
	return &api.ConfirmResult{SetCookie: mcpLinkProofCookie("", time.Time{})}, nil
}

func (s *mcpOAuthLinksService) Disconnect(ctx context.Context) (*api.DisconnectResult, error) {
	browser, _ := ctx.Value(mcpLinkBrowserContext{}).(oauth.BrowserSession)
	if err := s.linking.Disconnect(ctx, browser); err != nil {
		return nil, mcpLinkError(err)
	}
	return &api.DisconnectResult{SetCookie: mcpLinkProofCookie("", time.Time{})}, nil
}

func (s *mcpOAuthLinksService) Callback(ctx context.Context, input *api.CallbackPayload) (*api.CallbackResult, error) {
	if input.Iss != nil && *input.Iss != s.issuer {
		return nil, mcpLinkError(oauth.ErrInvalidFlow)
	}
	proof, err := s.linking.Complete(ctx, mcpLinkOptional(input.Code), input.State, mcpLinkOptional(input.Error))
	if err != nil {
		return nil, mcpLinkError(err)
	}
	cookie := mcpLinkProofCookie(proof.Secret, proof.ExpiresAt)
	return &api.CallbackResult{Location: s.returnURL, SetCookie: &cookie}, nil
}

func mcpLinkProofCookie(secret string, expires time.Time) string {
	cookie := http.Cookie{Name: "__Host-vector-mcp-link", Value: secret, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, Expires: expires}
	if secret == "" {
		cookie.MaxAge = -1
	}
	return cookie.String()
}

func mcpLinkOptional(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mcpLinkError(err error) *api.LinkError {
	switch {
	case errors.Is(err, oauth.ErrInvalidCredential):
		return &api.LinkError{Kind: "status_401", Message: "Sign in again to manage MCP access."}
	case errors.Is(err, oauth.ErrInvalidFlow):
		return &api.LinkError{Kind: "status_400", Message: "This connection attempt is invalid or expired. Start again."}
	case errors.Is(err, oauth.ErrLinkConflict):
		return &api.LinkError{Kind: "status_409", Message: "This account connection cannot be confirmed. Check its current status."}
	default:
		return &api.LinkError{Kind: "status_503", Message: "MCP account linking is temporarily unavailable."}
	}
}

var _ api.Service = (*mcpOAuthLinksService)(nil)
var _ api.Authorizer = (*mcpOAuthLinksService)(nil)
