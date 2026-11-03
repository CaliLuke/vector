package projectmembershipreads

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/CaliLuke/go-plane/internal/apicontract"
	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/requestinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func isProjectMemberWrite(c LifecycleCommand) bool {
	return strings.HasPrefix(c.Operation, "v1-create-") || strings.HasPrefix(c.Operation, "v1-patch-") || strings.HasPrefix(c.Operation, "v1-delete-") || c.Operation == "legacy-create-members" || c.Operation == "legacy-patch-members" || c.Operation == "legacy-delete-members" || c.Operation == "legacy-leave-members"
}

func authorizeProjectMemberWrite(ctx context.Context, db database, c LifecycleCommand) error {
	var allowed bool
	err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id
  JOIN workspace_members wm ON wm.workspace_id=w.id JOIN users u ON u.id=wm.member_id
  JOIN project_members actor_pm ON actor_pm.project_id=p.id AND actor_pm.workspace_id=w.id AND actor_pm.member_id=wm.member_id AND actor_pm.is_active AND actor_pm.deleted_at IS NULL AND actor_pm.role IN (5,15,20)
  WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL AND w.deleted_at IS NULL
  AND ($5::uuid IS NULL OR w.id=$5) AND wm.member_id=$3 AND wm.is_active AND wm.deleted_at IS NULL AND u.is_active
  AND wm.role IN (5,15,20) AND ($4 OR wm.role=20 OR (wm.role=15 AND EXISTS(
   SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.workspace_id=w.id AND pm.member_id=$3 AND pm.role=20 AND pm.is_active AND pm.deleted_at IS NULL))))`, c.ProjectID, c.Slug, c.UserID, c.Operation == "legacy-leave-members", c.ExpectedWorkspaceID).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return inviteFailure(http.StatusForbidden, "Project administrator permission is required")
	}
	return nil
}

type projectMemberInput struct {
	member string
	role   *int
	active *bool
}

func parseProjectMembers(c LifecycleCommand) ([]projectMemberInput, error) {
	if c.Method == http.MethodDelete || c.Operation == "legacy-leave-members" {
		return nil, nil
	}
	bulk := c.Operation == "legacy-create-members"
	allowed := []string{"role", "is_active"}
	create := strings.HasPrefix(c.Operation, "v1-create-")
	if create {
		allowed = []string{"member", "role"}
	}
	if bulk {
		allowed = []string{"members"}
	}
	fields, err := memberFields(c.Body, c.ContentType, allowed...)
	if err != nil {
		return nil, err
	}
	if !bulk {
		item, parseErr := parseProjectMember(fields, create, "member")
		return []projectMemberInput{item}, parseErr
	}
	values, ok := fields["members"].([]any)
	if !ok || len(values) == 0 {
		return nil, inviteFailure(http.StatusBadRequest, "At least one member is required")
	}
	if len(values) > 1000 {
		return nil, inviteFailure(http.StatusBadRequest, "At most 1000 members can be added at once")
	}
	result := make([]projectMemberInput, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		fields, ok := value.(map[string]any)
		if !ok {
			return nil, inviteFailure(http.StatusBadRequest, "Each member must be an object")
		}
		for field := range fields {
			if field != "member_id" && field != "role" {
				return nil, inviteFailure(http.StatusBadRequest, "Unsupported membership field: "+field)
			}
		}
		item, parseErr := parseProjectMember(fields, true, "member_id")
		if parseErr != nil {
			return nil, parseErr
		}
		if seen[item.member] {
			return nil, inviteFailure(http.StatusBadRequest, "A member can appear only once in a batch")
		}
		seen[item.member] = true
		result = append(result, item)
	}
	return result, nil
}

func parseProjectMember(fields map[string]any, create bool, identity string) (projectMemberInput, error) {
	item := projectMemberInput{}
	if create {
		raw, ok := fields[identity].(string)
		if !ok {
			return item, inviteFailure(http.StatusBadRequest, "A member UUID is required")
		}
		id, err := requestinput.UUID(raw)
		if err != nil {
			return item, inviteFailure(http.StatusBadRequest, "A valid member UUID is required")
		}
		item.member = id.String()
		role := 5
		item.role = &role
	}
	if value, present := fields["role"]; present {
		role, err := memberRole(value)
		if err != nil {
			return item, err
		}
		item.role = &role
	}
	if value, present := fields["is_active"]; present {
		active, ok := value.(bool)
		if !ok {
			return item, inviteFailure(http.StatusBadRequest, "is_active must be a boolean")
		}
		item.active = &active
	}
	return item, nil
}

func (r *LifecycleRepository) mutateProjectMembers(ctx context.Context, c LifecycleCommand) (_ any, resultErr error) {
	if _, parseErr := requestinput.UUID(c.ProjectID); parseErr != nil {
		return nil, inviteFailure(http.StatusBadRequest, "A valid project UUID is required")
	}
	inputs, err := parseProjectMembers(c)
	if err != nil {
		return nil, err
	}
	m, err := beginMemberMutation(ctx, r.writer, c.Slug)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, m.tx.Rollback, resultErr) }()
	if err = authorizeProjectMemberWrite(ctx, m.tx, c); err != nil {
		return nil, err
	}
	m.projects = append(m.projects, c.ProjectID)
	leave := c.Operation == "legacy-leave-members"
	bulk := c.Operation == "legacy-create-members"
	create := strings.HasPrefix(c.Operation, "v1-create-")
	var result any
	switch {
	case bulk || create:
		ids := make([]string, 0, len(inputs))
		for _, input := range inputs {
			id, writeErr := r.addProjectMember(ctx, m, c, input, bulk)
			if writeErr != nil {
				return nil, writeErr
			}
			ids = append(ids, id)
		}
		if bulk {
			result, err = apicontract.JSON(ctx, m.tx, `SELECT COALESCE(jsonb_agg(`+projectMembershipJSON+` ORDER BY pm.created_at DESC,pm.id),'[]'::jsonb) FROM project_members pm WHERE pm.id::text=ANY($1)`, ids)
		} else {
			result, err = projectMemberRepresentation(ctx, m.tx, ids[0])
		}
	default:
		var id, member string
		var currentRole int
		var active bool
		err = m.tx.QueryRow(ctx, `SELECT id::text,member_id::text,role,is_active FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND (($3 AND member_id=$4) OR (NOT $3 AND id::text=$5))`, c.ProjectID, m.workspace, leave, c.UserID, c.MemberID).Scan(&id, &member, &currentRole, &active)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, inviteMissing()
		}
		if err != nil {
			return nil, err
		}
		if leave || c.Method == http.MethodDelete {
			active = false
		} else {
			input := inputs[0]
			if input.role != nil {
				if *input.role != currentRole {
					if err = m.checkSelfRoleChange(ctx, c.UserID, member); err != nil {
						return nil, err
					}
				}
				if err = m.validateProjectRole(ctx, member, *input.role); err != nil {
					return nil, err
				}
				currentRole = *input.role
			}
			if input.active != nil {
				active = *input.active
			}
			if active {
				if err = m.validateProjectRole(ctx, member, currentRole); err != nil {
					return nil, err
				}
			}
		}
		if _, err = m.tx.Exec(ctx, `UPDATE project_members SET role=$2,is_active=$3,updated_at=clock_timestamp(),updated_by_id=$4 WHERE id=$1`, id, currentRole, active, c.UserID); err != nil {
			return nil, err
		}
		if !active {
			if _, err = m.tx.Exec(ctx, `UPDATE project_member_invites SET deleted_at=clock_timestamp(),updated_at=clock_timestamp() WHERE project_id=$1 AND workspace_id=$2 AND lower(email)=lower((SELECT email FROM users WHERE id=$3)) AND deleted_at IS NULL`, c.ProjectID, m.workspace, member); err != nil {
				return nil, err
			}
		}
		if c.Method != http.MethodDelete && !leave {
			result, err = projectMemberRepresentation(ctx, m.tx, id)
		}
	}
	if err != nil {
		return nil, err
	}
	return result, m.commit(ctx)
}

func (m *memberMutation) validateProjectRole(ctx context.Context, member string, role int) error {
	var workspaceRole int
	err := m.tx.QueryRow(ctx, `SELECT wm.role FROM workspace_members wm JOIN users u ON u.id=wm.member_id WHERE wm.workspace_id=$1 AND wm.member_id=$2 AND wm.is_active AND wm.deleted_at IS NULL AND u.is_active AND wm.role IN (5,15,20)`, m.workspace, member).Scan(&workspaceRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return inviteFailure(http.StatusBadRequest, "Member must have active workspace access")
	}
	if err != nil {
		return err
	}
	if workspaceRole == 5 && role != 5 {
		return inviteFailure(http.StatusBadRequest, "Workspace guests must remain project guests")
	}
	return nil
}

func (r *LifecycleRepository) addProjectMember(ctx context.Context, m *memberMutation, c LifecycleCommand, input projectMemberInput, bulk bool) (string, error) {
	if err := m.validateProjectRole(ctx, input.member, *input.role); err != nil {
		return "", err
	}
	var id string
	var role int
	var active bool
	err := m.tx.QueryRow(ctx, `SELECT id::text,role,is_active FROM project_members WHERE project_id=$1 AND workspace_id=$2 AND member_id=$3 AND deleted_at IS NULL`, c.ProjectID, m.workspace, input.member).Scan(&id, &role, &active)
	missing := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !missing {
		return "", err
	}
	if !missing && !bulk {
		return "", inviteFailure(http.StatusBadRequest, "Member already belongs to this project")
	}
	if !missing && role != *input.role {
		if err = m.checkSelfRoleChange(ctx, c.UserID, input.member); err != nil {
			return "", err
		}
	}
	if missing {
		id = uuid.NewString()
		_, err = m.tx.Exec(ctx, `INSERT INTO project_members(id,project_id,workspace_id,member_id,role,view_props,default_props,preferences,sort_order,is_active,created_at,updated_at,created_by_id) VALUES($1,$2,$3,$4,$5,$6,$6,$7,65535,true,clock_timestamp(),clock_timestamp(),$8)`, id, c.ProjectID, m.workspace, input.member, *input.role, memberDefaultProps, memberPreferences, c.UserID)
	} else {
		_, err = m.tx.Exec(ctx, `UPDATE project_members SET role=$2,is_active=true,updated_at=clock_timestamp(),updated_by_id=$3 WHERE id=$1`, id, *input.role, c.UserID)
	}
	if err != nil {
		return "", lifecycleDBError(err)
	}
	if err = insertProjectProperty(ctx, m.tx, c.ProjectID, c.UserID, input.member); err != nil {
		return "", err
	}
	if missing || !active {
		err = (projectmutations.Effects{Tx: m.tx}).Email(ctx, "plane.bgtasks.project_add_user_email_task.project_add_user_email", []any{r.baseHost, id, c.UserID})
	}
	return id, err
}

func (m *memberMutation) checkSelfRoleChange(ctx context.Context, actor, member string) error {
	if actor != member {
		return nil
	}
	role, err := m.actorRole(ctx, actor)
	if err != nil {
		return err
	}
	if role != 20 {
		return inviteFailure(http.StatusBadRequest, "Use leave project to remove your own membership")
	}
	return nil
}
