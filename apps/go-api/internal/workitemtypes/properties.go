package workitemtypes

import (
	"context"
	"encoding/json"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

type PropertyRecord struct {
	Definition property.Definition `json:"definition"`
	Used       bool                `json:"used"`
}
type PropertyAttachment struct {
	TypeID      string   `json:"type_id"`
	PropertyIDs []string `json:"property_ids"`
}
type PropertyChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type PropertyCatalog struct {
	Enabled      bool                 `json:"enabled"`
	ProjectCount int                  `json:"project_count"`
	Types        []Type               `json:"types"`
	Revision     int64                `json:"revision"`
	CanManage    bool                 `json:"can_manage"`
	Properties   []PropertyRecord     `json:"properties"`
	Attachments  []PropertyAttachment `json:"attachments"`
	Projects     []PropertyChoice     `json:"projects"`
	Members      []PropertyChoice     `json:"members"`
}
type PropertyInput struct {
	Revision    int64
	Action      string
	ID          string
	Definition  property.Definition
	TypeID      string
	PropertyIDs []string
	ReviewID    string
	Permanent   bool
}

func (r *Repository) PropertyCatalog(ctx context.Context, actor, slug string) (_ PropertyCatalog, resultErr error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return PropertyCatalog{}, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return PropertyCatalog{}, err
	}
	workspace, admin, err := authorize(ctx, tx, actor, slug, false)
	if err != nil {
		return PropertyCatalog{}, err
	}
	out, err := loadProperties(ctx, tx, workspace, actor, admin)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}
func loadProperties(ctx context.Context, tx pgx.Tx, workspace, actor string, admin bool) (PropertyCatalog, error) {
	out := PropertyCatalog{CanManage: admin, Properties: []PropertyRecord{}, Attachments: []PropertyAttachment{}, Projects: []PropertyChoice{}, Members: []PropertyChoice{}}
	metadata, err := loadCatalog(ctx, tx, workspace, admin)
	if err != nil {
		return out, err
	}
	out.Enabled, out.Revision, out.ProjectCount, out.Types = metadata.Enabled, metadata.Revision, metadata.ProjectCount, metadata.Types
	rows, err := tx.Query(ctx, `SELECT definition,used FROM work_item_properties WHERE workspace_id=$1 ORDER BY created_at,id`, workspace)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var raw []byte
		var item PropertyRecord
		if err = rows.Scan(&raw, &item.Used); err == nil {
			err = json.Unmarshal(raw, &item.Definition)
		}
		if err != nil {
			rows.Close()
			return out, err
		}
		out.Properties = append(out.Properties, item)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}
	rows, err = tx.Query(ctx, `SELECT t.id::text,COALESCE(array_agg(l.property_id::text ORDER BY l.sort_order,l.property_id) FILTER (WHERE l.property_id IS NOT NULL),'{}') FROM issue_types t LEFT JOIN work_item_type_properties l ON l.type_id=t.id WHERE t.workspace_id=$1 AND t.deleted_at IS NULL GROUP BY t.id ORDER BY t.id`, workspace)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var link PropertyAttachment
		if err = rows.Scan(&link.TypeID, &link.PropertyIDs); err != nil {
			rows.Close()
			return out, err
		}
		out.Attachments = append(out.Attachments, link)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}
	// Context configuration never expands visibility into a private project's name.
	out.Projects, err = propertyChoices(ctx, tx, `SELECT p.id::text,p.name FROM projects p WHERE p.workspace_id=$1 AND p.deleted_at IS NULL AND EXISTS(SELECT 1 FROM project_members m WHERE m.project_id=p.id AND m.member_id=$2 AND m.is_active AND m.deleted_at IS NULL) ORDER BY p.name,p.id`, workspace, actor)
	if err != nil {
		return out, err
	}
	out.Members, err = propertyChoices(ctx, tx, `SELECT u.id::text,COALESCE(NULLIF(u.display_name,''),NULLIF(trim(u.first_name||' '||u.last_name),''),u.username) FROM workspace_members m JOIN users u ON u.id=m.member_id WHERE m.workspace_id=$1 AND m.is_active AND m.deleted_at IS NULL AND u.is_active ORDER BY u.id`, workspace)
	return out, err
}
func propertyChoices(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]PropertyChoice, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PropertyChoice{}
	for rows.Next() {
		var c PropertyChoice
		if err = rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Empty collections are arrays in the persisted contract, never JSON null:
// PostgreSQL's effective configuration resolver distinguishes those values.
func canonicalProperty(def property.Definition) property.Definition {
	def.Configuration = canonicalConfiguration(def.Configuration)
	def.Contexts = append([]property.Context{}, def.Contexts...)
	for i := range def.Contexts {
		c := &def.Contexts[i]
		c.Projects = append([]string{}, c.Projects...)
		c.Types = append([]string{}, c.Types...)
		c.Overrides = append([]string{}, c.Overrides...)
		c.Configuration = canonicalConfiguration(c.Configuration)
	}
	return def
}
func canonicalConfiguration(config property.Configuration) property.Configuration {
	config.Options = append([]property.Option{}, config.Options...)
	config.Levels = append([]property.Level{}, config.Levels...)
	return config
}
