package workspaceprojectreads

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// EnsureProjectIntake is shared by project settings and feature mutations. The
// caller authorizes and updates the project in this transaction before calling,
// so its project row lock serializes competing enable operations.
func EnsureProjectIntake(ctx context.Context, db interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, projectID, actorID string) error {
	_, err := db.Exec(ctx, `INSERT INTO intakes(id,project_id,workspace_id,name,description,is_default,view_props,logo_props,created_at,updated_at,created_by_id)
 SELECT $1,p.id,p.workspace_id,p.name || ' Intake','',true,'{}','{}',clock_timestamp(),clock_timestamp(),$2
 FROM projects p WHERE p.id=$3 AND p.intake_view AND p.deleted_at IS NULL
 AND NOT EXISTS(SELECT 1 FROM intakes intake WHERE intake.project_id=p.id AND intake.is_default AND intake.deleted_at IS NULL)`, uuid.NewString(), actorID, projectID)
	return err
}
