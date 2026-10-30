package assets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/url"
	"path"
	"strings"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const AttachmentUploadLimit int64 = 5 << 20
const AttachmentTextLimit int64 = 1 << 20

type PublicFileFetcher func(context.Context, string, int64) ([]byte, string, error)
type AttachmentApplication struct {
	repository *PostgresRepository
	receipts   mutationreceipts.Database
	storage    *Storage
	fetch      PublicFileFetcher
	origin     string
	limit      int64
}
type AttachmentQuery struct {
	ActorID, WorkspaceID, ProjectID, IssueID, AttachmentID uuid.UUID
	WorkspaceSlug, Action                                  string
	IncludeRevision                                        bool
}
type AttachmentContent struct {
	Kind     string  `json:"kind"`
	MIMEType string  `json:"mime_type"`
	Text     *string `json:"text,omitempty"`
	Data     []byte  `json:"data,omitempty"`
}

func NewAttachmentApplication(repository *PostgresRepository, storage *Storage, fetch PublicFileFetcher, origin string, limit int64) (*AttachmentApplication, error) {
	if repository == nil || storage == nil {
		return nil, errors.New("attachment database and storage are required")
	}
	if limit <= 0 {
		limit = AttachmentUploadLimit
	}
	database, ok := repository.database.(mutationreceipts.Database)
	if !ok {
		return nil, errors.New("attachment receipt database is required")
	}
	return &AttachmentApplication{repository: repository, receipts: database, storage: storage, fetch: fetch, origin: origin, limit: min(limit, AttachmentUploadLimit)}, nil
}

func (a *AttachmentApplication) Read(ctx context.Context, q AttachmentQuery) (_ any, resultErr error) {
	tx, err := a.repository.database.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	if _, err = tx.Exec(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
		return nil, err
	}
	if err = authorizeAttachmentRead(ctx, tx, q); err != nil {
		return nil, err
	}
	reader := &PostgresRepository{database: tx}
	if q.Action == "list" {
		rows, visibleIssueAttachmentsErr := reader.VisibleIssueAttachments(ctx, q.ActorID, q.WorkspaceID, q.ProjectID, q.IssueID, true)
		if visibleIssueAttachmentsErr != nil {
			return nil, visibleIssueAttachmentsErr
		}
		for _, row := range rows {
			normalizeAttachment(row)
			if q.IncludeRevision {
				if visibleIssueAttachmentsErr = attachmentRevision(ctx, tx, row); visibleIssueAttachmentsErr != nil {
					return nil, visibleIssueAttachmentsErr
				}
			}
		}
		return rows, tx.Commit(ctx)
	}
	asset, row, err := reader.IssueAttachment(ctx, q.ActorID, q.WorkspaceID, q.ProjectID, q.IssueID, q.AttachmentID)
	if err != nil {
		return nil, err
	}
	if !asset.IsUploaded || asset.EntityType != "ISSUE_ATTACHMENT" {
		return nil, ErrNotFound
	}
	normalizeAttachment(row)
	if q.IncludeRevision {
		if err = attachmentRevision(ctx, tx, row); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	name, _ := row["name"].(string)
	if name == "" {
		name = q.AttachmentID.String()
	}
	if q.Action == "download_url" {
		origin, _ := url.Parse(a.origin)
		location, presignGetErr := a.storage.PresignGet(ctx, asset.Asset, "attachment", &name, origin.Scheme, origin.Host)
		out := map[string]any{"download_url": location, "attachment_id": q.AttachmentID.String(), "name": name}
		if revision, ok := row["revision"]; ok {
			out["revision"] = revision
		}
		return out, presignGetErr
	}
	if q.Action != "read" {
		return nil, projectmutations.ErrInvalid
	}
	contentType, _ := row["content_type"].(string)
	contentType = attachmentMIME(contentType, name)
	kind, limit := "", AttachmentTextLimit
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		kind, limit = "image", AttachmentUploadLimit
	case "text/plain", "text/markdown", "text/csv", "text/html", "text/xml", "text/yaml", "application/json", "application/xml", "application/yaml", "application/x-yaml":
		kind = "text"
	default:
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Unsupported inline content type. Use download_url for this attachment."}}
	}
	body, err := a.storage.readAttachment(ctx, asset.Asset, limit)
	if err != nil {
		return nil, err
	}
	result := AttachmentContent{Kind: kind, MIMEType: contentType}
	if kind == "image" {
		result.Data = body
	} else {
		text := strings.ToValidUTF8(string(body), "\uFFFD")
		result.Text = &text
	}
	return result, nil
}

func authorizeAttachmentRead(ctx context.Context, tx pgx.Tx, q AttachmentQuery) error {
	var allowed bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces w JOIN workspace_members wm ON wm.workspace_id=w.id JOIN users u ON u.id=wm.member_id
 WHERE w.id=$1 AND w.slug=$2 AND w.deleted_at IS NULL AND u.id=$3 AND u.is_active AND wm.is_active AND wm.deleted_at IS NULL AND wm.role IN(5,15,20)
 AND EXISTS(SELECT 1 FROM issues i JOIN projects p ON p.id=i.project_id AND p.workspace_id=i.workspace_id WHERE i.id=$4 AND i.project_id=$5 AND p.archived_at IS NULL AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "w.id")+`))`, q.WorkspaceID, q.WorkspaceSlug, q.ActorID, q.IssueID, q.ProjectID).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return projectmutations.ErrForbidden
	}
	return nil
}
func normalizeAttachment(row map[string]any) {
	attrs, _ := row["attributes"].(map[string]any)
	row["name"], row["content_type"] = attrs["name"], attrs["type"]
	size := attrs["size"]
	var number float64
	switch value := size.(type) {
	case json.Number:
		var err error
		number, err = value.Float64()
		if err != nil {
			return
		}
	case float64:
		number = value
	default:
		return
	}
	if number != 0 && !math.IsNaN(number) && !math.IsInf(number, 0) {
		row["size"] = size
	}
}
func attachmentMIME(raw, name string) string {
	contentType, _, err := mime.ParseMediaType(raw)
	if err != nil || contentType == "" || contentType == "application/octet-stream" {
		contentType, _, _ = mime.ParseMediaType(mime.TypeByExtension(strings.ToLower(path.Ext(name))))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}
func (s *Storage) readAttachment(ctx context.Context, key string, limit int64) ([]byte, error) {
	response, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, fmt.Errorf("read attachment storage: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.ContentLength != nil && *response.ContentLength > limit {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Attachment exceeds inline size limit. Use download_url."}}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err == nil && int64(len(body)) > limit {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "Attachment exceeds inline size limit. Use download_url."}}
	}
	return body, err
}

func attachmentRevision(ctx context.Context, tx pgx.Tx, row map[string]any) error {
	var revision int64
	if err := tx.QueryRow(ctx, `SELECT revision FROM file_asset_edit_revisions WHERE asset_id=$1`, row["id"]).Scan(&revision); err != nil {
		return err
	}
	row["revision"] = fmt.Sprintf("workitem_attachment:%s:%d", row["id"], revision)
	return nil
}
