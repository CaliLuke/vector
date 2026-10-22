package workspacewebhooks

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
)

type IPResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type URLValidator struct {
	Resolver IPResolver
	// PublicOnly disables webhook-specific allowlists for generic link previews.
	PublicOnly bool
}

func ValidateURL(ctx context.Context, raw, requestHost string) (string, error) {
	return (URLValidator{Resolver: net.DefaultResolver}).Validate(ctx, raw, requestHost)
}

func (validator URLValidator) Validate(ctx context.Context, raw, requestHost string) (string, error) {
	if raw == "" || utf8.RuneCountInString(raw) > 1024 || !djangoinput.ValidURL(raw) {
		return "Enter a valid URL.", fmt.Errorf("invalid URL syntax")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return "Enter a valid URL.", fmt.Errorf("invalid URL syntax")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "Invalid schema. Only HTTP and HTTPS are allowed.", fmt.Errorf("invalid URL schema")
	}
	host := strings.TrimRight(strings.ToLower(parsed.Hostname()), ".")
	if parsed.User == nil && (parsed.Host == "localhost" || parsed.Host == "127.0.0.1") {
		return "Local URLs are not allowed.", fmt.Errorf("localhost is not allowed")
	}
	allowedHosts := splitEnv("WEBHOOK_ALLOWED_HOSTS")
	if validator.PublicOnly {
		allowedHosts = nil
	}
	for _, allowed := range allowedHosts {
		if host == strings.TrimRight(strings.ToLower(allowed), ".") {
			return "", nil
		}
	}
	var ips []netip.Addr
	if literal, parseErr := netip.ParseAddr(host); parseErr == nil {
		// Preserve IPv4-mapped IPv6 identity: the source blocks transition
		// addresses even when their embedded IPv4 address is publicly routed.
		ips = []netip.Addr{literal.WithZone("")}
	} else {
		resolver := validator.Resolver
		if resolver == nil {
			resolver = net.DefaultResolver
		}
		ips, err = resolver.LookupNetIP(ctx, "ip", host)
	}
	if err != nil || len(ips) == 0 {
		if err == nil {
			err = fmt.Errorf("no IP addresses found")
		}
		return "Invalid or disallowed webhook URL.", fmt.Errorf("resolve webhook host: %w", err)
	}
	allowedNets := parseAllowedNetworks(os.Getenv("WEBHOOK_ALLOWED_IPS"))
	if validator.PublicOnly {
		allowedNets = nil
	}
	for _, addr := range ips {
		if !addr.IsValid() {
			return "Invalid or disallowed webhook URL.", fmt.Errorf("parse resolved webhook IP")
		}
		addr = addr.WithZone("")
		allowed := false
		for _, prefix := range allowedNets {
			if prefix.Contains(addr) {
				allowed = true
				break
			}
		}
		if !allowed && blockedIP(addr) {
			return "Invalid or disallowed webhook URL.", fmt.Errorf("webhook host resolves to a private or reserved IP")
		}
	}
	disallowed := splitEnv("WEBHOOK_DISALLOWED_DOMAINS")
	if requestHost != "" {
		// Django's source uses get_host().split(":")[0] here.
		requestDomain, _, _ := strings.Cut(requestHost, ":")
		disallowed = append(disallowed, requestDomain)
	}
	for _, domain := range disallowed {
		domain = strings.TrimRight(strings.ToLower(domain), ".")
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return "URL domain or its subdomain is not allowed.", fmt.Errorf("disallowed URL domain")
		}
	}
	return "", nil
}

func splitEnv(name string) []string {
	var out []string
	for _, value := range strings.Split(os.Getenv(name), ",") {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func parseAllowedNetworks(raw string) []netip.Prefix {
	var out []netip.Prefix
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if addr, err := netip.ParseAddr(value); err == nil {
			out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		if prefix, err := netip.ParsePrefix(value); err == nil {
			out = append(out, prefix.Masked())
		}
	}
	return out
}
