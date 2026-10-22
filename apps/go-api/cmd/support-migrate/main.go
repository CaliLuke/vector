package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

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
		fmt.Fprintln(stderr, "usage: support-migrate --database-url URL [--migrations-path PATH] up|version")
		return 2
	}
	if flags.Arg(0) != "up" && flags.Arg(0) != "version" {
		fmt.Fprintln(stderr, "command must be up or version")
		return 2
	}
	driverURL, err := supportDatabaseURL(databaseURL)
	if err != nil {
		fmt.Fprintf(stderr, "configure support migrations: %v\n", err)
		return 1
	}
	migration, err := migrate.New("file://"+migrationsPath, driverURL)
	if err != nil {
		fmt.Fprintf(stderr, "configure support migrations: %v\n", err)
		return 1
	}
	defer migration.Close()

	switch flags.Arg(0) {
	case "up":
		if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintf(stderr, "apply support migrations: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "support migrations are current")
	case "version":
		version, dirty, err := migration.Version()
		if err != nil {
			fmt.Fprintf(stderr, "read support migration version: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "version=%d dirty=%t\n", version, dirty)
	}
	return 0
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
