package workspacewebhooks

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

func TestValidateURLSchemaDomainAndTrustedHost(t *testing.T) {
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "test-webhook.invalid")
	t.Setenv("WEBHOOK_DISALLOWED_DOMAINS", "blocked.example")
	for _, test := range []struct {
		name, raw, wantMessage string
		wantErr                bool
	}{
		{name: "invalid URL", raw: "not a URL", wantMessage: "Enter a valid URL.", wantErr: true},
		{name: "unsupported schema", raw: "ftp://example.com/hook", wantMessage: "Invalid schema. Only HTTP and HTTPS are allowed.", wantErr: true},
		{name: "localhost", raw: "http://localhost/hook", wantMessage: "Local URLs are not allowed.", wantErr: true},
		{name: "disallowed domain", raw: "https://child.blocked.example/hook", wantMessage: "URL domain or its subdomain is not allowed.", wantErr: true},
		{name: "trusted host bypasses DNS", raw: "https://test-webhook.invalid/hook", wantErr: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			message, err := (URLValidator{Resolver: fixedResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}}).Validate(context.Background(), test.raw, "")
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateURL(%q) err=%v, wantErr=%v", test.raw, err, test.wantErr)
			}
			if message != test.wantMessage {
				t.Fatalf("ValidateURL(%q) message=%q, want %q", test.raw, message, test.wantMessage)
			}
		})
	}
}

func TestValidateURLRejectsPrivateResolvedAddress(t *testing.T) {
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "")
	message, err := ValidateURL(context.Background(), "http://127.0.0.2/hook", "")
	if err == nil || message != "Invalid or disallowed webhook URL." {
		t.Fatalf("ValidateURL private IP = (%q, %v), want source SSRF rejection", message, err)
	}
}

type fixedResolver struct {
	addresses []netip.Addr
	err       error
}

func (r fixedResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return r.addresses, r.err
}

func TestURLValidatorResolutionAndRequestHostPolicy(t *testing.T) {
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "")
	t.Setenv("WEBHOOK_DISALLOWED_DOMAINS", "blocked.example")
	for _, test := range []struct {
		name, url, requestHost, allowedIPs, message string
		addresses                                   []string
		resolveErr                                  error
	}{
		{name: "mixed public and private DNS", url: "https://external.example/hook", addresses: []string{"8.8.8.8", "10.0.0.1"}, message: "Invalid or disallowed webhook URL."},
		{name: "mapped public IPv4 remains blocked", url: "https://[::ffff:8.8.8.8]/hook", message: "Invalid or disallowed webhook URL."},
		{name: "IPv4 allowlist cannot allow mapped IPv6", url: "https://[::ffff:8.8.8.8]/hook", allowedIPs: "8.8.8.8", message: "Invalid or disallowed webhook URL."},
		{name: "operator allows internal network", url: "https://external.example/hook", addresses: []string{"10.0.0.1"}, allowedIPs: "10.0.0.0/8"},
		{name: "DNS failure before domain restriction", url: "https://blocked.example/hook", resolveErr: errors.New("NXDOMAIN"), message: "Invalid or disallowed webhook URL."},
		{name: "request host domain", url: "https://child.plane.example/hook", requestHost: "PLANE.EXAMPLE.:8000", addresses: []string{"8.8.8.8"}, message: "URL domain or its subdomain is not allowed."},
		{name: "domain suffix boundary", url: "https://notplane.example/hook", requestHost: "plane.example", addresses: []string{"8.8.8.8"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("WEBHOOK_ALLOWED_IPS", test.allowedIPs)
			resolver := fixedResolver{err: test.resolveErr}
			for _, value := range test.addresses {
				resolver.addresses = append(resolver.addresses, netip.MustParseAddr(value))
			}
			message, err := (URLValidator{Resolver: resolver}).Validate(context.Background(), test.url, test.requestHost)
			if message != test.message || (err != nil) != (test.message != "") {
				t.Fatalf("Validate = (%q, %v), want %q", message, err, test.message)
			}
		})
	}
}
