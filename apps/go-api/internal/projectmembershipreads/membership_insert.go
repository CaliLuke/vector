package projectmembershipreads

import (
	"context"

	"github.com/google/uuid"
)

// WorkspaceDefaultProps is the default Plane view configuration assigned to a new workspace member.
const WorkspaceDefaultProps = `{"filters":` + memberFilters + `,"display_filters":` + memberDisplayFilters + `,"display_properties":` + memberDisplayProperties + `}`

// WorkspaceIssueProps is the default issue subscription configuration for a new workspace member.
const WorkspaceIssueProps = `{"subscribed":true,"assigned":true,"created":true,"all_issues":true}`

func insertWorkspaceMembership(ctx context.Context, db database, workspace, user string, role int, creator string) error {
	_, err := db.Exec(ctx, `INSERT INTO workspace_members(id,workspace_id,member_id,role,view_props,default_props,issue_props,is_active,getting_started_checklist,tips,explored_features,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$5,$6,true,'{}','{}','{}',clock_timestamp(),clock_timestamp(),$7) ON CONFLICT DO NOTHING`, uuid.NewString(), workspace, user, role, WorkspaceDefaultProps, WorkspaceIssueProps, creator)
	return err
}

func insertProjectMembership(ctx context.Context, db database, project, workspace, user string, role int, creator string, property bool) error {
	if property {
		if err := insertProjectProperty(ctx, db, project, creator, user); err != nil {
			return err
		}
	}
	_, err := db.Exec(ctx, `INSERT INTO project_members(id,project_id,workspace_id,member_id,role,view_props,default_props,preferences,sort_order,is_active,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$6,$6,$7,65535,true,clock_timestamp(),clock_timestamp(),$8) ON CONFLICT DO NOTHING`, uuid.NewString(), project, workspace, user, role, memberDefaultProps, memberPreferences, creator)
	return err
}

func insertProjectProperty(ctx context.Context, db database, project, actor, user string) error {
	_, err := db.Exec(ctx, `INSERT INTO project_user_properties(id,project_id,workspace_id,user_id,filters,display_filters,display_properties,rich_filters,preferences,sort_order,created_at,updated_at,created_by_id,updated_by_id) VALUES($1,$2,(SELECT workspace_id FROM projects WHERE id=$2),$3,$4,$5,$6,'{}',$7,COALESCE((SELECT MIN(sort_order)-10000 FROM project_user_properties WHERE user_id=$3 AND workspace_id=(SELECT workspace_id FROM projects WHERE id=$2) AND deleted_at IS NULL),65535),clock_timestamp(),clock_timestamp(),$8,NULL) ON CONFLICT DO NOTHING`, uuid.NewString(), project, user, memberFilters, memberDisplayFilters, memberDisplayProperties, memberPreferences, actor)
	return err
}
