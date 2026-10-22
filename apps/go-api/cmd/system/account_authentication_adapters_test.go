package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/accountauthentication/gen/account_authentication"
	accountauthsvr "github.com/CaliLuke/go-plane/accountauthentication/gen/http/account_authentication/server"
	store "github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	"github.com/CaliLuke/go-plane/internal/authentication/password"
	"github.com/CaliLuke/go-plane/internal/authredirects"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/redis/go-redis/v9"
)

func TestDjangoPasswordResetTokenIssuerMatchesDjangoTokenFormat(t *testing.T) {
	// Django PasswordResetTokenGenerator in the pinned API image produced this
	// token for SECRET_KEY=django-secret, _now=2026-10-21 12:34:56, and the user
	// fields below. Its UID helper produced YWJj for the same primary key.
	issuer, err := newDjangoPasswordResetTokenIssuer("django-secret")
	if err != nil {
		t.Fatal(err)
	}
	issuer.now = func() time.Time { return time.Date(2026, 10, 21, 12, 34, 56, 0, time.UTC) }
	lastLogin := time.Date(2025, 1, 2, 3, 4, 5, 999_000_000, time.FixedZone("offset", 2*60*60))
	user := store.User{ID: "abc", Email: "a@example.com", Password: "pbkdf2_sha256$1$salt$hash", LastLogin: &lastLogin}
	uid, token, err := issuer.Issue(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	if uid != "YWJj" {
		t.Fatalf("uidb64 = %q, want YWJj", uid)
	}
	if token != "dgstm8-ae1467e6cad5dcde3f56602290912746" {
		t.Fatalf("token = %q, want Django-compatible fixture", token)
	}
	valid, err := issuer.Verify(context.Background(), user, uid, token)
	if err != nil || !valid {
		t.Fatalf("Django reset token did not verify: valid=%t err=%v", valid, err)
	}
	valid, err = issuer.Verify(context.Background(), user, uid, token+"0")
	if err != nil || valid {
		t.Fatalf("tampered reset token verified: valid=%t err=%v", valid, err)
	}
}

func TestRedisMagicCodeIssuerPayloadVerifiesAndConsumes(t *testing.T) {
	redisURL := os.Getenv("ACCOUNT_AUTHENTICATION_TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("set ACCOUNT_AUTHENTICATION_TEST_REDIS_URL to run the Redis issuer/consumer round trip")
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(options)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect test Redis: %v", err)
	}
	issuer, err := newRedisMagicCodeIssuer(client, func(context.Context, string) (bool, error) { return true, nil })
	if err != nil {
		t.Fatal(err)
	}
	email := "adapter-round-trip-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.invalid"
	key, code, err := issuer.Issue(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Del(context.Background(), key, key+":verify_attempts")
	encoded, err := client.Get(ctx, key).Bytes()
	if err != nil {
		t.Fatalf("read issued record: %v", err)
	}
	var record struct {
		CurrentAttempt int    `json:"current_attempt"`
		Email          string `json:"email"`
		Token          string `json:"token"`
	}
	if err := json.Unmarshal(encoded, &record); err != nil {
		t.Fatalf("decode issued record: %v", err)
	}
	if record.CurrentAttempt != 0 || record.Email != email || record.Token != code {
		t.Fatalf("issued Redis record = %#v", record)
	}
	verifier, err := authredirects.NewRedisMagicCodeVerifier(client)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.Verify(ctx, key, code, true); err != nil {
		t.Fatalf("consumer rejected issued code: %v", err)
	}
	if _, err := client.Get(ctx, key).Result(); !errors.Is(err, redis.Nil) {
		t.Fatalf("consumed code lookup error = %v, want redis.Nil", err)
	}
}

type accountAuthenticationHTTPRepository struct{}

func (accountAuthenticationHTTPRepository) InstanceConfigured(context.Context) (bool, error) {
	return true, nil
}
func (accountAuthenticationHTTPRepository) ConfigValue(_ context.Context, key, fallback string) (string, error) {
	if key == "EMAIL_HOST" {
		return "smtp.example", nil
	}
	if key == "ENABLE_MAGIC_LINK_LOGIN" {
		return "1", nil
	}
	return fallback, nil
}
func (accountAuthenticationHTTPRepository) UserByEmail(context.Context, string) (store.User, error) {
	return store.User{}, store.ErrUserNotFound
}
func (accountAuthenticationHTTPRepository) UserByID(context.Context, string) (store.User, error) {
	return store.User{}, store.ErrUserNotFound
}
func (accountAuthenticationHTTPRepository) UpdatePassword(context.Context, string, string, bool, time.Time) error {
	return nil
}
func (accountAuthenticationHTTPRepository) SetPassword(context.Context, string, string, bool) error {
	return nil
}
func (accountAuthenticationHTTPRepository) UpdateLoginMetadata(context.Context, string, time.Time, string, string, string, time.Time) (bool, error) {
	return false, nil
}
func (accountAuthenticationHTTPRepository) UserJSON(context.Context, string) (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

type accountAuthenticationHTTPThrottle struct{}

func (accountAuthenticationHTTPThrottle) AllowWithWait(context.Context, string) (bool, time.Duration, error) {
	return true, 0, nil
}

type accountAuthenticationHTTPCache struct{}

func (accountAuthenticationHTTPCache) Invalidate(context.Context, string, string, bool) error {
	return nil
}

type accountAuthenticationHTTPTasks struct{}

func (accountAuthenticationHTTPTasks) Task(context.Context, string, []any, map[string]any) error {
	return nil
}

type accountAuthenticationHTTPMagicIssuer struct{}

func (accountAuthenticationHTTPMagicIssuer) Issue(context.Context, string) (string, string, error) {
	return "", "", nil
}

type accountAuthenticationHTTPResetIssuer struct{}

func (accountAuthenticationHTTPResetIssuer) Issue(context.Context, store.User) (string, string, error) {
	return "", "", nil
}

func TestAccountAuthenticationGeneratedEmailCheckAcceptsRequest(t *testing.T) {
	service, err := systemapi.NewAccountAuthentication(
		accountAuthenticationHTTPRepository{}, accountAuthenticationHTTPThrottle{}, accountAuthenticationHTTPCache{},
		accountAuthenticationHTTPTasks{}, accountAuthenticationHTTPMagicIssuer{}, accountAuthenticationHTTPResetIssuer{},
		password.Default(), &djangosession.Validator{}, "https://app.example", "https://space.example",
	)
	if err != nil {
		t.Fatal(err)
	}
	mux := loomhttp.NewMuxer()
	endpoints := api.NewEndpoints(service)
	cors, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{Pattern: "http://localhost:3000"}}})
	if err != nil {
		t.Fatal(err)
	}
	server := accountauthsvr.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, cors)
	accountauthsvr.Mount(mux, server)
	request := httptest.NewRequest("POST", "/auth/email-check/", strings.NewReader(`{"email":"alice@example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("generated operation returned %d: %s", response.Code, response.Body.String())
	}
	if got := strings.TrimSpace(response.Body.String()); got != `{"existing":false,"status":"MAGIC_CODE"}` {
		t.Fatalf("generated operation response = %s", got)
	}
}

func TestAccountAuthenticationSessionMatcherCoversOptionalSessionAuthentication(t *testing.T) {
	for _, test := range []struct {
		path string
		want bool
	}{
		{path: "/auth/change-password/", want: true},
		{path: "/auth/set-password", want: true},
		{path: "/auth/email-check/", want: true},
		{path: "/auth/spaces/email-check/", want: true},
		{path: "/auth/forgot-password/", want: true},
		{path: "/auth/spaces/forgot-password/", want: true},
		{path: "/auth/magic-generate/", want: true},
		{path: "/auth/spaces/magic-generate/", want: true},
		{path: "/auth/unrelated/", want: false},
	} {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.path, nil)
			if got := accountAuthenticationSessionPath(request); got != test.want {
				t.Fatalf("accountAuthenticationSessionPath(%q) = %t, want %t", test.path, got, test.want)
			}
		})
	}
}
