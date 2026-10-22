package celery

import (
	"context"
	"encoding/json"
	"fmt"
)

// Task publishes a protocol-v2 message through the same confirmed, durable
// channel used for API audit and soft deletion. Callers supply server-owned task
// names and match the existing worker's argument contract.
func (publisher *Publisher) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	if name == "" {
		return fmt.Errorf("Celery task name is required")
	}
	if args == nil {
		args = []any{}
	}
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	body, err := json.Marshal([]any{args, kwargs, map[string]any{
		"callbacks": nil, "errbacks": nil, "chain": nil, "chord": nil,
	}})
	if err != nil {
		return fmt.Errorf("encode Celery task: %w", err)
	}
	// Argument representations are diagnostics, not worker input. Never put
	// arbitrary submitted issue content into broker headers.
	return publisher.publish(ctx, taskMessage(name, body, "(<task arguments>)", "{<task keyword arguments>}"))
}
