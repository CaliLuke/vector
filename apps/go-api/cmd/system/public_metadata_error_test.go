package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	systemapi "github.com/CaliLuke/go-plane"
	api "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	"github.com/CaliLuke/go-plane/gen/system"
	store "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type publicMetadataFailureDatabase struct {
	calls      int
	cause      error
	failAnchor bool
}

func (db *publicMetadataFailureDatabase) QueryRow(context.Context, string, ...any) pgx.Row {
	db.calls++
	return db
}
func (db *publicMetadataFailureDatabase) Scan(destinations ...any) error {
	if db.calls == 1 && !db.failAnchor {
		if len(destinations) != 2 {
			return errors.New("unexpected anchor destinations")
		}
		slug, ok := destinations[0].(*string)
		if !ok {
			return errors.New("unexpected anchor slug destination")
		}
		project, ok := destinations[1].(**uuid.UUID)
		if !ok {
			return errors.New("unexpected anchor project destination")
		}
		*slug = "demo"
		*project = new(uuid.New())
		return nil
	}
	return db.cause
}

func TestPublicMetadataRepositoryFailuresKeepHTTPPhaseAndSanitization(t *testing.T) {
	for _, endpoint := range []string{"cycles", "labels", "modules", "members", "states"} {
		for _, tc := range []struct {
			name        string
			failAnchor  bool
			cause       error
			status      int
			body        string
			contentType string
			queries     int
		}{
			{"render database error", false, &pgconn.PgError{Code: "08006", Message: "private database diagnostic"}, 500, string(djangoServerErrorBody), "text/html; charset=utf-8", 2},
			{"render wrapped lookup marker", false, fmt.Errorf("private diagnostic: %w", store.ErrInvalidAnchor), 500, string(djangoServerErrorBody), "text/html; charset=utf-8", 2},
			{"anchor not found", true, pgx.ErrNoRows, 404, `{"error":"Invalid anchor"}`, "application/json", 1},
			{"anchor database error", true, &pgconn.PgError{Code: "08006", Message: "private anchor diagnostic"}, 500, `{"error":"Something went wrong please try again later"}`, "application/json", 1},
		} {
			t.Run(endpoint+"/"+tc.name, func(t *testing.T) {
				db := &publicMetadataFailureDatabase{cause: tc.cause, failAnchor: tc.failAnchor}
				repository, err := store.NewRepository(db)
				require.NoError(t, err)
				service, err := systemapi.NewPublicProjectMetadataReads(repository)
				require.NoError(t, err)
				handler := newHTTPHandlerWithPendingReplacements(
					newLogContext(t.Context(), io.Discard, false), system.NewEndpoints(systemapi.NewSystem()), false,
					nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
					pendingReplacementEndpoints{
						publicProjectMetadataReads:    api.NewEndpoints(service),
						publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
					},
				)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/anchor/example/"+endpoint+"/", nil))
				require.Equal(t, tc.status, response.Code, response.Body.String())
				require.Equal(t, tc.body, response.Body.String())
				require.Equal(t, tc.contentType, response.Header().Get("Content-Type"))
				require.Equal(t, tc.queries, db.calls)
			})
		}
	}
}
