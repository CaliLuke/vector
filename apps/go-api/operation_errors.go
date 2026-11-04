package systemapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	"github.com/CaliLuke/loom/clue/log"
	"github.com/jackc/pgx/v5/pgconn"
)

func logOperationError(ctx context.Context, message, operation, stage string, err error) {
	var httpError *apicontract.HTTPError
	if errors.As(err, &httpError) && httpError.Status < 500 {
		return
	}
	fields := []log.Fielder{
		log.KV{K: "msg", V: message},
		log.KV{K: "operation", V: operation},
		log.KV{K: "stage", V: stage},
		log.KV{K: "error_type", V: fmt.Sprintf("%T", err)},
		log.KV{K: "error_kind", V: operationErrorKind(err)},
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		fields = append(fields,
			log.KV{K: "sqlstate", V: postgres.Code},
			log.KV{K: "table", V: postgres.TableName},
			log.KV{K: "column", V: postgres.ColumnName},
			log.KV{K: "constraint", V: postgres.ConstraintName},
		)
	}
	log.Print(ctx, fields...)
}

func operationErrorKind(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		return "postgres"
	}
	return "other"
}
