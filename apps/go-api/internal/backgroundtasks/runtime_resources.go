package backgroundtasks

import (
	"errors"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
)

// runtimeResources owns clients while workflows are being assembled. Successful
// construction transfers them to Runtime; failure and shutdown share this order.
type runtimeResources struct {
	database   *pgxpool.Pool
	activities io.Closer
	emails     io.Closer
}

func (resources runtimeResources) close() error {
	var closeErr error
	if resources.activities != nil {
		closeErr = resources.activities.Close()
	}
	if resources.emails != nil {
		closeErr = errors.Join(closeErr, resources.emails.Close())
	}
	if resources.database != nil {
		resources.database.Close()
	}
	return closeErr
}
