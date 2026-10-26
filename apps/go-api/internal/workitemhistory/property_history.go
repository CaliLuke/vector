package workitemhistory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CaliLuke/go-plane/internal/stickies/content"
)

// A retained field remains discoverable after its definition or attachment has
// gone. Only metadata is returned here; content requires a scoped version read.
func (repository *Repository) propertyHistory(ctx context.Context, request Request) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, visibleItemHistorySQL(`
 WITH matching AS (SELECT v.* FROM work_item_property_versions v JOIN workspaces w ON w.id=v.workspace_id WHERE w.slug=$1 AND v.issue_id=$3),
 latest AS (SELECT DISTINCT ON(property_id) property_id,property_title,created_at FROM matching ORDER BY property_id,created_at DESC,id DESC)
 SELECT COALESCE(jsonb_agg(jsonb_build_object('property_id',l.property_id,'title',l.property_title,'latest_at',l.created_at,'version_count',(SELECT count(*) FROM matching m WHERE m.property_id=l.property_id)) ORDER BY l.created_at DESC,l.property_id),'[]'::jsonb) FROM latest l`, "$3", "$1", "$4::uuid"), request.Slug, request.ProjectID, request.ResourceID, request.UserID).Scan(&data)
	return scannedJSON(data, err, "list retained property histories")
}

// UUID cursors identify immutable revisions, so a concurrent append cannot shift
// or duplicate entries on later pages. The cursor must belong to this field/item.
func (repository *Repository) propertyVersions(ctx context.Context, request Request) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, visibleItemHistorySQL(propertyVersionListSQL, "$3", "$1", "$6::uuid"), request.Slug, request.ProjectID, request.ResourceID, request.PropertyID, request.BeforeID, request.UserID).Scan(&data)
	return scannedJSON(data, err, "list property revisions")
}

func (repository *Repository) propertyVersionDetail(ctx context.Context, request Request) (json.RawMessage, error) {
	var data []byte
	err := repository.database.QueryRow(ctx, visibleItemHistorySQL(`SELECT `+propertyVersionMetadataSQL+` || jsonb_build_object('value',v.value)
 FROM work_item_property_versions v JOIN workspaces w ON w.id=v.workspace_id
 WHERE w.slug=$1 AND v.issue_id=$3 AND v.property_id=$4 AND v.id=$5`, "$3", "$1", "$6::uuid"), request.Slug, request.ProjectID, request.ResourceID, request.PropertyID, request.VersionID, request.UserID).Scan(&data)
	raw, err := scannedJSON(data, err, "read property revision")
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if value, ok := result["value"].(string); ok {
		clean, _, cleanErr := content.ValidateHTML(value)
		if cleanErr != nil {
			return nil, fmt.Errorf("sanitize historical rich text: %w", cleanErr)
		}
		result["value"] = clean
	}
	return json.Marshal(result)
}

const propertyVersionMetadataSQL = `jsonb_build_object('id',v.id,'property_id',v.property_id,'property_title',v.property_title,'source',v.source,'created_at',v.created_at,
 'actor',CASE WHEN v.actor_id IS NULL THEN NULL ELSE jsonb_build_object('id',v.actor_id,'display_name',COALESCE((SELECT NULLIF(u.display_name,'') FROM users u WHERE u.id=v.actor_id),'Former member')) END)`

const propertyVersionListSQL = `WITH matching AS (
 SELECT v.* FROM work_item_property_versions v JOIN workspaces w ON w.id=v.workspace_id
 WHERE w.slug=$1 AND v.issue_id=$3 AND v.property_id=$4
), boundary AS (SELECT created_at,id FROM matching WHERE id=$5),
 selected AS (SELECT * FROM matching WHERE $5::uuid IS NULL OR (created_at,id)<(SELECT created_at,id FROM boundary) ORDER BY created_at DESC,id DESC LIMIT 51),
 page AS (SELECT * FROM selected ORDER BY created_at DESC,id DESC LIMIT 50)
 SELECT jsonb_build_object('results',COALESCE((SELECT jsonb_agg(` + propertyVersionMetadataSQL + ` ORDER BY v.created_at DESC,v.id DESC) FROM page v),'[]'::jsonb),
 'next_cursor',CASE WHEN (SELECT count(*) FROM selected)>50 THEN (SELECT id FROM page ORDER BY created_at,id LIMIT 1) ELSE NULL END)
 WHERE $5::uuid IS NULL OR EXISTS(SELECT 1 FROM boundary)`
