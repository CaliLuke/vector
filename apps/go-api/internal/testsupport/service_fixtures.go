package testsupport

import (
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartRedis owns an isolated Redis-compatible server and its client.
func StartRedis(t testing.TB) *redis.Client {
	t.Helper()
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "valkey/valkey:9.1.2-alpine", ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForListeningPort("6379/tcp"),
		}, Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, testcontainers.TerminateContainer(container)) })
	host, err := container.Host(t.Context())
	require.NoError(t, err)
	port, err := container.MappedPort(t.Context(), "6379/tcp")
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: net.JoinHostPort(host, port.Port())})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	require.NoError(t, client.Ping(t.Context()).Err())
	return client
}

// RustFSFixture exposes only the connection settings of an owned object store.
type RustFSFixture struct {
	Endpoint, AccessKey, SecretKey string
}

// StartRustFS uses the deployment's pinned S3 implementation in a disposable container.
func StartRustFS(t testing.TB) RustFSFixture {
	t.Helper()
	fixture := RustFSFixture{AccessKey: "integration-access", SecretKey: uuid.NewString()}
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "rustfs/rustfs:1.0.3@sha256:9ca9559533b537d10482fe923288b676ad08f92a7069d9a17327fbae3cab688b",
			ExposedPorts: []string{"9000/tcp"}, Cmd: []string{"/data"},
			Env: map[string]string{
				"RUSTFS_ACCESS_KEY": fixture.AccessKey, "RUSTFS_SECRET_KEY": fixture.SecretKey,
				"RUSTFS_ADDRESS": ":9000", "RUSTFS_CONSOLE_ENABLE": "false",
				"RUSTFS_CORS_ALLOWED_ORIGINS": "http://localhost:8000",
			},
			WaitingFor: wait.ForHTTP("/health/ready").WithPort("9000/tcp").WithStartupTimeout(90 * time.Second),
		}, Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, testcontainers.TerminateContainer(container)) })
	fixture.Endpoint, err = container.PortEndpoint(t.Context(), "9000/tcp", "http")
	require.NoError(t, err)
	return fixture
}

// ConfigureAssets directs environment-based production entry points to this store.
func (fixture RustFSFixture) ConfigureAssets(t *testing.T, bucket string) {
	t.Helper()
	for name, value := range map[string]string{
		"PLANE_ASSETS_S3_ACCESS_KEY_ID": fixture.AccessKey, "PLANE_ASSETS_S3_SECRET_ACCESS_KEY": fixture.SecretKey,
		"PLANE_ASSETS_S3_INTERNAL_ENDPOINT": fixture.Endpoint, "PLANE_ASSETS_S3_PUBLIC_ENDPOINT": fixture.Endpoint,
		"PLANE_ASSETS_S3_BUCKET": bucket, "PLANE_ASSETS_S3_REGION": "us-east-1", "PLANE_ASSETS_S3_PATH_STYLE": "true",
	} {
		t.Setenv(name, value)
	}
}
