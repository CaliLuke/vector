package notifications

import (
	"context"
	"errors"
	"slices"

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

// MarkAllRead updates only unread notifications from the selected view.
func (r *ActionRepository) MarkAllRead(ctx context.Context, p Principal, slug string, selection Selection) error {
	_, err := r.writer.Exec(ctx, `UPDATE notifications n SET read_at=CURRENT_TIMESTAMP FROM workspaces w WHERE w.id=n.workspace_id AND n.read_at IS NULL AND `+selection.where(), p.UserID, slug)
	return err
}
