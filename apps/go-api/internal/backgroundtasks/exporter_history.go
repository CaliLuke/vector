package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	awsretry "github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/smithy-go"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ExporterHistoryWorkflowName   = "plane.exporter_history.delete_expired_objects"
	ExporterHistoryActivityName   = "plane.exporter_history.delete_expired_batch"
	ExporterHistoryQueue          = "plane-background"
	exporterHistoryRetention      = 8 * 24 * time.Hour
	exporterHistoryBatchSize      = 500
	exporterHistoryActivityTimout = 5 * time.Minute
)

type exporterHistoryBeatSpec struct {
	name   string
	hour   int
	minute int
}

// Plane currently schedules this cleanup at both 01:30 and 03:45 UTC. Keep
// both source firings until the deployment changes its Beat contract.
var exporterHistoryBeatSchedules = []exporterHistoryBeatSpec{
	{name: "delete-exporter-history-0130", hour: 1, minute: 30},
	{name: "delete-exporter-history-0345", hour: 3, minute: 45},
}

var ErrPermanentExporterHistory = errors.New("permanent exporter history failure")

type ExporterHistoryWorkflowInput struct {
	ScheduledAt time.Time `json:"scheduled_at"`
	Cutoff      time.Time `json:"cutoff"`
}

type ExporterHistoryBatchInput struct {
	Cutoff time.Time `json:"cutoff"`
}

type ExporterHistoryBatchOutput struct {
	Processed int64 `json:"processed"`
}

type exporterObjectDeleter interface {
	Delete(context.Context, string) error
}

type exporterHistoryStore interface {
	DeleteExpiredBatch(context.Context, time.Time) (int64, error)
}

type PostgresExporterHistoryStore struct {
	pool    *pgxpool.Pool
	objects exporterObjectDeleter
}

func NewPostgresExporterHistoryStore(pool *pgxpool.Pool, objects exporterObjectDeleter) (*PostgresExporterHistoryStore, error) {
	if pool == nil || objects == nil {
		return nil, errors.New("PostgreSQL pool and object storage are required")
	}
	return &PostgresExporterHistoryStore{pool: pool, objects: objects}, nil
}

type expiredExporterHistory struct {
	ID  string
	Key string
}

func (store *PostgresExporterHistoryStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time) (int64, error) {
	rows, err := store.pool.Query(ctx, `
SELECT id::text, key
FROM exporters
WHERE url IS NOT NULL AND created_at <= $1 AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT $2`, cutoff, exporterHistoryBatchSize)
	if err != nil {
		return 0, fmt.Errorf("select expired exporter history: %w", err)
	}
	batch := make([]expiredExporterHistory, 0, exporterHistoryBatchSize)
	for rows.Next() {
		var item expiredExporterHistory
		if err := rows.Scan(&item.ID, &item.Key); err != nil {
			rows.Close()
			return 0, fmt.Errorf("read expired exporter history: %w", err)
		}
		batch = append(batch, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate expired exporter history: %w", err)
	}
	rows.Close()

	var processed int64
	for _, item := range batch {
		if item.Key != "" {
			if err := store.objects.Delete(ctx, item.Key); err != nil {
				return processed, fmt.Errorf("delete exported object %q for history %s: %w", item.Key, item.ID, err)
			}
		}
		result, err := store.pool.Exec(ctx, `
UPDATE exporters SET url=NULL
WHERE id=$1::uuid AND url IS NOT NULL AND created_at <= $2 AND deleted_at IS NULL`, item.ID, cutoff)
		if err != nil {
			return processed, fmt.Errorf("clear expired exporter URL for %s: %w", item.ID, err)
		}
		processed += result.RowsAffected()
	}
	return processed, nil
}

type ExporterHistoryDispatcher struct {
	app       *selvedge.App
	workflow  *selvedge.WorkflowFunc[ExporterHistoryWorkflowInput, ExporterHistoryBatchOutput]
	schedules map[string]*selvedge.WorkflowFunc[struct{}, struct{}]
}

func RegisterExporterHistoryWorkflow(app *selvedge.App, store exporterHistoryStore, policy *retry.Policy) (*ExporterHistoryDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("workflow app and exporter history store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	policyCopy.NonRetryableErrors = append(append([]error(nil), policy.NonRetryableErrors...), ErrPermanentExporterHistory)
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	activity := execution.Activity[ExporterHistoryBatchInput, ExporterHistoryBatchOutput]{Name: ExporterHistoryActivityName}
	if err := executionSelvedge.Register(activities, activity, func(ctx context.Context, input ExporterHistoryBatchInput) (ExporterHistoryBatchOutput, error) {
		processed, err := store.DeleteExpiredBatch(ctx, input.Cutoff)
		if err != nil {
			return ExporterHistoryBatchOutput{}, classifyExporterHistoryError(err)
		}
		return ExporterHistoryBatchOutput{Processed: processed}, nil
	}, selvedge.ActivityOptions{Queue: ExporterHistoryQueue, StartToCloseTimeout: exporterHistoryActivityTimout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register exporter history activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(ExporterHistoryWorkflowName, func(ctx *selvedge.WorkflowContext, input ExporterHistoryWorkflowInput) (ExporterHistoryBatchOutput, error) {
		var result ExporterHistoryBatchOutput
		for {
			batch, err := activity.Execute(ctx.Context(), activities.Scope(ctx), ExporterHistoryBatchInput{Cutoff: input.Cutoff}, execution.ActivityOptions{})
			if err != nil {
				return ExporterHistoryBatchOutput{}, err
			}
			result.Processed += batch.Processed
			if batch.Processed < exporterHistoryBatchSize {
				return result, nil
			}
		}
	})
	selvedge.RegisterWorkflow(app, workflow)
	dispatcher := &ExporterHistoryDispatcher{
		app: app, workflow: workflow,
		schedules: make(map[string]*selvedge.WorkflowFunc[struct{}, struct{}], len(exporterHistoryBeatSchedules)),
	}
	for _, spec := range exporterHistoryBeatSchedules {
		spec := spec
		schedule := selvedge.DefineWorkflow("plane.beat."+spec.name, func(ctx *selvedge.WorkflowContext, _ struct{}) (struct{}, error) {
			for {
				now, err := selvedge.Now(ctx)
				if err != nil {
					return struct{}{}, err
				}
				next := nextDailyUTC(now, spec.hour, spec.minute)
				if err := selvedge.SleepUntil(ctx, next, selvedge.WithSleepID("exporter-history-beat:"+spec.name+":"+next.Format(time.RFC3339))); err != nil {
					return struct{}{}, err
				}
				for attempt := 0; ; attempt++ {
					childID := fmt.Sprintf("plane-exporter-history-scheduled:%s:%s:%d", spec.name, next.Format(time.RFC3339), attempt)
					child, err := selvedge.StartChild[ExporterHistoryBatchOutput](ctx, ExporterHistoryWorkflowName,
						ExporterHistoryWorkflowInput{ScheduledAt: next, Cutoff: next.Add(-exporterHistoryRetention)}, selvedge.WithChildID(childID))
					if err != nil {
						return struct{}{}, err
					}
					if _, err := child.Get(ctx); err == nil {
						break
					}
					if err := selvedge.Sleep(ctx, time.Minute, selvedge.WithSleepID(fmt.Sprintf("exporter-history-retry:%s:%d", next.Format(time.RFC3339), attempt))); err != nil {
						return struct{}{}, err
					}
				}
			}
		})
		selvedge.RegisterWorkflow(app, schedule)
		dispatcher.schedules[spec.name] = schedule
	}
	return dispatcher, nil
}

func (dispatcher *ExporterHistoryDispatcher) StartBeatSchedules(ctx context.Context) error {
	if dispatcher == nil || dispatcher.app == nil {
		return errors.New("exporter history Beat schedules are not configured")
	}
	for _, spec := range exporterHistoryBeatSchedules {
		workflow := dispatcher.schedules[spec.name]
		if workflow == nil {
			return fmt.Errorf("exporter history Beat schedule %q is not registered", spec.name)
		}
		if _, err := admitOrReconcile(ctx, dispatcher.app, workflow, "plane-beat:"+spec.name, struct{}{}); err != nil {
			return fmt.Errorf("start exporter history Beat schedule %s: %w", spec.name, err)
		}
	}
	return nil
}

func (dispatcher *ExporterHistoryDispatcher) Admit(ctx context.Context, scheduledAt time.Time) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("durable exporter history dispatcher is not configured")
	}
	scheduledAt = scheduledAt.UTC()
	input := ExporterHistoryWorkflowInput{ScheduledAt: scheduledAt, Cutoff: scheduledAt.Add(-exporterHistoryRetention)}
	instanceID := "plane-exporter-history:" + scheduledAt.Format(time.RFC3339Nano)
	startedID, err := admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, instanceID, input)
	if err != nil {
		return "", fmt.Errorf("admit exporter history cleanup: %w", err)
	}
	return startedID, nil
}

func classifyExporterHistoryError(err error) error {
	if errors.Is(err, ErrPermanentExporterHistory) {
		return err
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		if len(databaseError.Code) < 2 {
			return fmt.Errorf("%w: %v", ErrPermanentExporterHistory, err)
		}
		switch databaseError.Code[:2] {
		case "08", "40", "53", "57":
		default:
			return fmt.Errorf("%w: %w", ErrPermanentExporterHistory, err)
		}
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		if awsretry.NewStandard().IsErrorRetryable(err) || apiError.ErrorFault() == smithy.FaultServer {
			return err
		}
		return fmt.Errorf("%w: %w", ErrPermanentExporterHistory, err)
	}
	return err
}
