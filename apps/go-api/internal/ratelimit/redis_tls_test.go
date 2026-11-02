package ratelimit

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func redisOptionsForTest(t *testing.T, address string) (*redis.Options, error) {
	t.Helper()
	store, err := NewRedisStore(address)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	return store.client.Options(), nil
}
func TestRedisThrottleOptionsVerifyTLSPeersByDefault(t *testing.T) {
	options, err := redisOptionsForTest(t, "rediss://cache.example.test:6380/1")
	if err != nil {
		t.Fatal(err)
	}
	if options.TLSConfig == nil || options.TLSConfig.InsecureSkipVerify || options.TLSConfig.ServerName != "cache.example.test" {
		t.Fatalf("TLSConfig=%+v", options.TLSConfig)
	}
	plainOptions, err := redisOptionsForTest(t, "redis://cache.example.test:6379/1")
	if err != nil {
		t.Fatal(err)
	}
	if plainOptions.TLSConfig != nil {
		t.Fatalf("plain Redis TLSConfig=%+v", plainOptions.TLSConfig)
	}
}

func TestRedisThrottleTLSRejectsUntrustedCertificates(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	address := server.Listener.Addr().String()
	options, err := redisOptionsForTest(t, "rediss://"+address+"/1")
	if err != nil {
		t.Fatal(err)
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	connection, err := tls.DialWithDialer(dialer, "tcp", address, options.TLSConfig)
	if err == nil {
		if closeErr := connection.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("Redis TLS accepted a certificate outside the trust store")
	}
	var untrusted x509.UnknownAuthorityError
	if !errors.As(err, &untrusted) {
		t.Fatalf("TLS error = %v, want an untrusted-certificate failure", err)
	}
	// A deployment can trust its private CA without disabling peer verification.
	trusted := options.TLSConfig.Clone()
	trusted.RootCAs = x509.NewCertPool()
	trusted.RootCAs.AddCert(server.Certificate())
	connection, err = tls.DialWithDialer(dialer, "tcp", address, trusted)
	if err != nil {
		t.Fatalf("TLS handshake with a trusted certificate: %v", err)
	}
	if closeErr := connection.Close(); closeErr != nil {
		t.Error(closeErr)
	}
}
