package workitemtypes

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ItemValueInput must come from an authorized, locked work item. Stored values
// and submitted keys are separate so omission never clears an inactive value.
type ItemValueInput struct {
	WorkspaceID, ProjectID, PreviousProjectID, TypeID, PreviousTypeID, CreatorID, ParentID, ActorID string
	Stored, Submitted                                                                               map[string]any
	Draft, Creating, Existing, TypeSpecified, ProjectChanged                                        bool
}

type ItemValueResult struct {
	TypeID      string
	Values      map[string]any
	Subscribers []string
}

type ItemValueError struct{ Fields map[string]string }

func (e *ItemValueError) Error() string { return "Correct the custom property values before saving." }

// ValidateItemValues runs inside the caller's mutation transaction. It assumes
// authorization, and acquires the schema lock before loading definitions. The
// caller must acquire that same lock before locking/reloading an existing item.
func ValidateItemValues(ctx context.Context, tx pgx.Tx, in ItemValueInput) (ItemValueResult, error) {
	in.ProjectID, in.PreviousProjectID = strings.ToLower(in.ProjectID), strings.ToLower(in.PreviousProjectID)
	in.TypeID, in.PreviousTypeID = strings.ToLower(in.TypeID), strings.ToLower(in.PreviousTypeID)
	out := ItemValueResult{TypeID: in.TypeID, Values: maps.Clone(in.Stored), Subscribers: []string{}}
	if out.Values == nil {
		out.Values = map[string]any{}
	}
	if _, err := tx.Exec(ctx, `SELECT work_item_schema_lock($1::uuid)`, in.WorkspaceID); err != nil {
		return out, err
	}
	var enabled bool
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT enabled FROM work_item_type_settings WHERE workspace_id=$1),false)`, in.WorkspaceID).Scan(&enabled); err != nil {
		return out, err
	}
	if in.Draft && in.TypeID == "" && in.PreviousTypeID != "" {
		for _, value := range in.Stored {
			if nonemptyPropertyValue(value) {
				return out, reject(409, "Review preservation before clearing this draft’s type.")
			}
		}
		out.Values = map[string]any{}
	}
	if !enabled {
		if len(in.Submitted) > 0 {
			return out, reject(422, "Select an available work item type before entering custom properties.")
		}
		return out, nil
	}
	var err error
	out.TypeID, err = resolveItemType(ctx, tx, in)
	if err != nil {
		return out, err
	}
	if out.TypeID == "" {
		if len(in.Submitted) > 0 {
			return out, reject(422, "Select an available work item type before entering custom properties.")
		}
		return out, nil
	}
	definitions, err := itemPropertyDefinitions(ctx, tx, in.WorkspaceID, out.TypeID)
	if err != nil {
		return out, err
	}
	fields := map[string]string{}
	known := map[string]bool{}
	for _, def := range definitions {
		known[def.ID] = true
	}
	for id := range in.Submitted {
		if !known[id] {
			fields[id] = "This property is not attached to the selected type."
		}
	}
	// The separate reviewed preservation operation must handle nonempty old-only
	// values. A normal type update cannot silently discard them.
	if out.TypeID != in.PreviousTypeID {
		for id, value := range in.Stored {
			if known[id] {
				continue
			}
			if nonemptyPropertyValue(value) {
				return out, reject(409, "Review preservation of the previous type's properties before switching type.")
			}
			delete(out.Values, id)
		}
	}
	if err = validateItemContextTransition(in, out.TypeID, definitions); err != nil {
		return out, err
	}
	allowed, err := itemValueReferences(ctx, tx, in, definitions, true)
	if err != nil {
		return out, err
	}
	for _, def := range definitions {
		config, _, configErr := property.Effective(def, in.ProjectID, out.TypeID)
		if configErr != nil {
			return out, configErr
		}
		value, present := out.Values[def.ID]
		submitted, edited := in.Submitted[def.ID]
		if !config.Active || def.Kind == property.Formula || def.Kind == property.Text && config.Format == "Read-only" {
			if edited {
				fields[def.ID] = "This property is inactive or read-only."
			}
			continue
		}
		if in.Draft && def.Kind == property.Cascading {
			if edited {
				fields[def.ID] = "Cascading properties are available when publishing this draft."
			}
			continue
		}
		if edited {
			value, present = submitted, true
		}
		if in.Creating && !present && config.Default != nil {
			value = config.Default
		}
		canonical, validationErr := property.ValidateValue(def.Kind, config, value, in.Draft, allowed)
		if validationErr != nil {
			fields[def.ID] = validationErr.Error()
			continue
		}
		// Retain explicit clears to distinguish them from an absent creation value,
		// including when a private draft is later published.
		if present || canonical != nil {
			out.Values[def.ID] = canonical
		}
		if def.Kind == property.Member && !in.Draft && (in.Creating || edited) {
			out.Subscribers = append(out.Subscribers, memberValueIDs(canonical)...)
		}
	}
	if len(fields) > 0 {
		return out, &ItemValueError{Fields: fields}
	}
	slices.Sort(out.Subscribers)
	out.Subscribers = slices.Compact(out.Subscribers)
	return out, nil
}

func resolveItemType(ctx context.Context, tx pgx.Tx, in ItemValueInput) (string, error) {
	id := in.TypeID
	if id == "" {
		var err error
		id, err = suggestedItemType(ctx, tx, in)
		if err != nil {
			return "", err
		}
		if id == "" {
			if in.Draft {
				return "", nil
			}
			return "", reject(422, "Choose an available work item type before saving.")
		}
	}
	var active, available bool
	err := tx.QueryRow(ctx, `SELECT t.is_active,t.is_default OR t.mandatory OR $3='' OR EXISTS(SELECT 1 FROM project_issue_types p WHERE p.issue_type_id=t.id AND p.project_id=NULLIF($3,'')::uuid AND p.is_active AND p.deleted_at IS NULL) FROM issue_types t WHERE t.id=$1 AND t.workspace_id=$2 AND t.deleted_at IS NULL`, id, in.WorkspaceID, in.ProjectID).Scan(&active, &available)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", reject(422, "The work item type is not available in this workspace.")
	}
	if err != nil {
		return "", err
	}
	if (in.Creating || id != in.PreviousTypeID || in.ProjectChanged) && (!active || !available) {
		return "", reject(422, "The work item type is inactive or has not been imported into this project.")
	}
	return id, nil
}

func itemPropertyDefinitions(ctx context.Context, tx pgx.Tx, workspace, typeID string) ([]property.Definition, error) {
	rows, err := tx.Query(ctx, `SELECT p.definition FROM work_item_type_properties l JOIN work_item_properties p ON p.id=l.property_id WHERE p.workspace_id=$1 AND l.type_id=$2 ORDER BY l.sort_order,p.id`, workspace, typeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	definitions := []property.Definition{}
	for rows.Next() {
		var raw []byte
		var def property.Definition
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &def); err != nil {
			return nil, err
		}
		definitions = append(definitions, def)
	}
	return definitions, rows.Err()
}

func itemValueReferences(ctx context.Context, tx pgx.Tx, in ItemValueInput, definitions []property.Definition, lock bool) (property.ReferenceAllowed, error) {
	allowed := map[property.Kind]map[string]bool{property.Member: {}, property.Release: {}, property.Issue: {}}
	for _, kind := range []property.Kind{property.Member, property.Release, property.Issue} {
		if !slices.ContainsFunc(definitions, func(def property.Definition) bool { return def.Kind == kind }) {
			continue
		}
		query := `SELECT r.id::text FROM work_item_releases r WHERE r.workspace_id=$1 AND (r.project_id=NULLIF($2,'')::uuid OR r.project_id IS NULL) AND r.deleted_at IS NULL AND $3::text IS NOT NULL AND $4::boolean IS NOT NULL`
		if kind == property.Member {
			query = `SELECT u.id::text FROM users u JOIN workspace_members w ON w.member_id=u.id WHERE w.workspace_id=$1 AND w.is_active AND w.deleted_at IS NULL AND u.is_active AND (($2='' AND $4) OR EXISTS(SELECT 1 FROM project_members m JOIN projects p ON p.id=m.project_id WHERE m.member_id=u.id AND m.workspace_id=$1 AND m.project_id=NULLIF($2,'')::uuid AND m.is_active AND m.deleted_at IS NULL AND p.deleted_at IS NULL AND (m.role>=15 OR m.role=5 AND (p.guest_view_all_features OR u.id=NULLIF($3,'')::uuid))))`
		}
		args := []any{in.WorkspaceID, in.ProjectID, in.CreatorID, in.Draft}
		if kind == property.Issue {
			ids := []string{}
			for _, def := range definitions {
				if def.Kind == property.Issue {
					value, present := in.Submitted[def.ID]
					if !present {
						value = in.Stored[def.ID]
					}
					ids = append(ids, memberValueIDs(value)...)
				}
			}
			for _, id := range ids {
				if _, err := uuid.Parse(id); err != nil {
					return nil, reject(422, "Choose valid work item references.")
				}
			}
			// Lock selected targets and the actor's corresponding project memberships
			// so revocation or movement cannot race an accepted reference write.
			query = `SELECT i.id::text FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id AND pm.member_id=NULLIF($2,'')::uuid AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN(5,15,20) WHERE i.id=ANY($3::uuid[]) AND ` + djangoapi.IssueVisibleSQL("i", "NULLIF($2,'')::uuid", "$1::uuid") + ` ORDER BY i.id`
			if lock {
				query += " FOR SHARE OF i,p,pm"
			}
			args = []any{in.WorkspaceID, in.ActorID, ids}
		}
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			allowed[kind][id] = true
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return nil, err
		}
	}
	return func(kind property.Kind, id string) bool { return allowed[kind][id] }, nil
}

func nonemptyPropertyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case []string:
		return len(v) > 0
	default:
		return true
	}
}
func memberValueIDs(value any) []string {
	switch v := value.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []any:
		out := []string{}
		for _, id := range v {
			if s, ok := id.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func validateItemContextTransition(in ItemValueInput, typeID string, definitions []property.Definition) error {
	if in.Creating && len(in.Stored) == 0 || in.PreviousProjectID == in.ProjectID && in.PreviousTypeID == typeID {
		return nil
	}
	for _, def := range definitions {
		if def.Kind != property.Cascading || !nonemptyPropertyValue(in.Stored[def.ID]) {
			continue
		}
		previous := property.Scope{ProjectID: in.PreviousProjectID, TypeID: in.PreviousTypeID}
		current := property.Scope{ProjectID: in.ProjectID, TypeID: typeID}
		removed, err := property.CascadingSelectionRequiresPreservation(def, def, previous, current, in.Stored[def.ID])
		if err != nil {
			return reject(422, err.Error())
		}
		if removed {
			return reject(409, "Review preservation of cascading selections before changing this item's project or type.")
		}
	}
	return nil
}
