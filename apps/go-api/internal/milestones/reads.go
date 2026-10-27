package milestones

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/issuequery"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Record struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	TargetDate     *string `json:"target_date"`
	ExternalSource *string `json:"external_source"`
	ExternalID     *string `json:"external_id"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	Revision       *string `json:"revision,omitempty"`
}

type Link struct {
	ID        string `json:"id"`
	Issue     string `json:"issue"`
	Milestone string `json:"milestone"`
}

// Page retains native records and delegates cursor arithmetic to the shared
// HTTP envelope. MCP projects that envelope to its seven pinned fields.
type Page[T any] struct {
	Results             []T
	Total, Limit, Index int
}

func (page Page[T]) Envelope() map[string]any {
	rows := make([]any, len(page.Results))
	for i, row := range page.Results {
		rows[i] = row
	}
	return djangoapi.Envelope(rows, page.Total, page.Limit, page.Index)
}

func pageWindow(query djangoapi.Query) (limit, page, offset int, err error) {
	limit, err = djangoapi.PageSizeWithMinimum(query, 1)
	if err != nil {
		return 0, 0, 0, ErrInvalidInput
	}
	page, err = djangoapi.CursorPage(query, limit)
	if err != nil || page < 0 || page > math.MaxInt/limit {
		return 0, 0, 0, ErrInvalidInput
	}
	return limit, page, page * limit, nil
}

const recordColumns = `m.id::text,m.title,m.target_date::text,m.external_source,m.external_id,m.created_at,m.updated_at`
const ownedMilestone = `m.project_id=$1 AND m.workspace_id=$2 AND m.deleted_at IS NULL`

func scanRecord(row pgx.Row, timezone string) (Record, error) {
	var record Record
	var created, updated time.Time
	err := row.Scan(&record.ID, &record.Title, &record.TargetDate, &record.ExternalSource, &record.ExternalID, &created, &updated)
	if err != nil {
		return Record{}, err
	}
	times := map[string]any{"created_at": created.UTC().Format(time.RFC3339Nano), "updated_at": updated.UTC().Format(time.RFC3339Nano)}
	djangoapi.Timestamps(times, timezone)
	record.CreatedAt, record.UpdatedAt = times["created_at"].(string), times["updated_at"].(string)
	return record, nil
}

func (r *Repository) List(ctx context.Context, scope Scope, query djangoapi.Query) (Page[Record], error) {
	limit, page, offset, err := pageWindow(query)
	if err != nil {
		return Page[Record]{}, err
	}
	return readSnapshot(ctx, r, scope, func(tx pgx.Tx) (Page[Record], error) {
		result := Page[Record]{Results: []Record{}, Limit: limit, Index: page}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM milestones m WHERE `+ownedMilestone, scope.ProjectID, scope.WorkspaceID).Scan(&result.Total); err != nil {
			return result, err
		}
		rows, err := tx.Query(ctx, `SELECT `+recordColumns+` FROM milestones m WHERE `+ownedMilestone+` ORDER BY m.target_date ASC NULLS LAST,m.id LIMIT $3 OFFSET $4`, scope.ProjectID, scope.WorkspaceID, limit, offset)
		if err != nil {
			return result, err
		}
		defer rows.Close()
		for rows.Next() {
			record, err := scanRecord(rows, scope.Timezone)
			if err != nil {
				return result, err
			}
			result.Results = append(result.Results, record)
		}
		return result, rows.Err()
	})
}

func (r *Repository) Retrieve(ctx context.Context, scope Scope, id uuid.UUID, includeRevision bool) (Record, error) {
	if id == uuid.Nil {
		return Record{}, ErrInvalidInput
	}
	return readSnapshot(ctx, r, scope, func(tx pgx.Tx) (Record, error) {
		record, err := retrieve(ctx, tx, scope, id)
		if err != nil || !includeRevision {
			return record, err
		}
		var revision int64
		if err = tx.QueryRow(ctx, `SELECT revision FROM milestone_edit_revisions WHERE milestone_id=$1`, id).Scan(&revision); err != nil {
			return Record{}, err
		}
		value := revisionToken(id, revision)
		record.Revision = &value
		return record, nil
	})
}

func retrieve(ctx context.Context, tx pgx.Tx, scope Scope, id uuid.UUID) (Record, error) {
	record, err := scanRecord(tx.QueryRow(ctx, `SELECT `+recordColumns+` FROM milestones m WHERE `+ownedMilestone+` AND m.id=$3`, scope.ProjectID, scope.WorkspaceID, id), scope.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	return record, err
}

func revisionToken(id uuid.UUID, revision int64) string {
	return fmt.Sprintf("milestone:%s:%d", id, revision)
}

var visibleLinks = ` FROM milestone_issues link JOIN issues issue ON issue.id=link.issue_id
 AND issue.project_id=link.project_id AND issue.workspace_id=link.workspace_id
 WHERE link.milestone_id=$1 AND link.workspace_id=$2 AND link.project_id=$3 AND link.deleted_at IS NULL AND ` +
	issuequery.IssueVisibleSQL("issue", "$4::uuid", "$2::uuid")

func (r *Repository) ListWorkItems(ctx context.Context, scope Scope, id uuid.UUID, query djangoapi.Query) (Page[Link], error) {
	if id == uuid.Nil {
		return Page[Link]{}, ErrInvalidInput
	}
	limit, page, offset, err := pageWindow(query)
	if err != nil {
		return Page[Link]{}, err
	}
	return readSnapshot(ctx, r, scope, func(tx pgx.Tx) (Page[Link], error) {
		result := Page[Link]{Results: []Link{}, Limit: limit, Index: page}
		if _, err := retrieve(ctx, tx, scope, id); err != nil {
			return result, err
		}
		args := []any{id, scope.WorkspaceID, scope.ProjectID, scope.ActorID}
		if err := tx.QueryRow(ctx, `SELECT count(*)`+visibleLinks, args...).Scan(&result.Total); err != nil {
			return result, err
		}
		args = append(args, limit, offset)
		rows, err := tx.Query(ctx, `SELECT link.id::text,link.issue_id::text,link.milestone_id::text`+visibleLinks+` ORDER BY link.created_at,link.id LIMIT $5 OFFSET $6`, args...)
		if err != nil {
			return result, err
		}
		defer rows.Close()
		for rows.Next() {
			var link Link
			if err := rows.Scan(&link.ID, &link.Issue, &link.Milestone); err != nil {
				return result, err
			}
			result.Results = append(result.Results, link)
		}
		return result, rows.Err()
	})
}
