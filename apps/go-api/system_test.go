package systemapi

import (
	"context"
	"strings"
	"testing"

	csrftoken "github.com/CaliLuke/go-plane/gen/csrf_token"
	"github.com/CaliLuke/loom/clue/log"
)

func TestCSRFTokenUsesStableCookieSecretAndFreshMask(t *testing.T) {
	const secret = "abcdefghijklmnopqrstuvwxyzABCDEF"
	cookie := secret
	service := NewCSRFToken()
	first, err := service.CsrfToken(context.Background(), &csrftoken.CsrfTokenPayload{CsrfCookie: &cookie})
	if err != nil {
		t.Fatalf("first CsrfToken() error = %v", err)
	}
	second, err := service.CsrfToken(context.Background(), &csrftoken.CsrfTokenPayload{CsrfCookie: &cookie})
	if err != nil {
		t.Fatalf("second CsrfToken() error = %v", err)
	}
	if len(first.CsrfToken) != csrfTokenLength || len(second.CsrfToken) != csrfTokenLength {
		t.Fatalf("token lengths = %d, %d; want %d", len(first.CsrfToken), len(second.CsrfToken), csrfTokenLength)
	}
	if first.CsrfToken == second.CsrfToken {
		t.Fatal("successive responses reused the same CSRF mask")
	}
	if unmaskCSRFToken(first.CsrfToken) != secret || unmaskCSRFToken(second.CsrfToken) != secret {
		t.Fatal("masked tokens do not decode to the cookie secret")
	}
	if first.CsrfCookie != secret || second.CsrfCookie != secret {
		t.Fatalf("cookie secrets = %q, %q; want %q", first.CsrfCookie, second.CsrfCookie, secret)
	}
}

func TestNormalizeCSRFSecretHandlesDjangoCookieFormats(t *testing.T) {
	const secret = "abcdefghijklmnopqrstuvwxyzABCDEF"
	masked, err := maskCSRFSecret(secret)
	if err != nil {
		t.Fatalf("maskCSRFSecret() error = %v", err)
	}
	normalized, err := NormalizeCSRFSecret(masked)
	if err != nil {
		t.Fatalf("normalizeCSRFSecret(masked) error = %v", err)
	}
	if normalized != secret {
		t.Fatalf("masked cookie normalized to %q, want %q", normalized, secret)
	}

	for _, invalid := range []string{"", "short", strings.Repeat("!", csrfSecretLength)} {
		replacement, err := NormalizeCSRFSecret(invalid)
		if err != nil {
			t.Fatalf("normalizeCSRFSecret(%q) error = %v", invalid, err)
		}
		if !validCSRFValue(replacement, csrfSecretLength) {
			t.Fatalf("replacement for %q = %q, want a valid secret", invalid, replacement)
		}
	}
}

func TestValidCSRFTokenAcceptsDjangoCookieAndMaskedToken(t *testing.T) {
	const secret = "abcdefghijklmnopqrstuvwxyzABCDEF"
	token, err := maskCSRFSecret(secret)
	if err != nil {
		t.Fatal(err)
	}
	for name, submitted := range map[string]string{"plain secret": secret, "masked token": token} {
		t.Run(name, func(t *testing.T) {
			if !ValidCSRFToken(secret, submitted) {
				t.Fatal("matching Django CSRF value was rejected")
			}
		})
	}
	for name, values := range map[string][2]string{
		"mismatch":          {secret, strings.Repeat("b", csrfSecretLength)},
		"invalid cookie":    {"!", secret},
		"invalid submitted": {secret, "!"},
	} {
		t.Run(name, func(t *testing.T) {
			if ValidCSRFToken(values[0], values[1]) {
				t.Fatal("invalid or mismatched Django CSRF value was accepted")
			}
		})
	}
}

func TestSystemLive(t *testing.T) {
	ctx := log.Context(context.Background())
	result, err := NewSystem().Live(ctx)
	if err != nil {
		t.Fatalf("Live() error = %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("Live() status = %q, want %q", result.Status, "ok")
	}
}

func TestSystemReadyRequiresDatabase(t *testing.T) {
	ctx := log.Context(context.Background())
	if _, err := NewSystem().Ready(ctx); err == nil {
		t.Fatal("Ready() error = nil, want missing database error")
	}
}
