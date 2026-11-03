package main

import (
	"context"
	"fmt"

	workitemstore "github.com/CaliLuke/go-plane/internal/workitems"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/database"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/CaliLuke/selvedge"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (assembly *systemAssembly) database() error {
	var err error
	if assembly.cfg.databaseURL == "" {
		return fmt.Errorf("database-url is required")
	}
	if !assembly.cfg.goBackgroundTasks {
		return fmt.Errorf("native background tasks are required for transactional work item effects; start with --go-background-tasks")
	}
	poolConfig, err := pgxpool.ParseConfig(assembly.cfg.databaseURL)
	if err != nil {
		return fmt.Errorf("configure database: %w", err)
	}
	if configurePoolErr := database.ConfigurePool(poolConfig); configurePoolErr != nil {
		return fmt.Errorf("configure database adapters: %w", configurePoolErr)
	}
	assembly.pool, err = pgxpool.NewWithConfig(assembly.ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	assembly.onClose(func(_ *error) { assembly.pool.Close() })
	if pingErr := assembly.pool.Ping(assembly.ctx); pingErr != nil {
		return fmt.Errorf("connect database: %w", pingErr)
	}

	return nil
}

func (assembly *systemAssembly) backgroundRuntime() error {
	var err error
	var publicProjectMetadataActivityLogger publicprojectmetadatareadsstore.ActivityLogger
	var runtimeOptions []selvedge.Option
	if assembly.cfg.goBackgroundTasksNoWorkers {
		runtimeOptions = append(runtimeOptions, selvedge.WithActivityQueues())
	}
	assembly.backgroundTaskRuntime, err = backgroundtasks.NewRuntime(assembly.ctx, assembly.cfg.databaseURL, runtimeOptions...)
	if err != nil {
		return err
	}
	assembly.onClose(func(_ *error) {
		shutdownContext, cancel := context.WithTimeout(context.Background(), assembly.cfg.shutdownTimeout)
		defer cancel()
		if shutdownErr := assembly.backgroundTaskRuntime.Shutdown(shutdownContext); shutdownErr != nil {
			log.Error(assembly.ctx, shutdownErr)
		}
	})
	if startErr := assembly.backgroundTaskRuntime.Start(assembly.ctx); startErr != nil {
		return startErr
	}
	assembly.pageTaskAdmission, err = backgroundtasks.NewPageTaskAdmission(assembly.backgroundTaskRuntime.PageTransactions, assembly.backgroundTaskRuntime.Versions, assembly.backgroundTaskRuntime.DescriptionAssets)
	if err != nil {
		return err
	}
	assembly.issueDescriptionVersionAdmission, err = backgroundtasks.NewIssueDescriptionVersionAdmission(assembly.backgroundTaskRuntime.Versions)
	if err != nil {
		return err
	}
	assembly.recentVisitAdmission, err = backgroundtasks.NewRecentVisitAdmission(assembly.backgroundTaskRuntime.RecentVisits)
	if err != nil {
		return err
	}
	assembly.issueActivityAdmission, err = backgroundtasks.NewIssueActivityAdmission(assembly.backgroundTaskRuntime.IssueActivities)
	if err != nil {
		return err
	}
	assembly.emailTaskAdmission = assembly.backgroundTaskRuntime.EmailTaskAdmission
	assembly.webhookTaskAdmission, err = backgroundtasks.NewWebhookTaskAdmission(assembly.backgroundTaskRuntime.Webhooks)
	if err != nil {
		return err
	}
	assembly.issueExportAdmission = assembly.backgroundTaskRuntime.IssueExports
	assembly.analyticsExportDispatcher = assembly.backgroundTaskRuntime.AnalyticsExports
	assembly.eventTrackingAdmission = assembly.backgroundTaskRuntime.EventTracking
	assembly.assetMetadataDispatcher = assembly.backgroundTaskRuntime.Assets
	assembly.workspaceSeedDispatcher = assembly.backgroundTaskRuntime.WorkspaceSeed
	assembly.linkPreviewDispatcher = assembly.backgroundTaskRuntime.LinkPreviews
	assembly.deletionDispatcher = assembly.backgroundTaskRuntime.Deletion
	publicProjectMetadataActivityLogger = assembly.backgroundTaskRuntime.Logger
	assembly.routes.userIdentityActivity = publicProjectMetadataActivityLogger
	assembly.routes.publicProjectMetadataActivity = publicProjectMetadataActivityLogger
	assembly.routes.activityReadsActivity = publicProjectMetadataActivityLogger
	return nil
}

func (assembly *systemAssembly) readReplica() error {
	assembly.activityReadsPool = assembly.pool
	if assembly.cfg.readReplicaDatabaseURL != "" {
		readReplicaConfig, poolErr := pgxpool.ParseConfig(assembly.cfg.readReplicaDatabaseURL)
		if poolErr != nil {
			return fmt.Errorf("configure read replica database: %w", poolErr)
		}
		if configurePoolErr := database.ConfigurePool(readReplicaConfig); configurePoolErr != nil {
			return fmt.Errorf("configure read replica database adapters: %w", configurePoolErr)
		}
		assembly.activityReadsPool, poolErr = pgxpool.NewWithConfig(assembly.ctx, readReplicaConfig)
		if poolErr != nil {
			return fmt.Errorf("create read replica database pool: %w", poolErr)
		}
		assembly.onClose(func(_ *error) { assembly.activityReadsPool.Close() })
		if pingErr := assembly.activityReadsPool.Ping(assembly.ctx); pingErr != nil {
			return fmt.Errorf("connect read replica database: %w", pingErr)
		}
	}

	return nil
}

func (assembly *systemAssembly) activityPublisher() error {
	var err error
	assembly.activityTaskPublisher, err = assembly.commonTaskPublisher()
	if err != nil {
		return err
	}
	if assembly.assetMetadataDispatcher != nil {
		assembly.activityTaskPublisher, err = backgroundtasks.NewFileAssetMetadataTaskPublisher(assembly.activityTaskPublisher, assembly.assetMetadataDispatcher)
		if err != nil {
			return err
		}
	}
	if assembly.workspaceSeedDispatcher != nil {
		assembly.activityTaskPublisher, err = backgroundtasks.NewWorkspaceSeedTaskPublisher(assembly.activityTaskPublisher, assembly.workspaceSeedDispatcher)
		if err != nil {
			return err
		}
	}
	if assembly.linkPreviewDispatcher != nil {
		assembly.activityTaskPublisher, err = backgroundtasks.NewLinkPreviewTaskPublisher(assembly.activityTaskPublisher, assembly.linkPreviewDispatcher)
		if err != nil {
			return err
		}
	}
	assembly.activityTaskPublisher, err = assembly.withVersionTasks(assembly.activityTaskPublisher)
	return err
}

func (assembly *systemAssembly) commonTaskPublisher() (workitemstore.Publisher, error) {
	publisher, err := issueActivityTaskPublisher(rejectingTaskPublisher{}, assembly.issueActivityAdmission)
	if err != nil {
		return nil, err
	}
	if assembly.emailTaskAdmission != nil {
		publisher, err = backgroundtasks.NewTemplatedEmailTaskPublisher(publisher, assembly.emailTaskAdmission)
		if err != nil {
			return nil, err
		}
	}
	if assembly.webhookTaskAdmission != nil {
		publisher, err = backgroundtasks.NewWebhookTaskPublisher(publisher, assembly.webhookTaskAdmission)
		if err != nil {
			return nil, err
		}
	}
	if assembly.deletionDispatcher != nil {
		publisher, err = backgroundtasks.NewDeletionTaskPublisher(publisher, assembly.deletionDispatcher)
		if err != nil {
			return nil, err
		}
	}
	publisher, err = backgroundtasks.NewEventTrackingTaskPublisher(publisher, assembly.eventTrackingAdmission)
	if err != nil {
		return nil, err
	}
	return publisher, nil
}

func (assembly *systemAssembly) withVersionTasks(publisher workitemstore.Publisher) (workitemstore.Publisher, error) {
	var err error
	if assembly.backgroundTaskRuntime != nil {
		publisher, err = backgroundtasks.NewVersionTaskPublisher(
			publisher,
			assembly.backgroundTaskRuntime.IssueVersions,
			assembly.backgroundTaskRuntime.IssueVersionSync,
			assembly.backgroundTaskRuntime.IssueDescriptionSync,
			assembly.backgroundTaskRuntime.Logger,
		)
		if err != nil {
			return nil, err
		}
	}
	return publisher, nil
}
