//go:build integration

package celery

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestPublisherTaskEmitsPersistentCeleryProtocolV2Message(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	brokerURL := startDisposableRabbitMQ(t, ctx)
	queue := "go-plane-task-test-" + uuid.NewString()
	consumer, err := amqp.Dial(brokerURL)
	if err != nil {
		t.Fatalf("connect consumer: %v", err)
	}
	channel, err := consumer.Channel()
	if err != nil {
		_ = consumer.Close()
		t.Fatalf("open consumer channel: %v", err)
	}
	if declareCeleryQueueErr := declareCeleryQueue(channel, queue); declareCeleryQueueErr != nil {
		_ = channel.Close()
		_ = consumer.Close()
		t.Fatal(declareCeleryQueueErr)
	}
	t.Cleanup(func() {
		if _, queueDeleteErr := channel.QueueDelete(queue, false, false, false); queueDeleteErr != nil {
			t.Errorf("delete test queue: %v", queueDeleteErr)
		}
		_ = channel.Close()
		_ = consumer.Close()
	})
	publisher, err := newPublisher(brokerURL, queue)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = publisher.Close() })

	secret := "user supplied sensitive argument"
	kwargs := map[string]any{"payload": map[string]any{"text": secret}, "enabled": true}
	if err := publisher.Task(ctx, "plane.workitems.process", []any{"item-17", 3}, kwargs); err != nil {
		t.Fatalf("Task() error = %v", err)
	}
	delivery := consumeDelivery(t, channel, queue)
	if delivery.Exchange != "" || delivery.RoutingKey != queue {
		t.Fatalf("route = exchange %q, key %q; want default exchange and %q", delivery.Exchange, delivery.RoutingKey, queue)
	}
	if delivery.DeliveryMode != amqp.Persistent {
		t.Fatalf("delivery mode = %d, want persistent", delivery.DeliveryMode)
	}
	if delivery.ContentType != "application/json" || delivery.ContentEncoding != "utf-8" {
		t.Fatalf("content = %q; encoding = %q", delivery.ContentType, delivery.ContentEncoding)
	}
	if delivery.Headers["task"] != "plane.workitems.process" {
		t.Fatalf("task header = %v", delivery.Headers["task"])
	}
	for _, key := range []string{"argsrepr", "kwargsrepr"} {
		value, ok := delivery.Headers[key].(string)
		if !ok || value == "" || strings.Contains(value, secret) {
			t.Fatalf("diagnostic header %q = %v; want non-empty redacted string", key, delivery.Headers[key])
		}
	}
	if delivery.CorrelationId == "" || delivery.CorrelationId != delivery.Headers["id"] {
		t.Fatalf("correlation ID = %q, task ID = %v", delivery.CorrelationId, delivery.Headers["id"])
	}
	var body []any
	if err := json.Unmarshal(delivery.Body, &body); err != nil {
		t.Fatalf("decode Celery body: %v", err)
	}
	if len(body) != 3 {
		t.Fatalf("body sections = %d, want args, kwargs, and protocol metadata", len(body))
	}
	if got, want := body[0], []any{"item-17", float64(3)}; !jsonEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
	gotKwargs, ok := body[1].(map[string]any)
	if !ok || !jsonEqual(gotKwargs, kwargs) {
		t.Fatalf("kwargs = %#v, want %#v", body[1], kwargs)
	}
	metadata, ok := body[2].(map[string]any)
	if !ok {
		t.Fatalf("protocol metadata = %#v, want object", body[2])
	}
	for _, key := range []string{"callbacks", "errbacks", "chain", "chord"} {
		if value, exists := metadata[key]; !exists || value != nil {
			t.Errorf("metadata[%q] = %#v, exists=%t; want explicit null", key, value, exists)
		}
	}
}

func jsonEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}
