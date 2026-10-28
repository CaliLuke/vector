package systemapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	oauthredirectssvr "github.com/CaliLuke/go-plane/oauthredirects/gen/http/oauth_redirects/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type oauthRedirectTestRepository struct{ config map[string]string }

func TestOAuthInitiateDecodesGETQueryWithoutRequestBody(t *testing.T) {
	decode := oauthredirectssvr.DecodeGoogleInitiateRequest(nil, loomhttp.RequestDecoder)
	request := httptest.NewRequest(http.MethodGet, "/auth/google/?next_path=%2Fworkspace%2Fhome", nil)
	payload, err := decode(request)
	if err != nil {
		t.Fatalf("decode GET query: %v", err)
	}
	if payload.NextPath == nil || *payload.NextPath != "/workspace/home" {
		t.Fatalf("next_path=%v, want /workspace/home", payload.NextPath)
	}
}

func (r *oauthRedirectTestRepository) InstanceConfigured(context.Context) (bool, error) {
	return true, nil
}
func (r *oauthRedirectTestRepository) ConfigValue(_ context.Context, key, fallback string) (string, error) {
	if value, ok := r.config[key]; ok {
		return value, nil
	}
	return fallback, nil
}
func (*oauthRedirectTestRepository) AuthenticateOAuth(context.Context, oauthLoginRequest) (oauthLoginUser, error) {
	return oauthLoginUser{}, nil
}
func (*oauthRedirectTestRepository) SaveOAuthAccount(context.Context, oauthLoginUser, string, string, oauthTokens, time.Time) error {
	return nil
}
func (*oauthRedirectTestRepository) LoginRedirectPath(context.Context, oauthLoginUser) (string, error) {
	return "", nil
}

func TestOAuthProviderAuthorizationConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		config        map[string]string
		wantAuthURL   string
		wantTokenURL  string
		wantRedirect  string
		wantScope     string
		wantPrompt    string
		wantAccess    string
		wantConfigErr bool
	}{
		{name: "google offline consent and verified profile scopes", provider: "google", config: map[string]string{"GOOGLE_CLIENT_ID": "client", "GOOGLE_CLIENT_SECRET": "secret"}, wantAuthURL: "https://accounts.google.com/o/oauth2/v2/auth", wantTokenURL: "https://oauth2.googleapis.com/token", wantRedirect: "https://signin.example/auth/google/callback/", wantScope: "https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile", wantPrompt: "consent", wantAccess: "offline"},
		{name: "github adds organization scope only when configured", provider: "github", config: map[string]string{"GITHUB_CLIENT_ID": "client", "GITHUB_CLIENT_SECRET": "secret", "GITHUB_ORGANIZATION_ID": "plane"}, wantAuthURL: "https://github.com/login/oauth/authorize", wantTokenURL: "https://github.com/login/oauth/access_token", wantRedirect: "https://signin.example/auth/github/callback/", wantScope: "read:user user:email read:org"},
		{name: "gitlab configured host", provider: "gitlab", config: map[string]string{"GITLAB_CLIENT_ID": "client", "GITLAB_CLIENT_SECRET": "secret", "GITLAB_HOST": "https://gitlab.example/"}, wantAuthURL: "https://gitlab.example/oauth/authorize", wantTokenURL: "https://gitlab.example/oauth/token", wantRedirect: "https://signin.example/auth/gitlab/callback/", wantScope: "read_user"},
		{name: "gitea configured host", provider: "gitea", config: map[string]string{"GITEA_CLIENT_ID": "client", "GITEA_CLIENT_SECRET": "secret", "GITEA_HOST": "https://git.example/"}, wantAuthURL: "https://git.example/login/oauth/authorize", wantTokenURL: "https://git.example/login/oauth/access_token", wantRedirect: "https://signin.example/auth/gitea/callback/", wantScope: "openid email profile read:user"},
		{name: "gitea rejects a non-http scheme", provider: "gitea", config: map[string]string{"GITEA_CLIENT_ID": "client", "GITEA_CLIENT_SECRET": "secret", "GITEA_HOST": "file:///tmp/provider"}, wantConfigErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &oauthRedirectTestRepository{config: test.config}
			service := &oauthRedirectService{repository: repository}
			ctx := oauthRequestContext(t, "https://signin.example/auth/"+test.provider+"/")
			config, err := service.providerConfig(ctx, test.provider)
			if test.wantConfigErr {
				if err == nil {
					t.Fatal("providerConfig() unexpectedly accepted invalid host")
				}
				return
			}
			if err != nil {
				t.Fatalf("providerConfig() error = %v", err)
			}
			if config.Endpoint.AuthURL != test.wantAuthURL || config.Endpoint.TokenURL != test.wantTokenURL || config.RedirectURL != test.wantRedirect {
				t.Fatalf("provider endpoints/redirect = %q %q %q", config.Endpoint.AuthURL, config.Endpoint.TokenURL, config.RedirectURL)
			}
			if got := strings.Join(config.Scopes, " "); got != test.wantScope {
				t.Fatalf("scopes = %q, want %q", got, test.wantScope)
			}
			query, err := url.ParseQuery(oauthSourceOrderAuthURL(config.AuthCodeURL("fixed-state", config.options...), test.provider)[strings.Index(oauthSourceOrderAuthURL(config.AuthCodeURL("fixed-state", config.options...), test.provider), "?")+1:])
			if err != nil {
				t.Fatal(err)
			}
			if query.Get("state") != "fixed-state" || query.Get("redirect_uri") != test.wantRedirect || query.Get("scope") != test.wantScope {
				t.Fatalf("authorization query = %#v", query)
			}
			if test.wantPrompt != "" && query.Get("prompt") != test.wantPrompt {
				t.Fatalf("prompt = %q", query.Get("prompt"))
			}
			if test.wantAccess != "" && query.Get("access_type") != test.wantAccess {
				t.Fatalf("access_type = %q", query.Get("access_type"))
			}
		})
	}
}

func TestOAuthNextPathAndRedirectConstruction(t *testing.T) {
	for input, want := range map[string]string{
		"/workspace/home":                   "/workspace/home",
		"https://outside.test/x":            "/x",
		"javascript:alert(1)":               "",
		"/a/../b":                           "",
		"//outside.test/redirect":           "/redirect",
		"/long/" + strings.Repeat("x", 501): "",
	} {
		if got := validateOAuthNextPath(input); got != want {
			t.Errorf("validateOAuthNextPath(%q) = %q, want %q", input, got, want)
		}
	}
	repository := &oauthRedirectTestRepository{config: map[string]string{"APP_BASE_URL": "https://app.example", "WEB_URL": "https://api.example"}}
	service := &oauthRedirectService{repository: repository}
	got := service.safeOAuthRedirect(context.Background(), "https://app.example", "/workspace/home", map[string]string{"error_code": "5000", "error_message": "INSTANCE_NOT_CONFIGURED"})
	want := "https://app.example/?next_path=%2Fworkspace%2Fhome&error_code=5000&error_message=INSTANCE_NOT_CONFIGURED"
	if got != want {
		t.Fatalf("safeOAuthRedirect() = %q, want %q", got, want)
	}
}

func TestOAuthResponseUsesDjangoRedirectContentType(t *testing.T) {
	response, _ := oauthResponse("https://app.example.test/")
	if response.ContentType != "text/html; charset=utf-8" {
		t.Fatalf("redirect content type = %q", response.ContentType)
	}
}

func TestOAuthGiteaAppCallbackWithoutInitiationSessionUsesRelativeErrorRedirect(t *testing.T) {
	repository := &oauthRedirectTestRepository{config: map[string]string{
		"APP_BASE_URL": "https://app.example.test",
		"WEB_URL":      "https://api.example.test",
	}}
	service := &oauthRedirectService{repository: repository}
	ctx := oauthRequestContext(t, "https://api.example.test/auth/gitea/callback/?code=fixture-code&state=invalid-fixture-state")
	response, body, err := service.callback(ctx, "gitea", false, "fixture-code", "invalid-fixture-state")
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, body)
	want := "?error_code=5123&error_message=GITEA_OAUTH_PROVIDER_ERROR"
	if response.Location != want {
		t.Fatalf("application Gitea callback Location=%q, want %q", response.Location, want)
	}
}

func oauthRequestContext(t *testing.T, rawURL string) context.Context {
	t.Helper()
	policy, err := loomhttp.NewRequestMetadataPolicy(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var captured context.Context
	middleware := loomhttp.RequestMetadataMiddleware(policy)(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		captured = request.Context()
	}))
	middleware.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, rawURL, nil))
	return captured
}
