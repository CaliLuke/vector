package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-plane/internal/instancecipher"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Installation describes the idempotent operations needed before the Go
// application starts serving requests. External effects are injected so the
// bootstrap sequence can use the production storage and durable task clients.
type Installation struct {
	DatabaseURL      string
	MachineSignature string
	CurrentVersion   string
	IsTest           bool
	SecretKey        string
	LookupEnv        func(string) (string, bool)
	EnsureBucket     func(context.Context) error
	LatestVersion    LatestVersion
	AdmitMetrics     MetricsAdmission
}

// Install prepares the schema, encrypted configuration, storage bucket,
// registration row, and source-required metrics task. Missing task admission
// is checked before any state is changed.
func Install(ctx context.Context, installation Installation) error {
	if err := ValidateDatabaseURL(installation.DatabaseURL); err != nil {
		return err
	}
	if strings.TrimSpace(installation.MachineSignature) == "" {
		return errors.New("machine signature is required")
	}
	if installation.SecretKey == "" {
		return errors.New("SECRET_KEY env variable is required.")
	}
	if installation.LookupEnv == nil {
		return errors.New("environment lookup is required")
	}
	if installation.EnsureBucket == nil {
		return errors.New("application bucket initializer is required")
	}
	if installation.AdmitMetrics == nil {
		return ErrMetricsAdmissionUnavailable
	}
	// Prepare encrypted values before applying any migration or object-store
	// effect. ConfigureInstance recomputes these values within its transaction.
	cipher, err := instancecipher.New(installation.SecretKey)
	if err != nil {
		return fmt.Errorf("initialize instance configuration encryption: %w", err)
	}
	if _, instanceConfigurationsErr := InstanceConfigurations(installation.LookupEnv, cipher); instanceConfigurationsErr != nil {
		return instanceConfigurationsErr
	}
	if migrateSchemaErr := MigrateSchema(ctx, installation.DatabaseURL); migrateSchemaErr != nil {
		return migrateSchemaErr
	}
	pool, err := pgxpool.New(ctx, installation.DatabaseURL)
	if err != nil {
		return fmt.Errorf("configure bootstrap PostgreSQL pool: %w", err)
	}
	defer pool.Close()
	if pingErr := pool.Ping(ctx); pingErr != nil {
		return fmt.Errorf("connect bootstrap PostgreSQL pool: %w", pingErr)
	}
	if migrateRuntimeSchemasErr := MigrateRuntimeSchemas(ctx, installation.DatabaseURL, pool); migrateRuntimeSchemasErr != nil {
		return migrateRuntimeSchemasErr
	}
	if configureInstanceErr := ConfigureInstance(ctx, pool, installation.SecretKey, installation.LookupEnv); configureInstanceErr != nil {
		return configureInstanceErr
	}
	if ensureBucketErr := installation.EnsureBucket(ctx); ensureBucketErr != nil {
		return ensureBucketErr
	}
	_, err = RegisterInstance(ctx, pool, Registration{
		MachineSignature: installation.MachineSignature,
		CurrentVersion:   installation.CurrentVersion,
		IsTest:           installation.IsTest,
	}, installation.LatestVersion, installation.AdmitMetrics)
	return err
}
