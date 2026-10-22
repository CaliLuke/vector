package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	APIActivityLogWorkflowName = "plane.api_activity_logs.persist"
	APIActivityLogActivityName = "plane.api_activity_logs.insert"
	APIActivityLogQueue        = "plane-background"
	apiActivityLogTimeout      = 15 * time.Second
)

var ErrPermanentAPIActivityLog = errors.New("permanent API activity log failure")

// APIActivityLogInput is the durable, typed form of the existing process_logs
// payload. The ID remains stable across activity retries, making a crash after
// the insert safe to replay.
type APIActivityLogInput struct {
	ID              string `json:"id"`
	TokenIdentifier string `json:"token_identifier"`
	Path            string `json:"path"`
	Method          string `json:"method"`
	QueryParams     any    `json:"query_params"`
	Headers         any    `json:"headers"`
	Body            any    `json:"body"`
	ResponseCode    int    `json:"response_code"`
	ResponseBody    any    `json:"response_body"`
	IPAddress       any    `json:"ip_address"`
	UserAgent       any    `json:"user_agent"`
}

type apiActivityLogStore interface {
	InsertAPIActivityLog(context.Context, APIActivityLogInput) error
}

// APIActivityLogger admits the same API-token audit record that the Go API
// previously sent to Celery's process_logs task.
type APIActivityLogger struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[APIActivityLogInput, struct{}]
}

func NewAPIActivityLogger(app *selvedge.App, workflow *selvedge.WorkflowFunc[APIActivityLogInput, struct{}]) (*APIActivityLogger, error) {
	if app == nil || workflow == nil {
		return nil, errors.New("Selvedge app and API activity log workflow are required")
	}
	return &APIActivityLogger{app: app, workflow: workflow}, nil
}

func (logger *APIActivityLogger) Log(ctx context.Context, record publicprojectmetadatareads.APIActivityLog) error {
	_, err := logger.Admit(ctx, record)
	return err
}

// Admit starts a uniquely identified workflow and returns its durable identity.
func (logger *APIActivityLogger) Admit(ctx context.Context, record publicprojectmetadatareads.APIActivityLog) (string, error) {
	input := APIActivityLogInput{
		TokenIdentifier: record.TokenIdentifier, Path: record.Path,
		Method: record.Method, QueryParams: record.QueryParams, Headers: record.Headers,
		Body: record.Body, ResponseCode: record.ResponseCode,
		ResponseBody: record.ResponseBody, IPAddress: record.IPAddress, UserAgent: record.UserAgent,
	}
	return logger.AdmitInput(ctx, input)
}

// AdmitInput is the task-payload form used by process_logs compatibility
// publishers. A missing ID is replaced with a fresh event identity.
func (logger *APIActivityLogger) AdmitInput(ctx context.Context, input APIActivityLogInput) (string, error) {
	if logger == nil || logger.app == nil || logger.workflow == nil {
		return "", errors.New("durable API activity logger is not configured")
	}
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if _, err := uuid.Parse(input.ID); err != nil {
		return "", fmt.Errorf("invalid API activity log ID: %w", err)
	}
	startedID, err := selvedge.StartWorkflow(ctx, logger.app, logger.workflow, input,
		selvedge.WithInstanceID("api-activity-log:"+input.ID),
	)
	if err != nil {
		return "", fmt.Errorf("admit API activity log workflow: %w", err)
	}
	return startedID, nil
}

// RegisterAPIActivityLogWorkflow registers a workflow before App.Start.
func RegisterAPIActivityLogWorkflow(app *selvedge.App, store apiActivityLogStore, policy *retry.Policy) (*selvedge.WorkflowFunc[APIActivityLogInput, struct{}], error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and API activity log store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentAPIActivityLog)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[APIActivityLogInput, struct{}]{Name: APIActivityLogActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input APIActivityLogInput) (struct{}, error) {
		if _, err := uuid.Parse(input.ID); err != nil {
			return struct{}{}, fmt.Errorf("%w: invalid record ID", ErrPermanentAPIActivityLog)
		}
		if err := store.InsertAPIActivityLog(ctx, input); err != nil {
			return struct{}{}, classifyAPIActivityLogError(err)
		}
		return struct{}{}, nil
	}, selvedge.ActivityOptions{Queue: APIActivityLogQueue, StartToCloseTimeout: apiActivityLogTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register API activity log activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(APIActivityLogWorkflowName, func(ctx *selvedge.WorkflowContext, input APIActivityLogInput) (struct{}, error) {
		return activity.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return workflow, nil
}

// PostgresAPIActivityLogStore writes the existing Django table directly.
type PostgresAPIActivityLogStore struct {
	pool interface {
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	}
}

func NewPostgresAPIActivityLogStore(pool interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}) (*PostgresAPIActivityLogStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL pool is required")
	}
	return &PostgresAPIActivityLogStore{pool: pool}, nil
}

func (store *PostgresAPIActivityLogStore) InsertAPIActivityLog(ctx context.Context, record APIActivityLogInput) error {
	id, err := uuid.Parse(record.ID)
	if err != nil {
		return fmt.Errorf("parse API activity log ID: %w", err)
	}
	ipAddress, err := nullableIPAddress(record.IPAddress)
	if err != nil {
		return fmt.Errorf("parse API activity log IP address: %w", err)
	}
	queryParams, err := nullableText(record.QueryParams)
	if err != nil {
		return fmt.Errorf("encode API activity log query parameters: %w", err)
	}
	headers, err := nullableText(record.Headers)
	if err != nil {
		return fmt.Errorf("encode API activity log headers: %w", err)
	}
	body, err := nullableText(record.Body)
	if err != nil {
		return fmt.Errorf("encode API activity log request body: %w", err)
	}
	responseBody, err := nullableText(record.ResponseBody)
	if err != nil {
		return fmt.Errorf("encode API activity log response body: %w", err)
	}
	userAgent, err := nullableText(record.UserAgent)
	if err != nil {
		return fmt.Errorf("encode API activity log user agent: %w", err)
	}
	_, err = store.pool.Exec(ctx, `
INSERT INTO api_activity_logs (
 id, token_identifier, path, method, query_params, headers, body,
 response_code, response_body, ip_address, user_agent,
 created_at, updated_at, created_by_id, updated_by_id, deleted_at
) VALUES (
 $1,$2,$3,$4,$5,$6,$7,$8,$9,$10::inet,$11,
 clock_timestamp(),clock_timestamp(),NULL,NULL,NULL
)
ON CONFLICT (id) DO NOTHING`,
		id, record.TokenIdentifier, record.Path, record.Method, queryParams, headers, body,
		record.ResponseCode, responseBody, ipAddress, userAgent,
	)
	if err != nil {
		return fmt.Errorf("insert API activity log %s: %w", record.ID, err)
	}
	return nil
}

func nullableText(value any) (any, error) {
	switch value := value.(type) {
	case nil:
		return nil, nil
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		return nil, fmt.Errorf("expected text or null, got %T", value)
	}
}

func nullableIPAddress(value any) (any, error) {
	text, err := nullableText(value)
	if err != nil || text == nil {
		return text, err
	}
	address := text.(string)
	if net.ParseIP(address) == nil {
		return nil, fmt.Errorf("invalid IP address")
	}
	return address, nil
}

func classifyAPIActivityLogError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		if len(databaseError.Code) < 2 {
			return fmt.Errorf("%w: %v", ErrPermanentAPIActivityLog, err)
		}
		switch databaseError.Code[:2] {
		case "08", "40", "53", "57":
		default:
			return fmt.Errorf("%w: %w", ErrPermanentAPIActivityLog, err)
		}
	}
	return err
}
