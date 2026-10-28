package bootstrap

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/CaliLuke/go-plane/internal/workitemsearch"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
)

//go:embed schema_migrations/*.sql schema_baseline.json schema_native_deletion.json schema_catalog.sql
var schemaFiles embed.FS

const schemaMigrationTable = "plane_schema_migrations"

// MigrateSchema installs the application schema without Python. Existing Django
// databases are adopted only after every baseline definition has been verified.
// Application data and the independent support/River migration ledgers remain intact.
func MigrateSchema(ctx context.Context, databaseURL string) error {
	if err := ValidateDatabaseURL(databaseURL); err != nil {
		return err
	}
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect schema migrator: %w", err)
	}
	// Closing this dedicated connection releases its session lock even on error.
	defer connection.Close(context.WithoutCancel(ctx))
	if _, err := connection.Exec(ctx, `SELECT pg_advisory_lock($1)`, registrationLockID+2); err != nil {
		return fmt.Errorf("lock schema bootstrap: %w", err)
	}
	if _, err := connection.Exec(ctx, `SET search_path TO pg_catalog`); err != nil {
		return err
	}
	expected, err := expectedSchema()
	if err != nil {
		return err
	}
	actual, err := readSchema(ctx, connection)
	if err != nil {
		return err
	}
	_, managed := actual["table:"+schemaMigrationTable]
	if managed {
		var version int64
		var dirty bool
		err := connection.QueryRow(ctx, `SELECT version,dirty FROM public.plane_schema_migrations LIMIT 1`).Scan(&version, &dirty)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("read application migration baseline: %w", err)
		}
		managed = dirty || err == nil && version >= 1
	}
	adopt := false
	adoptVersion := 1
	if !managed {
		for key := range expected {
			if strings.HasPrefix(key, "table:") {
				if _, exists := actual[key]; exists {
					adopt = true
					break
				}
			}
		}
		if adopt {
			if err := validateSchema(expected, actual); err != nil {
				native, nativeErr := schemaManifest("schema_native_deletion.json")
				if nativeErr != nil {
					return nativeErr
				}
				if validateSchema(native, actual) != nil {
					return fmt.Errorf("cannot adopt existing Django schema: %w", err)
				}
				adoptVersion = 2
			}
		}
	}
	parsed, _ := url.Parse(databaseURL) // Validated above.
	parsed.Scheme = "pgx5"
	query := parsed.Query()
	query.Set("x-migrations-table", schemaMigrationTable)
	query.Set("x-statement-timeout", "120000")
	query.Set("search_path", "public")
	parsed.RawQuery = query.Encode()
	source, err := iofs.New(schemaFiles, "schema_migrations")
	if err != nil {
		return fmt.Errorf("open application migrations: %w", err)
	}
	migrator, err := migrate.NewWithSourceInstance("iofs", source, parsed.String())
	if err != nil {
		_ = source.Close()
		return fmt.Errorf("configure application migrations: %w", err)
	}
	defer migrator.Close()
	if adopt {
		// Force is the library's baseline-adoption operation. It is safe only
		// after the complete source schema comparison above has succeeded.
		if err := migrator.Force(adoptVersion); err != nil {
			return fmt.Errorf("record verified Django baseline: %w", err)
		}
	}
	if !managed && !adopt {
		// Verify the original schema before applying native constraint policies.
		if err := migrator.Steps(1); err != nil {
			return fmt.Errorf("install source baseline: %w", err)
		}
		actual, err = readSchema(ctx, connection)
		if err != nil {
			return err
		}
		if err := validateSchema(expected, actual); err != nil {
			return fmt.Errorf("verify source baseline: %w", err)
		}
	}
	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply application schema migrations: %w", err)
	}
	native, err := schemaManifest("schema_native_deletion.json")
	if err != nil {
		return err
	}
	// Migration 31 permits one live edge per relation kind. Keep the original
	// manifest for adoption of pre-migration databases, and verify the replacement
	// constraint after upgrading instead of requiring the retired pair-only index.
	delete(native, "index:issue_relations.issue_relation_unique_issue_related_issue_when_deleted_at_null")
	native["index:issue_relations.issue_relation_unique_kind_live"] = `["CREATE UNIQUE INDEX issue_relation_unique_kind_live ON public.issue_relations USING btree (issue_id, related_issue_id, relation_type) WHERE (deleted_at IS NULL)", true, true]`
	actual, err = readSchema(ctx, connection)
	if err != nil {
		return err
	}
	if err := validateSchema(native, actual); err != nil {
		return fmt.Errorf("verify application schema: %w", err)
	}

	return workitemsearch.Backfill(ctx, connection)
}

func expectedSchema() (map[string]string, error) {
	return schemaManifest("schema_baseline.json")
}

func schemaManifest(name string) (map[string]string, error) {
	data, err := schemaFiles.ReadFile(name)
	if err != nil {
		return nil, err
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode source schema manifest: %w", err)
	}
	return result, nil
}

func readSchema(ctx context.Context, connection *pgx.Conn) (map[string]string, error) {
	query, err := schemaFiles.ReadFile("schema_catalog.sql")
	if err != nil {
		return nil, err
	}
	var data string
	if err := connection.QueryRow(ctx, string(query)).Scan(&data); err != nil {
		return nil, fmt.Errorf("inspect application schema: %w", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("decode application schema: %w", err)
	}
	return result, nil
}

func validateSchema(expected, actual map[string]string) error {
	var differences []string
	for key, definition := range expected {
		if actual[key] != definition {
			differences = append(differences, key)
		}
	}
	if len(differences) == 0 {
		return nil
	}
	sort.Strings(differences)
	count := len(differences)
	if len(differences) > 8 {
		differences = differences[:8]
	}
	return fmt.Errorf("%d missing or incompatible schema definitions (first: %s)", count, strings.Join(differences, ", "))
}
