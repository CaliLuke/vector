//go:build integration

package celery

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	activitylog "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestPublisherEmitsCeleryConsumableProcessLogsMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	brokerSetting := startDisposableRabbitMQ(t, ctx)

	queue := "go-plane-celery-test-" + uuid.NewString()
	consumerConnection, err := amqp.Dial(brokerSetting)
	if err != nil {
		t.Fatalf("connect consumer: %v", err)
	}
	consumerChannel, err := consumerConnection.Channel()
	if err != nil {
		testsupport.Close(t, consumerConnection)
		t.Fatalf("open consumer channel: %v", err)
	}
	if declareCeleryQueueErr := declareCeleryQueue(consumerChannel, queue); declareCeleryQueueErr != nil {
		testsupport.Close(t, consumerChannel)
		testsupport.Close(t, consumerConnection)
		t.Fatal(declareCeleryQueueErr)
	}
	t.Cleanup(func() {
		if _, queueDeleteErr := consumerChannel.QueueDelete(queue, false, false, false); queueDeleteErr != nil {
			t.Errorf("delete disposable queue: %v", queueDeleteErr)
		}
		if closeErr := consumerChannel.Close(); closeErr != nil {
			t.Errorf("close consumer channel: %v", closeErr)
		}
		if closeErr := consumerConnection.Close(); closeErr != nil {
			t.Errorf("close consumer connection: %v", closeErr)
		}
	})

	publisher, err := newPublisher(brokerSetting, queue)
	if err != nil {
		t.Fatalf("newPublisher() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := publisher.Close(); closeErr != nil {
			t.Errorf("close publisher: %v", closeErr)
		}
	})

	publishCtx, cancelPublish := context.WithTimeout(ctx, 5*time.Second)
	defer cancelPublish()
	record := activitylog.APIActivityLog{Path: "/api/v1/example/", ResponseCode: 200}
	if logErr := publisher.Log(publishCtx, record); logErr != nil {
		t.Fatalf("Log() error = %v", logErr)
	}
	if _, queueDeleteErr := consumerChannel.QueueDelete(queue, false, false, false); queueDeleteErr != nil {
		t.Fatalf("delete queue between publishes: %v", queueDeleteErr)
	}
	record.Path = "/api/v1/redeclared-queue/"
	if logErr := publisher.Log(publishCtx, record); logErr != nil {
		t.Fatalf("Log() after queue deletion error = %v", logErr)
	}

	delivery := consumeDelivery(t, consumerChannel, queue)
	if delivery.Exchange != "" || delivery.RoutingKey != queue {
		t.Fatalf("route = exchange %q, key %q; want default exchange and %q", delivery.Exchange, delivery.RoutingKey, queue)
	}
	if delivery.ContentType != "application/json" || delivery.ContentEncoding != "utf-8" {
		t.Fatalf("content = %q; encoding = %q", delivery.ContentType, delivery.ContentEncoding)
	}
	if delivery.DeliveryMode != amqp.Persistent {
		t.Fatalf("delivery mode = %d, want persistent", delivery.DeliveryMode)
	}
	if delivery.Headers["task"] != processLogsTask {
		t.Fatalf("task = %v, want %q", delivery.Headers["task"], processLogsTask)
	}
	if delivery.CorrelationId == "" || delivery.CorrelationId != delivery.Headers["id"] {
		t.Fatalf("correlation ID = %q, task ID = %v", delivery.CorrelationId, delivery.Headers["id"])
	}

	var body []any
	if unmarshalErr := json.Unmarshal(delivery.Body, &body); unmarshalErr != nil {
		t.Fatalf("decode body: %v", unmarshalErr)
	}
	kwargs := body[1].(map[string]any)
	got := kwargs["log_data"].(map[string]any)
	if got["path"] != record.Path || got["response_code"] != float64(200) {
		t.Fatalf("log_data = %#v, want path %q and response code 200", got, record.Path)
	}

	message, err := processLogsMessage(record)
	if err != nil {
		t.Fatalf("processLogsMessage() error = %v", err)
	}
	current, err := publisher.sessionForPublish(publishCtx)
	if err != nil {
		t.Fatalf("sessionForPublish() error = %v", err)
	}
	if err := publishConfirmedCeleryTask(publishCtx, current.channel, current.returns, queue+"-missing", message); err == nil || !strings.Contains(err.Error(), "returned by broker") {
		t.Fatalf("unroutable publish error = %v, want broker return", err)
	}

	if err := current.transport.Close(); err != nil {
		t.Fatalf("close established publisher transport: %v", err)
	}
	record.Path = "/api/v1/reconnected-session/"
	if err := publisher.Log(publishCtx, record); err != nil {
		t.Fatalf("Log() after stale session error = %v", err)
	}
	recoveredDelivery := consumeDelivery(t, consumerChannel, queue)
	body = nil
	if err := json.Unmarshal(recoveredDelivery.Body, &body); err != nil {
		t.Fatalf("decode recovered body: %v", err)
	}
	recoveredLog := body[1].(map[string]any)["log_data"].(map[string]any)
	if recoveredLog["path"] != record.Path {
		t.Fatalf("recovered log_data = %#v, want path %q", recoveredLog, record.Path)
	}
}

func consumeDelivery(t *testing.T, channel *amqp.Channel, queue string) amqp.Delivery {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		delivery, ok, err := channel.Get(queue, true)
		if err != nil {
			t.Fatalf("consume task: %v", err)
		}
		if ok {
			return delivery
		}
		if time.Now().After(deadline) {
			t.Fatal("consume task: queue remained empty")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
