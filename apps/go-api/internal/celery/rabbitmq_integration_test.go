//go:build integration

package celery

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startDisposableRabbitMQ(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := testcontainers.Run(ctx, "rabbitmq:3.13.6-management-alpine",
		testcontainers.WithExposedPorts("5672/tcp"),
		// The broker shares a Docker VM with the other integration fixtures. Do
		// not create a busy-waiting Erlang scheduler for every visible host CPU.
		testcontainers.WithEnv(map[string]string{
			"RABBITMQ_SERVER_ADDITIONAL_ERL_ARGS": "+S 2:2 +sbwt none +sbwtdcpu none +sbwtdio none",
		}),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5672/tcp").WithStartupTimeout(time.Minute)),
	)
	// Run can return a container even when its readiness check fails.
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatal(err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5672/tcp")
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port())
}
