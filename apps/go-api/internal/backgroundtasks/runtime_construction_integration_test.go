//go:build integration

package backgroundtasks

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configureRuntimeTestEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("PLANE_ASSETS_S3_BUCKET", "runtime-fixture")
	t.Setenv("PLANE_ASSETS_S3_REGION", "us-east-1")
	t.Setenv("PLANE_ASSETS_S3_ACCESS_KEY_ID", "fixture")
	t.Setenv("PLANE_ASSETS_S3_SECRET_ACCESS_KEY", "fixture-secret")
	t.Setenv("PLANE_ASSETS_S3_INTERNAL_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("PLANE_ASSETS_S3_PUBLIC_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("PLANE_ASSETS_S3_PATH_STYLE", "true")
	t.Setenv("SIGNED_URL_EXPIRATION", "3600")
	t.Setenv("AWS_CONFIG_FILE", "/dev/null")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/dev/null")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("SECRET_KEY", "runtime-fixture-secret")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:1")
	t.Setenv("HARD_DELETE_AFTER_DAYS", "60")
	t.Setenv("WEB_URL", "https://app.example.test")
}

func TestRuntimeConstructionTransfersResourcesUntilShutdown(t *testing.T) {
	configureRuntimeTestEnvironment(t)
	databaseURL, _, observer := openBackgroundTaskDatabase(t)
	_, err := observer.Exec(t.Context(), `CREATE TABLE runtime_fixture_probe(value text); INSERT INTO runtime_fixture_probe VALUES ('keep')`)
	require.NoError(t, err)
	runtime, err := NewRuntime(t.Context(), databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() {
		if runtime != nil {
			require.NoError(t, runtime.Shutdown(t.Context()))
		}
	})
	activityStore, ok := runtime.IssueActivities.store.(*PostgresIssueActivityStore)
	require.True(t, ok)
	emailOrigin, activityOrigin, database := runtime.emailStore.origin, activityStore.origin, runtime.pool
	require.NotNil(t, emailOrigin)
	require.NotNil(t, activityOrigin)
	require.NoError(t, database.Ping(t.Context()))
	var value string
	require.NoError(t, database.QueryRow(t.Context(), `SELECT value FROM runtime_fixture_probe`).Scan(&value))
	require.Equal(t, "keep", value)

	err = runtime.Shutdown(t.Context())
	runtime = nil
	require.NoError(t, err)
	require.ErrorIs(t, activityOrigin.Ping(t.Context()).Err(), redis.ErrClosed)
	require.ErrorIs(t, emailOrigin.Ping(t.Context()).Err(), redis.ErrClosed)
	require.Error(t, database.Ping(t.Context()))
	require.NoError(t, observer.QueryRow(t.Context(), `SELECT value FROM runtime_fixture_probe`).Scan(&value))
	require.Equal(t, "keep", value)
}

func TestRuntimeConstructionFailureReleasesDatabaseAtEachStage(t *testing.T) {
	configureRuntimeTestEnvironment(t)
	databaseURL, _, observer := openBackgroundTaskDatabase(t)
	parsed, err := url.Parse(databaseURL)
	require.NoError(t, err)
	query := parsed.Query()
	query.Set("application_name", "runtime-construction-failure")
	parsed.RawQuery = query.Encode()
	for _, test := range []struct{ name, variable, value, want string }{
		{"before stores", "PLANE_ASSETS_S3_PATH_STYLE", "invalid", "parse PLANE_ASSETS_S3_PATH_STYLE"},
		{"activity store", "REDIS_URL", ":invalid", "configure issue activity origin Redis"},
		{"after both stores", "HARD_DELETE_AFTER_DAYS", "0", "HARD_DELETE_AFTER_DAYS must be between"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.variable, test.value)
			runtime, err := NewRuntime(t.Context(), parsed.String())
			require.Nil(t, runtime)
			require.ErrorContains(t, err, test.want)
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				var connections int
				err := observer.QueryRow(t.Context(), `SELECT count(*) FROM pg_stat_activity WHERE application_name='runtime-construction-failure'`).Scan(&connections)
				assert.NoError(collect, err)
				assert.Zero(collect, connections)
			}, 5*time.Second, 20*time.Millisecond)
		})
	}
}

func TestRuntimeShutdownClosesDatabaseAfterStoreFailure(t *testing.T) {
	databaseURL, _, observer := openBackgroundTaskDatabase(t)
	database, err := pgxpool.New(t.Context(), databaseURL)
	require.NoError(t, err)
	t.Cleanup(database.Close)
	require.NoError(t, database.Ping(t.Context()))
	activityErr := errors.New("activity cleanup failed")
	runtime := &Runtime{
		pool:            database,
		IssueActivities: &IssueActivityDispatcher{store: failingRuntimeActivityStore{err: activityErr}},
	}
	require.ErrorIs(t, runtime.Shutdown(t.Context()), activityErr)
	require.Error(t, database.Ping(t.Context()))
	require.NoError(t, observer.Ping(t.Context()), "shutdown must only close its own pool")
}
