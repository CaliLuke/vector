package backgroundtasks

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/CaliLuke/go-plane/internal/workspacewebhooks"
)

// All network I/O remains standard net/http. The narrow hook pins sockets to
// validated addresses and retains the original Host, TLS SNI and certificate checks.
func pinnedTaskRequest(ctx context.Context, method, rawURL string, headers http.Header, body []byte, timeout time.Duration, publicOnly bool, upstream workspacewebhooks.IPResolver, dial func(context.Context, string, string) (net.Conn, error)) (*http.Response, error) {
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, &webhookURLPolicyError{message: "invalid URL"}
	}
	resolver := &webhookPinnedResolver{resolved: make(map[string][]netip.Addr), upstream: upstream}
	if _, validationErr := (workspacewebhooks.URLValidator{Resolver: resolver, PublicOnly: publicOnly}).Validate(ctx, rawURL, ""); validationErr != nil {
		return nil, &webhookURLPolicyError{message: validationErr.Error()}
	}
	ips, err := resolver.lookup(ctx, parsed.Hostname())
	if err != nil {
		return nil, &webhookURLPolicyError{message: err.Error()}
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	transport := &http.Transport{Proxy: nil, TLSClientConfig: nil, DisableKeepAlives: true}
	transport.DialContext = func(dialCtx context.Context, network, address string) (net.Conn, error) {
		var last error
		for _, ip := range ips {
			conn, dialErr := dial(dialCtx, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			last = dialErr
		}
		return nil, last
	}
	client := &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header = headers.Clone()
	return client.Do(request)
}

type webhookPinnedResolver struct {
	upstream workspacewebhooks.IPResolver
	mu       sync.Mutex
	resolved map[string][]netip.Addr
}

func (resolver *webhookPinnedResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return resolver.lookup(ctx, host)
}

func (resolver *webhookPinnedResolver) lookup(ctx context.Context, host string) ([]netip.Addr, error) {
	key := strings.TrimRight(strings.ToLower(host), ".")
	resolver.mu.Lock()
	if addresses, ok := resolver.resolved[key]; ok {
		resolver.mu.Unlock()
		return addresses, nil
	}
	resolver.mu.Unlock()
	source := resolver.upstream
	if source == nil {
		source = net.DefaultResolver
	}
	addresses, err := source.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	resolver.mu.Lock()
	resolver.resolved[key] = addresses
	resolver.mu.Unlock()
	return addresses, nil
}
