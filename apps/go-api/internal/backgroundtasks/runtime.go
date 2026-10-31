package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
	"github.com/CaliLuke/selvedge"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runtime owns the Selvedge app and the PostgreSQL connection used by task
// handlers. The app registers its workers when Start is called.
type Runtime struct {
	App                  *selvedge.App
	Logger               *APIActivityLogger
	Cleanup              *CleanupDispatcher
	Assets               *FileAssetMetadataDispatcher
	Exporter             *ExporterHistoryDispatcher
	Versions             *VersionHistoryDispatcher
	IssueDescriptionSync *IssueDescriptionSyncDispatcher
	IssueVersionSync     *IssueVersionSyncDispatcher
	IssueVersions        *IssueVersionTaskDispatcher
	PageTransactions     *PageTransactionDispatcher
	DescriptionAssets    *DescriptionAssetCopyDispatcher
	RecentVisits         *RecentVisitDispatcher
	IssueActivities      *IssueActivityDispatcher
	IssueAutomation      *IssueAutomationDispatcher
	EmailNotifications   *IssueEmailDispatcher
	TemplatedEmails      *TemplatedEmailTaskDispatcher
	EmailTaskAdmission   *TemplatedEmailTaskAdmission
	InstanceMetrics      *InstanceMetricsDispatcher
	IssueExports         *IssueExportDispatcher
	AnalyticsExports     *AnalyticsExportDispatcher
	EventTracking        *EventTrackingDispatcher
	WorkspaceSeed        *WorkspaceSeedDispatcher
	LinkPreviews         *LinkPreviewDispatcher
	Webhooks             *WebhookTaskDispatcher
	Deletion             *DeletionDispatcher
	workItemEffects      *workItemEffectWorker
	pool                 *pgxpool.Pool
	emailStore           *issueEmailStore
}

// NewRuntime uses the application database but places Selvedge's migration
// journal and runtime tables in a dedicated schema. It only creates that
// schema; it never drops or replaces application data.
func NewRuntime(ctx context.Context, databaseURL string, options ...selvedge.Option) (_ *Runtime, err error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("database URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("configure background task PostgreSQL pool: %w", err)
	}
	resources := runtimeResources{database: pool}
	closeOnError := true
	defer func() {
		if closeOnError {
			if closeErr := resources.close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}
	}()
	if pingErr := pool.Ping(ctx); pingErr != nil {
		return nil, fmt.Errorf("connect background task PostgreSQL pool: %w", pingErr)
	}
	if _, execErr := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS plane_execution`); execErr != nil {
		return nil, fmt.Errorf("ensure dedicated Selvedge schema: %w", execErr)
	}
	selvedgeURL, err := withExecutionSearchPath(databaseURL)
	if err != nil {
		return nil, err
	}
	appOptions := []selvedge.Option{
		selvedge.WithDatabase(selvedgeURL),
		selvedge.WithServiceName("plane-background-tasks"),
	}
	appOptions = append(appOptions, options...)
	app := selvedge.NewApp(appOptions...)
	runtime := &Runtime{App: app, pool: pool}
	store, err := NewPostgresAPIActivityLogStore(pool)
	if err != nil {
		return nil, err
	}
	workflow, err := RegisterAPIActivityLogWorkflow(app, store, nil)
	if err != nil {
		return nil, err
	}
	runtime.Logger, err = NewAPIActivityLogger(app, workflow)
	if err != nil {
		return nil, err
	}
	cleanupStore, err := NewPostgresCleanupStore(pool)
	if err != nil {
		return nil, err
	}
	runtime.Cleanup, err = RegisterCleanupWorkflows(app, cleanupStore, nil)
	if err != nil {
		return nil, err
	}
	assetWorkflows, err := runtime.registerAssetWorkflows()
	if err != nil {
		return nil, err
	}
	if err = runtime.registerContentWorkflows(assetWorkflows.storage); err != nil {
		return nil, err
	}
	recentVisitStore, err := NewPostgresRecentVisitStore(pool)
	if err != nil {
		return nil, err
	}
	runtime.RecentVisits, err = RegisterRecentVisitWorkflow(app, recentVisitStore, nil)
	if err != nil {
		return nil, err
	}
	issueActivityStore, err := NewPostgresIssueActivityStore(pool)
	if err != nil {
		return nil, err
	}
	resources.activities = issueActivityStore
	runtime.IssueActivities, err = RegisterIssueActivityWorkflow(app, issueActivityStore, nil)
	if err != nil {
		return nil, err
	}
	issueAutomationStore, err := NewPostgresIssueAutomationStore(pool)
	if err != nil {
		return nil, err
	}
	runtime.IssueAutomation, err = RegisterIssueAutomationWorkflow(app, issueAutomationStore, runtime.IssueActivities, nil)
	if err != nil {
		return nil, err
	}
	cipher, err := instancecipher.New(os.Getenv("SECRET_KEY"))
	if err != nil {
		return nil, fmt.Errorf("configure instance email settings cipher: %w", err)
	}
	instanceRepository, err := instancemanagement.NewRepository(pool, cipher)
	if err != nil {
		return nil, fmt.Errorf("configure instance email settings repository: %w", err)
	}
	runtime.emailStore, err = NewPostgresIssueEmailStore(pool, instanceRepository, instancemail.Sender{})
	if err != nil {
		return nil, err
	}
	resources.emails = runtime.emailStore
	if err = runtime.registerEmailWorkflows(instanceRepository); err != nil {
		return nil, err
	}
	instanceMetricsStore, err := NewPostgresInstanceMetricsStore(pool, otlpInstanceMetricsExporter{})
	if err != nil {
		return nil, err
	}
	runtime.InstanceMetrics, err = RegisterInstanceMetricsWorkflow(app, instanceMetricsStore, nil)
	if err != nil {
		return nil, err
	}
	eventTrackingStore, err := NewPostgresEventTrackingStore(pool, cipher)
	if err != nil {
		return nil, err
	}
	runtime.EventTracking, err = RegisterEventTrackingWorkflow(app, eventTrackingStore, nil)
	if err != nil {
		return nil, err
	}
	workspaceSeedStore, err := NewPostgresWorkspaceSeedStore(pool, os.Getenv("WEB_URL"))
	if err != nil {
		return nil, err
	}
	runtime.WorkspaceSeed, err = RegisterWorkspaceSeedWorkflow(app, workspaceSeedStore, nil)
	if err != nil {
		return nil, err
	}
	linkPreviewStore, err := NewPostgresLinkPreviewStore(pool)
	if err != nil {
		return nil, err
	}
	runtime.LinkPreviews, err = RegisterLinkPreviewWorkflow(app, linkPreviewStore, nil)
	if err != nil {
		return nil, err
	}
	if err = runtime.registerExportWorkflows(assetWorkflows.config, instanceRepository); err != nil {
		return nil, err
	}
	webhookStore, err := NewPostgresWebhookTaskStore(pool)
	if err != nil {
		return nil, err
	}
	runtime.Webhooks, err = RegisterWebhookTaskWorkflows(app, webhookStore, nil)
	if err != nil {
		return nil, err
	}
	deletionStore, err := NewPostgresDeletionStore(pool)
	if err != nil {
		return nil, err
	}
	deleteDays, err := HardDeleteDaysFromEnv()
	if err != nil {
		return nil, err
	}
	runtime.Deletion, err = RegisterDeletionWorkflows(app, deletionStore, deleteDays, nil)
	if err != nil {
		return nil, err
	}
	if err = runtime.configureWorkItemEffects(assetWorkflows); err != nil {
		return nil, err
	}
	closeOnError = false
	return runtime, nil
}

func (runtime *Runtime) Start(ctx context.Context) error {
	if runtime == nil || runtime.App == nil {
		return errors.New("background task runtime is not configured")
	}
	if err := runtime.App.Start(ctx); err != nil {
		return fmt.Errorf("start Selvedge background task runtime: %w", err)
	}
	if runtime.EmailNotifications != nil {
		if _, err := runtime.EmailNotifications.StartBeatSchedule(ctx); err != nil {
			return fmt.Errorf("start durable email notification Beat schedule: %w", err)
		}
	}
	if runtime.Cleanup != nil {
		if err := runtime.Cleanup.StartBeatSchedules(ctx); err != nil {
			return fmt.Errorf("start durable cleanup Beat schedules: %w", err)
		}
	}
	if runtime.Exporter != nil {
		if err := runtime.Exporter.StartBeatSchedules(ctx); err != nil {
			return fmt.Errorf("start durable exporter history Beat schedules: %w", err)
		}
	}
	if runtime.IssueAutomation != nil {
		if _, err := runtime.IssueAutomation.StartBeatSchedule(ctx); err != nil {
			return fmt.Errorf("start durable issue automation Beat schedule: %w", err)
		}
	}
	if runtime.InstanceMetrics != nil {
		if _, err := runtime.InstanceMetrics.StartBeatSchedule(ctx); err != nil {
			return fmt.Errorf("start durable instance metrics Beat schedule: %w", err)
		}
	}
	if runtime.Deletion != nil {
		if _, err := runtime.Deletion.StartBeatSchedule(ctx); err != nil {
			return fmt.Errorf("start durable hard deletion Beat schedule: %w", err)
		}
	}
	if runtime.workItemEffects != nil {
		if err := runtime.workItemEffects.Start(ctx); err != nil {
			return fmt.Errorf("start work item effect worker: %w", err)
		}
	}
	return nil
}

func (runtime *Runtime) Shutdown(ctx context.Context) error {
	if runtime == nil {
		return nil
	}
	var shutdownErr error
	if runtime.workItemEffects != nil {
		shutdownErr = runtime.workItemEffects.Shutdown(ctx)
	}
	if runtime.App != nil {
		shutdownErr = errors.Join(shutdownErr, runtime.App.Shutdown(ctx))
	}
	resources := runtimeResources{database: runtime.pool, activities: runtime.IssueActivities, emails: runtime.emailStore}
	shutdownErr = errors.Join(shutdownErr, resources.close())
	if shutdownErr != nil {
		return fmt.Errorf("stop Selvedge background task runtime: %w", shutdownErr)
	}
	return nil
}

func withExecutionSearchPath(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return "", errors.New("background tasks require a postgres:// or postgresql:// URL")
	}
	query := parsed.Query()
	if query.Has("options") {
		return "", errors.New("background task URL must not set PostgreSQL options; Plane owns the Selvedge search path")
	}
	query.Set("options", "-c search_path=plane_execution,public")
	// pgx accepts PostgreSQL's URI form, where spaces in `options` must be
	// percent-encoded. QueryEscape's `+` encoding is valid for form values but
	// libpq passes it through as a literal plus in PostgreSQL's startup options.
	parsed.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
	return parsed.String(), nil
}
