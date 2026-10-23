package workspacesidebarpreferences

import (
	"context"
	"encoding/json"
	"fmt"

	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	firstSortOrder = 65535.0
	sortOrderStep  = 10000.0
)

// Patch is the writable subset of a workspace sidebar preference. A nil
// field means that the field was omitted from the request and must be kept.
type Patch struct {
	Key       string   `json:"key"`
	IsPinned  *bool    `json:"is_pinned,omitempty"`
	SortOrder *float64 `json:"sort_order,omitempty"`
}

type Database interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct{ db Database }

func NewRepository(db Database) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("workspace sidebar preferences database is required")
	}
	return &Repository{db: db}, nil
}

// Authorized permits all active workspace roles supported by the Django
// view: guest (5), member (15), and administrator (20).
func (r *Repository) Authorized(ctx context.Context, user uuid.UUID, slug string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_members wm JOIN workspaces w ON w.id=wm.workspace_id WHERE wm.member_id=$1 AND w.slug=$2 AND wm.role IN (5,15,20) AND wm.is_active=true AND wm.deleted_at IS NULL AND w.deleted_at IS NULL)`, user, slug).Scan(&ok)
	return ok, err
}

// List lazily creates the seven Django preference keys. The row_number
// expression is important: Django assigns sort_order based on the order of
// only the missing keys, not based on all seven keys.
func (r *Repository) List(ctx context.Context, user uuid.UUID, slug string) (json.RawMessage, error) {
	ids := make([]uuid.UUID, 7)
	for i := range ids {
		ids[i] = uuid.NewV7()
	}
	var body []byte
	err := r.db.QueryRow(ctx, `WITH w AS (SELECT id FROM workspaces WHERE slug=$2 AND deleted_at IS NULL), keys(key,idx,id) AS (VALUES ('views',0,$3::uuid),('active_cycles',1,$4::uuid),('analytics',2,$5::uuid),('drafts',3,$6::uuid),('your_work',4,$7::uuid),('archives',5,$8::uuid),('stickies',6,$9::uuid)), missing AS (SELECT k.key,k.idx,k.id FROM keys k WHERE NOT EXISTS (SELECT 1 FROM workspace_user_preferences p JOIN w ON w.id=p.workspace_id WHERE p.user_id=$1 AND p.key=k.key AND p.deleted_at IS NULL)), ins AS (INSERT INTO workspace_user_preferences (id,created_at,updated_at,workspace_id,user_id,key,sort_order,is_pinned) SELECT m.id,NOW(),NOW(),w.id,$1,m.key,`+fmt.Sprintf("%g", firstSortOrder)+` + ((row_number() OVER (ORDER BY m.idx)-1)*`+fmt.Sprintf("%g", sortOrderStep)+`),m.key IN ('drafts','your_work','stickies') FROM w CROSS JOIN missing m ON CONFLICT DO NOTHING) SELECT COALESCE(json_object_agg(p.key,json_build_object('is_pinned',p.is_pinned,'sort_order',p.sort_order) ORDER BY p.sort_order),'{}'::json) FROM workspace_user_preferences p JOIN workspaces w ON w.id=p.workspace_id WHERE w.slug=$2 AND p.user_id=$1 AND p.deleted_at IS NULL`, user, slug, ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6]).Scan(&body)
	return body, err
}

// Patch applies the same partial-update and ignore-unknown-key behavior as
// Django. The last occurrence wins when a request repeats a key, matching the
// view's sequential loop.
func (r *Repository) Patch(ctx context.Context, user uuid.UUID, slug string, patches []Patch) (json.RawMessage, error) {
	payload, err := json.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("marshal workspace sidebar preference patch: %w", err)
	}
	var body []byte
	err = r.db.QueryRow(ctx, `WITH updates AS (SELECT DISTINCT ON (item->>'key') item->>'key' AS key,item,(ordinality) FROM jsonb_array_elements($3::jsonb) WITH ORDINALITY AS entries(item,ordinality) WHERE NULLIF(item->>'key','') IS NOT NULL ORDER BY item->>'key',ordinality DESC) UPDATE workspace_user_preferences p SET is_pinned=CASE WHEN updates.item ? 'is_pinned' THEN (updates.item->>'is_pinned')::boolean ELSE p.is_pinned END,sort_order=CASE WHEN updates.item ? 'sort_order' THEN (updates.item->>'sort_order')::double precision ELSE p.sort_order END FROM workspaces w, updates WHERE updates.key=p.key AND p.workspace_id=w.id AND w.slug=$2 AND p.user_id=$1 AND p.deleted_at IS NULL AND w.deleted_at IS NULL RETURNING json_build_object('message','Successfully updated')`, user, slug, payload).Scan(&body)
	// UPDATE ... RETURNING yields no row for an empty patch or unknown keys,
	// while Django still returns its success message.
	if err == pgx.ErrNoRows {
		return json.RawMessage(`{"message":"Successfully updated"}`), nil
	}
	return body, err
}
