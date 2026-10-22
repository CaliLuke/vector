// Package externalintegrations contains the storage boundary for workspace
// configuration and the small serializer projections used by AI assistance.
package externalintegrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrWorkspaceForbidden = errors.New("workspace role is not allowed")
	ErrProjectForbidden   = errors.New("project role is not allowed")
	ErrWorkspaceMissing   = errors.New("workspace not found")
	ErrProjectMissing     = errors.New("project not found")
)

type ConfigCipher interface{ Decrypt(string) (string, error) }

type Repository struct {
	db     *pgxpool.Pool
	cipher ConfigCipher
}

func NewRepository(db *pgxpool.Pool, cipher ConfigCipher) (*Repository, error) {
	if db == nil || cipher == nil {
		return nil, fmt.Errorf("external integrations database and configuration cipher are required")
	}
	return &Repository{db: db, cipher: cipher}, nil
}

func (r *Repository) Configuration(ctx context.Context, keys []string, environment map[string]string, preferDatabase bool) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	if !preferDatabase {
		for _, key := range keys {
			values[key] = environment[key]
		}
		return values, nil
	}
	rows, err := r.db.Query(ctx, `SELECT key, value, is_encrypted FROM instance_configurations WHERE key = ANY($1::text[]) AND deleted_at IS NULL`, keys)
	if err != nil {
		return nil, fmt.Errorf("read external integration configuration: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value *string
		var encrypted bool
		if err := rows.Scan(&key, &value, &encrypted); err != nil {
			return nil, err
		}
		if value == nil {
			values[key] = ""
			continue
		}
		if encrypted {
			decoded, err := r.cipher.Decrypt(*value)
			if err != nil {
				return nil, fmt.Errorf("decrypt external integration configuration %s: %w", key, err)
			}
			values[key] = decoded
		} else {
			values[key] = *value
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, key := range keys {
		if _, ok := values[key]; !ok {
			values[key] = environment[key]
		}
	}
	return values, nil
}

func (r *Repository) WorkspaceRole(ctx context.Context, userID, slug string) (int, error) {
	var role int
	err := r.db.QueryRow(ctx, `SELECT wm.role FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL LIMIT 1`, userID, slug).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrWorkspaceForbidden
	}
	if err != nil {
		return 0, fmt.Errorf("resolve workspace integration role: %w", err)
	}
	return role, nil
}

func (r *Repository) ProjectAllowed(ctx context.Context, userID, slug, projectID string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM project_members pm JOIN workspaces w ON w.id=pm.workspace_id
 WHERE pm.member_id=$1 AND w.slug=$2 AND pm.project_id=$3 AND pm.is_active=true AND pm.deleted_at IS NULL AND w.deleted_at IS NULL
 AND (pm.role IN (15,20) OR EXISTS (
   SELECT 1 FROM workspace_members wm WHERE wm.workspace_id=w.id AND wm.member_id=$1 AND wm.role=20 AND wm.is_active=true AND wm.deleted_at IS NULL
 ))
)`, userID, slug, projectID).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("authorize project AI integration: %w", err)
	}
	return allowed, nil
}

func (r *Repository) WorkspaceLite(ctx context.Context, slug string) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT json_build_object('name',w.name,'slug',w.slug,'id',w.id,'logo_url',CASE WHEN w.logo_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||w.logo_asset_id||'/' ELSE w.logo END) FROM workspaces w WHERE w.slug=$1 AND w.deleted_at IS NULL LIMIT 1`, slug).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWorkspaceMissing
	}
	if err != nil {
		return nil, fmt.Errorf("read workspace AI projection: %w", err)
	}
	return body, nil
}

func (r *Repository) ProjectLite(ctx context.Context, projectID string) (json.RawMessage, error) {
	var body []byte
	err := r.db.QueryRow(ctx, `SELECT json_build_object('id',p.id,'identifier',p.identifier,'name',p.name,'cover_image',p.cover_image,'cover_image_url',CASE WHEN p.cover_image_asset_id IS NOT NULL THEN '/api/assets/v2/static/'||p.cover_image_asset_id||'/' ELSE p.cover_image END,'logo_props',p.logo_props,'description',p.description) FROM projects p WHERE p.id=$1 AND p.deleted_at IS NULL LIMIT 1`, projectID).Scan(&body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProjectMissing
	}
	if err != nil {
		return nil, fmt.Errorf("read project AI projection: %w", err)
	}
	return body, nil
}
