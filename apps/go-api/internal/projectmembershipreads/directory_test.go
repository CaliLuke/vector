package projectmembershipreads

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type directoryFailureDB struct {
	calls int
	err   error
}

func (d *directoryFailureDB) QueryRow(context.Context, string, ...any) pgx.Row {
	d.calls++
	return rowFunc(func(...any) error { return d.err })
}

func TestMemberDirectoryRejectsInvalidQueriesWithoutReading(t *testing.T) {
	_, err := NewDirectoryReader(nil)
	require.Error(t, err)
	db := &directoryFailureDB{err: errors.New("unexpected database read")}
	reader, err := NewDirectoryReader(db)
	require.NoError(t, err)
	actor := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	for _, q := range []WorkspaceMembersQuery{
		{Limit: 0}, {Limit: -1}, {Limit: 1001}, {Limit: 100, Page: -1},
		{Limit: 1000, Page: int(^uint(0) >> 1)},
		{Limit: 10, OrderBy: "email; DROP TABLE users"},
		{Limit: 10, OrderBy: "email,-email"},
		{Limit: 10, OrderBy: "email,"},
		{Limit: 10, OrderBy: "--created_at"},
		{Limit: 10, OrderBy: "unknown"},
	} {
		_, err = reader.ListWorkspaceMembers(t.Context(), actor, "current", q)
		assert.ErrorIs(t, err, ErrInvalidMemberQuery)
	}
	for _, q := range []struct {
		ns          string
		limit, page int
	}{{"other", 10, 0}, {"", 0, 0}, {"", 1001, 0}, {"", 10, -1}} {
		_, _, err = reader.ListRoles(t.Context(), actor, "current", q.ns, q.limit, q.page)
		assert.ErrorIs(t, err, ErrInvalidMemberQuery)
	}
	assert.Zero(t, db.calls)
}

func TestMemberDirectoryPropagatesDatabaseCancellation(t *testing.T) {
	db := &directoryFailureDB{err: context.Canceled}
	reader, err := NewDirectoryReader(db)
	require.NoError(t, err)
	actor := uuid.MustParse("10000000-0000-4000-8000-000000000001")
	_, err = reader.ListWorkspaceMembers(t.Context(), actor, "current", WorkspaceMembersQuery{Limit: 10})
	assert.ErrorIs(t, err, context.Canceled)
	_, _, err = reader.ListRoles(t.Context(), actor, "current", "", 10, 0)
	assert.ErrorIs(t, err, context.Canceled)
	_, err = reader.RetrieveRole(t.Context(), actor, "current", actor)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 3, db.calls)
}
