// Command bootstrap prepares the services required by a Go-only installation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	timeout, err := bootstrapTimeout(os.Getenv("BOOTSTRAP_TIMEOUT"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func bootstrapTimeout(value string) (time.Duration, error) {
	if value == "" {
		return time.Hour, nil
	}
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return 0, fmt.Errorf("BOOTSTRAP_TIMEOUT must be a positive Go duration, for example 2h")
	}
	return timeout, nil
}

func run(ctx context.Context, args []string) error {
	if len(args) == 1 && args[0] == "install" {
		return install(ctx)
	}
	if len(args) == 1 && args[0] == "schema" {
		if err := bootstrap.MigrateSchema(ctx, os.Getenv("DATABASE_URL")); err != nil {
			return err
		}
		fmt.Println("Application schema is current.")
		return nil
	}
	if len(args) == 1 && args[0] == "bucket" {
		config, err := assets.StorageConfigFromEnv()
		if err != nil {
			return err
		}
		storage, err := assets.NewStorage(config)
		if err != nil {
			return err
		}
		if err := storage.EnsureBucket(ctx); err != nil {
			return err
		}
		fmt.Println("Application bucket is ready.")
		return nil
	}
	if len(args) < 2 || args[0] != "instance" {
		return fmt.Errorf("usage: bootstrap install | bootstrap schema | bootstrap bucket | bootstrap instance register <machine-signature> | bootstrap instance configure")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if err := bootstrap.ValidateDatabaseURL(databaseURL); err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	switch args[1] {
	case "register":
		if len(args) != 3 {
			return fmt.Errorf("usage: bootstrap instance register <machine-signature>")
		}
		created, err := bootstrap.RegisterInstance(ctx, pool, bootstrap.Registration{
			MachineSignature: args[2],
			CurrentVersion:   currentVersion(),
			IsTest:           os.Getenv("IS_TEST") == "1",
		}, nil, nil)
		if err != nil {
			return err
		}
		if created {
			fmt.Println("Instance registered.")
		} else {
			fmt.Println("Instance already registered.")
		}
		return nil
	case "configure":
		if len(args) != 2 {
			return fmt.Errorf("usage: bootstrap instance configure")
		}
		if err := bootstrap.ConfigureInstance(ctx, pool, os.Getenv("SECRET_KEY"), os.LookupEnv); err != nil {
			return err
		}
		fmt.Println("Instance configuration defaults are ready.")
		return nil
	default:
		return fmt.Errorf("unknown instance command %q", args[1])
	}
}

func install(ctx context.Context) error {
	databaseURL := os.Getenv("DATABASE_URL")
	storageConfig, err := assets.StorageConfigFromEnv()
	if err != nil {
		return err
	}
	storage, err := assets.NewStorage(storageConfig)
	if err != nil {
		return err
	}
	machineSignature := os.Getenv("MACHINE_SIGNATURE")
	if machineSignature == "" {
		// Registration persists its own instance ID; this bootstrap hint needs
		// no machine hardware fingerprint or Python startup command.
		machineSignature = uuid.NewString()
	}
	// Start the durable runtime only after Install has migrated the application
	// schema and registered the instance. Its periodic jobs query those tables.
	admitMetrics := func(ctx context.Context) error {
		runtime, err := backgroundtasks.NewRuntime(ctx, databaseURL)
		if err != nil {
			return fmt.Errorf("configure durable task runtime: %w", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = runtime.Shutdown(shutdownCtx)
		}()
		if startErr := runtime.Start(ctx); startErr != nil {
			return fmt.Errorf("start durable task runtime: %w", startErr)
		}
		if runtime.InstanceMetrics == nil {
			return fmt.Errorf("durable instance metrics dispatcher is unavailable")
		}
		_, err = runtime.InstanceMetrics.Admit(ctx, uuid.NewString())
		return err
	}
	if err := bootstrap.Install(ctx, bootstrap.Installation{
		DatabaseURL:      databaseURL,
		MachineSignature: machineSignature,
		CurrentVersion:   currentVersion(),
		IsTest:           os.Getenv("IS_TEST") == "1",
		SecretKey:        os.Getenv("SECRET_KEY"),
		LookupEnv:        os.LookupEnv,
		EnsureBucket:     storage.EnsureBucket,
		AdmitMetrics:     admitMetrics,
	}); err != nil {
		return err
	}
	fmt.Println("Go installation bootstrap is ready.")
	return nil
}

func currentVersion() string {
	if version := os.Getenv("APP_VERSION"); version != "" {
		return version
	}
	data, err := os.ReadFile("package.json")
	if err != nil {
		return "v0.1.0"
	}
	var packageInfo struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &packageInfo); err != nil || packageInfo.Version == "" {
		return "v0.1.0"
	}
	return packageInfo.Version
}
