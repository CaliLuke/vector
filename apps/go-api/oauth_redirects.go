package systemapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/authentication/authworkflow"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	oauthredirects "github.com/CaliLuke/go-plane/oauthredirects/gen/oauth_redirects"
	loomhttp "github.com/CaliLuke/loom/http"
	"golang.org/x/oauth2"
)

const (
	oauthSessionBackend = "django.contrib.auth.backends.ModelBackend"
	oauthSessionAge     = int64(1209600)
)

type oauthIdentity struct {
	ProviderID string
	Email      string
	FirstName  string
	LastName   string
	Avatar     string
}

type oauthTokens struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        *time.Time
	RefreshExpiresAt *time.Time
	IDToken          string
}

type oauthLoginRequest struct {
	Provider  string
	Identity  oauthIdentity
	Tokens    oauthTokens
	ClientIP  string
	UserAgent string
	Host      string
	Now       time.Time
}

type oauthLoginUser struct {
	ID                string
	Email             string
	PasswordHash      string
	Active            bool
	LastLogoutTimeSet bool
	Bot               bool
	WasNew            bool
}

type oauthRedirectRepository interface {
	InstanceConfigured(context.Context) (bool, error)
	ConfigValue(context.Context, string, string) (string, error)
	AuthenticateOAuth(context.Context, oauthLoginRequest) (oauthLoginUser, error)
	SaveOAuthAccount(context.Context, oauthLoginUser, string, string, oauthTokens, time.Time) error
	LoginRedirectPath(context.Context, oauthLoginUser) (string, error)
}

type oauthRedirectService struct {
	repository oauthRedirectRepository
	sessions   *djangosession.Validator
	workflow   interface {
		Run(context.Context, string, string) error
	}
	client *http.Client
}

func NewOAuthRedirects(repository oauthRedirectRepository, sessions *djangosession.Validator, workflow *authworkflow.PostUserAuthWorkflow, client *http.Client) (oauthredirects.Service, error) {
	if repository == nil || sessions == nil || workflow == nil {
		return nil, errors.New("OAuth redirects require a repository, Django sessions, and post-login workflow")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &oauthRedirectService{repository: repository, sessions: sessions, workflow: workflow, client: client}, nil
}

func (s *oauthRedirectService) GoogleInitiate(ctx context.Context, p *oauthredirects.GoogleInitiatePayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.initiate(ctx, "google", false, oauthNextPath(p.NextPath))
}
func (s *oauthRedirectService) GoogleCallback(ctx context.Context, p *oauthredirects.GoogleCallbackPayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.callback(ctx, "google", false, oauthString(p.Code), oauthString(p.State))
}
func (s *oauthRedirectService) GoogleSpaceInitiate(ctx context.Context, p *oauthredirects.GoogleSpaceInitiatePayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.initiate(ctx, "google", true, oauthNextPath(p.NextPath))
}
func (s *oauthRedirectService) GoogleSpaceCallback(ctx context.Context, p *oauthredirects.GoogleSpaceCallbackPayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.callback(ctx, "google", true, oauthString(p.Code), oauthString(p.State))
}
func (s *oauthRedirectService) GitHubInitiate(ctx context.Context, p *oauthredirects.GitHubInitiatePayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.initiate(ctx, "github", false, oauthNextPath(p.NextPath))
}
func (s *oauthRedirectService) GitHubCallback(ctx context.Context, p *oauthredirects.GitHubCallbackPayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.callback(ctx, "github", false, oauthString(p.Code), oauthString(p.State))
}
func (s *oauthRedirectService) GitHubSpaceInitiate(ctx context.Context, p *oauthredirects.GitHubSpaceInitiatePayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.initiate(ctx, "github", true, oauthNextPath(p.NextPath))
}
func (s *oauthRedirectService) GitHubSpaceCallback(ctx context.Context, p *oauthredirects.GitHubSpaceCallbackPayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.callback(ctx, "github", true, oauthString(p.Code), oauthString(p.State))
}
func (s *oauthRedirectService) GitLabInitiate(ctx context.Context, p *oauthredirects.GitLabInitiatePayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.initiate(ctx, "gitlab", false, oauthNextPath(p.NextPath))
}
func (s *oauthRedirectService) GitLabCallback(ctx context.Context, p *oauthredirects.GitLabCallbackPayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.callback(ctx, "gitlab", false, oauthString(p.Code), oauthString(p.State))
}
func (s *oauthRedirectService) GitLabSpaceInitiate(ctx context.Context, p *oauthredirects.GitLabSpaceInitiatePayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.initiate(ctx, "gitlab", true, oauthNextPath(p.NextPath))
}
func (s *oauthRedirectService) GitLabSpaceCallback(ctx context.Context, p *oauthredirects.GitLabSpaceCallbackPayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.callback(ctx, "gitlab", true, oauthString(p.Code), oauthString(p.State))
}
func (s *oauthRedirectService) GiteaInitiate(ctx context.Context, p *oauthredirects.GiteaInitiatePayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.initiate(ctx, "gitea", false, oauthNextPath(p.NextPath))
}
func (s *oauthRedirectService) GiteaCallback(ctx context.Context, p *oauthredirects.GiteaCallbackPayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.callback(ctx, "gitea", false, oauthString(p.Code), oauthString(p.State))
}
func (s *oauthRedirectService) GiteaSpaceInitiate(ctx context.Context, p *oauthredirects.GiteaSpaceInitiatePayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.initiate(ctx, "gitea", true, oauthNextPath(p.NextPath))
}
func (s *oauthRedirectService) GiteaSpaceCallback(ctx context.Context, p *oauthredirects.GiteaSpaceCallbackPayload) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	return s.callback(ctx, "gitea", true, oauthString(p.Code), oauthString(p.State))
}

func (s *oauthRedirectService) initiate(ctx context.Context, provider string, space bool, nextPath string) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	host, err := s.baseHost(ctx, space)
	if err != nil {
		return nil, nil, err
	}
	current, _ := djangosession.CurrentSessionFromContext(ctx)
	values := map[string]any{"host": host}
	if !space && nextPath != "" {
		values["next_path"] = nextPath
	}
	if provider == "gitea" && nextPath != "" {
		values["next_path"] = validateOAuthNextPath(nextPath)
	}
	refresh, err := s.sessions.SaveValues(ctx, current, values, oauthSessionAge, time.Now())
	if err != nil {
		return nil, nil, err
	}
	djangosession.SetResponseSession(ctx, refresh)
	current = &refresh
	configured, err := s.repository.InstanceConfigured(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !configured {
		return s.errorRedirect(ctx, host, nextPath, 5000, "INSTANCE_NOT_CONFIGURED", provider == "gitea")
	}
	config, err := s.providerConfig(ctx, provider)
	if err != nil {
		var failure *oauthFailure
		if errors.As(err, &failure) {
			return s.failureRedirect(ctx, host, nextPath, failure, provider == "gitea")
		}
		return nil, nil, err
	}
	state, err := oauthRandomState()
	if err != nil {
		return nil, nil, err
	}
	refresh, err = s.sessions.SaveValues(ctx, current, map[string]any{"state": state}, oauthSessionAge, time.Now())
	if err != nil {
		return nil, nil, err
	}
	djangosession.SetResponseSession(ctx, refresh)
	response, body := oauthResponse(oauthSourceOrderAuthURL(config.AuthCodeURL(state, config.options...), provider))
	return response, body, nil
}

func (s *oauthRedirectService) callback(ctx context.Context, provider string, space bool, code, state string) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	current, _ := djangosession.CurrentSessionFromContext(ctx)
	baseHost, err := s.baseHost(ctx, space)
	if err != nil {
		return nil, nil, err
	}
	if provider == "gitea" && !space {
		if savedHost, ok := sessionString(current, "host"); ok && savedHost != "" {
			baseHost = savedHost
		}
	}
	redirectBase := baseHost
	if provider == "gitea" && !space {
		if _, ok := sessionString(current, "host"); !ok {
			// Django preserves this callback's relative query redirect when no
			// initiation session supplied a host.
			redirectBase = ""
		}
	}
	nextPath, _ := sessionString(current, "next_path")
	savedState, _ := sessionString(current, "state")
	if state != savedState || code == "" {
		return s.errorRedirect(ctx, redirectBase, nextPath, oauthProviderError(provider), oauthProviderErrorName(provider), provider == "gitea")
	}
	config, err := s.providerConfig(ctx, provider)
	if err != nil {
		var failure *oauthFailure
		if errors.As(err, &failure) {
			return s.failureRedirect(ctx, redirectBase, nextPath, failure, provider == "gitea")
		}
		return nil, nil, err
	}
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return s.errorRedirect(ctx, redirectBase, nextPath, oauthProviderError(provider), oauthProviderErrorName(provider), provider == "gitea")
	}
	identity, err := s.providerIdentity(ctx, provider, token, config)
	if err != nil {
		var failure *oauthFailure
		if errors.As(err, &failure) {
			return s.failureRedirect(ctx, redirectBase, nextPath, failure, provider == "gitea")
		}
		return s.errorRedirect(ctx, redirectBase, nextPath, oauthProviderError(provider), oauthProviderErrorName(provider), provider == "gitea")
	}
	metadata, _ := loomhttp.RequestMetadataFromContext(ctx)
	loginUser, err := s.repository.AuthenticateOAuth(ctx, oauthLoginRequest{
		Provider: provider, Identity: identity,
		Tokens:   oauthTokens{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: oauthExpiry(token), IDToken: oauthExtraString(token, "id_token")},
		ClientIP: metadata.ClientAddr, UserAgent: metadata.UserAgent, Host: baseHost, Now: time.Now(),
	})
	if err != nil {
		var failure *oauthFailure
		if errors.As(err, &failure) {
			return s.failureRedirect(ctx, redirectBase, nextPath, failure, provider == "gitea")
		}
		return nil, nil, err
	}
	if !space {
		if runErr := s.workflow.Run(ctx, loginUser.ID, loginUser.Email); runErr != nil {
			return nil, nil, runErr
		}
	}
	if saveOAuthAccountErr := s.repository.SaveOAuthAccount(ctx, loginUser, provider, identity.ProviderID, oauthTokens{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: oauthExpiry(token), IDToken: oauthExtraString(token, "id_token")}, time.Now()); saveOAuthAccountErr != nil {
		return nil, nil, saveOAuthAccountErr
	}
	path := nextPath
	if path == "" && !space {
		path, err = s.repository.LoginRedirectPath(ctx, loginUser)
		if err != nil {
			return nil, nil, err
		}
	}
	if space {
		path = validateOAuthNextPath(path)
		location := strings.TrimRight(baseHost, "/") + path
		if provider == "gitea" {
			location = baseHost + path
		}
		if _, err := s.loginSession(ctx, current, loginUser, baseHost, metadata); err != nil {
			return nil, nil, err
		}
		response, body := oauthResponse(location)
		return response, body, nil
	}
	location := s.safeOAuthRedirect(ctx, baseHost, path, nil)
	if _, err := s.loginSession(ctx, current, loginUser, baseHost, metadata); err != nil {
		return nil, nil, err
	}
	response, body := oauthResponse(location)
	return response, body, nil
}

// The helpers below keep provider-specific wire details on oauth2.Config and
// leave Plane-specific identity, session, and redirect policy in the service.
type oauthConfig struct {
	*oauth2.Config
	options  []oauth2.AuthCodeOption
	callback string
	provider string
}

type oauthFailure struct {
	code  int
	name  string
	email string
}

func (e *oauthFailure) Error() string { return e.name }

func oauthResponse(location string) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser) {
	body := "<html><body><p>You are being redirected to <a href=\"" + html.EscapeString(location) + "\">" + html.EscapeString(location) + "</a>.</p></body></html>"
	return &oauthredirects.OAuthRedirectResponse{ContentType: "text/html; charset=utf-8", Location: location}, io.NopCloser(strings.NewReader(body))
}

func oauthRandomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func oauthString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func oauthNextPath(value *string) string { return oauthString(value) }
func sessionString(session *djangosession.SessionRefresh, key string) (string, bool) {
	if session == nil {
		return "", false
	}
	value, ok := session.Get(key)
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}
func oauthExtraString(token *oauth2.Token, key string) string {
	value, _ := token.Extra(key).(string)
	return value
}

func oauthExpiry(token *oauth2.Token) *time.Time {
	if token == nil || token.Expiry.IsZero() {
		return nil
	}
	expires := token.Expiry
	return &expires
}

func (s *oauthRedirectService) baseHost(ctx context.Context, space bool) (string, error) {
	web, err := s.repository.ConfigValue(ctx, "WEB_URL", "")
	if err != nil {
		return "", err
	}
	app, err := s.repository.ConfigValue(ctx, "APP_BASE_URL", "")
	if err != nil {
		return "", err
	}
	base := web
	if base == "" {
		base = app
	}
	if !space {
		if app != "" {
			base = app
		}
		return strings.TrimRight(base, "/"), nil
	}
	spaceURL, err := s.repository.ConfigValue(ctx, "SPACE_BASE_URL", "")
	if err != nil {
		return "", err
	}
	spacePath, err := s.repository.ConfigValue(ctx, "SPACE_BASE_PATH", "/spaces/")
	if err != nil {
		return "", err
	}
	if spaceURL != "" {
		base = spaceURL
	}
	if !strings.HasPrefix(spacePath, "/") {
		spacePath = "/" + spacePath
	}
	if !strings.HasSuffix(spacePath, "/") {
		spacePath += "/"
	}
	return strings.TrimRight(base, "/") + spacePath, nil
}

func (s *oauthRedirectService) loginSession(ctx context.Context, current *djangosession.SessionRefresh, user oauthLoginUser, host string, metadata loomhttp.RequestMetadata) (djangosession.SessionRefresh, error) {
	refresh, err := s.sessions.Login(ctx, current, djangosession.AuthenticatedSession{UserID: user.ID, PasswordHash: user.PasswordHash, Backend: oauthSessionBackend, DeviceInfo: map[string]string{"user_agent": metadata.UserAgent, "ip_address": metadata.ClientAddr, "domain": host}}, oauthSessionAge, time.Now())
	if err == nil {
		djangosession.SetResponseSession(ctx, refresh)
	}
	return refresh, err
}

func (s *oauthRedirectService) safeOAuthRedirect(ctx context.Context, baseURL, nextPath string, params map[string]string) string {
	path := validateOAuthNextPath(nextPath)
	base := strings.TrimRight(baseURL, "/")
	ordered := make([]string, 0, len(params)+1)
	if path != "" {
		// Keep nested query parameters and fragments inside the destination value.
		ordered = append(ordered, "next_path="+url.QueryEscape(path))
	}
	for _, key := range []string{"error_code", "error_message", "email"} {
		if value, ok := params[key]; ok {
			ordered = append(ordered, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	if len(ordered) == 0 {
		return base
	}
	result := base + "/?" + strings.Join(ordered, "&")
	if s.allowedOAuthHost(ctx, result) {
		return result
	}
	if len(params) > 0 {
		return base + "?" + oauthOrderedEncode(params, []string{"error_code", "error_message", "email"})
	}
	return base
}

func (s *oauthRedirectService) allowedOAuthHost(ctx context.Context, value string) bool {
	if strings.HasPrefix(value, "/") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	for _, key := range []string{"WEB_URL", "APP_BASE_URL", "ADMIN_BASE_URL", "SPACE_BASE_URL"} {
		raw, e := s.repository.ConfigValue(ctx, key, "")
		if e == nil && raw != "" {
			configured, e := url.Parse(raw)
			if e == nil && configured.Host == parsed.Host {
				return true
			}
		}
	}
	return false
}

func validateOAuthNextPath(value string) string {
	if value == "" || len(value) > 500 {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "")
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		value = parsed.Path
	}
	if !strings.HasPrefix(value, "/") || strings.Contains(value, "..") {
		return ""
	}
	lower := strings.ToLower(value)
	for _, bad := range []string{"javascript:", "data:", "vbscript:", "file:", "ftp:", "%2e%2e", "%2f%2f", "%5c%5c", "<script", "<iframe", "<object", "<embed", "<form", "onload=", "onerror=", "onclick="} {
		if strings.Contains(lower, bad) {
			return ""
		}
	}
	return value
}

func (s *oauthRedirectService) errorRedirect(ctx context.Context, base, next string, code int, name string, gitea bool) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	params := map[string]string{"error_code": fmt.Sprint(code), "error_message": name}
	if gitea {
		if next != "" {
			params["next_path"] = validateOAuthNextPath(next)
		}
		baseURL, parseErr := url.Parse(base)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		reference := &url.URL{RawQuery: oauthOrderedEncode(params, []string{"error_code", "error_message", "email", "next_path"})}
		u := baseURL.ResolveReference(reference).String()
		r, b := oauthResponse(u)
		return r, b, nil
	}
	u := s.safeOAuthRedirect(ctx, base, next, params)
	r, b := oauthResponse(u)
	return r, b, nil
}

func (s *oauthRedirectService) failureRedirect(ctx context.Context, base, next string, failure *oauthFailure, gitea bool) (*oauthredirects.OAuthRedirectResponse, io.ReadCloser, error) {
	if failure == nil {
		return s.errorRedirect(ctx, base, next, 5999, "AUTHENTICATION_FAILED", gitea)
	}
	params := map[string]string{"error_code": fmt.Sprint(failure.code), "error_message": failure.name}
	if failure.email != "" {
		params["email"] = failure.email
	}
	if gitea && next != "" {
		params["next_path"] = validateOAuthNextPath(next)
	}
	if gitea {
		baseURL, err := url.Parse(base)
		if err != nil {
			return nil, nil, err
		}
		reference := &url.URL{RawQuery: oauthOrderedEncode(params, []string{"error_code", "error_message", "email", "next_path"})}
		location := baseURL.ResolveReference(reference).String()
		response, body := oauthResponse(location)
		return response, body, nil
	}
	location := s.safeOAuthRedirect(ctx, base, next, params)
	response, body := oauthResponse(location)
	return response, body, nil
}

func oauthOrderedEncode(values map[string]string, order []string) string {
	parts := make([]string, 0, len(values))
	for _, key := range order {
		if value, ok := values[key]; ok {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	return strings.Join(parts, "&")
}

func oauthSourceOrderAuthURL(rawURL, provider string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	order := []string{"client_id", "scope", "redirect_uri", "response_type", "access_type", "prompt", "state"}
	switch provider {
	case "github":
		order = []string{"client_id", "redirect_uri", "scope", "state"}
	case "gitlab":
		order = []string{"client_id", "redirect_uri", "response_type", "scope", "state"}
	case "gitea":
		order = []string{"client_id", "scope", "redirect_uri", "response_type", "state"}
	}
	ordered := make(map[string]string, len(query))
	for key, values := range query {
		if len(values) > 0 {
			ordered[key] = values[0]
		}
	}
	parts := make([]string, 0, len(ordered))
	for _, key := range order {
		if value, ok := ordered[key]; ok {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
			delete(ordered, key)
		}
	}
	for key, value := range ordered {
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
	}
	parsed.RawQuery = strings.Join(parts, "&")
	return parsed.String()
}

func oauthProviderError(provider string) int {
	switch provider {
	case "google":
		return 5115
	case "github":
		return 5120
	case "gitlab":
		return 5121
	default:
		return 5123
	}
}
func oauthProviderErrorName(provider string) string {
	return strings.ToUpper(provider) + "_OAUTH_PROVIDER_ERROR"
}
func oauthConfigError(provider string) int {
	switch provider {
	case "google":
		return 5105
	case "github":
		return 5110
	case "gitlab":
		return 5111
	default:
		return 5112
	}
}
func oauthConfigErrorName(provider string) string {
	return strings.ToUpper(provider) + "_NOT_CONFIGURED"
}

func (s *oauthRedirectService) providerConfig(ctx context.Context, provider string) (oauthConfig, error) {
	values := func(key, fallback string) (string, error) { return s.repository.ConfigValue(ctx, key, fallback) }
	clientID, err := values(strings.ToUpper(provider)+"_CLIENT_ID", "")
	if err != nil {
		return oauthConfig{}, err
	}
	clientSecret, err := values(strings.ToUpper(provider)+"_CLIENT_SECRET", "")
	if err != nil {
		return oauthConfig{}, err
	}
	if clientID == "" || clientSecret == "" {
		return oauthConfig{}, &oauthFailure{code: oauthConfigError(provider), name: oauthConfigErrorName(provider)}
	}
	metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
	if !ok || metadata.Host == "" || (metadata.Scheme != "http" && metadata.Scheme != "https") {
		return oauthConfig{}, errors.New("OAuth request metadata is missing effective host or scheme")
	}
	callback := "/auth/" + provider + "/callback/"
	endpoint := oauth2.Endpoint{}
	scopes := []string{}
	options := []oauth2.AuthCodeOption{}
	switch provider {
	case "google":
		endpoint = oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"}
		scopes = []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}
		options = []oauth2.AuthCodeOption{oauth2.AccessTypeOffline, oauth2.ApprovalForce}
	case "github":
		endpoint = oauth2.Endpoint{AuthURL: "https://github.com/login/oauth/authorize", TokenURL: "https://github.com/login/oauth/access_token"}
		scopes = []string{"read:user", "user:email"}
		organization, readErr := values("GITHUB_ORGANIZATION_ID", "")
		if readErr != nil {
			return oauthConfig{}, readErr
		}
		if organization != "" {
			scopes = append(scopes, "read:org")
		}
	case "gitlab":
		host, readErr := values("GITLAB_HOST", "https://gitlab.com")
		if readErr != nil {
			return oauthConfig{}, readErr
		}
		host = strings.TrimRight(host, "/")
		parsed, parseErr := url.Parse(host)
		if parseErr != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return oauthConfig{}, &oauthFailure{code: 5111, name: "GITLAB_NOT_CONFIGURED"}
		}
		endpoint = oauth2.Endpoint{AuthURL: host + "/oauth/authorize", TokenURL: host + "/oauth/token"}
		scopes = []string{"read_user"}
	case "gitea":
		host, readErr := values("GITEA_HOST", "")
		if readErr != nil {
			return oauthConfig{}, readErr
		}
		host = strings.TrimRight(host, "/")
		parsed, parseErr := url.Parse(host)
		if parseErr != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return oauthConfig{}, &oauthFailure{code: oauthConfigError(provider), name: oauthConfigErrorName(provider)}
		}
		endpoint = oauth2.Endpoint{AuthURL: host + "/login/oauth/authorize", TokenURL: host + "/login/oauth/access_token"}
		scopes = []string{"openid", "email", "profile", "read:user"}
	default:
		return oauthConfig{}, errors.New("unknown OAuth provider")
	}
	config := &oauth2.Config{ClientID: clientID, ClientSecret: clientSecret, RedirectURL: metadata.Scheme + "://" + metadata.Host + callback, Scopes: scopes, Endpoint: endpoint}
	return oauthConfig{Config: config, options: options, callback: metadata.Scheme + "://" + metadata.Host + callback, provider: provider}, nil
}

func (s *oauthRedirectService) providerIdentity(ctx context.Context, provider string, token *oauth2.Token, config oauthConfig) (oauthIdentity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.client)
	client := config.Client(ctx, token)
	var profile map[string]any
	var err error
	switch provider {
	case "google":
		profile, err = oauthGetJSON(ctx, client, "https://www.googleapis.com/oauth2/v2/userinfo")
	case "github":
		profile, err = oauthGetJSON(ctx, client, "https://api.github.com/user")
		if err == nil {
			organization, _ := s.repository.ConfigValue(ctx, "GITHUB_ORGANIZATION_ID", "")
			if organization != "" {
				username := oauthStringField(profile, "login")
				resp, requestErr := client.Get("https://api.github.com/orgs/" + url.PathEscape(organization) + "/memberships/" + url.PathEscape(username))
				if requestErr != nil {
					return oauthIdentity{}, &oauthFailure{code: 5122, name: "GITHUB_USER_NOT_IN_ORG"}
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					return oauthIdentity{}, &oauthFailure{code: 5122, name: "GITHUB_USER_NOT_IN_ORG"}
				}
			}
			var emails []map[string]any
			err = oauthGetJSONList(ctx, client, "https://api.github.com/user/emails", &emails)
			if err == nil {
				for _, entry := range emails {
					if entry["primary"] == true && entry["verified"] == true {
						profile["email"] = entry["email"]
						break
					}
				}
			}
		}
	case "gitlab":
		host, _ := s.repository.ConfigValue(ctx, "GITLAB_HOST", "https://gitlab.com")
		profile, err = oauthGetJSON(ctx, client, strings.TrimRight(host, "/")+"/api/v4/user")
	case "gitea":
		host, _ := s.repository.ConfigValue(ctx, "GITEA_HOST", "")
		base := strings.TrimRight(host, "/")
		profile, err = oauthGetJSON(ctx, client, base+"/api/v1/user")
		if err == nil {
			var emails []map[string]any
			err = oauthGetJSONList(ctx, client, base+"/api/v1/user/emails", &emails)
			if err == nil {
				email := ""
				for _, entry := range emails {
					if entry["primary"] == true && entry["verified"] == true {
						email = oauthStringField(entry, "email")
						break
					}
				}
				if email == "" {
					for _, entry := range emails {
						if entry["verified"] == true {
							email = oauthStringField(entry, "email")
							break
						}
					}
				}
				profile["email"] = email
			}
		}
	}
	if err != nil {
		return oauthIdentity{}, err
	}
	identity := oauthIdentity{}
	switch provider {
	case "google":
		if profile["verified_email"] != true {
			return identity, &oauthFailure{code: 5124, name: "OAUTH_PROVIDER_UNVERIFIED_EMAIL"}
		}
		identity = oauthIdentity{ProviderID: oauthStringField(profile, "id"), Email: oauthStringField(profile, "email"), FirstName: oauthStringField(profile, "given_name"), LastName: oauthStringField(profile, "family_name"), Avatar: oauthStringField(profile, "picture")}
	case "github":
		if oauthStringField(profile, "email") == "" {
			return identity, &oauthFailure{code: 5124, name: "OAUTH_PROVIDER_UNVERIFIED_EMAIL"}
		}
		identity = oauthIdentity{ProviderID: oauthID(profile["id"]), Email: oauthStringField(profile, "email"), FirstName: oauthStringField(profile, "name"), LastName: oauthStringField(profile, "family_name"), Avatar: oauthStringField(profile, "avatar_url")}
	case "gitlab":
		if oauthStringField(profile, "confirmed_at") == "" {
			return identity, &oauthFailure{code: 5124, name: "OAUTH_PROVIDER_UNVERIFIED_EMAIL"}
		}
		identity = oauthIdentity{ProviderID: oauthID(profile["id"]), Email: oauthStringField(profile, "email"), FirstName: oauthStringField(profile, "name"), LastName: oauthStringField(profile, "family_name"), Avatar: oauthStringField(profile, "avatar_url")}
	case "gitea":
		identity = oauthIdentity{ProviderID: oauthID(profile["id"]), Email: oauthStringField(profile, "email"), FirstName: firstNonempty(oauthStringField(profile, "full_name"), oauthStringField(profile, "login")), Avatar: oauthStringField(profile, "avatar_url")}
	}
	if identity.Email == "" || identity.ProviderID == "" {
		return oauthIdentity{}, &oauthFailure{code: oauthProviderError(provider), name: oauthProviderErrorName(provider)}
	}
	return identity, nil
}

func oauthGetJSON(ctx context.Context, client *http.Client, endpoint string) (map[string]any, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		response.Body.Close()
		return nil, fmt.Errorf("OAuth provider returned HTTP %d", response.StatusCode)
	}
	return oauthDecodeObject(response)
}
func oauthGetJSONList(ctx context.Context, client *http.Client, endpoint string, dest any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("OAuth provider returned HTTP %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(dest)
}
func oauthStringField(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return value
}
func oauthID(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return ""
	}
}
func firstNonempty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// Keep JSON shape checking centralized so provider response decoding rejects
// non-object bodies instead of silently treating them as zero-valued profiles.
func oauthDecodeObject(response *http.Response) (map[string]any, error) {
	defer response.Body.Close()
	var value map[string]any
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("OAuth provider returned a non-object response")
	}
	return value, nil
}
