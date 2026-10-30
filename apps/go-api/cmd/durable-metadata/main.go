// durable-metadata is an isolated prototype entry point. It requires an
// explicitly configured database URL so it cannot attach to Vector's live
// application database by accident.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/clioutput"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/urfave/cli/v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := newCommand().Run(ctx, os.Args); err != nil {
		os.Exit(clioutput.Report(os.Stderr, 1, "%v\n", err))
	}
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:  "durable-metadata",
		Usage: "run the isolated Selvedge FileAsset metadata prototype",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "database-url", Sources: cli.EnvVars("PLANE_DURABLE_METADATA_DATABASE_URL"), Required: true},
		},
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "admit a stable asset/event run and wait for its result",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "asset-id", Required: true},
					&cli.StringFlag{Name: "event-id", Required: true},
					&cli.DurationFlag{Name: "timeout", Value: 10 * time.Minute},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					assetID, err := uuid.Parse(command.String("asset-id"))
					if err != nil {
						return fmt.Errorf("parse asset-id: %w", err)
					}
					if command.String("event-id") == "" {
						return errors.New("event-id is required")
					}
					runID := fmt.Sprintf("asset-metadata:%s:%s", assetID, command.String("event-id"))
					waitCtx, cancel := context.WithTimeout(ctx, command.Duration("timeout"))
					defer cancel()
					app, workflow, closeApp, err := startApp(waitCtx, command.String("database-url"))
					if err != nil {
						return err
					}
					input := assets.StorageMetadataWorkflowInput{AssetID: assetID.String(), EventID: command.String("event-id")}
					return runMetadata(command.Root().Writer,
						func() (string, bool, error) { return startOrReconcile(waitCtx, app, workflow, runID, input) },
						func(id string) (*selvedge.WorkflowResult[struct{}], error) {
							return selvedge.WaitWorkflow[struct{}](waitCtx, app, id)
						}, closeApp)
				},
			},
			{
				Name:  "worker",
				Usage: "start a worker and recover pending metadata runs",
				Action: func(ctx context.Context, command *cli.Command) error {
					_, _, closeApp, err := startApp(ctx, command.String("database-url"))
					if err != nil {
						return err
					}
					return runMetadataWorker(ctx, command.Root().Writer, closeApp)
				},
			},
			{
				Name:  "status",
				Usage: "read persisted workflow status without starting workers",
				Flags: []cli.Flag{&cli.StringFlag{Name: "run-id", Required: true}},
				Action: func(ctx context.Context, command *cli.Command) error {
					pool, err := pgxpool.New(ctx, command.String("database-url"))
					if err != nil {
						return fmt.Errorf("configure isolated Vector database: %w", err)
					}
					defer pool.Close()
					if pingErr := pool.Ping(ctx); pingErr != nil {
						return fmt.Errorf("connect isolated Vector database: %w", pingErr)
					}
					var workflowName, status string
					err = pool.QueryRow(ctx, `SELECT workflow_name, status FROM workflow_instances WHERE instance_id=$1`, command.String("run-id")).Scan(&workflowName, &status)
					if err != nil {
						return fmt.Errorf("read Selvedge workflow instance %s: %w", command.String("run-id"), err)
					}
					return writeStatus(command.Root().Writer, command.String("run-id"), workflowName, status)
				},
			},
		},
	}
}

func startOrReconcile(ctx context.Context, app *selvedge.App, workflow *selvedge.WorkflowFunc[assets.StorageMetadataWorkflowInput, struct{}], runID string, input assets.StorageMetadataWorkflowInput) (string, bool, error) {
	startedID, startErr := selvedge.StartWorkflow(ctx, app, workflow, input, selvedge.WithInstanceID(runID))
	if startErr == nil {
		return startedID, false, nil
	}
	instance, lookupErr := app.GetInstance(ctx, runID)
	if lookupErr != nil {
		return "", false, fmt.Errorf("admit workflow %s: %w (reconcile stored run: %v)", runID, startErr, lookupErr)
	}
	expectedInput, marshalErr := json.Marshal(input)
	if marshalErr != nil {
		return "", false, fmt.Errorf("encode expected workflow input: %w", marshalErr)
	}
	if instance.WorkflowName != workflow.Name() || !bytes.Equal(instance.InputData, expectedInput) {
		return "", false, fmt.Errorf("workflow ID %s already belongs to workflow %q with different input; original admission error: %w", runID, instance.WorkflowName, startErr)
	}
	return runID, true, nil
}

func startApp(ctx context.Context, databaseURL string) (*selvedge.App, *selvedge.WorkflowFunc[assets.StorageMetadataWorkflowInput, struct{}], func() error, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("configure isolated Vector database: %w", err)
	}
	if pingErr := pool.Ping(ctx); pingErr != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("connect isolated Vector database: %w", pingErr)
	}
	store, err := assets.NewPostgresRepository(pool)
	if err != nil {
		pool.Close()
		return nil, nil, nil, err
	}
	storageConfig, err := assets.StorageConfigFromEnv()
	if err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("configure isolated assets object store: %w", err)
	}
	objects, err := assets.NewStorage(storageConfig)
	if err != nil {
		pool.Close()
		return nil, nil, nil, err
	}
	errPolicy := retry.Exponential(8, time.Second, time.Minute, 2)
	app := selvedge.NewApp(
		selvedge.WithDatabase(databaseURL),
		selvedge.WithServiceName("plane-durable-metadata-prototype"),
		selvedge.WithActivityPollInterval(100*time.Millisecond),
		selvedge.WithWorkflowResumptionInterval(100*time.Millisecond),
	)
	workflow, err := assets.RegisterStorageMetadataWorkflow(app, store, objects, errPolicy)
	if err != nil {
		pool.Close()
		return nil, nil, nil, err
	}
	if err := app.Start(ctx); err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("start Selvedge app: %w", err)
	}
	closeApp := func() error { return closeMetadataApp(ctx, app.Shutdown, pool.Close) }
	return app, workflow, closeApp, nil
}

func writeResult(output interface{ Write([]byte) (int, error) }, runID, status string, runErr error) error {
	result := map[string]any{"run_id": runID, "status": status}
	if runErr != nil {
		result["error"] = runErr.Error()
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(encoded))
	return err
}

func writeStatus(output interface{ Write([]byte) (int, error) }, runID, workflowName, status string) error {
	encoded, err := json.Marshal(map[string]string{"run_id": runID, "workflow": workflowName, "status": status})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(encoded))
	return err
}

func runMetadata(output io.Writer, admit func() (string, bool, error), wait func(string) (*selvedge.WorkflowResult[struct{}], error), closeApp func() error) (resultErr error) {
	defer func() { resultErr = errors.Join(resultErr, closeApp()) }()
	startedID, reconciled, err := admit()
	if err != nil {
		return err
	}
	if _, outputErr := fmt.Fprintf(output, "admitted run_id=%s reconciled=%t\n", startedID, reconciled); outputErr != nil {
		return fmt.Errorf("write workflow %s admission: %w", startedID, outputErr)
	}
	result, err := wait(startedID)
	if err != nil {
		return fmt.Errorf("wait for workflow %s: %w", startedID, err)
	}
	var workflowErr error
	if result.Status != "completed" {
		workflowErr = result.Error
		if workflowErr == nil {
			workflowErr = fmt.Errorf("workflow %s ended with status %q", result.InstanceID, result.Status)
		}
	}
	outputErr := writeResult(output, result.InstanceID, result.Status, result.Error)
	if outputErr != nil {
		outputErr = fmt.Errorf("write workflow %s result: %w", result.InstanceID, outputErr)
	}
	return errors.Join(workflowErr, outputErr)
}

func runMetadataWorker(ctx context.Context, output io.Writer, closeApp func() error) (resultErr error) {
	defer func() { resultErr = errors.Join(resultErr, closeApp()) }()
	if _, err := fmt.Fprintln(output, "worker ready"); err != nil {
		return fmt.Errorf("write metadata worker readiness: %w", err)
	}
	<-ctx.Done()
	return nil
}

func closeMetadataApp(ctx context.Context, shutdown func(context.Context) error, closePool func()) error {
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	defer closePool()
	if err := shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down metadata app: %w", err)
	}
	return nil
}
