//go:build integration

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func TestRunServesApplicationAndShutsDown(t *testing.T) {
	databaseURL := migratedDatabases.SeedDatabase(t, func(context.Context, string) error { return nil })
	redisClient := testsupport.StartRedis(t)
	for name, value := range map[string]string{
		"OTEL_EXPORTER_OTLP_ENDPOINT": "", "MCP_ENABLED": "false", "ENABLE_READ_REPLICA": "0", "SKIP_ENV_VAR": "false",
		"SECRET_KEY": "startup-fixture-secret", "REDIS_URL": "redis://" + redisClient.Options().Addr,
		"WEB_URL": "http://localhost:8000", "HARD_DELETE_AFTER_DAYS": "60",
		"PLANE_ASSETS_S3_BUCKET": "startup-fixture", "PLANE_ASSETS_S3_REGION": "us-east-1",
		"PLANE_ASSETS_S3_ACCESS_KEY_ID": "fixture", "PLANE_ASSETS_S3_SECRET_ACCESS_KEY": "fixture-secret",
		"PLANE_ASSETS_S3_INTERNAL_ENDPOINT": "http://127.0.0.1:1", "PLANE_ASSETS_S3_PUBLIC_ENDPOINT": "http://127.0.0.1:1",
		"PLANE_ASSETS_S3_PATH_STYLE": "true", "SIGNED_URL_EXPIRATION": "3600",
		"AWS_CONFIG_FILE": "/dev/null", "AWS_SHARED_CREDENTIALS_FILE": "/dev/null", "AWS_PROFILE": "", "AWS_EC2_METADATA_DISABLED": "true",
	} {
		t.Setenv(name, value)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	reader, writer := io.Pipe()
	address := make(chan string, 1)
	scanned := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			var entry struct{ Msg, Address string }
			if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil && entry.Msg == "HTTP server listening" {
				address <- entry.Address
			}
		}
		scanned <- scanner.Err()
	}()
	done := make(chan struct{})
	var runErr error
	go func() {
		runErr = errors.Join(run(ctx, []string{
			"--database-url", databaseURL, "--http-address", "127.0.0.1:0",
			"--go-background-tasks", "--go-background-tasks-no-workers", "--shutdown-timeout", "5s",
		}, writer), writer.Close())
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		defer func() { require.NoError(t, reader.Close()) }()
		select {
		case <-done:
			require.NoError(t, runErr)
		case <-time.After(10 * time.Second):
			t.Error("application did not shut down after cancellation")
			require.NoError(t, reader.Close())
		}
		require.NoError(t, <-scanned)
	})
	var baseURL string
	select {
	case bound := <-address:
		baseURL = "http://" + bound
	case <-done:
		t.Fatalf("application exited before listening: %v", runErr)
	case <-ctx.Done():
		t.Fatal("application did not start before deadline")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	for _, check := range []struct {
		path   string
		status int
	}{
		{"/health/live", http.StatusOK},
		{"/health/ready", http.StatusOK},
		{"/auth/get-csrf-token/", http.StatusOK},
		{"/api/users/me/", http.StatusUnauthorized},
	} {
		response, err := client.Get(baseURL + check.path)
		require.NoError(t, err, check.path)
		body, readErr := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		require.NoError(t, readErr)
		require.Equal(t, check.status, response.StatusCode, "%s: %s", check.path, body)
		if check.path == "/auth/get-csrf-token/" {
			require.NotEmpty(t, response.Cookies(), "assembled CSRF route must issue its cookie")
		}
	}
}
