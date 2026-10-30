package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/clioutput"
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
		Name:  "background-worker",
		Usage: "run Vector's durable Go background jobs",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "database-url", Sources: cli.EnvVars("DATABASE_URL"), Required: true},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			runtime, err := backgroundtasks.NewRuntime(ctx, command.String("database-url"))
			if err != nil {
				return err
			}
			return runWorker(ctx, runtime, command.Root().Writer)
		},
	}
}

type workerRuntime interface {
	Start(context.Context) error
	Shutdown(context.Context) error
}

func runWorker(ctx context.Context, runtime workerRuntime, writer io.Writer) (resultErr error) {
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := runtime.Shutdown(shutdownCtx); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("shut down background worker: %w", err))
		}
	}()
	if err := runtime.Start(ctx); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, "durable background worker ready"); err != nil {
		return fmt.Errorf("write background worker readiness: %w", err)
	}
	<-ctx.Done()
	return nil
}
