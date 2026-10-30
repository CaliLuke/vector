package database

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestPermanentPostgresError(t *testing.T) {
	for _, test := range []struct {
		code      string
		permanent bool
	}{
		{"08006", false}, {"40001", false}, {"40P01", false},
		{"53300", false}, {"57P01", false}, {"23505", true},
		{"42601", true}, {"XX000", true}, {"", true}, {"X", true},
	} {
		t.Run("code_"+test.code, func(t *testing.T) {
			cause := &pgconn.PgError{Code: test.code}
			require.Equal(t, test.permanent, IsPermanentPostgresError(cause))
			require.Equal(t, test.permanent, IsPermanentPostgresError(fmt.Errorf("query: %w", cause)))
		})
	}
	require.False(t, IsPermanentPostgresError(nil))
	require.False(t, IsPermanentPostgresError(errors.New("network unavailable")))
}
