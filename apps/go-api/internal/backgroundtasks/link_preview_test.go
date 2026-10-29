package backgroundtasks

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
)

type previewResolver struct {
	calls     atomic.Int32
	addresses []netip.Addr
}

func (r *previewResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	r.calls.Add(1)
	return r.addresses, nil
}
func TestLinkPreviewPinnedRedirectCharsetAndFavicon(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Host != "preview.example.test" {
			t.Errorf("lost original Host: %s", r.Host)
		}
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/page", http.StatusFound)
		case "/page":
			w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
			io.WriteString(w, "<title> Caf\xe9 &amp; tea </title><link rel='apple-touch-icon' href='/other'><link rel='icon' href='fav.ico'>")
		case "/fav.ico":
			w.Header().Set("Content-Type", "image/png")
			io.WriteString(w, "icon")
		case "/private":
			http.Redirect(w, r, "http://127.0.0.1/private", http.StatusFound)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	resolver := &previewResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}
	store := &PostgresLinkPreviewStore{resolver: resolver, dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "8.8.8.8:80" {
			return nil, fmt.Errorf("unvalidated dial: %s", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
	}}
	got := store.preview(t.Context(), "http://preview.example.test/")
	if got.Title == nil || *got.Title != "Café & tea" || got.Favicon != "data:image/png;base64,aWNvbg==" || got.FaviconURL == nil || *got.FaviconURL != "http://preview.example.test/fav.ico" {
		t.Fatalf("bad preview %#v", got)
	}
	if hits.Load() != 3 || resolver.calls.Load() != 3 {
		t.Fatalf("expected one resolution per request: hits=%d DNS=%d", hits.Load(), resolver.calls.Load())
	}
	// A link preview must never inherit webhook-specific private host exceptions.
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "127.0.0.1,preview.example.test")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "0.0.0.0/0")
	got = store.preview(t.Context(), "http://preview.example.test/private")
	if got.Title != nil || got.FaviconURL != nil {
		t.Fatalf("private redirect accepted %#v", got)
	}
	// Any private answer poisons the complete DNS set, before opening a socket.
	resolver.addresses = append(resolver.addresses, netip.MustParseAddr("10.0.0.1"))
	before := hits.Load()
	got = store.preview(t.Context(), "http://preview.example.test/")
	if hits.Load() != before || got.Title != nil || got.FaviconURL != nil {
		t.Fatal("mixed public/private DNS reached network")
	}
}
func TestLinkPreviewBoundsResponseBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "0123456789") }))
	defer server.Close()
	store := &PostgresLinkPreviewStore{resolver: &previewResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}, dial: func(ctx context.Context, n, a string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, n, strings.TrimPrefix(server.URL, "http://"))
	}}
	if _, _, _, _, err := store.fetch(t.Context(), "GET", "http://preview.example.test/", 5); err == nil {
		t.Fatal("accepted oversized response")
	}
}
