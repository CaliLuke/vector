package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type lifecycleBulkRole struct {
	identity any
	role     any
	id       string
}

// Original records preserve repeated entries and their submitted identity spelling.
// The role dictionary retains the first equal key and position, with the last role.
type lifecycleBulkBatch struct {
	original []map[string]any
	roles    map[string]*lifecycleBulkRole
	ordered  []*lifecycleBulkRole
}

func (r *LifecycleRepository) lifecycleBulkCreate(ctx context.Context, c LifecycleCommand) (any, error) {
	input, err := lifecycleParse(c)
	if err != nil {
		return nil, err
	}
	var project bool
	if scanErr := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2 AND p.deleted_at IS NULL)`, c.ProjectID, c.Slug).Scan(&project); scanErr != nil {
		return nil, scanErr
	}
	if !project {
		return nil, lifecycleMissing(true)
	}
	batch, err := indexLifecycleBulkMembers(input.Fields)
	if err != nil {
		return nil, err
	}
	queryIDs, err := r.resolveLifecycleBulkMembers(ctx, c.Slug, batch)
	if err != nil {
		return nil, err
	}
	// Keep the independent bulk update, membership insert, and property insert
	// commits. A later failure must not roll back an earlier completed stage.
	if err = r.updateLifecycleBulkMembers(ctx, c.ProjectID, queryIDs, batch.roles); err != nil {
		return nil, err
	}
	if err = r.createLifecycleBulkMembers(ctx, c.ProjectID, batch); err != nil {
		return nil, err
	}
	value, err := djangoapi.JSON(ctx, r.database, `SELECT COALESCE(jsonb_agg(jsonb_build_object('id',id,'member',member_id,'role',role,'project',project_id,'original_role',role,'created_at',created_at) ORDER BY created_at DESC),'[]'::jsonb) FROM project_members WHERE project_id=$1 AND member_id::text=ANY($2) AND deleted_at IS NULL`, c.ProjectID, queryIDs)
	if err != nil {
		return nil, err
	}
	for _, item := range value.([]any) {
		row := item.(map[string]any)
		if err = r.publisher.Task(ctx, "plane.bgtasks.project_add_user_email_task.project_add_user_email", []any{r.baseHost, row["id"], c.UserID}, nil); err != nil {
			return nil, err
		}
	}
	return value, nil
}

func indexLifecycleBulkMembers(fields map[string]any) (*lifecycleBulkBatch, error) {
	if raw, present := fields["members"]; present {
		if _, ok := raw.([]any); !ok {
			switch value := raw.(type) {
			case string:
				if value == "" {
					return nil, lifecycleFailure(400, "error", "At least one member is required")
				}
			case map[string]any:
				if len(value) == 0 {
					return nil, lifecycleFailure(400, "error", "At least one member is required")
				}
			}
			return nil, lifecycleFailure(400, "error", "Members must be a list.")
		}
	}
	members, _ := fields["members"].([]any)
	if len(members) == 0 {
		return nil, lifecycleFailure(400, "error", "At least one member is required")
	}
	roles := map[string]*lifecycleBulkRole{}
	ordered := []*lifecycleBulkRole{}
	original := make([]map[string]any, 0, len(members))
	for _, item := range members {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, lifecycleFailure(400, "error", "Each member must be an object.")
		}
		original = append(original, m)
		key, hashable := lifecycleBulkIdentityKey(m["member_id"])
		if !hashable {
			return nil, lifecycleFailure(400, "error", "Please provide valid detail")
		}
		entry, exists := roles[key]
		if !exists {
			entry = &lifecycleBulkRole{identity: m["member_id"]}
			roles[key] = entry
			ordered = append(ordered, entry)
		}
		entry.role = m["role"]
	}
	return &lifecycleBulkBatch{original: original, roles: roles, ordered: ordered}, nil
}

func (r *LifecycleRepository) resolveLifecycleBulkMembers(ctx context.Context, slug string, batch *lifecycleBulkBatch) ([]string, error) {
	// Python constructs the complete dictionary before any ORM lookup. Equal
	// numeric keys replace the role but retain the first key and its position.
	for _, entry := range batch.ordered {
		id, membershipErr := lifecycleBulkMemberUUID(entry.identity)
		if membershipErr != nil {
			return nil, membershipErr
		}
		entry.id = id
		var role int
		membershipErr = r.database.QueryRow(ctx, `SELECT wm.role FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$1 AND wm.member_id=$2 AND wm.is_active AND wm.deleted_at IS NULL`, slug, entry.id).Scan(&role)
		if errors.Is(membershipErr, pgx.ErrNoRows) {
			return nil, lifecycleMissing(true)
		}
		if membershipErr != nil {
			return nil, membershipErr
		}
		submitted, numeric := lifecycleNumericRole(entry.role)
		if role == 20 && numeric && (submitted == 5 || submitted == 15) {
			return nil, lifecycleFailure(400, "error", "You cannot add a user with role lower than the workspace role")
		}
		if role == 5 && numeric && (submitted == 15 || submitted == 20) {
			return nil, lifecycleFailure(400, "error", "You cannot add a user with role higher than the workspace role")
		}
	}
	// The ORM's next __in lookup prepares every original value, even values
	// whose equal dictionary keys were collapsed during the role checks.
	queryIDs := make([]string, 0, len(batch.original))
	for _, item := range batch.original {
		id, lifecycleBulkMemberUUIDErr := lifecycleBulkMemberUUID(item["member_id"])
		if lifecycleBulkMemberUUIDErr != nil {
			return nil, lifecycleBulkMemberUUIDErr
		}
		queryIDs = append(queryIDs, id)
	}
	return queryIDs, nil
}

func (r *LifecycleRepository) updateLifecycleBulkMembers(ctx context.Context, projectID string, queryIDs []string, roles map[string]*lifecycleBulkRole) error {
	existing, err := djangoapi.JSON(ctx, r.database, `SELECT COALESCE(jsonb_agg(member_id::text),'[]'::jsonb) FROM project_members WHERE project_id=$1 AND member_id::text=ANY($2) AND deleted_at IS NULL`, projectID, queryIDs)
	if err != nil {
		return err
	}
	updates := make([]map[string]any, 0, len(existing.([]any)))
	for _, item := range existing.([]any) {
		id := item.(string)
		key, _ := lifecycleBulkIdentityKey(id)
		entry, exists := roles[key]
		if !exists {
			return lifecycleFailure(400, "error", "The required key does not exist.")
		}
		role, lifecycleDatabaseRoleErr := lifecycleDatabaseRole(entry.role)
		if lifecycleDatabaseRoleErr != nil {
			return lifecycleDatabaseRoleErr
		}
		updates = append(updates, map[string]any{"member_id": id, "role": role})
	}
	updateJSON, _ := json.Marshal(updates)
	_, err = r.database.Exec(ctx, `UPDATE project_members pm SET role=incoming.role,is_active=true FROM jsonb_to_recordset($2::jsonb) AS incoming(member_id uuid,role smallint) WHERE pm.project_id=$1 AND pm.member_id=incoming.member_id AND pm.deleted_at IS NULL`, projectID, string(updateJSON))
	if err != nil {
		return lifecycleDBError(err)
	}
	return nil
}

func (r *LifecycleRepository) createLifecycleBulkMembers(ctx context.Context, projectID string, batch *lifecycleBulkBatch) error {
	inserts := make([]map[string]any, 0, len(batch.original))
	properties := make([]map[string]any, 0, len(batch.original))
	for _, m := range batch.original {
		key, _ := lifecycleBulkIdentityKey(m["member_id"])
		id := batch.roles[key].id
		role, exists := m["role"]
		if !exists {
			role = 5
		}
		role, err := lifecycleDatabaseRole(role)
		if err != nil {
			return err
		}
		inserts = append(inserts, map[string]any{"id": uuid.NewString(), "member_id": id, "role": role})
		properties = append(properties, map[string]any{"id": uuid.NewString(), "user_id": id, "lookup_key": djangoinput.ScalarString(m["member_id"])})
	}
	insertJSON, _ := json.Marshal(inserts)
	_, err := r.database.Exec(ctx, `INSERT INTO project_members(id,project_id,workspace_id,member_id,role,comment,view_props,default_props,preferences,sort_order,is_active,created_at,updated_at,created_by_id,updated_by_id)
 SELECT incoming.id,$1,(SELECT workspace_id FROM projects WHERE id=$1),incoming.member_id,incoming.role,NULL,$3,$3,$4,65535,true,clock_timestamp(),clock_timestamp(),NULL,NULL FROM jsonb_to_recordset($2::jsonb) AS incoming(id uuid,member_id uuid,role smallint) ON CONFLICT DO NOTHING`, projectID, string(insertJSON), memberDefaultProps, memberPreferences)
	if err != nil {
		return lifecycleDBError(err)
	}
	propertyJSON, _ := json.Marshal(properties)
	_, err = r.database.Exec(ctx, `INSERT INTO project_user_properties(id,project_id,workspace_id,user_id,filters,display_filters,display_properties,rich_filters,preferences,sort_order,created_at,updated_at,created_by_id,updated_by_id)
 SELECT incoming.id,$1,(SELECT workspace_id FROM projects WHERE id=$1),incoming.user_id,$3,$4,$5,'{}',$6,COALESCE((SELECT MIN(sort_order)-10000 FROM project_user_properties WHERE user_id::text=incoming.lookup_key AND workspace_id=(SELECT workspace_id FROM projects WHERE id=$1) AND deleted_at IS NULL),65535),clock_timestamp(),clock_timestamp(),NULL,NULL
 FROM jsonb_to_recordset($2::jsonb) AS incoming(id uuid,user_id uuid,lookup_key text) ON CONFLICT DO NOTHING`, projectID, string(propertyJSON), memberFilters, memberDisplayFilters, memberDisplayProperties, memberPreferences)
	if err != nil {
		return lifecycleDBError(err)
	}

	return nil
}

func lifecycleBulkMemberUUID(value any) (string, error) {
	if value == nil {
		return "", lifecycleMissing(true)
	}
	if boolean, ok := value.(bool); ok {
		value = json.Number("0")
		if boolean {
			value = json.Number("1")
		}
	}
	id, err := djangoinput.UUID(value)
	if err != nil {
		return "", lifecycleFailure(400, "error", "Please provide valid detail")
	}
	return id.String(), nil
}

// JSON object/list values are unhashable in Python. Its numeric dictionary
// keys compare by value, including bool/int/float equality, unlike strings.
func lifecycleBulkIdentityKey(value any) (string, bool) {
	switch value := value.(type) {
	case nil:
		return "none", true
	case string:
		return "str:" + value, true
	case bool:
		if value {
			return "number:1", true
		}
		return "number:0", true
	case json.Number:
		if !strings.ContainsAny(string(value), ".eE") {
			integer, ok := new(big.Int).SetString(string(value), 10)
			if !ok {
				return "", false
			}
			return "number:" + integer.String(), true
		}
		number, err := value.Float64()
		if err != nil && !math.IsInf(number, 0) {
			return "", false
		}
		if math.IsInf(number, 1) {
			return "number:inf", true
		}
		if math.IsInf(number, -1) {
			return "number:-inf", true
		}
		return "number:" + new(big.Rat).SetFloat64(number).RatString(), true
	}
	return "", false
}

func lifecycleDatabaseRole(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	integer, err := lifecyclePythonInt(value)
	if err != nil {
		return nil, err
	}
	return json.Number(integer.String()), nil
}
