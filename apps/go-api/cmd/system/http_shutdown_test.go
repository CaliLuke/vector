package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPShutdownDeadlineForcesInFlightRequestClosedAndRetainsCause(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	started, stopped := make(chan struct{}), make(chan struct{})
	handler := http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
		close(stopped)
	})
	serverErr := make(chan error, 1)
	go func() { serverErr <- serveHTTP(ctx, listener, handler, 10*time.Millisecond) }()
	clientErr := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		response, requestErr := client.Get("http://" + listener.Addr().String())
		if response != nil {
			requestErr = errors.Join(requestErr, response.Body.Close())
		}
		clientErr <- requestErr
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not reach the server")
	}
	stop()
	select {
	case err := <-serverErr:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not finish forced shutdown")
	}
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("forced shutdown left the request handler running")
	}
	select {
	case err := <-clientErr:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("forced shutdown left the client waiting")
	}
}
