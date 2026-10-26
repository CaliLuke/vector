package workitemtypes

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Discovery identities contain no issue counts, private-draft usage or administration data.
type TypeIdentity struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Color   string `json:"color"`
	Active  bool   `json:"active"`
	Default bool   `json:"default"`
	Level   int    `json:"level"`
}

type TypeIdentityCatalog struct {
	Enabled  bool           `json:"enabled"`
	Revision int64          `json:"revision"`
	Types    []TypeIdentity `json:"types"`
}

const discoveryTypeLimit = 1000

func (c TypeIdentityCatalog) TypeIDs() map[string]bool {
	ids := make(map[string]bool, len(c.Types))
	for _, item := range c.Types {
		ids[item.ID] = true
	}
	return ids
}

// WithTypeDiscoveryRead keeps authorization, identities and the caller's list/count
// queries in one snapshot. The callback must use tx for every discovery read.
func WithTypeDiscoveryRead[T any](ctx context.Context, db Database, actor, slug, project string, read func(pgx.Tx, TypeIdentityCatalog) (T, error)) (T, error) {
	var zero T
	tx, err := db.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return zero, err
	}
	catalog, err := readTypeIdentities(ctx, tx, actor, slug, project)
	if err != nil {
		return zero, err
	}
	result, err := read(tx, catalog)
	if err != nil {
		return zero, err
	}
	return result, tx.Commit(ctx)
}

func (r *Repository) TypeIdentities(ctx context.Context, actor, slug, project string) (TypeIdentityCatalog, error) {
	return WithTypeDiscoveryRead(ctx, r.db, actor, slug, project, func(_ pgx.Tx, catalog TypeIdentityCatalog) (TypeIdentityCatalog, error) {
		return catalog, nil
	})
}

func readTypeIdentities(ctx context.Context, tx pgx.Tx, actor, slug, project string) (TypeIdentityCatalog, error) {
	out := TypeIdentityCatalog{Types: []TypeIdentity{}}
	var workspace string
	err := tx.QueryRow(ctx, `SELECT w.id::text,COALESCE(s.enabled,false),COALESCE(s.revision,0)
 FROM workspaces w JOIN workspace_members member ON member.workspace_id=w.id
 LEFT JOIN work_item_type_settings s ON s.workspace_id=w.id
 WHERE w.slug=$1 AND w.deleted_at IS NULL AND member.member_id=$2
 AND member.is_active AND member.deleted_at IS NULL AND member.role IN (5,15,20)
 AND ($3='' OR EXISTS (SELECT 1 FROM projects p JOIN project_members pm ON pm.project_id=p.id AND pm.workspace_id=p.workspace_id
 WHERE p.id::text=$3 AND p.workspace_id=w.id AND p.deleted_at IS NULL
 AND pm.member_id=$2 AND pm.is_active AND pm.deleted_at IS NULL AND pm.role IN (5,15,20)))`, slug, actor, project).Scan(&workspace, &out.Enabled, &out.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, reject(403, "You do not have access to these work item types.")
	}
	if err != nil {
		return out, err
	}
	// Paused and unimported identities remain meaningful in saved filters.
	rows, err := tx.Query(ctx, `SELECT id::text,name,COALESCE(logo_props->>'name','task'),COALESCE(logo_props->>'color','#8b5cf6'),is_active,is_default,level::integer
 FROM issue_types WHERE workspace_id=$1 AND deleted_at IS NULL ORDER BY is_default DESC,created_at,id LIMIT $2`, workspace, discoveryTypeLimit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item TypeIdentity
		if err = rows.Scan(&item.ID, &item.Name, &item.Icon, &item.Color, &item.Active, &item.Default, &item.Level); err != nil {
			return out, err
		}
		out.Types = append(out.Types, item)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.Types) > discoveryTypeLimit {
		return TypeIdentityCatalog{}, reject(422, "The workspace has too many work item types to load this filter catalog.")
	}
	return out, nil
}
