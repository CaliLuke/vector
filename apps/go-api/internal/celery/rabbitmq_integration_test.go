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
		testcontainers.WithWaitStrategy(wait.ForLog("Server startup complete").WithStartupTimeout(time.Minute)),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if terminateErr := testcontainers.TerminateContainer(container); terminateErr != nil {
			t.Errorf("terminate RabbitMQ: %v", terminateErr)
		}
	})

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
