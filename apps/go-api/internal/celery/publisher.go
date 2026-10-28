package celery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	activitylog "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultQueue         = "celery"
	defaultIOTimeout     = 5 * time.Second
	publisherPoolSize    = 8
	maximumBrokerAttempt = 2 * time.Second
	connectionRetryCount = 3
	connectionRetryStep  = 200 * time.Millisecond
	maximumHeartbeat     = 1<<16 - 1
	processLogsTask      = "plane.bgtasks.logger_task.process_logs"
)

var errPublisherClosed = errors.New("Celery publisher is closed")

type broker struct {
	url   string
	label string
}

type session struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	transport  net.Conn
	returns    <-chan amqp.Return
}

// Publisher sends protocol-v2 task messages to Plane's existing Celery queue.
type Publisher struct {
	brokers []broker
	queue   string
	permit  chan struct{}

	lifetimeContext context.Context
	cancelLifetime  context.CancelFunc
	closeOnce       sync.Once
	closeDone       chan struct{}
	closeErr        error

	stateMu    sync.Mutex
	session    *session
	closed     bool
	nextBroker int
}

var _ activitylog.ActivityLogger = (*Publisher)(nil)

// NewPublisher validates the Celery broker setting, including Celery's
// semicolon-separated failover URLs. Connections are established lazily.
func NewPublisher(brokerSetting string) (*Publisher, error) {
	return newPublisher(brokerSetting, defaultQueue)
}

type publisherWorker interface {
	Task(context.Context, string, []any, map[string]any) error
	SoftDelete(context.Context, string, string, string) error
	Log(context.Context, activitylog.APIActivityLog) error
	Close() error
}

// PublisherPool bounds concurrent broker work while allowing independent AMQP
// channels to publish and await confirmations in parallel.
type PublisherPool struct {
	available chan publisherWorker
	workers   []publisherWorker
	closed    chan struct{}
	closeOnce sync.Once
	closeDone chan struct{}
	closeErr  error
}

var _ activitylog.ActivityLogger = (*PublisherPool)(nil)

// NewPublisherPool creates a fixed-size, lazy AMQP publisher pool.
func NewPublisherPool(brokerSetting string) (*PublisherPool, error) {
	workers := make([]publisherWorker, 0, publisherPoolSize)
	for range publisherPoolSize {
		publisher, err := NewPublisher(brokerSetting)
		if err != nil {
			for _, worker := range workers {
				_ = worker.Close()
			}
			return nil, err
		}
		workers = append(workers, publisher)
	}
	return newPublisherPool(workers), nil
}

func newPublisherPool(workers []publisherWorker) *PublisherPool {
	available := make(chan publisherWorker, len(workers))
	for _, worker := range workers {
		available <- worker
	}
	return &PublisherPool{
		available: available,
		workers:   workers,
		closed:    make(chan struct{}),
		closeDone: make(chan struct{}),
	}
}

// Log waits for one publisher lane or the caller deadline.
func (pool *PublisherPool) Log(ctx context.Context, record activitylog.APIActivityLog) error {
	var worker publisherWorker
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-pool.closed:
		return errPublisherClosed
	case worker = <-pool.available:
	}
	defer func() { pool.available <- worker }()

	select {
	case <-pool.closed:
		return errPublisherClosed
	default:
		return worker.Log(ctx, record)
	}
}

// Task publishes a durable Celery task through one lane of the pool. Task
// workers share the same queue and confirmation guarantees as activity logs.
func (pool *PublisherPool) Task(ctx context.Context, name string, args []any, kwargs map[string]any) error {
	var worker publisherWorker
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-pool.closed:
		return errPublisherClosed
	case worker = <-pool.available:
	}
	defer func() { pool.available <- worker }()
	select {
	case <-pool.closed:
		return errPublisherClosed
	default:
		return worker.Task(ctx, name, args, kwargs)
	}
}

// SoftDelete publishes Django's typed UUID deletion task through one pool lane.
func (pool *PublisherPool) SoftDelete(ctx context.Context, app, model, id string) error {
	var worker publisherWorker
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-pool.closed:
		return errPublisherClosed
	case worker = <-pool.available:
	}
	defer func() { pool.available <- worker }()
	select {
	case <-pool.closed:
		return errPublisherClosed
	default:
		return worker.SoftDelete(ctx, app, model, id)
	}
}

// Close cancels and closes every independent publisher lane.
func (pool *PublisherPool) Close() error {
	pool.closeOnce.Do(func() {
		close(pool.closed)
		errorsByWorker := make(chan error, len(pool.workers))
		for _, worker := range pool.workers {
			go func() { errorsByWorker <- worker.Close() }()
		}
		var closeErrors []error
		for range pool.workers {
			if err := <-errorsByWorker; err != nil {
				closeErrors = append(closeErrors, err)
			}
		}
		pool.closeErr = errors.Join(closeErrors...)
		close(pool.closeDone)
	})
	<-pool.closeDone
	return pool.closeErr
}

func newPublisher(brokerSetting, queue string) (*Publisher, error) {
	if strings.TrimSpace(brokerSetting) == "" {
		return nil, fmt.Errorf("Celery broker URL is required")
	}
	if queue == "" {
		return nil, fmt.Errorf("Celery queue is required")
	}

	parts := strings.Split(brokerSetting, ";")
	brokers := make([]broker, 0, len(parts))
	for _, part := range parts {
		brokerURL := strings.TrimSpace(part)
		if brokerURL == "" {
			return nil, fmt.Errorf("Celery broker URL list contains an empty entry")
		}
		parsedURL, err := url.Parse(brokerURL)
		if err != nil || parsedURL.Opaque != "" || parsedURL.Hostname() == "" {
			return nil, fmt.Errorf("invalid Celery broker URL")
		}
		if parsedURL.Scheme == "pyamqp" {
			parsedURL.Scheme = "amqp"
			brokerURL = parsedURL.String()
		}
		if port := parsedURL.Port(); port != "" {
			value, atoiErr := strconv.Atoi(port)
			if atoiErr != nil || value < 1 || value > 1<<16-1 {
				return nil, fmt.Errorf("invalid Celery broker URL")
			}
		}
		uri, err := amqp.ParseURI(brokerURL)
		if err != nil {
			return nil, fmt.Errorf("invalid Celery broker URL")
		}
		if uri.Scheme != "amqp" && uri.Scheme != "amqps" {
			return nil, fmt.Errorf("Celery broker URL scheme %q is not AMQP", uri.Scheme)
		}
		for _, heartbeat := range parsedURL.Query()["heartbeat"] {
			value, err := strconv.Atoi(heartbeat)
			if err != nil || value < 0 || value > maximumHeartbeat {
				return nil, fmt.Errorf("invalid Celery broker URL")
			}
		}
		brokers = append(brokers, broker{url: brokerURL, label: brokerLabel(parsedURL)})
	}

	permit := make(chan struct{}, 1)
	permit <- struct{}{}
	lifetimeContext, cancelLifetime := context.WithCancel(context.Background())
	return &Publisher{
		brokers:         brokers,
		queue:           queue,
		permit:          permit,
		lifetimeContext: lifetimeContext,
		cancelLifetime:  cancelLifetime,
		closeDone:       make(chan struct{}),
	}, nil
}

// Log confirms that RabbitMQ accepted one process_logs task after the durable
// target queue was declared. It never retries after publishing because the
// Django consumer is not idempotent.
func (publisher *Publisher) Log(ctx context.Context, record activitylog.APIActivityLog) error {
	message, err := processLogsMessage(record)
	if err != nil {
		return err
	}
	return publisher.publish(ctx, message)
}

func (publisher *Publisher) publish(ctx context.Context, message amqp.Publishing) error {
	ctx, cancel := boundedContext(ctx, defaultIOTimeout)
	stopLifetime := context.AfterFunc(publisher.lifetimeContext, cancel)
	defer func() {
		stopLifetime()
		cancel()
	}()

	select {
	case <-ctx.Done():
		if publisher.isClosed() {
			return errPublisherClosed
		}
		return ctx.Err()
	case <-publisher.permit:
	}
	defer func() { publisher.permit <- struct{}{} }()

	if publisher.isClosed() {
		return errPublisherClosed
	}
	var preparationErrors []error
	for retry := 0; retry <= connectionRetryCount; retry++ {
		if retry > 0 {
			delay := time.Duration(retry-1) * connectionRetryStep
			if err := waitForRetry(ctx, delay); err != nil {
				if publisher.isClosed() {
					return errPublisherClosed
				}
				return err
			}
		}

		current, stopIO, err := publisher.prepareSessionForPublish(ctx)
		if err != nil {
			if publisher.isClosed() {
				return errPublisherClosed
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			preparationErrors = append(preparationErrors, err)
			continue
		}
		if err := publishConfirmedCeleryTask(ctx, current.channel, current.returns, publisher.queue, message); err != nil {
			stopIO()
			publisher.reset(current)
			return err
		}

		// The broker has accepted the persistent message for the durable queue. A
		// concurrent close or local deadline-reset failure only invalidates reuse of
		// this session; it must not turn the confirmed delivery into a retry signal.
		if !stopIO() {
			publisher.reset(current)
			return nil
		}
		if err := current.transport.SetWriteDeadline(time.Time{}); err != nil {
			publisher.reset(current)
		}
		return nil
	}
	return fmt.Errorf("prepare Celery publisher: %w", errors.Join(preparationErrors...))
}

func (publisher *Publisher) prepareSessionForPublish(ctx context.Context) (*session, func() bool, error) {
	current, err := publisher.sessionForPublish(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := setTransportWriteDeadline(current.transport, ctx); err != nil {
		publisher.reset(current)
		return nil, nil, fmt.Errorf("set Celery publish write deadline: %w", err)
	}
	stopIO := context.AfterFunc(ctx, func() { _ = current.transport.Close() })
	if err := declareCeleryQueue(current.channel, publisher.queue); err != nil {
		stopIO()
		publisher.reset(current)
		if publisher.isClosed() {
			return nil, nil, errPublisherClosed
		}
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, err
	}
	return current, stopIO, nil
}

func publishConfirmedCeleryTask(
	ctx context.Context,
	channel *amqp.Channel,
	returns <-chan amqp.Return,
	queue string,
	message amqp.Publishing,
) error {
	confirmation, err := channel.PublishWithDeferredConfirmWithContext(ctx, "", queue, true, false, message)
	if err != nil {
		return fmt.Errorf("publish Celery task: %w", err)
	}
	if confirmation == nil {
		return fmt.Errorf("Celery delivery status is unknown: publisher confirmation is unavailable")
	}
	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("Celery delivery status is unknown: %w", err)
	}
	if !acked {
		return fmt.Errorf("Celery delivery status is unknown: task was not confirmed")
	}

	// amqp091-go notifies buffered return listeners before acknowledging an
	// unroutable mandatory publish, so this receive cannot race the confirmation.
	select {
	case returned, ok := <-returns:
		if !ok {
			return nil
		}
		if returned.CorrelationId != message.CorrelationId {
			return fmt.Errorf("Celery delivery status is unknown: publisher return correlation is unavailable")
		}
		return fmt.Errorf("Celery task was returned by broker: %d %s", returned.ReplyCode, returned.ReplyText)
	default:
		return nil
	}
}

// Close cancels in-flight broker I/O, waits for it to stop, and releases the
// shared connection. Concurrent calls wait for the same close operation.
func (publisher *Publisher) Close() error {
	publisher.closeOnce.Do(func() {
		publisher.stateMu.Lock()
		publisher.closed = true
		publisher.stateMu.Unlock()
		publisher.cancelLifetime()

		<-publisher.permit
		publisher.stateMu.Lock()
		current := publisher.session
		publisher.session = nil
		publisher.stateMu.Unlock()
		if current != nil {
			if err := current.transport.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				publisher.closeErr = fmt.Errorf("close Celery transport: %w", err)
			}
		}
		publisher.permit <- struct{}{}
		close(publisher.closeDone)
	})
	<-publisher.closeDone
	return publisher.closeErr
}

func (publisher *Publisher) sessionForPublish(ctx context.Context) (*session, error) {
	publisher.stateMu.Lock()
	if publisher.closed {
		publisher.stateMu.Unlock()
		return nil, errPublisherClosed
	}
	current := publisher.session
	publisher.stateMu.Unlock()
	if current != nil && !current.connection.IsClosed() && !current.channel.IsClosed() {
		return current, nil
	}
	if current != nil {
		publisher.reset(current)
	}

	var connectionErrors []error
	for remaining := len(publisher.brokers); remaining > 0; remaining-- {
		if publisher.isClosed() {
			return nil, errPublisherClosed
		}
		index := publisher.nextBroker % len(publisher.brokers)
		publisher.nextBroker = (index + 1) % len(publisher.brokers)
		attemptContext, cancelAttempt := brokerAttemptContext(ctx, remaining)
		current, err := publisher.connect(attemptContext, publisher.brokers[index])
		cancelAttempt()
		if err == nil {
			return current, nil
		}
		if publisher.isClosed() {
			return nil, errPublisherClosed
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		connectionErrors = append(connectionErrors, err)
	}
	return nil, fmt.Errorf("connect to Celery brokers: %w", errors.Join(connectionErrors...))
}

func (publisher *Publisher) connect(ctx context.Context, target broker) (*session, error) {
	var transport net.Conn
	var stopIO func() bool
	config := amqp.Config{
		Heartbeat: 30 * time.Second,
		Locale:    "en_US",
		Dial: func(network, address string) (net.Conn, error) {
			dialer := net.Dialer{}
			connection, err := dialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			transport = connection
			stopIO = context.AfterFunc(ctx, func() { _ = connection.Close() })
			if err := setTransportWriteDeadline(connection, ctx); err != nil {
				stopIO()
				_ = connection.Close()
				return nil, err
			}
			return connection, nil
		},
	}

	connection, err := amqp.DialConfig(target.url, config)
	if err != nil {
		if connection != nil {
			_ = connection.CloseDeadline(time.Now())
		}
		stopAndClose(stopIO, transport)
		return nil, fmt.Errorf("connect to Celery broker %s: %w", target.label, err)
	}
	channel, err := connection.Channel()
	if err != nil {
		stopAndClose(stopIO, transport)
		return nil, fmt.Errorf("open Celery channel: %w", err)
	}
	returns := channel.NotifyReturn(make(chan amqp.Return, 1))
	if err := channel.Confirm(false); err != nil {
		stopAndClose(stopIO, transport)
		return nil, fmt.Errorf("enable Celery publisher confirms: %w", err)
	}
	if stopIO == nil || !stopIO() || ctx.Err() != nil {
		_ = transport.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("Celery connection setup was interrupted")
	}
	if err := transport.SetWriteDeadline(time.Time{}); err != nil {
		_ = transport.Close()
		return nil, fmt.Errorf("clear Celery connection write deadline: %w", err)
	}

	current := &session{connection: connection, channel: channel, transport: transport, returns: returns}
	publisher.stateMu.Lock()
	if publisher.closed || ctx.Err() != nil {
		publisher.stateMu.Unlock()
		_ = transport.Close()
		if publisher.isClosed() {
			return nil, errPublisherClosed
		}
		return nil, ctx.Err()
	}
	publisher.session = current
	publisher.stateMu.Unlock()
	return current, nil
}

func (publisher *Publisher) isClosed() bool {
	publisher.stateMu.Lock()
	defer publisher.stateMu.Unlock()
	return publisher.closed
}

func (publisher *Publisher) reset(current *session) {
	publisher.stateMu.Lock()
	if publisher.session == current {
		publisher.session = nil
	}
	publisher.stateMu.Unlock()
	if current != nil {
		_ = current.transport.Close()
	}
}

func declareCeleryQueue(channel *amqp.Channel, queue string) error {
	if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare Celery queue: %w", err)
	}
	return nil
}

func boundedContext(parent context.Context, limit time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := parent.Deadline(); ok && time.Until(deadline) <= limit {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, limit)
}

func brokerAttemptContext(parent context.Context, remainingBrokers int) (context.Context, context.CancelFunc) {
	deadline, ok := parent.Deadline()
	if !ok || remainingBrokers <= 0 {
		return context.WithCancel(parent)
	}
	budget := time.Until(deadline) / time.Duration(remainingBrokers)
	if budget > maximumBrokerAttempt {
		budget = maximumBrokerAttempt
	}
	if budget <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, budget)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func setTransportWriteDeadline(connection net.Conn, ctx context.Context) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("context has no deadline")
	}
	return connection.SetWriteDeadline(deadline)
}

func stopAndClose(stop func() bool, connection net.Conn) {
	if stop != nil {
		stop()
	}
	if connection != nil {
		_ = connection.Close()
	}
}

func brokerLabel(parsedURL *url.URL) string {
	return parsedURL.Scheme + "://" + parsedURL.Host
}

func processLogsMessage(record activitylog.APIActivityLog) (amqp.Publishing, error) {
	body, err := json.Marshal([]any{
		[]any{},
		map[string]any{"log_data": map[string]any{
			"token_identifier": record.TokenIdentifier,
			"path":             record.Path,
			"method":           record.Method,
			"query_params":     record.QueryParams,
			"headers":          record.Headers,
			"body":             record.Body,
			"response_code":    record.ResponseCode,
			"response_body":    record.ResponseBody,
			"ip_address":       record.IPAddress,
			"user_agent":       record.UserAgent,
		}},
		map[string]any{"callbacks": nil, "errbacks": nil, "chain": nil, "chord": nil},
	})
	if err != nil {
		return amqp.Publishing{}, fmt.Errorf("encode Celery process_logs task: %w", err)
	}

	return taskMessage(processLogsTask, body, "()", "{'log_data': <Plane API activity log>}"), nil
}

func taskMessage(task string, body []byte, argsrepr, kwargsrepr string) amqp.Publishing {
	taskID := uuid.NewString()
	return amqp.Publishing{
		Headers: amqp.Table{
			"lang":                  "py",
			"task":                  task,
			"id":                    taskID,
			"shadow":                nil,
			"eta":                   nil,
			"expires":               nil,
			"group":                 nil,
			"group_index":           nil,
			"retries":               int32(0),
			"timelimit":             []any{nil, nil},
			"root_id":               taskID,
			"parent_id":             nil,
			"argsrepr":              argsrepr,
			"kwargsrepr":            kwargsrepr,
			"origin":                "gen1@go-plane-api",
			"ignore_result":         false,
			"replaced_task_nesting": int32(0),
			"stamped_headers":       nil,
			"stamps":                amqp.Table{},
		},
		ContentType:     "application/json",
		ContentEncoding: "utf-8",
		DeliveryMode:    amqp.Persistent,
		CorrelationId:   taskID,
		Body:            body,
	}
}
