package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func closeMigration(name string, close func() (error, error), operationErr error) error {
	sourceErr, databaseErr := close()
	if sourceErr != nil {
		operationErr = errors.Join(operationErr, fmt.Errorf("close %s migration source: %w", name, sourceErr))
	}
	if databaseErr != nil {
		operationErr = errors.Join(operationErr, fmt.Errorf("close %s migration database: %w", name, databaseErr))
	}
	return operationErr
}

func closeMigrationSource(name string, close func() error, operationErr error) error {
	if err := close(); err != nil {
		return errors.Join(operationErr, fmt.Errorf("close %s migration source: %w", name, err))
	}
	return operationErr
}

func closeSchemaConnection(ctx context.Context, close func(context.Context) error, operationErr error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := close(cleanupCtx); err != nil {
		return errors.Join(operationErr, fmt.Errorf("close schema bootstrap connection: %w", err))
	}
	return operationErr
}
