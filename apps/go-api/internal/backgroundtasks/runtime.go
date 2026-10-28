package backgroundtasks

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
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
func NewRuntime(ctx context.Context, databaseURL string, options ...selvedge.Option) (*Runtime, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("Plane PostgreSQL URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("configure background task PostgreSQL pool: %w", err)
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			pool.Close()
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
	store, err := NewPostgresAPIActivityLogStore(pool)
	if err != nil {
		return nil, err
	}
	workflow, err := RegisterAPIActivityLogWorkflow(app, store, nil)
	if err != nil {
		return nil, err
	}
	logger, err := NewAPIActivityLogger(app, workflow)
	if err != nil {
		return nil, err
	}
	cleanupStore, err := NewPostgresCleanupStore(pool)
	if err != nil {
		return nil, err
	}
	cleanup, err := RegisterCleanupWorkflows(app, cleanupStore, nil)
	if err != nil {
		return nil, err
	}
	assetRepository, err := assets.NewPostgresRepository(pool)
	if err != nil {
		return nil, err
	}
	storageConfig, err := assets.StorageConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("configure FileAsset S3 metadata activity: %w", err)
	}
	storage, err := assets.NewStorage(storageConfig)
	if err != nil {
		return nil, fmt.Errorf("create FileAsset S3 metadata client: %w", err)
	}
	metadataWorkflow, err := assets.RegisterStorageMetadataWorkflow(app, assetRepository, storage, nil)
	if err != nil {
		return nil, err
	}
	assetDispatcher, err := NewFileAssetMetadataDispatcher(app, metadataWorkflow)
	if err != nil {
		return nil, err
	}
	assetDeleteWorkflow, err := assets.RegisterStorageObjectDeleteWorkflow(app, assetRepository, storage, nil)
	if err != nil {
		return nil, err
	}
	assetDeleteDispatcher, err := NewFileAssetStorageDeleteDispatcher(app, assetDeleteWorkflow)
	if err != nil {
		return nil, err
	}
	exporterStore, err := NewPostgresExporterHistoryStore(pool, storage)
	if err != nil {
		return nil, err
	}
	exporterHistory, err := RegisterExporterHistoryWorkflow(app, exporterStore, nil)
	if err != nil {
		return nil, err
	}
	versionStore, err := NewPostgresVersionHistoryStore(pool)
	if err != nil {
		return nil, err
	}
	versions, err := RegisterVersionHistoryWorkflows(app, versionStore, nil)
	if err != nil {
		return nil, err
	}
	issueDescriptionSyncStore, err := NewPostgresIssueDescriptionSyncStore(pool)
	if err != nil {
		return nil, err
	}
	issueDescriptionSync, err := RegisterIssueDescriptionSyncWorkflow(app, issueDescriptionSyncStore, nil)
	if err != nil {
		return nil, err
	}
	issueVersionSyncStore, err := NewPostgresIssueVersionSyncStore(pool)
	if err != nil {
		return nil, err
	}
	issueVersionSync, err := RegisterIssueVersionSyncWorkflow(app, issueVersionSyncStore, nil)
	if err != nil {
		return nil, err
	}
	issueVersionTaskStore, err := NewPostgresIssueVersionTaskStore(pool)
	if err != nil {
		return nil, err
	}
	issueVersions, err := RegisterIssueVersionTaskWorkflow(app, issueVersionTaskStore, nil)
	if err != nil {
		return nil, err
	}
	pageTransactionStore, err := NewPostgresPageTransactionStore(pool)
	if err != nil {
		return nil, err
	}
	pageTransactions, err := RegisterPageTransactionWorkflow(app, pageTransactionStore, nil)
	if err != nil {
		return nil, err
	}
	descriptionAssetStore, err := NewPostgresDescriptionAssetCopyStore(pool, storage)
	if err != nil {
		return nil, err
	}
	descriptionAssets, err := RegisterDescriptionAssetCopyWorkflow(app, descriptionAssetStore, nil)
	if err != nil {
		return nil, err
	}
	recentVisitStore, err := NewPostgresRecentVisitStore(pool)
	if err != nil {
		return nil, err
	}
	recentVisits, err := RegisterRecentVisitWorkflow(app, recentVisitStore, nil)
	if err != nil {
		return nil, err
	}
	issueActivityStore, err := NewPostgresIssueActivityStore(pool)
	if err != nil {
		return nil, err
	}
	issueActivities, err := RegisterIssueActivityWorkflow(app, issueActivityStore, nil)
	if err != nil {
		return nil, err
	}
	issueAutomationStore, err := NewPostgresIssueAutomationStore(pool)
	if err != nil {
		return nil, err
	}
	issueAutomation, err := RegisterIssueAutomationWorkflow(app, issueAutomationStore, issueActivities, nil)
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
	emailStore, err := NewPostgresIssueEmailStore(pool, instanceRepository, instancemail.Sender{})
	if err != nil {
		return nil, err
	}
	emailNotifications, err := RegisterIssueEmailNotificationWorkflow(app, emailStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	templatedEmailStore, err := NewPostgresTemplatedEmailStore(pool, instanceRepository, instancemail.Sender{})
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	templatedEmails, err := RegisterTemplatedEmailTaskWorkflow(app, templatedEmailStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	emailTaskAdmission, err := NewTemplatedEmailTaskAdmission(templatedEmails)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	instanceMetricsStore, err := NewPostgresInstanceMetricsStore(pool, otlpInstanceMetricsExporter{})
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	instanceMetrics, err := RegisterInstanceMetricsWorkflow(app, instanceMetricsStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	eventTrackingStore, err := NewPostgresEventTrackingStore(pool, cipher)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	eventTracking, err := RegisterEventTrackingWorkflow(app, eventTrackingStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	workspaceSeedStore, err := NewPostgresWorkspaceSeedStore(pool, os.Getenv("WEB_URL"))
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	workspaceSeed, err := RegisterWorkspaceSeedWorkflow(app, workspaceSeedStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	linkPreviewStore, err := NewPostgresLinkPreviewStore(pool)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	linkPreviews, err := RegisterLinkPreviewWorkflow(app, linkPreviewStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	exportStorageConfig := storageConfig
	exportStorageConfig.SignedURLTTL = 7 * 24 * time.Hour
	exportStorage, err := assets.NewStorage(exportStorageConfig)
	if err != nil {
		_ = emailStore.Close()
		return nil, fmt.Errorf("create issue export S3 client: %w", err)
	}
	issueExportStore, err := NewPostgresIssueExportStore(pool, exportStorage)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	issueExports, err := RegisterIssueExportWorkflow(app, issueExportStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	analyticsExportStore, err := NewPostgresAnalyticsExportStore(pool, instanceRepository, instancemail.Sender{})
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	analyticsExports, err := RegisterAnalyticsExportWorkflow(app, analyticsExportStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	webhookStore, err := NewPostgresWebhookTaskStore(pool, emailTaskAdmission)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	webhooks, err := RegisterWebhookTaskWorkflows(app, webhookStore, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	deletionStore, err := NewPostgresDeletionStore(pool)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	deleteDays, err := HardDeleteDaysFromEnv()
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	deletion, err := RegisterDeletionWorkflows(app, deletionStore, deleteDays, nil)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	effectActivities, err := NewIssueActivityAdmission(issueActivities)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	effectVersions, err := NewIssueDescriptionVersionAdmission(versions)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	effectWebhooks, err := NewWebhookTaskAdmission(webhooks)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	attachmentUploads, err := assets.NewAttachmentApplication(assetRepository, storage, nil, "", assets.AttachmentUploadLimit)
	if err != nil {
		return nil, err
	}
	effectPages, err := NewPageTaskAdmission(pageTransactions, versions, descriptionAssets)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	effectStore, err := NewPostgresWorkItemEffectStore(pool, effectActivities, effectVersions, deletion, effectWebhooks, assetDeleteDispatcher, assetDispatcher, linkPreviews, attachmentUploads, effectPages)
	if err != nil {
		_ = emailStore.Close()
		return nil, err
	}
	workItemEffects := &workItemEffectWorker{store: effectStore}
	closeOnError = false
	return &Runtime{App: app, Logger: logger, Cleanup: cleanup, Assets: assetDispatcher, Exporter: exporterHistory, Versions: versions, IssueDescriptionSync: issueDescriptionSync, IssueVersionSync: issueVersionSync, IssueVersions: issueVersions, PageTransactions: pageTransactions, DescriptionAssets: descriptionAssets, RecentVisits: recentVisits, IssueActivities: issueActivities, IssueAutomation: issueAutomation, EmailNotifications: emailNotifications, TemplatedEmails: templatedEmails, EmailTaskAdmission: emailTaskAdmission, InstanceMetrics: instanceMetrics, IssueExports: issueExports, AnalyticsExports: analyticsExports, EventTracking: eventTracking, WorkspaceSeed: workspaceSeed, LinkPreviews: linkPreviews, Webhooks: webhooks, Deletion: deletion, workItemEffects: workItemEffects, emailStore: emailStore, pool: pool}, nil
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
	if runtime.IssueActivities != nil {
		if err := runtime.IssueActivities.Close(); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
	}
	if runtime.emailStore != nil {
		if err := runtime.emailStore.Close(); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
	}
	if runtime.pool != nil {
		runtime.pool.Close()
	}
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
