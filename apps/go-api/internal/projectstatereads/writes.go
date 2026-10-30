package projectstatereads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"golang.org/x/text/unicode/norm"
)

// WriteFailure carries the Django response associated with a state write.
type WriteFailure struct {
	Status int
	Body   any
}

func (failure *WriteFailure) Error() string {
	return fmt.Sprintf("project state write failed (%d)", failure.Status)
}

func missingStateFailure(input StateWriteInput) *WriteFailure {
	if input.App && input.Method != "POST" {
		return &WriteFailure{Status: 404, Body: map[string]string{"error": "The required object does not exist."}}
	}
	if !input.App {
		return &WriteFailure{Status: 404, Body: map[string]string{"error": "The requested resource does not exist."}}
	}
	return &WriteFailure{Status: 404, Body: map[string]string{"detail": "No State matches the given query."}}
}

type stateWriteDatabase interface {
	database
	Begin(context.Context) (pgx.Tx, error)
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type WriteRepository struct {
	*Repository
	database stateWriteDatabase
	cache    StateWriteCache
}

type StateWriteCache interface {
	Invalidate(context.Context, string) error
}

type redisStateWriteCache struct{ client *redis.Client }

func NewStateWriteCache(redisURL string) (*redisStateWriteCache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &redisStateWriteCache{client: redis.NewClient(options)}, nil
}

func (cache *redisStateWriteCache) Close() error { return cache.client.Close() }

func (cache *redisStateWriteCache) Invalidate(ctx context.Context, slug string) error {
	return cache.client.Del(ctx, ":1:workspaces/"+slug+"/states/").Err()
}

func (repository *WriteRepository) SetCache(cache StateWriteCache) { repository.cache = cache }

func (repository *WriteRepository) InvalidateAppCache(ctx context.Context, slug string) {
	if repository.cache != nil {
		_ = repository.cache.Invalidate(ctx, slug)
	}
}

func NewWriteRepository(database stateWriteDatabase) (*WriteRepository, error) {
	if database == nil {
		return nil, errors.New("project state writes database is required")
	}
	base, err := NewRepository(database)
	if err != nil {
		return nil, err
	}
	return &WriteRepository{Repository: base, database: database}, nil
}

// AuthorizeApp applies the session-authenticated app permission decorators.
func (repository *WriteRepository) AuthorizeApp(ctx context.Context, user, slug, project, method string) (bool, error) {
	allowedRole := 20
	if method == "PATCH" {
		allowedRole = 0 // app partial_update also permits members and guests.
	}
	var allowed bool
	err := repository.database.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM project_members pm
 JOIN projects p ON p.id=pm.project_id
 JOIN workspaces w ON w.id=p.workspace_id
 WHERE pm.member_id=$1 AND pm.project_id=$2 AND w.slug=$3
   AND pm.is_active=true AND pm.deleted_at IS NULL
   AND (($4=0 AND pm.role IN (5,15,20)) OR pm.role=$4 OR EXISTS (
     SELECT 1 FROM workspace_members wm
     WHERE wm.member_id=$1 AND wm.workspace_id=w.id AND wm.role=20 AND wm.is_active=true
       AND wm.deleted_at IS NULL
   ))
)`, user, project, slug, allowedRole).Scan(&allowed)
	return allowed, err
}

// AuthorizeAPIV1 applies ProjectEntityPermission's active admin/member write policy.
type StateWriteInput struct {
	UserID, Slug, ProjectID, StateID string
	Method                           string
	App                              bool
	Fields                           map[string]any
	Parsed                           *djangoinput.Input
}

func (repository *WriteRepository) Create(ctx context.Context, input StateWriteInput) (json.RawMessage, error) {
	return repository.mutate(ctx, input, "create")
}

func (repository *WriteRepository) Update(ctx context.Context, input StateWriteInput) (json.RawMessage, error) {
	return repository.mutate(ctx, input, "update")
}

// ResolveForUpdate matches the view's get_object ordering: PATCH resolves the
// scoped, live state before it reads or validates the request body. Update
// repeats the lookup while holding the project lock before changing anything.
func (repository *WriteRepository) ResolveForUpdate(ctx context.Context, input StateWriteInput) error {
	var id string
	err := repository.database.QueryRow(ctx, `SELECT s.id::text FROM states s
 JOIN projects p ON p.id=s.project_id JOIN workspaces w ON w.id=p.workspace_id
 WHERE s.id=$1 AND p.id=$2 AND w.slug=$3 AND s.deleted_at IS NULL AND s."group"<>'triage'`,
		input.StateID, input.ProjectID, input.Slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return missingStateFailure(input)
	}
	if err != nil {
		return fmt.Errorf("resolve project state for update: %w", err)
	}
	return nil
}

func (repository *WriteRepository) mutate(ctx context.Context, input StateWriteInput, action string) (_ json.RawMessage, resultErr error) {
	fields := cloneStateFields(input.Fields)
	if action == "create" {
		if err := validateStateFields(fields, true, input.App, input.Parsed); err != nil {
			return nil, err
		}
	}
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin project state write: %w", err)
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if lockStateProjectErr := lockStateProject(ctx, tx, input.Slug, input.ProjectID); lockStateProjectErr != nil {
		return nil, lockStateProjectErr
	}
	var raw json.RawMessage
	if action == "create" {
		raw, err = createState(ctx, tx, input, fields)
	} else {
		raw, err = updateState(ctx, tx, input, fields)
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit project state write: %w", err)
	}
	return raw, nil
}

func lockStateProject(ctx context.Context, tx pgx.Tx, slug, project string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT p.id::text FROM projects p JOIN workspaces w ON w.id=p.workspace_id
 WHERE p.id=$1 AND w.slug=$2 FOR UPDATE OF p`, project, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return &WriteFailure{Status: 404, Body: map[string]string{"detail": "No Project matches the given query."}}
	}
	if err != nil {
		return fmt.Errorf("lock project for state write: %w", err)
	}
	return nil
}

func createState(ctx context.Context, tx pgx.Tx, input StateWriteInput, fields map[string]any) (json.RawMessage, error) {
	name := stateString(fields, "name", "")
	color := stateString(fields, "color", "")
	group := stateString(fields, "group", "backlog")
	description := stateString(fields, "description", "")
	defaultValue := stateBool(fields, "default")
	isTriage := stateBool(fields, "is_triage")
	externalID := nullableStateString(fields, "external_id")
	externalSource := nullableStateString(fields, "external_source")
	if source, sourceOK := externalSource.(string); sourceOK && source != "" {
		if id, idOK := externalID.(string); idOK && id != "" {
			var existingID string
			err := tx.QueryRow(ctx, `SELECT s.id::text FROM states s JOIN workspaces w ON w.id=s.workspace_id
 WHERE s.project_id=$1 AND w.slug=$2 AND s.external_source=$3 AND s.external_id=$4 AND s.deleted_at IS NULL
 ORDER BY s.sequence LIMIT 1`, input.ProjectID, input.Slug, source, id).Scan(&existingID)
			if err == nil {
				return nil, &WriteFailure{Status: 409, Body: map[string]any{"error": "State with the same external id and external source already exists", "id": existingID}}
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
		}
	}
	if !input.App {
		var existingID string
		err := tx.QueryRow(ctx, `SELECT s.id::text FROM states s JOIN workspaces w ON w.id=s.workspace_id
 WHERE s.project_id=$1 AND w.slug=$2 AND s.name=$3 AND s.deleted_at IS NULL
 ORDER BY s.sequence LIMIT 1`, input.ProjectID, input.Slug, name).Scan(&existingID)
		if err == nil {
			return nil, &WriteFailure{Status: 409, Body: map[string]any{"error": "State with the same name already exists in the project", "id": existingID}}
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find existing state name: %w", err)
		}
	}
	if defaultValue {
		if _, err := tx.Exec(ctx, `UPDATE states SET "default"=false WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND deleted_at IS NULL AND "group"<>'triage'`, input.ProjectID, input.Slug); err != nil {
			return nil, err
		}
	}
	var data []byte
	err := tx.QueryRow(ctx, `INSERT INTO states
 (id,project_id,workspace_id,name,color,slug,"group","default",description,sequence,is_triage,external_source,external_id,created_at,updated_at,created_by_id)
 SELECT gen_random_uuid(),p.id,p.workspace_id,@name,@color,@slug,@group,@default,@description,
   COALESCE((SELECT MAX(s.sequence)+15000 FROM states s WHERE s.project_id=p.id AND s.workspace_id=p.workspace_id AND s.deleted_at IS NULL AND s."group"<>'triage'),65535),
   @is_triage,@external_source,@external_id,clock_timestamp(),clock_timestamp(),@user_id
 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=@project_id AND w.slug=@workspace_slug
 RETURNING `+stateJSONSQL(input.App), pgx.StrictNamedArgs{
		"project_id": input.ProjectID, "workspace_slug": input.Slug,
		"name": name, "color": color, "slug": stateSlug(name), "group": group,
		"default": defaultValue, "description": description, "is_triage": isTriage,
		"external_source": externalSource, "external_id": externalID, "user_id": input.UserID,
	}).Scan(&data)
	if err != nil {
		return nil, mapStateWriteDBError(err, input.App, "create")
	}
	return addAppOrderToResponse(data, input, fields)
}

func updateState(ctx context.Context, tx pgx.Tx, input StateWriteInput, fields map[string]any) (json.RawMessage, error) {
	if input.StateID == "" {
		return nil, missingStateFailure(input)
	}
	var currentDefault bool
	var currentExternalID, currentExternalSource *string
	err := tx.QueryRow(ctx, `SELECT s."default",s.external_id,s.external_source FROM states s JOIN workspaces w ON w.id=s.workspace_id
 WHERE s.id=$1 AND s.project_id=$2 AND w.slug=$3 AND s.deleted_at IS NULL AND s."group"<>'triage'
	FOR UPDATE OF s`, input.StateID, input.ProjectID, input.Slug).Scan(&currentDefault, &currentExternalID, &currentExternalSource)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, missingStateFailure(input)
	}
	if err != nil {
		return nil, err
	}
	if validateStateFieldsErr := validateStateFields(fields, false, input.App, input.Parsed); validateStateFieldsErr != nil {
		return nil, validateStateFieldsErr
	}
	if !input.App {
		if checkStateErr := checkStateExternalIdentity(ctx, tx, input, fields, currentExternalID, currentExternalSource); checkStateErr != nil {
			return nil, checkStateErr
		}
	}
	if hasStateField(fields, "default") && stateBool(fields, "default") {
		if _, execErr := tx.Exec(ctx, `UPDATE states SET "default"=false WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND deleted_at IS NULL AND "group"<>'triage'`, input.ProjectID, input.Slug); execErr != nil {
			return nil, execErr
		}
	}
	var data []byte
	err = tx.QueryRow(ctx, `UPDATE states s SET
 name=COALESCE($4,s.name), color=COALESCE($5,s.color), "group"=COALESCE($6,s."group"),
 "default"=COALESCE($7,s."default"), description=COALESCE($8,s.description),
 sequence=COALESCE($9,s.sequence), is_triage=COALESCE($10,s.is_triage),
 external_source=CASE WHEN $11 THEN $12 ELSE s.external_source END,
 external_id=CASE WHEN $13 THEN $14 ELSE s.external_id END,
 slug=CASE WHEN $4 IS NULL THEN s.slug ELSE $15 END,
	 updated_at=clock_timestamp(), updated_by_id=$3
	 WHERE s.id=$1 AND s.project_id=$2
	   AND EXISTS (SELECT 1 FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=s.project_id AND w.slug=$16)
	   AND s.deleted_at IS NULL AND s."group"<>'triage'
 RETURNING `+stateJSONSQL(input.App),
		input.StateID, input.ProjectID, input.UserID, stateField(fields, "name"), stateField(fields, "color"), stateField(fields, "group"), stateBoolPointer(fields, "default"), stateField(fields, "description"), stateField(fields, "sequence"), stateBoolPointer(fields, "is_triage"), hasStateField(fields, "external_source"), nullableStateString(fields, "external_source"), hasStateField(fields, "external_id"), nullableStateString(fields, "external_id"), stateSlug(stateString(fields, "name", "")), input.Slug).Scan(&data)
	if err != nil {
		return nil, mapStateWriteDBError(err, input.App, "update")
	}
	return addAppOrderToResponse(data, input, fields)
}

func (repository *WriteRepository) Delete(ctx context.Context, input StateWriteInput) (resultErr error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if err = lockStateProject(ctx, tx, input.Slug, input.ProjectID); err != nil {
		return err
	}
	if err = deleteState(ctx, tx, input); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (repository *WriteRepository) MarkDefault(ctx context.Context, input StateWriteInput) (resultErr error) {
	tx, err := repository.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if err = lockStateProject(ctx, tx, input.Slug, input.ProjectID); err != nil {
		return err
	}
	var found bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM states s JOIN workspaces w ON w.id=s.workspace_id
 WHERE s.id=$1 AND s.project_id=$2 AND w.slug=$3 AND s.deleted_at IS NULL AND s."group"<>'triage')`, input.StateID, input.ProjectID, input.Slug).Scan(&found)
	if err != nil {
		return err
	}
	if !found {
		return &WriteFailure{Status: 404, Body: map[string]string{"detail": "No State matches the given query."}}
	}
	if _, err = tx.Exec(ctx, `UPDATE states SET "default"=false WHERE project_id=$1 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$2) AND deleted_at IS NULL AND "group"<>'triage'`, input.ProjectID, input.Slug); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE states SET "default"=true WHERE id=$1 AND project_id=$2 AND workspace_id=(SELECT id FROM workspaces WHERE slug=$3) AND deleted_at IS NULL`, input.StateID, input.ProjectID, input.Slug); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func addAppOrderToResponse(data json.RawMessage, input StateWriteInput, fields map[string]any) (json.RawMessage, error) {
	order, present := fields["order"]
	if !input.App || !present {
		return data, nil
	}
	encodedOrder, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	var response map[string]json.RawMessage
	if unmarshalErr := json.Unmarshal(data, &response); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	response["order"] = encodedOrder
	encoded, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

func stateJSONSQL(app bool) string {
	if app {
		return `json_build_object('id',id,'project_id',project_id,'workspace_id',workspace_id,'name',name,'color',color,'group',"group",'default',"default",'description',description,'sequence',sequence)`
	}
	return `json_build_object('id',id,'created_at',created_at,'updated_at',updated_at,'deleted_at',deleted_at,'created_by',created_by_id,'updated_by',updated_by_id,'workspace',workspace_id,'project',project_id,'name',name,'description',description,'color',color,'slug',slug,'sequence',sequence,'group',"group",'is_triage',is_triage,'default',"default",'external_source',external_source,'external_id',external_id)`
}

func (repository *WriteRepository) SerializeAPIV1(ctx context.Context, data json.RawMessage, fields, expand string) (map[string]any, error) {
	var row map[string]any
	if err := json.Unmarshal(data, &row); err != nil {
		return nil, err
	}
	return djangoapi.Serialize(ctx, repository.database, row, apiStateFields, fields, expand, false)
}

func cloneStateFields(fields map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range fields {
		result[key] = value
	}
	return result
}
func hasStateField(fields map[string]any, key string) bool { _, ok := fields[key]; return ok }
func stateField(fields map[string]any, key string) any     { return fields[key] }
func stateString(fields map[string]any, key, fallback string) string {
	if v, ok := fields[key].(string); ok {
		return v
	}
	return fallback
}
func stateBool(fields map[string]any, key string) bool {
	value, _ := fields[key].(bool)
	return value
}
func stateBoolPointer(fields map[string]any, key string) any {
	if !hasStateField(fields, key) {
		return nil
	}
	return stateBool(fields, key)
}
func nullableStateString(fields map[string]any, key string) any {
	if value, ok := fields[key]; ok && value != nil {
		return value
	}
	return nil
}
func oneOf(value string, options ...string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

// stateSlug mirrors Django's default slugify path (NFKD, ASCII-ignore, then
// lowercase and retain word characters separated by collapsed whitespace or
// hyphens). Django's slugify library uses Python's Unicode database, so this
// small adapter keeps the source-specific ASCII behavior at the boundary.
func stateSlug(value string) string {
	value = norm.NFKD.String(value)
	var result strings.Builder
	dash := false
	for _, r := range value {
		if r > unicode.MaxASCII {
			continue
		}
		r = unicode.ToLower(r)
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			result.WriteRune(r)
			dash = false
		case unicode.IsSpace(r) || r == '-':
			if result.Len() > 0 && !dash {
				result.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(result.String(), "-")
}

func mapStateWriteDBError(err error, app bool, action string) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		if app {
			return &WriteFailure{Status: 400, Body: map[string]string{"name": "The state name is already taken"}}
		}
		return &WriteFailure{Status: 409, Body: map[string]any{"error": "State with the same name already exists in the project"}}
	}
	return fmt.Errorf("%s project state: %w", action, err)
}
