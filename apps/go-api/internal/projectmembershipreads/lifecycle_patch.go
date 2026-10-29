package projectmembershipreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/jackc/pgx/v5"
)

type legacyPatchRoles struct {
	targetRole, targetWorkspace, requesterRole int
	admin                                      bool
}

func (r *LifecycleRepository) lifecycleLegacyPatch(ctx context.Context, c LifecycleCommand) (any, error) {
	roles, err := r.readLegacyPatchRoles(ctx, c)
	if err != nil {
		return nil, err
	}
	input, err := lifecycleLegacyPatchInput(c)
	if err != nil {
		return nil, err
	}
	if err = validateLegacyPatchPermissions(input, roles); err != nil {
		return nil, err
	}
	assignments, args, err := r.prepareLegacyPatchAssignments(ctx, c, input)
	if err != nil {
		return nil, err
	}
	_, err = r.database.Exec(ctx, `UPDATE project_members SET `+strings.Join(assignments, ",")+` WHERE id=$1`, args...)
	if err != nil {
		return nil, lifecycleDBError(err)
	}
	return r.lifecycleLegacyRepresentation(ctx, c)
}

func (r *LifecycleRepository) readLegacyPatchRoles(ctx context.Context, c LifecycleCommand) (*legacyPatchRoles, error) {
	var targetUser *string
	var targetRole, targetWorkspace, requesterRole, requesterWorkspace int
	err := r.database.QueryRow(ctx, `SELECT pm.member_id::text,pm.role FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.id=$1 AND pm.project_id=$2 AND w.slug=$3 AND pm.is_active AND pm.deleted_at IS NULL`, c.MemberID, c.ProjectID, c.Slug).Scan(&targetUser, &targetRole)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, lifecycleMissing(true)
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range []struct {
		user   *string
		target *int
	}{{targetUser, &targetWorkspace}, {&c.UserID, &requesterWorkspace}} {
		err = r.database.QueryRow(ctx, `SELECT wm.role FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE w.slug=$1 AND wm.member_id=$2 AND wm.is_active AND wm.deleted_at IS NULL`, c.Slug, entry.user).Scan(entry.target)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, lifecycleMissing(true)
		}
		if err != nil {
			return nil, err
		}
	}
	admin := requesterWorkspace == 20
	if targetUser != nil && *targetUser == c.UserID && !admin {
		return nil, lifecycleFailure(400, "error", "You cannot update your own role")
	}
	err = r.database.QueryRow(ctx, `SELECT role FROM project_members WHERE project_id=$1 AND member_id=$2 AND is_active AND deleted_at IS NULL`, c.ProjectID, c.UserID).Scan(&requesterRole)
	if err != nil {
		return nil, err
	}
	return &legacyPatchRoles{targetRole: targetRole, targetWorkspace: targetWorkspace, requesterRole: requesterRole, admin: admin}, nil
}

func validateLegacyPatchPermissions(input djangoinput.Input, roles *legacyPatchRoles) error {
	if value, ok := input.Fields["role"]; ok {
		if roles.requesterRole < 20 && !roles.admin {
			return lifecycleFailure(403, "error", "You do not have permission to update roles")
		}
		if roles.targetRole >= roles.requesterRole && !roles.admin {
			return lifecycleFailure(403, "error", "You cannot update the role of a member with a role equal to or higher than your own")
		}
		if input.Kind != "" {
			return lifecycleFailure(500, "error", "Something went wrong please try again later")
		}
		number, e := lifecyclePythonInt(value)
		if e != nil {
			return lifecycleFailure(500, "error", "Something went wrong please try again later")
		}
		if number.Cmp(big.NewInt(int64(roles.requesterRole))) >= 0 && !roles.admin {
			return lifecycleFailure(403, "error", "You cannot assign a role equal to or higher than your own")
		}
		if roles.targetWorkspace == 5 && (number.Cmp(big.NewInt(15)) == 0 || number.Cmp(big.NewInt(20)) == 0) {
			return lifecycleFailure(400, "error", "You cannot add a user with role higher than the workspace role")
		}
	}
	if _, ok := input.Fields["is_active"]; ok {
		if roles.requesterRole < 20 && !roles.admin {
			return lifecycleFailure(403, "error", "You do not have permission to update member status")
		}
		if roles.targetRole >= roles.requesterRole && !roles.admin {
			return lifecycleFailure(403, "error", "You cannot update the status of a member with a role equal to or higher than your own")
		}
	}
	if input.Kind != "" {
		return &djangoapi.HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + input.Kind + "."}}}
	}
	return nil
}

func (r *LifecycleRepository) prepareLegacyPatchAssignments(ctx context.Context, c LifecycleCommand, input djangoinput.Input) ([]string, []any, error) {
	failures := map[string]any{}
	// BaseModel.save always uses the current actor; submitted updated_by is validated below but never assigned.
	assignments := []string{"updated_at=clock_timestamp()", "updated_by_id=$2", "workspace_id=(SELECT workspace_id FROM projects WHERE id=project_members.project_id)"}
	args := []any{c.MemberID, c.UserID}
	for _, field := range []string{"role", "is_active", "comment", "sort_order", "deleted_at", "view_props", "default_props", "preferences", "created_by", "updated_by"} {
		value, ok := input.Fields[field]
		if !ok {
			continue
		}
		// DRF's HTML get_value normalization belongs after the view's raw
		// role/status guards, which intentionally see the submitted form text.
		if input.Form && value == "" {
			switch field {
			case "role", "sort_order", "is_active":
				continue
			case "deleted_at", "created_by", "updated_by":
				value = nil
			}
		}
		column := field
		var failure string
		switch field {
		case "role":
			value, failure = lifecycleRole(value)
		case "is_active":
			value, failure = lifecycleBoolean(value)
		case "comment":
			if value != nil {
				var es []string
				value, es = djangoinput.TextField(value, true, 0, input.Surrogates[field])
				if len(es) > 0 {
					failure = es[0]
				}
			}
		case "sort_order":
			if value == nil {
				failure = "This field may not be null."
			} else {
				value, failure = djangoinput.FloatField(value)
			}
		case "deleted_at":
			if value != nil {
				location := time.UTC
				if c.Timezone != "" {
					if zone, e := time.LoadLocation(c.Timezone); e == nil {
						location = zone
					}
				}
				parsed, e := pythoncompat.DjangoDateTime(djangoinput.ScalarString(value), location)
				if e != nil {
					failure = "Datetime has wrong format. Use one of these formats instead: YYYY-MM-DDThh:mm[:ss[.uuuuuu]][+HH:MM|-HH:MM|Z]."
				} else {
					value = parsed
				}
			}
		case "view_props", "default_props", "preferences":
			if value == nil {
				failure = "This field may not be null."
			} else {
				if input.Form {
					decoder := json.NewDecoder(strings.NewReader(djangoinput.ScalarString(value)))
					decoder.UseNumber()
					if decodeErr := decoder.Decode(&value); decodeErr != nil {
						failure = "Value must be valid JSON."
					} else if trailingDataErr := decoder.Decode(new(any)); trailingDataErr != io.EOF {
						failure = "Value must be valid JSON."
					}
					if failure != "" {
						break
					}
				}
				raw, e := json.Marshal(value)
				if e != nil {
					return nil, nil, e
				}
				value = string(raw)
			}
		case "created_by", "updated_by":
			column += "_id"
			if _, boolean := value.(bool); boolean {
				failure = "Incorrect type. Expected pk value, received bool."
			} else if value != nil {
				parsed, e := djangoinput.UUID(value)
				if e != nil {
					failure = "“" + djangoinput.ScalarString(value) + "” is not a valid UUID."
				} else {
					value = parsed.String()
					var exists bool
					if scanErr := r.database.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, value).Scan(&exists); scanErr != nil {
						return nil, nil, scanErr
					}
					if !exists {
						failure = fmt.Sprintf("Invalid pk %q - object does not exist.", value)
					}
				}
			}
		}
		if failure != "" {
			failures[field] = []string{failure}
			continue
		}
		if field == "updated_by" {
			continue
		}
		args = append(args, value)
		assignments = append(assignments, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if len(failures) > 0 {
		return nil, nil, &djangoapi.HTTPError{Status: 400, Body: failures}
	}
	return assignments, args, nil
}

// The legacy view performs Python containment checks before passing data to
// its serializer. Retain those guard branches for non-dictionary JSON too.
func lifecycleLegacyPatchInput(c LifecycleCommand) (djangoinput.Input, error) {
	input, err := djangoinput.Parse(c.Body, c.ContentType)
	if err != nil {
		return input, err
	}
	if input.Kind == "" {
		return input, nil
	}
	if input.Null || (input.Kind != "list" && input.Kind != "str") {
		return input, lifecycleFailure(500, "error", "Something went wrong please try again later")
	}
	var raw any
	if err := json.Unmarshal(input.Raw, &raw); err != nil {
		return input, err
	}
	for _, field := range []string{"role", "is_active"} {
		present := false
		switch value := raw.(type) {
		case string:
			present = strings.Contains(value, field)
		case []any:
			for _, item := range value {
				if text, ok := item.(string); ok && text == field {
					present = true
					break
				}
			}
		}
		if present {
			input.Fields[field] = nil
		}
	}
	return input, nil
}
