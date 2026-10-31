//go:build integration

package mcpactions_test

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/require"
)

func configureCatalogueWorker(t *testing.T) {
	t.Helper()
	redis := testsupport.StartRedis(t)
	// These workflows use PostgreSQL and Redis. An unexpected storage request
	// must fail locally instead of inheriting a developer's S3 configuration.
	for key, value := range map[string]string{
		"PLANE_ASSETS_S3_BUCKET": "catalogue-worker-fixture", "PLANE_ASSETS_S3_REGION": "us-east-1",
		"PLANE_ASSETS_S3_ACCESS_KEY_ID": "fixture", "PLANE_ASSETS_S3_SECRET_ACCESS_KEY": "fixture-secret",
		"PLANE_ASSETS_S3_INTERNAL_ENDPOINT": "http://127.0.0.1:1", "PLANE_ASSETS_S3_PUBLIC_ENDPOINT": "http://127.0.0.1:1", "PLANE_ASSETS_S3_PATH_STYLE": "true",
		"SIGNED_URL_EXPIRATION": "3600", "AWS_CONFIG_FILE": "/dev/null", "AWS_SHARED_CREDENTIALS_FILE": "/dev/null", "AWS_PROFILE": "", "AWS_EC2_METADATA_DISABLED": "true",
		"SECRET_KEY": "catalogue-worker-fixture-secret", "REDIS_URL": "redis://" + redis.Options().Addr, "HARD_DELETE_AFTER_DAYS": "60", "WEB_URL": "https://app.example.test",
	} {
		t.Setenv(key, value)
	}
}

func startCatalogueWorker(t *testing.T, f milestoneFixture) *backgroundtasks.Runtime {
	t.Helper()
	runtime, err := backgroundtasks.NewRuntime(t.Context(), f.pool.Config().ConnString(), selvedge.WithActivityPollInterval(20*time.Millisecond), selvedge.WithWorkflowResumptionInterval(20*time.Millisecond))
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 20*time.Second)
		defer cancel()
		require.NoError(t, runtime.Shutdown(ctx))
	})
	require.NoError(t, runtime.Start(t.Context()))
	return runtime
}
