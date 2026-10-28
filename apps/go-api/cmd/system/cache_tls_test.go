package main

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/projectlabelreads"
	"github.com/CaliLuke/go-plane/internal/projectmembershipreads"
	"github.com/CaliLuke/go-plane/internal/projectstatereads"
	"github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/stretchr/testify/require"
)

func TestApplicationRedisCachesRejectUntrustedTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	address := "rediss://" + server.Listener.Addr().String() + "/1"
	for _, test := range []struct {
		name string
		open func(*testing.T, string) func(context.Context) error
	}{
		{"assets", func(t *testing.T, address string) func(context.Context) error {
			cache, err := assets.NewRedisCacheInvalidator(address)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, cache.Close()) })
			return func(ctx context.Context) error { return cache.Delete(ctx, "tls-proof") }
		}},
		{"labels", func(t *testing.T, address string) func(context.Context) error {
			cache, err := projectlabelreads.NewWriteCache(address)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, cache.Close()) })
			return func(ctx context.Context) error { return cache.Invalidate(ctx, "tls-proof", false) }
		}},
		{"invitations", func(t *testing.T, address string) func(context.Context) error {
			cache, err := projectmembershipreads.NewInvitationCache(address)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, cache.Close()) })
			return func(ctx context.Context) error { return cache.Invalidate(ctx, "tls-proof", "", false) }
		}},
		{"states", func(t *testing.T, address string) func(context.Context) error {
			cache, err := projectstatereads.NewStateWriteCache(address)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, cache.Close()) })
			return func(ctx context.Context) error { return cache.Invalidate(ctx, "tls-proof") }
		}},
		{"anonymous throttle", func(t *testing.T, address string) func(context.Context) error {
			cache, err := publicprojectmetadatareads.NewRedisThrottleCache(address)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, cache.Close()) })
			return cache.Ping
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := test.open(t, address)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			err := request(ctx)
			var untrusted x509.UnknownAuthorityError
			if !errors.As(err, &untrusted) {
				t.Fatalf("cache error = %v, want untrusted-certificate rejection before a Redis command", err)
			}
		})
	}
}
