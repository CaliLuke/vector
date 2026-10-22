package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/urfave/cli/v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := newCommand().Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:  "background-worker",
		Usage: "run Plane's durable Go background jobs",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "database-url", Sources: cli.EnvVars("DATABASE_URL"), Required: true},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			runtime, err := backgroundtasks.NewRuntime(ctx, command.String("database-url"))
			if err != nil {
				return err
			}
			if err := runtime.Start(ctx); err != nil {
				_ = runtime.Shutdown(context.Background())
				return err
			}
			fmt.Fprintln(command.Root().Writer, "durable background worker ready")
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return runtime.Shutdown(shutdownCtx)
		},
	}
}
