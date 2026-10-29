//go:build integration

package pagereads_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type failedPageDescriptionDatabase struct {
	*pgxpool.Pool
	failure error
}

func (db *failedPageDescriptionDatabase) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if strings.HasPrefix(query, "SELECT description_html FROM pages ") {
		return failedPageDescriptionRow{db.failure}
	}
	return db.Pool.QueryRow(ctx, query, args...)
}

type failedPageDescriptionRow struct{ failure error }

func (row failedPageDescriptionRow) Scan(...any) error { return row.failure }

func TestPageUpdateDescriptionReadFailureLeavesPageAndTasksUnchanged(t *testing.T) {
	for _, sample := range []struct {
		name   string
		fields map[string]any
	}{
		{"metadata update", map[string]any{"name": "After"}},
		{"description update", map[string]any{"description_html": "<p>After</p>"}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			fixture := newPageCreateFixture(t)
			_, raw, err := fixture.create(t, map[string]any{"name": "Before", "description_html": "<p>Before</p>"})
			require.NoError(t, err)
			var page struct {
				ID uuid.UUID `json:"id"`
			}
			require.NoError(t, json.Unmarshal(raw, &page))
			failure := errors.New("page description read failed")
			repository, err := pagereads.NewRepository(&failedPageDescriptionDatabase{Pool: fixture.pool, failure: failure}, fixture)
			require.NoError(t, err)

			_, _, err = repository.Lifecycle(t.Context(), "update", fixture.actor, "milestones", fixture.project, page.ID, "UTC", djangoinput.Input{Fields: sample.fields})

			require.ErrorIs(t, err, failure)
			var name, html, stripped string
			require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT name,description_html,description_stripped FROM pages WHERE id=$1`, page.ID).Scan(&name, &html, &stripped))
			require.Equal(t, "Before", name)
			require.Equal(t, "<p>Before</p>", html)
			require.Equal(t, "Before", stripped)
			require.Len(t, fixture.tasks, 1)
		})
	}
}
