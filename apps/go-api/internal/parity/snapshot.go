package parity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Column describes one PostgreSQL table column without exposing row data.
type Column struct {
	Name     string
	DataType string
	Nullable bool
}

// TableDump contains a complete, primary-key-ordered application table dump.
type TableDump struct {
	Name       string
	Columns    []Column
	PrimaryKey []string
	Rows       []json.RawMessage
}

// Snapshot contains every table in the selected Plane application-table inventory.
type Snapshot struct {
	Tables []TableDump
}

func snapshotDatabase(
	ctx context.Context,
	role string,
	postgresURL string,
	schema string,
) (Snapshot, error) {
	pool, err := pgxpool.New(ctx, postgresURL)
	if err != nil {
		return Snapshot{}, safeDatabaseError(role, "configure connection", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return Snapshot{}, safeDatabaseError(role, "connect", err)
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Snapshot{}, safeDatabaseError(role, "begin read-only snapshot", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	tableNames, err := PlaneTables()
	if err != nil {
		return Snapshot{}, err
	}
	snapshot := Snapshot{Tables: make([]TableDump, 0, len(tableNames))}
	for _, tableName := range tableNames {
		table, err := dumpTable(ctx, tx, schema, tableName)
		if err != nil {
			return Snapshot{}, safeDatabaseError(role, "dump table "+tableName, err)
		}
		snapshot.Tables = append(snapshot.Tables, table)
	}
	if err := tx.Commit(ctx); err != nil {
		return Snapshot{}, safeDatabaseError(role, "finish read-only snapshot", err)
	}
	return snapshot, nil
}

func dumpTable(
	ctx context.Context,
	tx pgx.Tx,
	schema string,
	tableName string,
) (TableDump, error) {
	columns, err := tableColumns(ctx, tx, schema, tableName)
	if err != nil {
		return TableDump{}, err
	}
	if len(columns) == 0 {
		return TableDump{}, fmt.Errorf("table is missing from selected schema")
	}
	primaryKey, err := tablePrimaryKey(ctx, tx, schema, tableName)
	if err != nil {
		return TableDump{}, err
	}
	if len(primaryKey) == 0 {
		return TableDump{}, fmt.Errorf("table has no primary key")
	}

	qualifiedTable := pgx.Identifier{schema, tableName}.Sanitize()
	orderColumns := make([]string, len(primaryKey))
	for index, column := range primaryKey {
		orderColumns[index] = pgx.Identifier{column}.Sanitize()
	}
	query := fmt.Sprintf(
		"SELECT to_jsonb(ordered_row) FROM (SELECT * FROM %s ORDER BY %s) AS ordered_row",
		qualifiedTable,
		strings.Join(orderColumns, ", "),
	)
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return TableDump{}, err
	}
	defer rows.Close()

	dump := TableDump{Name: tableName, Columns: columns, PrimaryKey: primaryKey}
	for rowIndex := 0; rows.Next(); rowIndex++ {
		var rawRow []byte
		if err := rows.Scan(&rawRow); err != nil {
			return TableDump{}, fmt.Errorf("scan row %d: %w", rowIndex, err)
		}
		dump.Rows = append(dump.Rows, json.RawMessage(rawRow))
	}
	if err := rows.Err(); err != nil {
		return TableDump{}, err
	}
	return dump, nil
}

func tableColumns(ctx context.Context, tx pgx.Tx, schema string, table string) ([]Column, error) {
	const query = `
SELECT attribute.attname,
       pg_catalog.format_type(attribute.atttypid, attribute.atttypmod),
       NOT attribute.attnotnull
FROM pg_catalog.pg_attribute AS attribute
JOIN pg_catalog.pg_class AS relation ON relation.oid = attribute.attrelid
JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
WHERE namespace.nspname = $1
  AND relation.relname = $2
  AND relation.relkind IN ('r', 'p')
  AND attribute.attnum > 0
  AND NOT attribute.attisdropped
ORDER BY attribute.attnum`
	rows, err := tx.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var column Column
		if err := rows.Scan(&column.Name, &column.DataType, &column.Nullable); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, rows.Err()
}

func tablePrimaryKey(ctx context.Context, tx pgx.Tx, schema string, table string) ([]string, error) {
	const query = `
SELECT attribute.attname
FROM pg_catalog.pg_index AS index
JOIN pg_catalog.pg_class AS relation ON relation.oid = index.indrelid
JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
JOIN LATERAL unnest(index.indkey) WITH ORDINALITY AS key(attnum, position) ON true
JOIN pg_catalog.pg_attribute AS attribute
  ON attribute.attrelid = relation.oid AND attribute.attnum = key.attnum
WHERE namespace.nspname = $1
  AND relation.relname = $2
  AND index.indisprimary
ORDER BY key.position`
	rows, err := tx.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, rows.Err()
}

func safeDatabaseError(role string, operation string, err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		return fmt.Errorf("%s database %s failed with PostgreSQL code %s", role, operation, postgresError.Code)
	}
	return fmt.Errorf("%s database %s failed (%T)", role, operation, err)
}
