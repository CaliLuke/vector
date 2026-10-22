package celery

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"strings"
)

// SoftDelete queues Django's existing recursive soft-deletion task. Publication
// follows the model save, matching Django's non-transactional request behavior.
func (publisher *Publisher) SoftDelete(ctx context.Context, app, model, id string) error {
	message, err := softDeleteMessage(app, model, id)
	if err != nil {
		return err
	}
	return publisher.publish(ctx, message)
}

func softDeleteMessage(app, model, id string) (amqp.Publishing, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return amqp.Publishing{}, err
	}
	body, err := json.Marshal([]any{
		[]any{app, model, map[string]any{"__type__": "uuid", "__value__": map[string]string{"hex": strings.ReplaceAll(parsed.String(), "-", "")}}},
		map[string]any{"using": nil},
		map[string]any{"callbacks": nil, "errbacks": nil, "chain": nil, "chord": nil},
	})
	if err != nil {
		return amqp.Publishing{}, err
	}
	return taskMessage("plane.bgtasks.deletion_task.soft_delete_related_objects", body, fmt.Sprintf("(%q, %q, UUID(%q))", app, model, id), "{'using': None}"), nil
}
