package celery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	activitylog "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
)

func TestNewPublisherValidatesBrokerSetting(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		brokerSetting string
	}{
		{name: "missing URL"},
		{name: "unsupported scheme", brokerSetting: "redis://localhost/0"},
		{name: "missing host", brokerSetting: "amqp:///plane"},
		{name: "port without host", brokerSetting: "amqp://:5672/plane"},
		{name: "credentials and port without host", brokerSetting: "amqp://audit-user:top-secret@:5672/plane"},
		{name: "opaque credentials", brokerSetting: "amqp:audit-user:top-secret@127.0.0.1:5672/plane"},
		{name: "empty failover entry", brokerSetting: "amqp://localhost/plane;"},
		{name: "negative heartbeat", brokerSetting: "amqp://localhost/plane?heartbeat=-1"},
		{name: "overflowing heartbeat", brokerSetting: "amqp://localhost/plane?heartbeat=65536"},
		{name: "zero port", brokerSetting: "amqp://localhost:0/plane"},
		{name: "overflowing port", brokerSetting: "amqp://localhost:65536/plane"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := NewPublisher(testCase.brokerSetting); err == nil {
				t.Fatal("NewPublisher() error = nil")
			}
		})
	}

	publisher, err := NewPublisher(" amqp://plane:plane@primary:5672/plane ; amqps://plane:plane@secondary:5671/plane ")
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	t.Cleanup(func() { _ = publisher.Close() })
	if publisher.queue != defaultQueue {
		t.Fatalf("queue = %q, want %q", publisher.queue, defaultQueue)
	}
	if len(publisher.brokers) != 2 {
		t.Fatalf("brokers = %#v, want two", publisher.brokers)
	}

	aliasPublisher, err := NewPublisher("pyamqp://plane:plane@rabbitmq:5672/plane")
	if err != nil {
		t.Fatalf("NewPublisher(pyamqp) error = %v", err)
	}
	t.Cleanup(func() { _ = aliasPublisher.Close() })
	if got, want := aliasPublisher.brokers[0].url, "amqp://plane:plane@rabbitmq:5672/plane"; got != want {
		t.Fatalf("normalized broker URL = %q, want %q", got, want)
	}
}

func TestProcessLogsMessageUsesCeleryProtocolV2(t *testing.T) {
	record := activitylog.APIActivityLog{
		TokenIdentifier: "token-hash",
		Path:            "/api/v1/workspaces/example/",
		Method:          "GET",
		QueryParams:     "<QueryDict: {}>",
		Headers:         "{'X-Api-Key': '[REDACTED]'}",
		Body:            "",
		ResponseCode:    200,
		ResponseBody:    "<tag>&",
		IPAddress:       "127.0.0.1",
		UserAgent:       "go-plane test",
	}
	message, err := processLogsMessage(record)
	if err != nil {
		t.Fatalf("processLogsMessage() error = %v", err)
	}

	if message.ContentType != "application/json" {
		t.Errorf("content type = %q", message.ContentType)
	}
	if message.ContentEncoding != "utf-8" {
		t.Errorf("content encoding = %q", message.ContentEncoding)
	}
	if message.DeliveryMode != 2 {
		t.Errorf("delivery mode = %d, want persistent", message.DeliveryMode)
	}
	if message.CorrelationId == "" || message.CorrelationId != message.Headers["id"] {
		t.Errorf("correlation ID = %q, task ID = %v", message.CorrelationId, message.Headers["id"])
	}
	if got := message.Headers["task"]; got != processLogsTask {
		t.Errorf("task = %v, want %q", got, processLogsTask)
	}

	var body []any
	if err := json.Unmarshal(message.Body, &body); err != nil {
		t.Fatalf("decode task body: %v", err)
	}
	if len(body) != 3 {
		t.Fatalf("task body sections = %d, want 3", len(body))
	}
	args, ok := body[0].([]any)
	if !ok || len(args) != 0 {
		t.Fatalf("task positional args = %#v, want empty array", body[0])
	}
	kwargs := body[1].(map[string]any)
	gotRecord := kwargs["log_data"].(map[string]any)
	for key, want := range map[string]any{
		"token_identifier": record.TokenIdentifier,
		"path":             record.Path,
		"method":           record.Method,
		"query_params":     record.QueryParams,
		"headers":          record.Headers,
		"body":             record.Body,
		"response_code":    float64(record.ResponseCode),
		"response_body":    record.ResponseBody,
		"ip_address":       record.IPAddress,
		"user_agent":       record.UserAgent,
	} {
		if got := gotRecord[key]; got != want {
			t.Errorf("log_data[%q] = %#v, want %#v", key, got, want)
		}
	}
	callbacks := body[2].(map[string]any)
	for _, key := range []string{"callbacks", "errbacks", "chain", "chord"} {
		if value, exists := callbacks[key]; !exists || value != nil {
			t.Errorf("%s = %#v, exists = %t; want explicit null", key, value, exists)
		}
	}
}

func TestProcessLogsMessagePreservesNullableFields(t *testing.T) {
	message, err := processLogsMessage(activitylog.APIActivityLog{})
	if err != nil {
		t.Fatalf("processLogsMessage() error = %v", err)
	}
	var body []any
	if err := json.Unmarshal(message.Body, &body); err != nil {
		t.Fatalf("decode task body: %v", err)
	}
	logData := body[1].(map[string]any)["log_data"].(map[string]any)
	for _, key := range []string{"body", "response_body", "ip_address", "user_agent"} {
		if logData[key] != nil {
			t.Errorf("log_data[%q] = %#v, want null", key, logData[key])
		}
	}
}

func TestProcessLogsMessageRejectsUnencodableRecord(t *testing.T) {
	_, err := processLogsMessage(activitylog.APIActivityLog{Body: make(chan struct{})})
	if err == nil {
		t.Fatal("processLogsMessage() error = nil")
	}
}

func TestPublisherRejectsLogAfterClose(t *testing.T) {
	publisher, err := NewPublisher("amqp://plane:plane@localhost:5672/plane")
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	if err := publisher.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := publisher.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if err := publisher.Log(context.Background(), activitylog.APIActivityLog{}); !errors.Is(err, errPublisherClosed) {
		t.Fatalf("Log() error = %v, want %v", err, errPublisherClosed)
	}
}

func TestPublisherConnectionErrorRedactsCredentials(t *testing.T) {
	publisher, err := NewPublisher("amqp://audit-user:top-secret@127.0.0.1:1/plane?connection_timeout=1")
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	t.Cleanup(func() { _ = publisher.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = publisher.Log(ctx, activitylog.APIActivityLog{})
	if err == nil {
		t.Fatal("Log() error = nil")
	}
	for _, secret := range []string{"audit-user", "top-secret", "connection_timeout"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("Log() error exposes %q: %v", secret, err)
		}
	}
}

func TestPublisherContextBoundsAMQPHandshake(t *testing.T) {
	listener, accepted := stalledAMQPListener(t)
	publisher, err := NewPublisher("amqp://guest:guest@" + listener.Addr().String() + "/")
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	t.Cleanup(func() { _ = publisher.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := publisher.Log(ctx, activitylog.APIActivityLog{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Log() error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("Log() elapsed = %v, want at most one second", elapsed)
	}
	select {
	case <-accepted:
	default:
		t.Fatal("publisher did not reach the stalled AMQP server")
	}
}

func TestCloseInterruptsAMQPHandshake(t *testing.T) {
	listener, accepted := stalledAMQPListener(t)
	publisher, err := NewPublisher("amqp://guest:guest@" + listener.Addr().String() + "/")
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}

	logDone := make(chan error, 1)
	go func() {
		logDone <- publisher.Log(context.Background(), activitylog.APIActivityLog{})
	}()
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("publisher did not reach the stalled AMQP server")
	}

	closed := make(chan error, 1)
	go func() { closed <- publisher.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close() did not interrupt the handshake")
	}
	select {
	case err := <-logDone:
		if !errors.Is(err, errPublisherClosed) {
			t.Fatalf("Log() error = %v, want %v", err, errPublisherClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("Log() remained blocked after Close()")
	}
}

func TestBoundedContextKeepsEarlierParentDeadline(t *testing.T) {
	parent, cancelParent := context.WithTimeout(context.Background(), time.Minute)
	defer cancelParent()
	bounded, cancelBounded := boundedContext(parent, time.Millisecond)
	defer cancelBounded()
	deadline, ok := bounded.Deadline()
	if !ok || time.Until(deadline) > time.Second {
		t.Fatalf("bounded deadline = %v, exists = %t", deadline, ok)
	}

	earlier, cancelEarlier := context.WithCancel(context.Background())
	cancelEarlier()
	boundedEarlier, cancelBoundedEarlier := boundedContext(earlier, time.Minute)
	defer cancelBoundedEarlier()
	if got, want := boundedEarlier.Err(), earlier.Err(); got != want {
		t.Fatalf("bounded earlier error = %v, want %v", got, want)
	}
}

type blockingPublisherWorker struct {
	entered chan<- struct{}
	release <-chan struct{}
}

func (worker *blockingPublisherWorker) Log(ctx context.Context, _ activitylog.APIActivityLog) error {
	worker.entered <- struct{}{}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-worker.release:
		return nil
	}
}

func (*blockingPublisherWorker) Close() error                                             { return nil }
func (*blockingPublisherWorker) SoftDelete(context.Context, string, string, string) error { return nil }

func TestPublisherPoolRunsIndependentLanesWithBoundedBackpressure(t *testing.T) {
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	pool := newPublisherPool([]publisherWorker{
		&blockingPublisherWorker{entered: entered, release: release},
		&blockingPublisherWorker{entered: entered, release: release},
	})
	t.Cleanup(func() { _ = pool.Close() })

	completed := make(chan error, 2)
	for range 2 {
		go func() { completed <- pool.Log(context.Background(), activitylog.APIActivityLog{}) }()
	}
	<-entered
	<-entered

	waitingContext, cancelWaiting := context.WithCancel(context.Background())
	cancelWaiting()
	if err := pool.Log(waitingContext, activitylog.APIActivityLog{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting Log() error = %v", err)
	}

	close(release)
	for range 2 {
		if err := <-completed; err != nil {
			t.Fatalf("concurrent Log() error = %v", err)
		}
	}
}
func stalledAMQPListener(t *testing.T) (net.Listener, <-chan struct{}) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	accepted := make(chan struct{})
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		close(accepted)
		_, _ = io.Copy(io.Discard, connection)
		_ = connection.Close()
	}()
	t.Cleanup(func() { _ = listener.Close() })
	return listener, accepted
}
