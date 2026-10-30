package assets

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"time"

	dbutil "github.com/CaliLuke/go-plane/internal/database"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/mutationreceipts"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (a *AttachmentApplication) MutateResource(ctx context.Context, cmd projectmutations.Command) (projectmutations.Result, error) {
	if cmd.ParentID == uuid.Nil || (cmd.Action != "create" && cmd.Action != "delete") {
		return projectmutations.Result{}, projectmutations.ErrInvalid
	}
	resource := projectmutations.Resource{Name: "workitem_attachment", Table: "file_assets", RevisionTable: "file_asset_edit_revisions", RevisionColumn: "asset_id", ParentColumn: "issue_id", SchemaLock: true}
	resource.Authorize = func(ctx context.Context, tx pgx.Tx) error {
		var id uuid.UUID
		err := tx.QueryRow(ctx, `SELECT i.id FROM issues i WHERE i.id=$1 AND i.project_id=$2 AND `+djangoapi.IssueVisibleSQL("i", "$3::uuid", "$4::uuid")+` FOR SHARE`, cmd.ParentID, cmd.ProjectID, cmd.ActorID, cmd.WorkspaceID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return projectmutations.ErrNotFound
		}
		return err
	}
	result, err := projectmutations.Execute(ctx, a.receipts, resource, cmd, func(ctx context.Context, tx pgx.Tx) (json.RawMessage, error) {
		repository := &PostgresRepository{database: tx}
		if cmd.Action == "delete" {
			asset, _, err := repository.IssueAttachment(ctx, cmd.ActorID, cmd.WorkspaceID, cmd.ProjectID, cmd.ParentID, cmd.ResourceID)
			if err != nil {
				return nil, err
			}
			if asset.EntityType != "ISSUE_ATTACHMENT" {
				return nil, projectmutations.ErrNotFound
			}
			err = repository.DeleteIssueAttachment(ctx, cmd.ActorID, cmd.WorkspaceID, cmd.ProjectID, cmd.ParentID, cmd.ResourceID, attachmentActivity(cmd, a.origin, "deleted"))
			return json.RawMessage("null"), err
		}
		return a.stageAttachment(ctx, repository, tx, cmd)
	})
	if err != nil || cmd.Action == "delete" {
		return result, err
	}
	// The committed intent contains the bytes. A crash or storage outage can be
	// retried by this request or the durable effect worker without re-fetching URL.
	id, err := uuid.Parse(result.ID)
	if err != nil {
		return result, err
	}
	if err = a.CompleteAttachmentUpload(ctx, id.String()); err != nil {
		return result, mutationreceipts.ErrIndeterminate
	}
	// A concurrent deletion can cancel a pending upload. Do not describe that
	// cancellation as a completed object upload to the importing caller.
	var uploaded bool
	err = a.repository.database.QueryRow(ctx, `SELECT is_uploaded FROM file_assets WHERE id=$1 AND workspace_id=$2 AND project_id=$3 AND issue_id=$4`, id, cmd.WorkspaceID, cmd.ProjectID, cmd.ParentID).Scan(&uploaded)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && !uploaded {
		return result, projectmutations.ErrNotFound
	}
	if err != nil {
		return result, mutationreceipts.ErrIndeterminate
	}
	return result, nil
}
func (a *AttachmentApplication) stageAttachment(ctx context.Context, repository *PostgresRepository, tx pgx.Tx, cmd projectmutations.Command) (json.RawMessage, error) {
	rawURL, ok := cmd.Fields["url"].(string)
	if !ok || rawURL == "" || a.fetch == nil {
		return nil, projectmutations.ErrInvalid
	}
	name, _ := cmd.Fields["name"].(string)
	body, contentType, err := a.fetch(ctx, rawURL, a.limit)
	if err != nil {
		return nil, &djangoapi.HTTPError{Status: 400, Body: map[string]any{"error": "The public file could not be imported. Check the URL and 5 MB size limit."}}
	}
	if len(body) == 0 || int64(len(body)) > a.limit {
		return nil, projectmutations.ErrInvalid
	}
	if name == "" {
		parsed, _ := url.Parse(rawURL)
		name = path.Base(parsed.Path)
	}
	name = sanitizeFilename(name)
	if name == "" || name == "." || name == "/" {
		name = "attachment"
	}
	contentType = attachmentMIME(contentType, name)
	if !allowedAttachmentMIME(contentType) {
		return nil, projectmutations.ErrInvalid
	}
	if cmd.Revalidate != nil {
		if err = cmd.Revalidate(ctx); err != nil {
			return nil, err
		}
	}
	id := uuid.New()
	key := cmd.WorkspaceID.String() + "/" + compactUUID(id.String()) + "-" + name
	attrs := uploadRequest{Name: name, Type: contentType}.Attributes(name, contentType, int64(len(body)))
	if err = repository.CreateIssueAttachment(ctx, id, cmd.ActorID, cmd.WorkspaceID, cmd.ProjectID, cmd.ParentID, key, attrs, int64(len(body)), "", ""); err != nil {
		return nil, err
	}
	activity, err := json.Marshal(attachmentActivity(cmd, a.origin, "created"))
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO file_asset_upload_intents(asset_id,object_key,content_type,contents,activity) VALUES($1,$2,$3,$4,$5::jsonb)`, id, key, contentType, body, activity); err != nil {
		return nil, err
	}
	if err = insertWorkItemEffect(ctx, tx, "asset_storage_upload", map[string]string{"asset_id": id.String()}); err != nil {
		return nil, err
	}
	_, row, err := repository.IssueAttachment(ctx, cmd.ActorID, cmd.WorkspaceID, cmd.ProjectID, cmd.ParentID, id)
	if err != nil {
		return nil, err
	}
	row["is_uploaded"] = true // Only released after CompleteAttachmentUpload succeeds.
	normalizeAttachment(row)
	return json.Marshal(row)
}
func attachmentActivity(cmd projectmutations.Command, origin, verb string) map[string]any {
	return map[string]any{"type": "attachment.activity." + verb, "requested_data": nil, "actor_id": cmd.ActorID.String(), "issue_id": cmd.ParentID.String(), "project_id": cmd.ProjectID.String(), "epoch": time.Now().Unix(), "notification": true, "origin": origin}
}

// CompleteAttachmentUpload is idempotent against an immutable staged object.
// Locks serialize object creation with deletion; bytes survive ambiguous commits.
func (a *AttachmentApplication) CompleteAttachmentUpload(ctx context.Context, rawID string) (resultErr error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return err
	}
	tx, err := a.repository.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	var key, contentType string
	var contents, activity []byte
	var deleted bool
	err = tx.QueryRow(ctx, `SELECT intent.object_key,intent.content_type,intent.contents,intent.activity,f.deleted_at IS NOT NULL OR f.is_deleted
 FROM file_asset_upload_intents intent JOIN file_assets f ON f.id=intent.asset_id AND f.asset=intent.object_key
 WHERE intent.asset_id=$1 AND intent.completed_at IS NULL FOR UPDATE OF f,intent`, id).Scan(&key, &contentType, &contents, &activity, &deleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if deleted {
		if err = insertWorkItemEffect(ctx, tx, "asset_storage_delete", map[string]string{"asset_id": id.String(), "object_key": key}); err != nil {
			return err
		}
	} else {
		if err = a.storage.Put(ctx, key, contentType, contents); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE file_assets SET is_uploaded=true WHERE id=$1 AND asset=$2`, id, key); err != nil {
			return err
		}
		if err = insertWorkItemEffect(ctx, tx, "activity", json.RawMessage(activity)); err != nil {
			return err
		}
		if err = insertWorkItemEffect(ctx, tx, "asset_storage_metadata", map[string]string{"asset_id": id.String()}); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE file_asset_upload_intents SET completed_at=clock_timestamp(),contents=''::bytea,activity='{}'::jsonb WHERE asset_id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
