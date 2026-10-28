package pagereads

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// A replacement starts a new collaboration document. Old live sessions and
// offline caches must not overwrite it when they next flush their Yjs state.
func checkPageCollaborationRevision(ctx context.Context, tx pgx.Tx, page, workspace uuid.UUID, fields map[string]any) error {
	expected := int64(0)
	if value, present := fields["collaboration_revision"]; present {
		parsed, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
		if err != nil || parsed < 0 {
			return lifecycleError(http.StatusBadRequest, "Invalid page collaboration revision")
		}
		expected = parsed
	}
	var current int64
	if err := tx.QueryRow(ctx, `SELECT collaboration_revision FROM pages WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR UPDATE`, page, workspace).Scan(&current); err != nil {
		return err
	}
	if current != expected {
		return lifecycleError(http.StatusConflict, "The page was replaced. Reload it before editing.")
	}
	return nil
}
