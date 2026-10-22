package backgroundtasks

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	htmlcompat "github.com/CaliLuke/go-plane/internal/htmlcompat"
	"github.com/CaliLuke/loom/execution"
	executionSelvedge "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DescriptionAssetCopyWorkflowName = "plane.descriptions.copy_assets"
	descriptionAssetPrepareActivity  = "plane.descriptions.load_assets"
	descriptionAssetCopyActivity     = "plane.descriptions.copy_asset"
	descriptionAssetRewriteActivity  = "plane.descriptions.rewrite_assets"
	descriptionAssetConvertActivity  = "plane.descriptions.convert_document"
	descriptionAssetQueue            = "plane-background"
	descriptionAssetActivityTimeout  = 5 * time.Minute
	descriptionAssetBatchTimeout     = 5 * time.Minute
)

type DescriptionAssetCopyInput struct {
	EventID       string `json:"event_id"`
	EntityName    string `json:"entity_name"`
	EntityID      string `json:"entity_id"`
	ProjectID     string `json:"project_id"`
	WorkspaceSlug string `json:"workspace_slug"`
	UserID        string `json:"user_id"`
}

type descriptionAssetRecord struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Attributes  json.RawMessage `json:"attributes"`
	Size        float64         `json:"size"`
	Metadata    json.RawMessage `json:"metadata"`
	WorkspaceID string          `json:"workspace_id"`
	EntityType  string          `json:"entity_type"`
}

type descriptionAssetCopyPreparation struct {
	Exists bool                     `json:"exists"`
	HTML   string                   `json:"html"`
	Assets []descriptionAssetRecord `json:"assets"`
}

type descriptionAssetCopyOutput struct {
	OriginalID string `json:"original_id"`
	CopyID     string `json:"copy_id"`
}

type descriptionAssetCopyStore interface {
	Prepare(context.Context, DescriptionAssetCopyInput) (descriptionAssetCopyPreparation, error)
	Copy(context.Context, DescriptionAssetCopyInput, descriptionAssetRecord) (descriptionAssetCopyOutput, error)
	Rewrite(context.Context, DescriptionAssetCopyInput, string, map[string]string) (string, error)
	Convert(context.Context, DescriptionAssetCopyInput, string) error
}

type DescriptionAssetCopyDispatcher struct {
	app      *selvedge.App
	workflow *selvedge.WorkflowFunc[DescriptionAssetCopyInput, struct{}]
}

type PostgresDescriptionAssetCopyStore struct {
	pool    *pgxpool.Pool
	storage interface {
		Copy(context.Context, string, string) error
	}
	client  *http.Client
	liveURL string
}

func NewPostgresDescriptionAssetCopyStore(pool *pgxpool.Pool, storage *assets.Storage) (*PostgresDescriptionAssetCopyStore, error) {
	if pool == nil || storage == nil {
		return nil, errors.New("PostgreSQL and S3 storage are required for description asset copies")
	}
	return &PostgresDescriptionAssetCopyStore{pool: pool, storage: storage, client: &http.Client{Timeout: 30 * time.Second}, liveURL: strings.TrimSpace(os.Getenv("LIVE_URL"))}, nil
}

func RegisterDescriptionAssetCopyWorkflow(app *selvedge.App, store descriptionAssetCopyStore, policy *retry.Policy) (*DescriptionAssetCopyDispatcher, error) {
	if app == nil || store == nil {
		return nil, errors.New("Selvedge app and description asset copy store are required")
	}
	if policy == nil {
		policy = retry.Exponential(8, time.Second, time.Minute, 2)
	}
	policyCopy := *policy
	activities, err := executionSelvedge.New(app)
	if err != nil {
		return nil, err
	}
	prepare := execution.Activity[DescriptionAssetCopyInput, descriptionAssetCopyPreparation]{Name: descriptionAssetPrepareActivity}
	if err := executionSelvedge.Register(activities, prepare, store.Prepare, selvedge.ActivityOptions{Queue: descriptionAssetQueue, StartToCloseTimeout: descriptionAssetActivityTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register description asset preparation activity: %w", err)
	}
	copyAsset := execution.Activity[descriptionAssetCopyInput, descriptionAssetCopyOutput]{Name: descriptionAssetCopyActivity}
	if err := executionSelvedge.Register(activities, copyAsset, func(ctx context.Context, input descriptionAssetCopyInput) (descriptionAssetCopyOutput, error) {
		return store.Copy(ctx, input.Input, input.Asset)
	}, selvedge.ActivityOptions{Queue: descriptionAssetQueue, StartToCloseTimeout: descriptionAssetActivityTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register description asset copy activity: %w", err)
	}
	rewrite := execution.Activity[descriptionAssetRewriteInput, string]{Name: descriptionAssetRewriteActivity}
	if err := executionSelvedge.Register(activities, rewrite, func(ctx context.Context, input descriptionAssetRewriteInput) (string, error) {
		return store.Rewrite(ctx, input.Input, input.HTML, input.Mapping)
	}, selvedge.ActivityOptions{Queue: descriptionAssetQueue, StartToCloseTimeout: descriptionAssetActivityTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register description asset rewrite activity: %w", err)
	}
	convert := execution.Activity[descriptionAssetRewriteInput, struct{}]{Name: descriptionAssetConvertActivity}
	if err := executionSelvedge.Register(activities, convert, func(ctx context.Context, input descriptionAssetRewriteInput) (struct{}, error) {
		return struct{}{}, store.Convert(ctx, input.Input, input.HTML)
	}, selvedge.ActivityOptions{Queue: descriptionAssetQueue, StartToCloseTimeout: descriptionAssetActivityTimeout, RetryPolicy: &policyCopy}); err != nil {
		return nil, fmt.Errorf("register description conversion activity: %w", err)
	}
	workflow := selvedge.DefineWorkflow(DescriptionAssetCopyWorkflowName, func(ctx *selvedge.WorkflowContext, input DescriptionAssetCopyInput) (struct{}, error) {
		if err := validateDescriptionAssetCopyInput(input); err != nil {
			return struct{}{}, err
		}
		preparation, err := prepare.Execute(ctx.Context(), activities.Scope(ctx), input, execution.ActivityOptions{})
		if err != nil {
			return struct{}{}, err
		}
		if !preparation.Exists {
			return struct{}{}, nil
		}
		mapping := make(map[string]string, len(preparation.Assets))
		for _, asset := range preparation.Assets {
			copied, err := copyAsset.Execute(ctx.Context(), activities.Scope(ctx), descriptionAssetCopyInput{Input: input, Asset: asset}, execution.ActivityOptions{})
			if err != nil {
				return struct{}{}, err
			}
			mapping[copied.OriginalID] = copied.CopyID
		}
		rewriteInput := descriptionAssetRewriteInput{Input: input, HTML: preparation.HTML, Mapping: mapping}
		updatedHTML, err := rewrite.Execute(ctx.Context(), activities.Scope(ctx), rewriteInput, execution.ActivityOptions{})
		if err != nil {
			return struct{}{}, err
		}
		rewriteInput.HTML = updatedHTML
		return convert.Execute(ctx.Context(), activities.Scope(ctx), rewriteInput, execution.ActivityOptions{})
	})
	selvedge.RegisterWorkflow(app, workflow)
	return &DescriptionAssetCopyDispatcher{app: app, workflow: workflow}, nil
}

type descriptionAssetCopyInput struct {
	Input DescriptionAssetCopyInput `json:"input"`
	Asset descriptionAssetRecord    `json:"asset"`
}

type descriptionAssetRewriteInput struct {
	Input   DescriptionAssetCopyInput `json:"input"`
	HTML    string                    `json:"html"`
	Mapping map[string]string         `json:"mapping"`
}

func (dispatcher *DescriptionAssetCopyDispatcher) Admit(ctx context.Context, input DescriptionAssetCopyInput) (string, error) {
	if dispatcher == nil || dispatcher.app == nil || dispatcher.workflow == nil {
		return "", errors.New("description asset copy dispatcher is not configured")
	}
	if err := validateDescriptionAssetCopyInput(input); err != nil {
		return "", err
	}
	return admitOrReconcile(ctx, dispatcher.app, dispatcher.workflow, "plane-description-assets:"+input.EventID, input)
}

func validateDescriptionAssetCopyInput(input DescriptionAssetCopyInput) error {
	if _, err := uuid.Parse(input.EventID); err != nil {
		return fmt.Errorf("description asset copy event ID must be a UUID: %w", err)
	}
	if _, err := uuid.Parse(input.EntityID); err != nil {
		return fmt.Errorf("description asset copy entity ID must be a UUID: %w", err)
	}
	if _, err := uuid.Parse(input.ProjectID); err != nil {
		return fmt.Errorf("description asset copy project ID must be a UUID: %w", err)
	}
	if _, err := uuid.Parse(input.UserID); err != nil {
		return fmt.Errorf("description asset copy user ID must be a UUID: %w", err)
	}
	if input.EntityName != "PAGE" && input.EntityName != "ISSUE" {
		return fmt.Errorf("unsupported description asset copy entity %q", input.EntityName)
	}
	return nil
}

func (store *PostgresDescriptionAssetCopyStore) Prepare(ctx context.Context, input DescriptionAssetCopyInput) (descriptionAssetCopyPreparation, error) {
	if err := validateDescriptionAssetCopyInput(input); err != nil {
		return descriptionAssetCopyPreparation{}, err
	}
	var workspaceID uuid.UUID
	var markup string
	table := descriptionEntityTable(input.EntityName)
	query := `SELECT workspace_id,description_html FROM ` + table + ` WHERE id=$1 AND deleted_at IS NULL`
	err := store.pool.QueryRow(ctx, query, input.EntityID).Scan(&workspaceID, &markup)
	if errors.Is(err, pgx.ErrNoRows) {
		return descriptionAssetCopyPreparation{Exists: false}, nil
	}
	if err != nil {
		return descriptionAssetCopyPreparation{}, fmt.Errorf("load %s for asset copy: %w", input.EntityName, err)
	}
	assetIDs, err := descriptionImageIDs(markup)
	if err != nil {
		return descriptionAssetCopyPreparation{}, err
	}
	preparation := descriptionAssetCopyPreparation{Exists: true, HTML: markup, Assets: make([]descriptionAssetRecord, 0, len(assetIDs))}
	if len(assetIDs) == 0 {
		return preparation, nil
	}
	rows, err := store.pool.Query(ctx, `SELECT id::text,asset,attributes,size,storage_metadata,workspace_id::text,COALESCE(entity_type,'') FROM file_assets WHERE workspace_id=$1 AND project_id=$2 AND id=ANY($3::uuid[]) AND deleted_at IS NULL ORDER BY created_at DESC`, workspaceID, input.ProjectID, assetIDs)
	if err != nil {
		return descriptionAssetCopyPreparation{}, fmt.Errorf("load source description assets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var asset descriptionAssetRecord
		if err := rows.Scan(&asset.ID, &asset.Key, &asset.Attributes, &asset.Size, &asset.Metadata, &asset.WorkspaceID, &asset.EntityType); err != nil {
			return descriptionAssetCopyPreparation{}, fmt.Errorf("read source description asset: %w", err)
		}
		preparation.Assets = append(preparation.Assets, asset)
	}
	if err := rows.Err(); err != nil {
		return descriptionAssetCopyPreparation{}, fmt.Errorf("iterate source description assets: %w", err)
	}
	return preparation, nil
}

func (store *PostgresDescriptionAssetCopyStore) Copy(ctx context.Context, input DescriptionAssetCopyInput, source descriptionAssetRecord) (descriptionAssetCopyOutput, error) {
	eventID, _ := uuid.Parse(input.EventID)
	sourceID, err := uuid.Parse(source.ID)
	if err != nil {
		return descriptionAssetCopyOutput{}, fmt.Errorf("parse source asset ID: %w", err)
	}
	copyID := uuid.NewSHA1(eventID, sourceID[:])
	workspaceID, err := uuid.Parse(source.WorkspaceID)
	if err != nil {
		return descriptionAssetCopyOutput{}, fmt.Errorf("parse source asset workspace ID: %w", err)
	}
	projectID, _ := uuid.Parse(input.ProjectID)
	userID, _ := uuid.Parse(input.UserID)
	filename, err := descriptionAssetFilename(source.Attributes)
	if err != nil {
		return descriptionAssetCopyOutput{}, err
	}
	copyKey := workspaceID.String() + "/" + strings.ReplaceAll(copyID.String(), "-", "") + "-" + filename
	entityColumn, err := descriptionAssetEntityColumn(source.EntityType)
	if err != nil {
		return descriptionAssetCopyOutput{}, err
	}
	var pageID, issueID, commentID, draftIssueID, assetUserID any
	switch entityColumn {
	case "page_id":
		pageID = input.EntityID
	case "issue_id":
		issueID = input.EntityID
	case "comment_id":
		commentID = input.EntityID
	case "draft_issue_id":
		draftIssueID = input.EntityID
	case "user_id":
		assetUserID = input.EntityID
	}
	if entityColumn == "workspace_id" {
		workspaceID, _ = uuid.Parse(input.EntityID)
	}
	if entityColumn == "project_id" {
		projectID, _ = uuid.Parse(input.EntityID)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO file_assets (created_at,updated_at,id,attributes,asset,created_by_id,workspace_id,is_deleted,deleted_at,is_archived,comment_id,entity_type,external_id,external_source,is_uploaded,issue_id,page_id,project_id,size,storage_metadata,user_id,draft_issue_id,entity_identifier) VALUES (clock_timestamp(),clock_timestamp(),$1,$2::jsonb,$3,$4,$5,false,NULL,false,$6,NULLIF($7,''),NULL,NULL,false,$8,$9,$10,$11,$12::jsonb,$13,$14,NULL) ON CONFLICT (id) DO UPDATE SET updated_at=EXCLUDED.updated_at,attributes=EXCLUDED.attributes,asset=EXCLUDED.asset,created_by_id=EXCLUDED.created_by_id,workspace_id=EXCLUDED.workspace_id,is_deleted=false,deleted_at=NULL,is_archived=false,comment_id=EXCLUDED.comment_id,entity_type=EXCLUDED.entity_type,is_uploaded=false,issue_id=EXCLUDED.issue_id,page_id=EXCLUDED.page_id,project_id=EXCLUDED.project_id,size=EXCLUDED.size,storage_metadata=EXCLUDED.storage_metadata,user_id=EXCLUDED.user_id,draft_issue_id=EXCLUDED.draft_issue_id,entity_identifier=NULL`, copyID, source.Attributes, copyKey, userID, workspaceID, commentID, source.EntityType, issueID, pageID, projectID, source.Size, nonemptyJSON(source.Metadata), assetUserID, draftIssueID); err != nil {
		return descriptionAssetCopyOutput{}, fmt.Errorf("create copied file asset metadata: %w", err)
	}
	if err := store.storage.Copy(ctx, source.Key, copyKey); err != nil {
		return descriptionAssetCopyOutput{}, fmt.Errorf("copy S3 description asset: %w", err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE file_assets SET is_uploaded=true,updated_at=clock_timestamp() WHERE id=$1`, copyID); err != nil {
		return descriptionAssetCopyOutput{}, fmt.Errorf("mark copied FileAsset uploaded: %w", err)
	}
	return descriptionAssetCopyOutput{OriginalID: source.ID, CopyID: copyID.String()}, nil
}

func (store *PostgresDescriptionAssetCopyStore) Rewrite(ctx context.Context, input DescriptionAssetCopyInput, markup string, mapping map[string]string) (string, error) {
	updated, err := replaceDescriptionImageIDs(markup, mapping)
	if err != nil {
		return "", err
	}
	if _, err := store.pool.Exec(ctx, `UPDATE `+descriptionEntityTable(input.EntityName)+` SET description_html=$2,updated_at=clock_timestamp() WHERE id=$1 AND deleted_at IS NULL`, input.EntityID, updated); err != nil {
		return "", fmt.Errorf("update %s description HTML after asset copy: %w", input.EntityName, err)
	}
	return updated, nil
}

func (store *PostgresDescriptionAssetCopyStore) Convert(ctx context.Context, input DescriptionAssetCopyInput, markup string) error {
	if store.liveURL == "" {
		return nil
	}
	parsed, err := url.Parse(store.liveURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("LIVE_URL must be an absolute URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/convert-document/"
	variant := "document"
	if input.EntityName == "PAGE" {
		variant = "rich"
	}
	body, err := json.Marshal(map[string]string{"description_html": markup, "variant": variant})
	if err != nil {
		return fmt.Errorf("encode description conversion request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create description conversion request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := store.client.Do(request)
	if err != nil {
		return fmt.Errorf("request description conversion: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}
	var converted struct {
		DescriptionJSON   json.RawMessage `json:"description_json"`
		DescriptionBinary string          `json:"description_binary"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&converted); err != nil {
		return fmt.Errorf("decode description conversion response: %w", err)
	}
	binary, err := base64.StdEncoding.DecodeString(converted.DescriptionBinary)
	if err != nil {
		return fmt.Errorf("decode converted description binary: %w", err)
	}
	if !json.Valid(converted.DescriptionJSON) {
		return errors.New("description conversion returned invalid JSON")
	}
	if _, err := store.pool.Exec(ctx, `UPDATE `+descriptionEntityTable(input.EntityName)+` SET description_json=$2::jsonb,description_binary=$3,updated_at=clock_timestamp() WHERE id=$1 AND deleted_at IS NULL`, input.EntityID, converted.DescriptionJSON, binary); err != nil {
		return fmt.Errorf("update converted %s description: %w", input.EntityName, err)
	}
	return nil
}

func descriptionEntityTable(entity string) string {
	if entity == "PAGE" {
		return "pages"
	}
	return "issues"
}

func descriptionImageIDs(markup string) ([]uuid.UUID, error) {
	nodes, err := htmlcompat.ParseFragment(strings.NewReader(markup), nil)
	if err != nil {
		return nil, fmt.Errorf("parse description HTML for assets: %w", err)
	}
	seen := make(map[uuid.UUID]struct{})
	var visit func(*htmlcompat.Node)
	visit = func(node *htmlcompat.Node) {
		if node.Type == htmlcompat.ElementNode && strings.EqualFold(node.Data, "image-component") {
			for _, attribute := range node.Attr {
				if strings.EqualFold(attribute.Key, "src") {
					if id, err := uuid.Parse(attribute.Val); err == nil {
						seen[id] = struct{}{}
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range nodes {
		visit(node)
	}
	result := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result, nil
}

func replaceDescriptionImageIDs(markup string, mapping map[string]string) (string, error) {
	nodes, err := htmlcompat.ParseFragment(strings.NewReader(markup), nil)
	if err != nil {
		return "", fmt.Errorf("parse description HTML for asset replacement: %w", err)
	}
	var visit func(*htmlcompat.Node)
	visit = func(node *htmlcompat.Node) {
		if node.Type == htmlcompat.ElementNode && strings.EqualFold(node.Data, "image-component") {
			for index := range node.Attr {
				if strings.EqualFold(node.Attr[index].Key, "src") {
					if replacement, ok := mapping[node.Attr[index].Val]; ok {
						node.Attr[index].Val = replacement
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	var output strings.Builder
	for _, node := range nodes {
		visit(node)
		if err := htmlcompat.Render(&output, node); err != nil {
			return "", fmt.Errorf("render rewritten description HTML: %w", err)
		}
	}
	return output.String(), nil
}

func descriptionAssetFilename(raw json.RawMessage) (string, error) {
	var attributes map[string]any
	if err := json.Unmarshal(raw, &attributes); err != nil {
		return "", fmt.Errorf("decode copied asset attributes: %w", err)
	}
	name, ok := attributes["name"]
	if !ok || name == nil {
		return "None", nil
	}
	return fmt.Sprint(name), nil
}

func descriptionAssetEntityColumn(entityType string) (string, error) {
	switch entityType {
	case "WORKSPACE_LOGO":
		return "workspace_id", nil
	case "PROJECT_COVER":
		return "project_id", nil
	case "USER_AVATAR", "USER_COVER":
		return "user_id", nil
	case "ISSUE_ATTACHMENT", "ISSUE_DESCRIPTION":
		return "issue_id", nil
	case "COMMENT_DESCRIPTION":
		return "comment_id", nil
	case "PAGE_DESCRIPTION":
		return "page_id", nil
	case "DRAFT_ISSUE_DESCRIPTION":
		return "draft_issue_id", nil
	case "DRAFT_ISSUE_ATTACHMENT":
		return "", nil
	case "":
		return "", nil
	default:
		return "", nil
	}
}

func nonemptyJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	if !json.Valid(raw) {
		return json.RawMessage(`{}`)
	}
	return raw
}
