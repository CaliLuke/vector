package publicprojectmetadatareads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestCatalogueRenderFailuresPreserveDatabaseCause(t *testing.T) {
	for _, tc := range []struct {
		name string
		read func(*Repository, context.Context, string) (json.RawMessage, error)
	}{
		{"cycles", (*Repository).Cycles}, {"labels", (*Repository).Labels},
		{"modules", (*Repository).Modules}, {"members", (*Repository).Members},
		{"states", (*Repository).States},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cause := &pgconn.PgError{Code: "08006", Message: "private database diagnostic"}
			database := &databaseStub{rows: []pgx.Row{anchorRow(nil), rowFunc(func(...any) error {
				return fmt.Errorf("read rows: %w", cause)
			})}}
			repository, err := NewRepository(database)
			require.NoError(t, err)
			data, err := tc.read(repository, t.Context(), "anchor")
			require.Empty(t, data)
			require.ErrorIs(t, err, ErrRender)
			require.ErrorIs(t, err, cause)
			var typed *pgconn.PgError
			require.ErrorAs(t, err, &typed)
			require.Same(t, cause, typed)
			require.Len(t, database.queries, 2)
		})
	}
}

func TestStatesRenderFailurePreservesDecodeCause(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		check   func(*testing.T, error)
	}{
		{"invalid json", `[{`, func(t *testing.T, err error) {
			var cause *json.SyntaxError
			require.ErrorAs(t, err, &cause)
		}},
		{"wrong state type", `[{"sequence":true}]`, func(t *testing.T, err error) {
			var cause *json.UnmarshalTypeError
			require.ErrorAs(t, err, &cause)
		}},
		{"invalid sequence", `[{"sequence":"invalid"}]`, func(t *testing.T, err error) {
			var cause *strconv.NumError
			require.ErrorAs(t, err, &cause)
			require.ErrorIs(t, err, strconv.ErrSyntax)
		}},
		{"overflow sequence", `[{"sequence":"1e1000"}]`, func(t *testing.T, err error) {
			require.ErrorIs(t, err, strconv.ErrRange)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database := &databaseStub{rows: []pgx.Row{anchorRow(nil), jsonRow(tc.payload)}}
			repository, err := NewRepository(database)
			require.NoError(t, err)
			data, err := repository.States(t.Context(), "anchor")
			require.Empty(t, data)
			require.ErrorIs(t, err, ErrRender)
			tc.check(t, err)
			require.Len(t, database.queries, 2)
		})
	}
}

func TestCatalogueAnchorFailureKeepsLookupPhase(t *testing.T) {
	cause := errors.New("private anchor lookup diagnostic")
	database := &databaseStub{rows: []pgx.Row{rowFunc(func(...any) error { return cause })}}
	repository, err := NewRepository(database)
	require.NoError(t, err)
	data, err := repository.Labels(t.Context(), "anchor")
	require.Empty(t, data)
	require.ErrorIs(t, err, cause)
	require.NotErrorIs(t, err, ErrRender)
	require.Len(t, database.queries, 1)
}
