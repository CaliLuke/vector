package workitemtypes

import (
	"context"
	"encoding/json"
	"time"

	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

// PropertyDiscoverySnapshot is request-local. Definitions may contain other
// contexts internally, so this type is not a client catalog or response DTO.
// Only plans resolved for authorized project/type scopes may leave this boundary.
type PropertyDiscoverySnapshot struct {
	tx          pgx.Tx
	workspace   string
	actor       string
	references  map[property.Kind]map[string]bool
	typeIDs     map[string]bool
	Types       TypeIdentityCatalog
	EvaluatedAt time.Time
	definitions map[string][]property.Definition
	projects    map[string]bool
	plans       map[property.Scope]*ItemProjectionPlan
}

const discoveryPropertyLimit = 1000

// WithPropertyDiscoveryRead loads definitions and scope once, under the same
// snapshot that the callback must use for candidate reads, counts and pagination.
func WithPropertyDiscoveryRead[T any](ctx context.Context, db Database, actor, slug, project string, read func(pgx.Tx, *PropertyDiscoverySnapshot) (T, error)) (T, error) {
	return withPropertyDiscoveryRead(ctx, db, actor, slug, project, false, read)
}

// WithPropertyProjectionRead permits disabled properties for presentation-only
// reads. Callers with type/property predicates must use the strict read above.
// Access, malformed definitions and all other failures remain errors.
func WithPropertyProjectionRead[T any](ctx context.Context, db Database, actor, slug, project string, read func(pgx.Tx, *PropertyDiscoverySnapshot) (T, error)) (T, error) {
	return withPropertyDiscoveryRead(ctx, db, actor, slug, project, true, read)
}

func withPropertyDiscoveryRead[T any](ctx context.Context, db Database, actor, slug, project string, allowDisabled bool, read func(pgx.Tx, *PropertyDiscoverySnapshot) (T, error)) (T, error) {
	return WithTypeDiscoveryRead(ctx, db, actor, slug, project, func(tx pgx.Tx, types TypeIdentityCatalog) (T, error) {
		var zero T
		if !types.Enabled && !allowDisabled {
			return zero, reject(400, "Work item properties are disabled. Update this filter.")
		}
		snapshot, err := loadPropertyDiscovery(ctx, tx, actor, slug, project, types)
		if err != nil {
			return zero, err
		}
		return read(tx, snapshot)
	})
}

func loadPropertyDiscovery(ctx context.Context, tx pgx.Tx, actor, slug, project string, types TypeIdentityCatalog) (*PropertyDiscoverySnapshot, error) {
	snapshot := &PropertyDiscoverySnapshot{tx: tx, actor: actor, Types: types, typeIDs: types.TypeIDs(), definitions: map[string][]property.Definition{}, projects: map[string]bool{}, plans: map[property.Scope]*ItemProjectionPlan{}}
	var workspace string
	if err := tx.QueryRow(ctx, `SELECT id::text,transaction_timestamp() FROM workspaces WHERE slug=$1 AND deleted_at IS NULL`, slug).Scan(&workspace, &snapshot.EvaluatedAt); err != nil {
		return nil, err
	}
	snapshot.workspace, snapshot.references = workspace, map[property.Kind]map[string]bool{}
	snapshot.EvaluatedAt = snapshot.EvaluatedAt.UTC().Truncate(time.Microsecond)
	rows, err := tx.Query(ctx, `SELECT p.id::text FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id WHERE p.workspace_id=$1 AND p.deleted_at IS NULL AND pm.member_id=$2 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN (5,15,20) AND ($3='' OR p.id::text=$3)`, workspace, actor, project)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		snapshot.projects[id] = true
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if !types.Enabled {
		return snapshot, nil
	}
	snapshot.definitions, err = discoveryDefinitions(ctx, tx, workspace, snapshot.typeIDs)
	return snapshot, err
}

// Read each definition once, even when attached to many types. The bounded
// definition count rejects overflow rather than silently omitting predicates.
func discoveryDefinitions(ctx context.Context, tx pgx.Tx, workspace string, typeIDs map[string]bool) (map[string][]property.Definition, error) {
	rows, err := tx.Query(ctx, `SELECT p.id::text,p.definition FROM work_item_properties p WHERE p.workspace_id=$1 AND EXISTS(SELECT 1 FROM work_item_type_properties link JOIN issue_types t ON t.id=link.type_id AND t.workspace_id=p.workspace_id WHERE link.property_id=p.id AND t.deleted_at IS NULL) ORDER BY p.id LIMIT $2`, workspace, discoveryPropertyLimit+1)
	if err != nil {
		return nil, err
	}
	definitions := map[string]property.Definition{}
	for rows.Next() {
		var id string
		var raw []byte
		var def property.Definition
		if err = rows.Scan(&id, &raw); err == nil {
			err = json.Unmarshal(raw, &def)
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		definitions[id] = def
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(definitions) > discoveryPropertyLimit {
		return nil, reject(422, "The workspace has too many work item properties to load this discovery catalog.")
	}
	links, err := tx.Query(ctx, `SELECT link.type_id::text,link.property_id::text FROM work_item_type_properties link JOIN work_item_properties p ON p.id=link.property_id JOIN issue_types t ON t.id=link.type_id AND t.workspace_id=p.workspace_id WHERE p.workspace_id=$1 AND t.deleted_at IS NULL ORDER BY link.type_id,link.sort_order,p.id`, workspace)
	if err != nil {
		return nil, err
	}
	defer links.Close()
	out := map[string][]property.Definition{}
	for links.Next() {
		var typeID, propertyID string
		if err = links.Scan(&typeID, &propertyID); err != nil {
			return nil, err
		}
		if def, exists := definitions[propertyID]; exists && typeIDs[typeID] {
			out[typeID] = append(out[typeID], def)
		}
	}
	return out, links.Err()
}

// Project accepts only a row already admitted by the caller's item-visibility
// predicate. Project membership alone is not sufficient for guest item access.
// Every row uses this snapshot's time and cached compiled scope plan.
func (snapshot *PropertyDiscoverySnapshot) Project(projectID, typeID string, standard ItemFormulaInputs, saved map[string]any) (ItemProjection, error) {
	if err := snapshot.validateItemScope(projectID, typeID); err != nil {
		return ItemProjection{}, err
	}
	if !snapshot.Types.Enabled || typeID == "" {
		return ItemProjection{Values: map[string]ProjectedValue{}, Computed: map[string]FormulaResult{}, EvaluatedAt: snapshot.EvaluatedAt}, nil
	}
	scope := property.Scope{ProjectID: projectID, TypeID: typeID}
	plan, exists := snapshot.plans[scope]
	if !exists {
		var err error
		plan, err = CompileItemProjection(snapshot.definitions[typeID], projectID, typeID)
		if err != nil {
			return ItemProjection{}, err
		}
		snapshot.plans[scope] = plan
	}
	return plan.Project(standard, saved, snapshot.EvaluatedAt, false), nil
}

func (snapshot *PropertyDiscoverySnapshot) validateItemScope(projectID, typeID string) error {
	if !snapshot.projects[projectID] {
		return reject(403, "Project is unavailable in this discovery scope.")
	}
	if snapshot.Types.Enabled && typeID != "" && !snapshot.typeIDs[typeID] {
		return reject(400, "Work item type is unavailable in this discovery scope.")
	}
	return nil
}
