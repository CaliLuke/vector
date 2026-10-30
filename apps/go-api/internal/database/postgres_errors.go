package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsPermanentPostgresError reports server failures that need operator or data
// correction. Connection, transaction rollback, resource exhaustion, and shutdown
// classes retain the workers' retry policy. Errors without a PostgreSQL cause
// remain available for classification by their owning subsystem.
func IsPermanentPostgresError(err error) bool {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return false
	}
	if len(postgresError.Code) < 2 {
		return true
	}
	switch postgresError.Code[:2] {
	case "08", "40", "53", "57":
		return false
	default:
		return true
	}
}
