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
	"time"

	"github.com/stretchr/testify/require"
)

// Public DNS answers are connected to a local fixture only after the production
// transport has validated and pinned the destination. No public network is used.
func publicFileFixture(t *testing.T, handler http.HandlerFunc) (*previewResolver, func(context.Context, string, string) (net.Conn, error)) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	resolver := &previewResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}}
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "8.8.8.8:80" {
			return nil, fmt.Errorf("unvalidated destination: %s", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, strings.TrimPrefix(server.URL, "http://"))
	}
	return resolver, dial
}

func TestPublicFileRedirectsPinEachRequestAndPreserveContent(t *testing.T) {
	var hits atomic.Int32
	resolver, dial := publicFileFixture(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Host != "files.example.test" || r.Method != http.MethodGet || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected request: %s %s authorization=%t", r.Method, r.Host, r.Header.Get("Authorization") != "")
		}
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/file", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, err := io.WriteString(w, "hello")
		if err != nil {
			t.Errorf("write file: %v", err)
		}
	})
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	body, mime, err := fetchPublicFile(t.Context(), "http://files.example.test/start", 5, resolver, dial)
	require.NoError(t, err)
	require.Equal(t, "hello", string(body))
	require.Equal(t, "text/plain; charset=utf-8", mime)
	require.EqualValues(t, 2, hits.Load())
	require.EqualValues(t, 2, resolver.calls.Load())
}

func TestPublicFileRejectsUnsafeTargetsBeforeDial(t *testing.T) {
	t.Setenv("WEBHOOK_ALLOWED_HOSTS", "files.example.test,127.0.0.1")
	t.Setenv("WEBHOOK_ALLOWED_IPS", "0.0.0.0/0,::/0")
	for _, raw := range []string{
		"http://127.0.0.1/private", "http://169.254.169.254/latest/meta-data/", "http://[::1]/", "file:///etc/passwd",
		"http://user:secret@files.example.test/file", "http://files.example.test/file#fragment", "http://%/", "http://files.example.test/file",
	} {
		t.Run(raw, func(t *testing.T) {
			resolver := &previewResolver{addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.1")}}
			var dials atomic.Int32
			body, mime, err := fetchPublicFile(t.Context(), raw, 5, resolver, func(context.Context, string, string) (net.Conn, error) {
				dials.Add(1)
				return nil, fmt.Errorf("unsafe target reached dial")
			})
			require.Error(t, err)
			require.Nil(t, body)
			require.Empty(t, mime)
			require.Zero(t, dials.Load())
		})
	}
}

func TestPublicFileRevalidatesRedirectsAndBoundsTheirCount(t *testing.T) {
	for _, tc := range []struct {
		name, location, want string
		hits                 int32
	}{
		{"private", "http://127.0.0.1/private", "could not be fetched", 1},
		{"credentials", "http://user:secret@files.example.test/file", "invalid public file URL", 1},
		{"fragment", "/file#fragment", "invalid public file URL", 1},
		{"missing location", "", "invalid file redirect", 1},
		{"cycle", "/start", "too many file redirects", 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int32
			resolver, dial := publicFileFixture(t, func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				if tc.location != "" {
					w.Header().Set("Location", tc.location)
				}
				w.WriteHeader(http.StatusFound)
			})
			body, mime, err := fetchPublicFile(t.Context(), "http://files.example.test/start", 5, resolver, dial)
			require.ErrorContains(t, err, tc.want)
			require.Nil(t, body)
			require.Empty(t, mime)
			require.Equal(t, tc.hits, hits.Load())
		})
	}
}

func TestPublicFileBoundsDeclaredAndStreamedBodies(t *testing.T) {
	for _, tc := range []struct {
		name, content, length, want string
		status                      int
		chunked                     bool
	}{
		{name: "exact declared", content: "hello", length: "5"},
		{name: "exact streamed", content: "hello", chunked: true},
		{name: "oversized declared", content: "hello!", length: "6", want: "exceeds upload limit"},
		{name: "oversized streamed", content: "hello!", chunked: true, want: "exceeds upload limit"},
		{name: "truncated", content: "hi", length: "5", want: "download was interrupted"},
		{name: "unsuccessful", status: http.StatusNotFound, want: "unsuccessful response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolver, dial := publicFileFixture(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if tc.length != "" {
					w.Header().Set("Content-Length", tc.length)
				}
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
				if tc.chunked {
					if err := http.NewResponseController(w).Flush(); err != nil {
						t.Errorf("flush: %v", err)
					}
				}
				if _, err := io.WriteString(w, tc.content); err != nil {
					t.Errorf("write: %v", err)
				}
			})
			body, mime, err := fetchPublicFile(t.Context(), "http://files.example.test/file", 5, resolver, dial)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
				require.Nil(t, body)
				require.Empty(t, mime)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.content, string(body))
			require.Equal(t, "text/plain", mime)
		})
	}
}

func TestPublicFileHonorsCancellation(t *testing.T) {
	started := make(chan struct{})
	resolver, dial := publicFileFixture(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, _, err := fetchPublicFile(ctx, "http://files.example.test/file", 5, resolver, dial)
		done <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("download did not start before deadline")
	}
	cancel()
	require.ErrorContains(t, <-done, "could not be fetched")
}

type publicFileRebindingResolver struct{ calls atomic.Int32 }

func (r *publicFileRebindingResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	if r.calls.Add(1) == 1 {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	return []netip.Addr{netip.MustParseAddr("10.0.0.1")}, nil
}

func TestPublicFileRejectsRebindingBetweenRedirects(t *testing.T) {
	var hits atomic.Int32
	_, dial := publicFileFixture(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, "/file", http.StatusFound)
	})
	resolver := &publicFileRebindingResolver{}
	body, mime, err := fetchPublicFile(t.Context(), "http://files.example.test/start", 5, resolver, dial)
	require.ErrorContains(t, err, "could not be fetched")
	require.Nil(t, body)
	require.Empty(t, mime)
	require.EqualValues(t, 1, hits.Load())
	require.EqualValues(t, 2, resolver.calls.Load())
}
