package main

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestServerRejectsRuntimeWithoutTransactionalEffectConsumer(t *testing.T) {
	t.Setenv("SECRET_KEY", "test-secret")
	err := run(context.Background(), []string{"--database-url", "postgres://unused@localhost/unused", "--go-background-tasks=false"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "native background tasks are required") {
		t.Fatalf("server accepted stranded work item effects: %v", err)
	}
}
