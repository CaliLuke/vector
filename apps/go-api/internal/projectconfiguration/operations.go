package projectconfiguration

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/labelhierarchy"
	"github.com/CaliLuke/go-plane/internal/pythoncompat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const estimateReadSQL = `SELECT jsonb_build_object(
 'id',e.id,'created_at',e.created_at,'updated_at',e.updated_at,'created_by',e.created_by_id,
 'updated_by',e.updated_by_id,'deleted_at',e.deleted_at,'workspace',e.workspace_id,
 'project',e.project_id,'name',e.name,'description',e.description,'type',e.type,
 'last_used',e.last_used,'points',COALESCE((SELECT jsonb_agg(jsonb_build_object(
 'id',ep.id,'created_at',ep.created_at,'updated_at',ep.updated_at,'created_by',ep.created_by_id,
 'updated_by',ep.updated_by_id,'deleted_at',ep.deleted_at,'estimate',ep.estimate_id,
 'workspace',ep.workspace_id,'project',ep.project_id,'key',ep.key,'value',ep.value,
 'description',ep.description) ORDER BY ep.value) FROM estimate_points ep
 WHERE ep.estimate_id=e.id AND ep.deleted_at IS NULL),'[]'::jsonb))
 FROM estimates e WHERE e.id=$1`

const estimatePointReadSQL = `SELECT jsonb_build_object('id',ep.id,'created_at',ep.created_at,
 'updated_at',ep.updated_at,'created_by',ep.created_by_id,'updated_by',ep.updated_by_id,
 'deleted_at',ep.deleted_at,'estimate',ep.estimate_id,'workspace',ep.workspace_id,
 'project',ep.project_id,'key',ep.key,'value',ep.value,'description',ep.description)
 FROM estimate_points ep WHERE ep.id=$1`

func (r *Repository) PatchEstimate(ctx context.Context, user, slug, project, estimateID string, body io.Reader, contentType string) (json.RawMessage, error) {
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	if err := requireObjectInput(input); err != nil {
		return nil, err
	}
	pointsValue, present := input.Fields["estimate_points"]
	points, ok := pointsValue.([]any)
	if !present || !ok || len(points) == 0 {
		return nil, failure(http.StatusBadRequest, "error", "Estimate points are required")
	}
	var exists bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM estimates e JOIN projects p ON p.id=e.project_id JOIN workspaces w ON w.id=e.workspace_id WHERE e.id=$1 AND p.id=$2 AND w.slug=$3 AND e.deleted_at IS NULL)`, estimateID, project, slug).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, failure(http.StatusNotFound, "error", "The required object does not exist.")
	}
	if estimate, ok := input.Fields["estimate"].(map[string]any); ok && len(estimate) > 0 {
		name, typ := any(nil), any(nil)
		if value, present := estimate["name"]; present {
			name = djangoinput.ScalarString(value)
		}
		if value, present := estimate["type"]; present {
			typ = djangoinput.ScalarString(value)
		}
		_, err = r.db.Exec(ctx, `UPDATE estimates SET name=COALESCE($4,name),type=COALESCE($5,type),updated_at=clock_timestamp(),updated_by_id=$6 WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3)`, estimateID, project, slug, name, typ, user)
		if err != nil {
			return nil, err
		}
	}
	for _, raw := range points {
		point, valid := raw.(map[string]any)
		if !valid {
			continue
		}
		id, _ := point["id"].(string)
		if _, err := uuid.Parse(id); err != nil {
			continue
		}
		var key, value any
		if v, ok := point["key"]; ok {
			n, valid := integerValue(v)
			if !valid {
				return nil, failure(400, "estimate_points", "A valid integer is required.")
			}
			key = n
		}
		if v, ok := point["value"]; ok {
			text, problems := djangoinput.CharField(v, false, nil)
			if len(problems) > 0 {
				return nil, failure(400, "estimate_points", problems[0])
			}
			if len([]rune(text)) > 20 {
				return nil, failure(400, "estimate_points", "Value can't be more than 20 characters")
			}
			value = text
		}
		_, err := r.db.Exec(ctx, `UPDATE estimate_points SET key=COALESCE($4,key),value=COALESCE($5,value),updated_at=clock_timestamp() WHERE id=$1 AND estimate_id=$2 AND project_id=$3 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$6) AND deleted_at IS NULL`, id, estimateID, project, key, value, slug)
		if err != nil {
			return nil, err
		}
	}
	return scanJSON(ctx, r.db, estimateReadSQLForID, estimateID)
}

const estimateReadSQLForID = `SELECT jsonb_build_object('id',e.id,'created_at',e.created_at,'updated_at',e.updated_at,'created_by',e.created_by_id,'updated_by',e.updated_by_id,'deleted_at',e.deleted_at,'workspace',e.workspace_id,'project',e.project_id,'name',e.name,'description',e.description,'type',e.type,'last_used',e.last_used,'points',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',ep.id,'created_at',ep.created_at,'updated_at',ep.updated_at,'created_by',ep.created_by_id,'updated_by',ep.updated_by_id,'deleted_at',ep.deleted_at,'estimate',ep.estimate_id,'workspace',ep.workspace_id,'project',ep.project_id,'key',ep.key,'value',ep.value,'description',ep.description) ORDER BY ep.value) FROM estimate_points ep WHERE ep.estimate_id=e.id AND ep.deleted_at IS NULL),'[]'::jsonb)) FROM estimates e WHERE e.id=$1`

func (r *Repository) DeleteEstimate(ctx context.Context, user, slug, project, estimateID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	writer := &Repository{db: tx}
	if err = writer.Authorize(ctx, user, slug, project, true); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock(p.workspace_id) FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2`, project, slug); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `UPDATE estimates SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1 AND project_id=$3 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$4) AND deleted_at IS NULL`, estimateID, user, project, slug)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return failure(404, "error", "The required object does not exist.")
	}
	if _, err = tx.Exec(ctx, `UPDATE projects SET estimate_id=NULL,updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND estimate_id=$4`, project, user, slug, estimateID); err != nil {
		return err
	}
	if err = writer.enqueueWorkItemEffect(ctx, "soft_delete", map[string]string{"app": "db", "model": "estimate", "id": estimateID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) CreateEstimatePoint(ctx context.Context, user, slug, project, estimateID string, body io.Reader, contentType string) (json.RawMessage, error) {
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	if err := requireObjectInput(input); err != nil {
		return nil, err
	}
	key, keyOK := input.Fields["key"]
	value, valueOK := input.Fields["value"]
	if !keyOK || !valueOK || !truthy(key) || !truthy(value) {
		return nil, failure(400, "error", "Key and value are required")
	}
	n, valid := integerValue(key)
	if !valid || n < 0 {
		return nil, failure(400, "key", "A valid integer is required.")
	}
	text, problems := djangoinput.CharField(value, false, nil)
	if len(problems) > 0 {
		return nil, failure(400, "value", problems[0])
	}
	// The source create call only passes estimate/key/value; a request
	// description is ignored and the model default remains in effect.
	description := ""
	var id string
	err = r.db.QueryRow(ctx, `INSERT INTO estimate_points(id,estimate_id,project_id,workspace_id,key,value,description,created_at,updated_at,created_by_id,updated_by_id) SELECT $1,e.id,p.id,p.workspace_id,$4,$5,$6,clock_timestamp(),clock_timestamp(),$7,NULL FROM estimates e JOIN projects p ON p.id=e.project_id JOIN workspaces w ON w.id=p.workspace_id WHERE e.id=$2 AND p.id=$3 AND w.slug=$8 AND e.deleted_at IS NULL RETURNING id::text`, uuid.NewString(), estimateID, project, n, text, description, user, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, failure(404, "error", "Estimate not found")
	}
	if err != nil {
		return nil, err
	}
	return scanJSON(ctx, r.db, estimatePointReadSQL, id)
}

func (r *Repository) PatchEstimatePoint(ctx context.Context, user, slug, project, estimateID, pointID string, body io.Reader, contentType string) (json.RawMessage, error) {
	if _, err := uuid.Parse(pointID); err != nil {
		return nil, failure(http.StatusBadRequest, "error", "Please provide valid detail")
	}
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM estimate_points ep JOIN projects p ON p.id=ep.project_id JOIN workspaces w ON w.id=ep.workspace_id WHERE ep.id=$1 AND ep.estimate_id=$2 AND p.id=$3 AND w.slug=$4 AND ep.deleted_at IS NULL)`, pointID, estimateID, project, slug).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, failure(404, "error", "The required object does not exist.")
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	sets := []string{}
	args := []any{pointID}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if value, ok := input.Fields["key"]; ok {
		n, valid := integerValue(value)
		if !valid || n < 0 {
			return nil, failure(400, "key", "A valid integer is required.")
		}
		add("key", n)
	}
	if value, ok := input.Fields["value"]; ok {
		text, problems := djangoinput.CharField(value, false, nil)
		if len(problems) > 0 {
			return nil, failure(400, "value", problems[0])
		}
		if len([]rune(text)) > 20 {
			return nil, failure(400, "value", "Value can't be more than 20 characters")
		}
		add("value", text)
	}
	if value, ok := input.Fields["description"]; ok {
		text, valid := value.(string)
		if !valid {
			return nil, failure(400, "description", "Not a valid string.")
		}
		add("description", text)
	}
	if len(sets) > 0 {
		args = append(args, user)
		sets = append(sets, "updated_by_id=$"+fmt.Sprint(len(args)))
		args = append(args, estimateID, project, slug)
		query := `UPDATE estimate_points SET ` + strings.Join(sets, ",") + `,updated_at=clock_timestamp() WHERE id=$1 AND estimate_id=$` + fmt.Sprint(len(args)-2) + ` AND project_id=$` + fmt.Sprint(len(args)-1) + ` AND workspace_id=(SELECT id FROM workspaces WHERE slug=$` + fmt.Sprint(len(args)) + `)`
		if _, err = r.db.Exec(ctx, query, args...); err != nil {
			return nil, err
		}
	}
	return scanJSON(ctx, r.db, estimatePointReadSQL, pointID)
}

func (r *Repository) DeleteEstimatePoint(ctx context.Context, user, slug, project, estimateID, pointID string, body io.Reader, contentType string) (json.RawMessage, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	writer := &Repository{db: tx}
	if err = writer.Authorize(ctx, user, slug, project, true); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `SELECT work_item_schema_lock(p.workspace_id) FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$1 AND w.slug=$2`, project, slug); err != nil {
		return nil, err
	}
	result, err := writer.deleteEstimatePoint(ctx, user, slug, project, estimateID, pointID, body, contentType)
	if err != nil {
		return nil, err
	}
	if err = writer.enqueueWorkItemEffect(ctx, "soft_delete", map[string]string{"app": "db", "model": "estimatepoint", "id": pointID}); err != nil {
		return nil, err
	}
	return result, tx.Commit(ctx)
}

func (r *Repository) deleteEstimatePoint(ctx context.Context, user, slug, project, estimateID, pointID string, body io.Reader, contentType string) (json.RawMessage, error) {
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	newEstimate := input.Fields["new_estimate_id"]
	requested := any(nil)
	if newEstimate != nil && truthy(newEstimate) {
		requested = fmt.Sprint(newEstimate)
	}
	var exists bool
	err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM estimate_points ep JOIN workspaces w ON w.id=ep.workspace_id WHERE ep.id=$1 AND ep.estimate_id=$2 AND ep.project_id=$3 AND w.slug=$4 AND ep.deleted_at IS NULL)`, pointID, estimateID, project, slug).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, failure(404, "error", "Estimate point not found")
	}
	if requested != nil {
		err = r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM estimate_points ep JOIN workspaces w ON w.id=ep.workspace_id WHERE ep.id=$1 AND ep.project_id=$2 AND w.slug=$3 AND ep.deleted_at IS NULL)`, requested, project, slug).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, failure(404, "error", "Estimate point not found")
		}
	}
	var issues []issueActivityRow
	rows, err := r.db.Query(ctx, `SELECT id::text,estimate_point_id::text FROM issues WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND estimate_point_id=$3 AND deleted_at IS NULL`, project, slug, pointID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var issue issueActivityRow
		if err = rows.Scan(&issue.ID, &issue.CurrentPoint); err != nil {
			rows.Close()
			return nil, err
		}
		issues = append(issues, issue)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for _, issue := range issues {
		requestedData, err := pythoncompat.Dumps(map[string]any{"estimate_point": requested}, nil)
		if err != nil {
			return nil, err
		}
		current, err := pythoncompat.Dumps(map[string]any{"estimate_point": issue.CurrentPoint}, nil)
		if err != nil {
			return nil, err
		}
		args := map[string]any{"type": "issue.activity.updated", "requested_data": requestedData, "actor_id": user, "issue_id": issue.ID, "project_id": project, "current_instance": current, "epoch": 0}
		if err := r.enqueueWorkItemEffect(ctx, "activity", args); err != nil {
			return nil, err
		}
	}
	if _, err = r.db.Exec(ctx, `UPDATE issues SET estimate_point_id=$1,updated_at=clock_timestamp() WHERE project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND estimate_point_id=$4 AND deleted_at IS NULL`, requested, project, slug, pointID); err != nil {
		return nil, err
	}
	var oldKey int64
	err = r.db.QueryRow(ctx, `SELECT key FROM estimate_points WHERE id=$1`, pointID).Scan(&oldKey)
	if err != nil {
		return nil, err
	}
	_, err = r.db.Exec(ctx, `UPDATE estimate_points SET key=key-1,updated_at=clock_timestamp() WHERE estimate_id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND key>$4 AND deleted_at IS NULL`, estimateID, project, slug, oldKey)
	if err != nil {
		return nil, err
	}
	_, err = r.db.Exec(ctx, `UPDATE estimate_points SET deleted_at=clock_timestamp(),updated_at=clock_timestamp(),updated_by_id=$2 WHERE id=$1`, pointID, user)
	if err != nil {
		return nil, err
	}
	data, err := djangoJSON(ctx, r.db, `SELECT COALESCE(jsonb_agg(jsonb_build_object('id',ep.id,'created_at',ep.created_at,'updated_at',ep.updated_at,'created_by',ep.created_by_id,'updated_by',ep.updated_by_id,'deleted_at',ep.deleted_at,'estimate',ep.estimate_id,'workspace',ep.workspace_id,'project',ep.project_id,'key',ep.key,'value',ep.value,'description',ep.description) ORDER BY ep.key),'[]'::jsonb) FROM estimate_points ep WHERE ep.estimate_id=$1 AND ep.project_id=$2 AND ep.workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND ep.deleted_at IS NULL`, estimateID, project, slug)
	return json.RawMessage(data), err
}

type issueActivityRow struct {
	ID           string
	CurrentPoint *string
}

func scanJSON(ctx context.Context, db database, query, id string) (json.RawMessage, error) {
	var data []byte
	err := db.QueryRow(ctx, query, id).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, failure(404, "error", "The required object does not exist.")
	}
	return json.RawMessage(data), err
}
func djangoJSON(ctx context.Context, db database, query string, args ...any) ([]byte, error) {
	var data []byte
	err := db.QueryRow(ctx, query, args...).Scan(&data)
	return data, err
}
func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case json.Number:
		return v != "0"
	case int:
		return v != 0
	case int64:
		return v != 0
	default:
		return true
	}
}

const labelReadSQL = `SELECT jsonb_build_object('id',l.id,'created_at',l.created_at,'updated_at',l.updated_at,'created_by',l.created_by_id,'updated_by',l.updated_by_id,'deleted_at',l.deleted_at,'workspace',l.workspace_id,'project',l.project_id,'name',l.name,'description',l.description,'color',l.color,'sort_order',l.sort_order,'external_source',l.external_source,'external_id',l.external_id,'parent',l.parent_id) FROM labels l WHERE l.id=$1`

func (r *Repository) WriteAPILabel(ctx context.Context, user, slug, project, labelID, method string, body io.Reader, contentType string) (json.RawMessage, error) {
	if method == http.MethodPatch || method == http.MethodDelete {
		var exists bool
		err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM labels l JOIN projects p ON p.id=l.project_id JOIN workspaces w ON w.id=l.workspace_id WHERE l.id=$1 AND p.id=$2 AND w.slug=$3 AND p.archived_at IS NULL AND l.deleted_at IS NULL AND EXISTS(SELECT 1 FROM project_members pm WHERE pm.project_id=p.id AND pm.member_id=$4 AND pm.is_active))`, labelID, project, slug, user).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, failure(404, "error", "The requested resource does not exist.")
		}
	}
	if method == http.MethodDelete {
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
		workspaceID, err := labelhierarchy.LockProject(ctx, tx, project, slug)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, failure(404, "error", "The requested resource does not exist.")
		}
		if err != nil {
			return nil, err
		}
		deleted, err := labelhierarchy.SoftDeleteTree(ctx, tx, labelID, user, project, workspaceID)
		if errors.Is(err, labelhierarchy.ErrOutOfScopeChild) {
			return nil, failure(409, "error", "The label hierarchy contains an out-of-scope child.")
		}
		if err != nil {
			return nil, err
		}
		if !deleted {
			return nil, failure(404, "error", "The requested resource does not exist.")
		}
		return nil, tx.Commit(ctx)
	}
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	if input.Null {
		return nil, &HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"No data provided"}}}
	}
	if input.Kind != "" {
		return nil, &HTTPError{Status: 400, Body: map[string]any{"non_field_errors": []string{"Invalid data. Expected a dictionary, but got " + input.Kind + "."}}}
	}
	targetID := labelID
	if method == http.MethodPost {
		targetID = uuid.NewString()
	}
	values := map[string]any{}
	fieldErrors := map[string][]string{}
	parentID := ""
	validateText := func(field string, required, allowBlank, allowNull bool, maximum int) {
		value, present := input.FieldValue(field, required, allowBlank, allowNull)
		if !present {
			if required {
				fieldErrors[field] = []string{"This field is required."}
			}
			return
		}
		if value == nil && allowNull {
			values[field] = nil
			return
		}
		text, problems := djangoinput.TextField(value, allowBlank, maximum, nil)
		if len(problems) != 0 {
			fieldErrors[field] = problems
			return
		}
		values[field] = text
	}
	validateText("name", method == http.MethodPost, false, false, 255)
	validateText("color", false, true, false, 255)
	validateText("description", false, true, false, 0)
	validateText("external_source", false, true, true, 255)
	validateText("external_id", false, true, true, 255)
	if parent, present := input.FieldValue("parent", false, false, true); present {
		if parent == nil {
			values["parent_id"] = nil
		} else {
			parsedParentID, parseErr := djangoinput.UUID(parent)
			if parseErr != nil {
				fieldErrors["parent"] = []string{"“" + djangoinput.ScalarString(parent) + "” is not a valid UUID."}
			} else {
				parentID = parsedParentID.String()
			}
		}
	}
	if sortOrder, present := input.FieldValue("sort_order", false, false, false); present {
		if sortOrder == nil {
			fieldErrors["sort_order"] = []string{"This field may not be null."}
		} else if parsed, message := djangoinput.FloatField(sortOrder); message != "" {
			fieldErrors["sort_order"] = []string{message}
		} else {
			values["sort_order"] = parsed
		}
	}
	if len(fieldErrors) != 0 {
		return nil, &HTTPError{Status: http.StatusBadRequest, Body: fieldErrors}
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	db := tx
	workspaceID, err := labelhierarchy.LockProject(ctx, tx, project, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, failure(404, "error", "The requested resource does not exist.")
	}
	if err != nil {
		return nil, err
	}
	if method != http.MethodPost {
		var found string
		err = tx.QueryRow(ctx, `SELECT label.id::text FROM labels label
 JOIN projects project ON project.id=label.project_id
 WHERE label.id=$1 AND label.project_id=$2 AND label.workspace_id=$3 AND label.deleted_at IS NULL
 AND project.archived_at IS NULL
 AND EXISTS (SELECT 1 FROM project_members member WHERE member.project_id=label.project_id
  AND member.member_id=$4 AND member.is_active)
 FOR UPDATE OF label`, labelID, project, workspaceID, user).Scan(&found)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, failure(404, "error", "The requested resource does not exist.")
		}
		if err != nil {
			return nil, err
		}
	}
	if parentID != "" {
		valid, validationErr := labelhierarchy.ValidateParent(ctx, tx, targetID, parentID, project, workspaceID)
		if validationErr != nil {
			return nil, validationErr
		}
		if !valid {
			parent, _ := input.FieldValue("parent", false, false, true)
			return nil, &HTTPError{Status: http.StatusBadRequest, Body: map[string][]string{
				"parent": {"Invalid pk \"" + djangoinput.ScalarString(parent) + "\" - object does not exist."},
			}}
		}
		values["parent_id"] = parentID
	}
	if src, srcOK := values["external_source"]; srcOK && truthy(src) {
		if ext, extOK := values["external_id"]; extOK && truthy(ext) {
			var conflictID string
			err := db.QueryRow(ctx, `SELECT id::text FROM labels WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND external_source=$3 AND external_id=$4 AND deleted_at IS NULL AND ($5='' OR id::text<>$5) LIMIT 1`, project, slug, src, ext, labelID).Scan(&conflictID)
			if err == nil {
				return nil, &HTTPError{Status: 409, Body: map[string]any{"error": "Label with the same external id and external source already exists", "id": conflictID}}
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
	}
	if method == http.MethodPost {
		name := values["name"]
		var conflictID string
		err := db.QueryRow(ctx, `SELECT id::text FROM labels WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND name=$3 AND deleted_at IS NULL LIMIT 1`, project, slug, name).Scan(&conflictID)
		if err == nil {
			return nil, &HTTPError{Status: 409, Body: map[string]any{"error": "Label with the same name already exists in the project", "id": conflictID}}
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		color, _ := values["color"].(string)
		description := stringValueOrDefault(values, "description", "")
		externalSource := nullableLabelValue(values, "external_source")
		externalID := nullableLabelValue(values, "external_id")
		var parent any
		if value, ok := values["parent_id"]; ok && value != nil {
			parent = value
		}
		_, err = db.Exec(ctx, `INSERT INTO labels(id,name,color,parent_id,project_id,workspace_id,sort_order,description,external_source,external_id,created_at,updated_at,created_by_id,updated_by_id) SELECT $1,$2,$3,$4,p.id,p.workspace_id,COALESCE((SELECT MAX(sort_order)+10000 FROM labels WHERE project_id=p.id AND deleted_at IS NULL),65535),$5,$6,$7,clock_timestamp(),clock_timestamp(),$8,NULL FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$9 AND w.slug=$10`, targetID, name, color, parent, description, externalSource, externalID, user, project, slug)
		if err != nil {
			return nil, err
		}
		data, err := scanJSON(ctx, db, labelReadSQL, targetID)
		if err != nil {
			return nil, err
		}
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
		return data, nil
	}
	args := []any{labelID}
	sets := []string{}
	for _, field := range []string{"name", "color", "description", "external_source", "external_id", "parent_id", "sort_order"} {
		value, ok := values[field]
		if !ok {
			continue
		}
		if (field == "external_source" || field == "external_id") && value == nil {
			value = nil
		}
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s=$%d", field, len(args)))
	}
	if len(sets) > 0 {
		args = append(args, user)
		_, err = db.Exec(ctx, `UPDATE labels SET `+strings.Join(sets, ",")+`,updated_by_id=$`+fmt.Sprint(len(args))+`,updated_at=clock_timestamp() WHERE id=$1`, args...)
		if err != nil {
			return nil, err
		}
	}
	data, err := scanJSON(ctx, db, labelReadSQL, labelID)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return data, nil
}

func stringValueOrDefault(values map[string]any, key, fallback string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return fallback
}

func nullableLabelValue(values map[string]any, key string) any {
	if value, ok := values[key]; ok {
		return value
	}
	return nil
}

func (r *Repository) BulkCreateLabels(ctx context.Context, user, slug, project string, body io.Reader, contentType string) (json.RawMessage, error) {
	input, err := djangoinput.Parse(body, contentType)
	if err != nil {
		return nil, err
	}
	if err := requireObjectInput(input); err != nil {
		return nil, err
	}
	value, present := input.Fields["label_data"]
	labels := []any{}
	if present {
		var ok bool
		labels, ok = value.([]any)
		if !ok {
			return nil, failure(400, "label_data", "Expected a list of items but got type \""+djangoinput.PythonType(value)+"\".")
		}
	}
	result := make([]any, 0)
	for _, raw := range labels {
		label, ok := raw.(map[string]any)
		if !ok {
			return nil, failure(400, "label_data", "Expected a dictionary of items.")
		}
		name := "Migrated"
		if value, ok := label["name"]; ok {
			name = djangoinput.ScalarString(value)
		}
		description := "Migrated Issue"
		if value, ok := label["description"]; ok {
			description = djangoinput.ScalarString(value)
		}
		colorNumber, err := rand.Int(rand.Reader, big.NewInt(0x1000001))
		if err != nil {
			return nil, err
		}
		color := fmt.Sprintf("#%06X", colorNumber.Int64())
		id := uuid.NewString()
		var data []byte
		err = r.db.QueryRow(ctx, `INSERT INTO labels(id,name,color,project_id,workspace_id,sort_order,description,created_at,updated_at,created_by_id,updated_by_id) SELECT $1,$2,$3,p.id,p.workspace_id,65535,$4,clock_timestamp(),clock_timestamp(),$5,$5 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=$6 AND w.slug=$7 ON CONFLICT DO NOTHING RETURNING jsonb_build_object('parent',parent_id,'name',name,'color',color,'id',id,'project_id',project_id,'workspace_id',workspace_id,'sort_order',sort_order)`, id, name, color, description, user, project, slug).Scan(&data)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var row any
		if err = json.Unmarshal(data, &row); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	encoded, err := json.Marshal(map[string]any{"labels": result})
	return json.RawMessage(encoded), err
}

func (r *Repository) AuthorizeBulkLabels(ctx context.Context, user, slug, project string) error {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3 AND pm.is_active=true AND pm.deleted_at IS NULL AND pm.role=20)`, user, project, slug).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return failure(403, "detail", "You do not have permission to perform this action.")
	}
	return nil
}

// Estimate reassignment uses the same durable work-item effect queue as edits.
// Callers bind this repository to the transaction that changes the item values.
func (r *Repository) enqueueWorkItemEffect(ctx context.Context, kind string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `INSERT INTO work_item_effects(id,kind,payload) VALUES($1,$2,$3)`, uuid.NewString(), kind, raw)
	return err
}
