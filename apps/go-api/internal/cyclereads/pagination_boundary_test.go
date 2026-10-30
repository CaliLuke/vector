package cyclereads

import (
	"context"
	"strconv"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestPaginationRejectsPageWhoseAdjacentCursorWouldOverflow(t *testing.T) {
	maxPage := int(^uint(0) >> 1)
	for _, page := range []int{maxPage, -maxPage - 1} {
		for _, group := range []string{"", "priority"} {
			t.Run(strconv.Itoa(page)+"/"+group, func(t *testing.T) {
				db := &databaseStub{rows: []pgx.Row{boolRow(true)}}
				repo, err := NewRepository(db)
				require.NoError(t, err)
				_, err = repo.Issues(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), IssueListOptions{
					GroupBy: group, PerPage: 1, PerPageSet: true, Cursor: "1:" + strconv.Itoa(page) + ":0", CursorSet: true,
				})
				require.ErrorIs(t, err, ErrPaginationOverflow)
				require.Len(t, db.queries, 1, "invalid adjacent cursor must be rejected before issue reads")
			})
		}
	}
}

func TestLastRepresentableOffsetDoesNotAdvertiseWrappedNextPage(t *testing.T) {
	page := int(^uint(0)>>1) / 2
	db := &databaseStub{rows: []pgx.Row{boolRow(true), pageRow(`[]`, 0)}}
	repo, err := NewRepository(db)
	require.NoError(t, err)
	result, err := repo.Issues(context.Background(), uuid.New(), "alpha", uuid.New(), uuid.New(), IssueListOptions{
		PerPage: 2, PerPageSet: true, Cursor: "2:" + strconv.Itoa(page) + ":0", CursorSet: true,
	})
	require.NoError(t, err)
	require.Contains(t, string(result), `"next_page_results":false`)
}
