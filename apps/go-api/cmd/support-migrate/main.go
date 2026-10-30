package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/CaliLuke/go-plane/internal/clioutput"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("support-migrate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var databaseURL string
	var migrationsPath string
	flags.StringVar(&databaseURL, "database-url", "", "PostgreSQL URL (required)")
	flags.StringVar(&migrationsPath, "migrations-path", "migrations/support", "support migration directory")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if databaseURL == "" || flags.NArg() != 1 {
		return clioutput.Report(stderr, 2, "usage: support-migrate --database-url URL [--migrations-path PATH] up|version\n")
	}
	if flags.Arg(0) != "up" && flags.Arg(0) != "version" {
		return clioutput.Report(stderr, 2, "command must be up or version\n")
	}
	driverURL, err := supportDatabaseURL(databaseURL)
	if err != nil {
		return clioutput.Report(stderr, 1, "configure support migrations: %v\n", err)
	}
	migration, err := migrate.New("file://"+migrationsPath, driverURL)
	if err != nil {
		return clioutput.Report(stderr, 1, "configure support migrations: %v\n", err)
	}
	if err := executeSupportMigration(migration, flags.Arg(0), stdout); err != nil {
		return clioutput.Report(stderr, 1, "%v\n", err)
	}
	return 0
}

type supportMigration interface {
	Up() error
	Version() (uint, bool, error)
	Close() (error, error)
}

func executeSupportMigration(migration supportMigration, command string, stdout io.Writer) (resultErr error) {
	defer func() {
		sourceErr, databaseErr := migration.Close()
		if sourceErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close migration source: %w", sourceErr))
		}
		if databaseErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close migration database: %w", databaseErr))
		}
	}()
	switch command {
	case "up":
		if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("apply support migrations: %w", err)
		}
		if _, err := fmt.Fprintln(stdout, "support migrations are current"); err != nil {
			return fmt.Errorf("write migration result: %w", err)
		}
	case "version":
		version, dirty, err := migration.Version()
		if err != nil {
			return fmt.Errorf("read support migration version: %w", err)
		}
		if _, err := fmt.Fprintf(stdout, "version=%d dirty=%t\n", version, dirty); err != nil {
			return fmt.Errorf("write migration version: %w", err)
		}
	}
	return nil
}

func supportDatabaseURL(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse PostgreSQL URL: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" && parsed.Scheme != "pgx5" {
		return "", fmt.Errorf("database URL must use postgres, postgresql, or pgx5 scheme")
	}
	parsed.Scheme = "pgx5"
	return parsed.String(), nil
}
