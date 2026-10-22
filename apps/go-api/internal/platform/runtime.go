// Package platform defines the shared process dependencies.
package platform

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// Runtime holds dependencies shared by API services and job workers.
type Runtime struct {
	Database *pgxpool.Pool
	Jobs     *river.Client[pgx.Tx]
}
