package backgroundtasks

import (
	"fmt"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/instancemail"
	"github.com/CaliLuke/go-plane/internal/instancemanagement"
)

// runtimeAssetWorkflows keeps object storage and its deletion admission together
// for export generation, content copies, and work-item effects.
type runtimeAssetWorkflows struct {
	repository *assets.PostgresRepository
	config     assets.StorageConfig
	storage    *assets.Storage
	deletion   *FileAssetStorageDeleteDispatcher
}

func (runtime *Runtime) registerAssetWorkflows() (runtimeAssetWorkflows, error) {
	assetRepository, err := assets.NewPostgresRepository(runtime.pool)
	if err != nil {
		return runtimeAssetWorkflows{}, err
	}
	storageConfig, err := assets.StorageConfigFromEnv()
	if err != nil {
		return runtimeAssetWorkflows{}, fmt.Errorf("configure FileAsset S3 metadata activity: %w", err)
	}
	storage, err := assets.NewStorage(storageConfig)
	if err != nil {
		return runtimeAssetWorkflows{}, fmt.Errorf("create FileAsset S3 metadata client: %w", err)
	}
	metadataWorkflow, err := assets.RegisterStorageMetadataWorkflow(runtime.App, assetRepository, storage, nil)
	if err != nil {
		return runtimeAssetWorkflows{}, err
	}
	runtime.Assets, err = NewFileAssetMetadataDispatcher(runtime.App, metadataWorkflow)
	if err != nil {
		return runtimeAssetWorkflows{}, err
	}
	assetDeleteWorkflow, err := assets.RegisterStorageObjectDeleteWorkflow(runtime.App, assetRepository, storage, nil)
	if err != nil {
		return runtimeAssetWorkflows{}, err
	}
	assetDeleteDispatcher, err := NewFileAssetStorageDeleteDispatcher(runtime.App, assetDeleteWorkflow)
	if err != nil {
		return runtimeAssetWorkflows{}, err
	}
	exporterStore, err := NewPostgresExporterHistoryStore(runtime.pool, storage)
	if err != nil {
		return runtimeAssetWorkflows{}, err
	}
	runtime.Exporter, err = RegisterExporterHistoryWorkflow(runtime.App, exporterStore, nil)
	if err != nil {
		return runtimeAssetWorkflows{}, err
	}
	return runtimeAssetWorkflows{repository: assetRepository, config: storageConfig, storage: storage, deletion: assetDeleteDispatcher}, nil
}

func (runtime *Runtime) registerContentWorkflows(storage *assets.Storage) error {
	versionStore, err := NewPostgresVersionHistoryStore(runtime.pool)
	if err != nil {
		return err
	}
	runtime.Versions, err = RegisterVersionHistoryWorkflows(runtime.App, versionStore, nil)
	if err != nil {
		return err
	}
	issueDescriptionSyncStore, err := NewPostgresIssueDescriptionSyncStore(runtime.pool)
	if err != nil {
		return err
	}
	runtime.IssueDescriptionSync, err = RegisterIssueDescriptionSyncWorkflow(runtime.App, issueDescriptionSyncStore, nil)
	if err != nil {
		return err
	}
	issueVersionSyncStore, err := NewPostgresIssueVersionSyncStore(runtime.pool)
	if err != nil {
		return err
	}
	runtime.IssueVersionSync, err = RegisterIssueVersionSyncWorkflow(runtime.App, issueVersionSyncStore, nil)
	if err != nil {
		return err
	}
	issueVersionTaskStore, err := NewPostgresIssueVersionTaskStore(runtime.pool)
	if err != nil {
		return err
	}
	runtime.IssueVersions, err = RegisterIssueVersionTaskWorkflow(runtime.App, issueVersionTaskStore, nil)
	if err != nil {
		return err
	}
	pageTransactionStore, err := NewPostgresPageTransactionStore(runtime.pool)
	if err != nil {
		return err
	}
	runtime.PageTransactions, err = RegisterPageTransactionWorkflow(runtime.App, pageTransactionStore, nil)
	if err != nil {
		return err
	}
	descriptionAssetStore, err := NewPostgresDescriptionAssetCopyStore(runtime.pool, storage)
	if err != nil {
		return err
	}
	runtime.DescriptionAssets, err = RegisterDescriptionAssetCopyWorkflow(runtime.App, descriptionAssetStore, nil)
	if err != nil {
		return err
	}
	return nil
}

func (runtime *Runtime) registerEmailWorkflows(instanceRepository *instancemanagement.Repository) error {
	var err error
	runtime.EmailNotifications, err = RegisterIssueEmailNotificationWorkflow(runtime.App, runtime.emailStore, nil)
	if err != nil {
		return err
	}
	templatedEmailStore, err := NewPostgresTemplatedEmailStore(runtime.pool, instanceRepository, instancemail.Sender{})
	if err != nil {
		return err
	}
	runtime.TemplatedEmails, err = RegisterTemplatedEmailTaskWorkflow(runtime.App, templatedEmailStore, nil)
	if err != nil {
		return err
	}
	runtime.EmailTaskAdmission, err = NewTemplatedEmailTaskAdmission(runtime.TemplatedEmails)
	if err != nil {
		return err
	}
	return nil
}

func (runtime *Runtime) registerExportWorkflows(storageConfig assets.StorageConfig, instanceRepository *instancemanagement.Repository) error {
	exportStorageConfig := storageConfig
	exportStorageConfig.SignedURLTTL = 7 * 24 * time.Hour
	exportStorage, err := assets.NewStorage(exportStorageConfig)
	if err != nil {
		return fmt.Errorf("create issue export S3 client: %w", err)
	}
	issueExportStore, err := NewPostgresIssueExportStore(runtime.pool, exportStorage)
	if err != nil {
		return err
	}
	runtime.IssueExports, err = RegisterIssueExportWorkflow(runtime.App, issueExportStore, nil)
	if err != nil {
		return err
	}
	analyticsExportStore, err := NewPostgresAnalyticsExportStore(runtime.pool, instanceRepository, instancemail.Sender{})
	if err != nil {
		return err
	}
	runtime.AnalyticsExports, err = RegisterAnalyticsExportWorkflow(runtime.App, analyticsExportStore, nil)
	if err != nil {
		return err
	}
	return nil
}

func (runtime *Runtime) configureWorkItemEffects(assetWorkflows runtimeAssetWorkflows) error {
	effectActivities, err := NewIssueActivityAdmission(runtime.IssueActivities)
	if err != nil {
		return err
	}
	effectVersions, err := NewIssueDescriptionVersionAdmission(runtime.Versions)
	if err != nil {
		return err
	}
	effectWebhooks, err := NewWebhookTaskAdmission(runtime.Webhooks)
	if err != nil {
		return err
	}
	attachmentUploads, err := assets.NewAttachmentApplication(assetWorkflows.repository, assetWorkflows.storage, nil, "", assets.AttachmentUploadLimit)
	if err != nil {
		return err
	}
	effectPages, err := NewPageTaskAdmission(runtime.PageTransactions, runtime.Versions, runtime.DescriptionAssets)
	if err != nil {
		return err
	}
	effectStore, err := NewPostgresWorkItemEffectStore(runtime.pool, effectActivities, effectVersions, runtime.Deletion, effectWebhooks, assetWorkflows.deletion, runtime.Assets, runtime.LinkPreviews, attachmentUploads, effectPages)
	if err != nil {
		return err
	}
	runtime.workItemEffects = &workItemEffectWorker{store: effectStore}
	return nil
}
