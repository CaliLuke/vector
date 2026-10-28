package notionsync

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNotionPageRejectsTruncatedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "2")
		if _, err := w.Write([]byte("{")); err != nil {
			t.Errorf("write truncated response: %v", err)
		}
	}))
	defer server.Close()
	notion, err := NewNotion("test-token", false)
	if err != nil {
		t.Fatal(err)
	}
	notion.api.base = server.URL

	_, err = notion.Page(context.Background(), rootID)

	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("truncated page response error = %v, want unexpected EOF", err)
	}
}

func TestTransientReadRetriesButUncertainWriteDoesNot(t *testing.T) {
	reads, writes := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writes++
			w.WriteHeader(503)
			return
		}
		reads++
		if reads == 1 {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write([]byte(`{"id":"recovered"}`))
	}))
	defer server.Close()
	client := newAPI(server.URL, nil, 0)
	var row Object
	if err := client.request(context.Background(), http.MethodGet, "/", nil, &row); err != nil || row["id"] != "recovered" || reads != 2 {
		t.Fatalf("read retry: %v %v", row, err)
	}
	if err := client.request(context.Background(), http.MethodPost, "/", map[string]any{"name": "example"}, nil); err == nil || writes != 1 {
		t.Fatal("uncertain write replayed")
	}
}

func TestCancellationStopsRateLimitWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := newAPI("http://localhost", nil, time.Hour)
	client.next = time.Now().Add(time.Hour)
	if err := client.request(ctx, http.MethodGet, "/", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestVectorRequestPacingHonorsCancellation(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	vector, err := NewVector(server.URL, "workspace", "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if err = vector.SetRequestInterval(-time.Second); err == nil {
		t.Fatal("negative pacing accepted")
	}
	if err = vector.SetRequestInterval(time.Hour); err != nil {
		t.Fatal(err)
	}
	if err = vector.api.request(context.Background(), http.MethodGet, "/", nil, nil); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err = vector.api.request(ctx, http.MethodGet, "/", nil, nil); !errors.Is(err, context.DeadlineExceeded) || requests != 1 {
		t.Fatalf("pacing did not stop request: requests=%d, err=%v", requests, err)
	}
}

func TestRedirectNeverReceivesCredentials(t *testing.T) {
	received := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { received = true }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	client := newAPI(origin.URL, http.Header{"Authorization": {"Bearer private-test-token"}}, 0)
	if err := client.request(context.Background(), http.MethodGet, "/", nil, nil); err == nil || received {
		t.Fatal("redirect followed with credentials")
	}
}
