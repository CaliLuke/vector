package notifications

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type ActionDatabase interface {
	Database
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}
type DeletionPublisher interface {
	SoftDelete(context.Context, string, string, string) error
}

// ActionRepository shares notification scope and serialization with reads.
// Writes require the existing durable publisher; read replicas do not.
type ActionRepository struct {
	*Repository
	writer    ActionDatabase
	publisher DeletionPublisher
}

func NewActionRepository(db ActionDatabase, publisher DeletionPublisher) (*ActionRepository, error) {
	if db == nil || publisher == nil {
		return nil, errors.New("notification write database and durable deletion publisher are required")
	}
	repository, err := NewRepository(db)
	if err != nil {
		return nil, err
	}
	return &ActionRepository{Repository: repository, writer: db, publisher: publisher}, nil
}

func (r *ActionRepository) Set(ctx context.Context, p Principal, slug, id, field string, value any) (map[string]any, error) {
	if !slices.Contains([]string{"read_at", "archived_at", "deleted_at", "snoozed_till"}, field) {
		return nil, errors.New("invalid notification action field")
	}
	// Recheck visibility at the write after the service lookup. Publishing remains
	// after the save, without a transaction that could undo it.
	tag, err := r.writer.Exec(ctx, `UPDATE notifications n SET `+field+`=$4,updated_at=clock_timestamp(),updated_by_id=$1 FROM workspaces w WHERE w.id=n.workspace_id AND `+scope+` AND n.id=$3`, p.UserID, slug, id, value)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, Failure(404, "error", "The required object does not exist.")
	}
	if field == "deleted_at" {
		return nil, r.publisher.SoftDelete(ctx, "db", "notification", id)
	}
	return r.Detail(ctx, p, slug, id, false)
}

// MarkAllRead applies the source viewset's filters and updates only read_at.
// Authorization is performed by the service before this method is called.
func (r *ActionRepository) MarkAllRead(ctx context.Context, p Principal, slug string, snoozed, archived bool, kind string) error {
	filters := []string{
		scope,
		"n.read_at IS NULL",
	}
	if snoozed {
		filters = append(filters, "n.snoozed_till IS NOT NULL")
	} else {
		filters = append(filters, "(n.snoozed_till IS NULL OR n.snoozed_till>=CURRENT_TIMESTAMP)")
	}
	if archived {
		filters = append(filters, "n.archived_at IS NOT NULL")
	} else {
		filters = append(filters, "n.archived_at IS NULL")
	}
	switch kind {
	case "watching":
		filters = append(filters, "n.entity_identifier IN (SELECT subscriber.issue_id FROM issue_subscribers subscriber JOIN workspaces sw ON sw.id=subscriber.workspace_id WHERE sw.slug=$2 AND subscriber.subscriber_id=$1 AND subscriber.deleted_at IS NULL)")
	case "assigned":
		filters = append(filters, "n.entity_identifier IN (SELECT assignment.issue_id FROM issue_assignees assignment JOIN workspaces aw ON aw.id=assignment.workspace_id WHERE aw.slug=$2 AND assignment.assignee_id=$1 AND assignment.deleted_at IS NULL)")
	case "created":
		var guest bool
		if err := r.writer.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members member JOIN workspaces workspace ON workspace.id=member.workspace_id WHERE workspace.slug=$2 AND member.member_id=$1 AND member.role<15 AND member.is_active AND member.deleted_at IS NULL)`, p.UserID, slug).Scan(&guest); err != nil {
			return err
		}
		if guest {
			filters = append(filters, "FALSE")
		} else {
			filters = append(filters, "n.entity_identifier IN (SELECT issue.id FROM issues issue JOIN workspaces iw ON iw.id=issue.workspace_id WHERE iw.slug=$2 AND issue.created_by_id=$1 AND issue.deleted_at IS NULL)")
		}
	}
	_, err := r.writer.Exec(ctx, `UPDATE notifications n SET read_at=CURRENT_TIMESTAMP FROM workspaces w WHERE w.id=n.workspace_id AND `+strings.Join(filters, " AND "), p.UserID, slug)
	return err
}
