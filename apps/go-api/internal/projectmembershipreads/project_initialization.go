package projectmembershipreads

import (
	"context"

	"github.com/google/uuid"
)

// InitializeProjectAdministrator persists ProjectMember.save's property side
// effect and the admin membership in the caller's creation transaction.
func InitializeProjectAdministrator(ctx context.Context, db database, actor, project, user string) error {
	r := &LifecycleRepository{Repository: &Repository{database: db}}
	if err := r.insertLifecycleProperty(ctx, LifecycleCommand{UserID: actor, ProjectID: project}, user, false); err != nil {
		return err
	}
	_, err := db.Exec(ctx, `INSERT INTO project_members(id,project_id,workspace_id,member_id,role,comment,view_props,default_props,preferences,sort_order,is_active,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,(SELECT workspace_id FROM projects WHERE id=$2),$3,20,NULL,$4,$4,$5,65535,true,clock_timestamp(),clock_timestamp(),$6,NULL)`, uuid.NewString(), project, user, memberDefaultProps, memberPreferences, actor)
	return err
}
