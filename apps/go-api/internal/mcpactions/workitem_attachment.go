package mcpactions

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/CaliLuke/go-plane/internal/assets"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/projectmutations"
	api "github.com/CaliLuke/go-plane/mcp/gen/vector_mcp"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/google/uuid"
)

type attachmentApplication interface {
	Read(context.Context, assets.AttachmentQuery) (any, error)
	MutateResource(context.Context, projectmutations.Command) (projectmutations.Result, error)
}

func (s *Service) WorkitemAttachment(ctx context.Context, input *api.WorkitemAttachmentRequest) (loom.Nullable[api.WorkitemAttachmentResult], error) {
	if input == nil {
		return loom.Nullable[api.WorkitemAttachmentResult]{}, api.MakeInvalidParams(errors.New("invalid attachment action"))
	}
	if value, ok := input.AsAttachmentList(); ok && value != nil && value.Action == "list" {
		return s.attachmentRead(ctx, assets.AttachmentQuery{Action: "list", IncludeRevision: value.IncludeRevision != nil && *value.IncludeRevision}, value.WorkspaceSlug, value.ProjectID, value.WorkitemID, "")
	}
	if value, ok := input.AsAttachmentRead(); ok && value != nil && (value.Action == "read" || value.Action == "download_url") {
		return s.attachmentRead(ctx, assets.AttachmentQuery{Action: value.Action, IncludeRevision: value.IncludeRevision != nil && *value.IncludeRevision}, value.WorkspaceSlug, value.ProjectID, value.WorkitemID, value.AttachmentID)
	}
	selected := resourceMutationInput{}
	if value, ok := input.AsAttachmentUpload(); ok && value != nil && value.Action == "upload_from_url" {
		selected = resourceMutationInput{action: "create", project: value.ProjectID, parent: value.WorkitemID, workspace: value.WorkspaceSlug, request: value.RequestID, fields: map[string]any{"url": value.URL, "name": ""}}
		if value.Name != nil {
			selected.fields["name"] = *value.Name
		}
	} else if value, ok := input.AsAttachmentDelete(); ok && value != nil && value.Action == "delete" {
		selected = resourceMutationInput{action: "delete", project: value.ProjectID, parent: value.WorkitemID, target: value.AttachmentID, workspace: value.WorkspaceSlug, request: value.RequestID, revision: value.ExpectedRevision, fields: map[string]any{}}
	} else {
		return loom.Nullable[api.WorkitemAttachmentResult]{}, api.MakeInvalidParams(errors.New("invalid attachment action"))
	}
	result, err := s.mutateResource(ctx, "workitem_attachment", selected, s.attachments)
	if err != nil {
		return loom.Nullable[api.WorkitemAttachmentResult]{}, err
	}
	var record *api.AttachmentRecord
	if selected.action == "create" {
		var row map[string]any
		if json.Unmarshal(result.Record, &row) != nil {
			return loom.Nullable[api.WorkitemAttachmentResult]{}, workitemResultError()
		}
		record, err = attachmentProjection(row)
		if err != nil {
			return loom.Nullable[api.WorkitemAttachmentResult]{}, err
		}
	}
	if selected.request == nil && selected.revision == nil {
		if record == nil {
			return loom.NullValue[api.WorkitemAttachmentResult](), nil
		}
		return loom.NullableValue(api.NewWorkitemAttachmentResultAttachmentRecord(record)), nil
	}
	receipt := &api.AttachmentMutationReceipt{AttachmentID: result.ID, Revision: result.Revision, Result: loom.NullValue[api.AttachmentRecord]()}
	if record != nil {
		receipt.Result = loom.NullableValue(*record)
	}
	if selected.request != nil {
		if result.RequestID == nil || result.ReplayExpiresAt == nil || result.Replayed == nil {
			return loom.Nullable[api.WorkitemAttachmentResult]{}, workitemResultError()
		}
		id, expires := result.RequestID.String(), result.ReplayExpiresAt.UTC().Format(time.RFC3339Nano)
		receipt.RequestID, receipt.ReplayExpiresAt, receipt.Replayed = &id, &expires, result.Replayed
	}
	return loom.NullableValue(api.NewWorkitemAttachmentResultAttachmentMutationReceipt(receipt)), nil
}
func (s *Service) attachmentRead(ctx context.Context, q assets.AttachmentQuery, workspace *string, project, item, attachment string) (loom.Nullable[api.WorkitemAttachmentResult], error) {
	var err error
	q.ProjectID, err = uuid.Parse(project)
	if err != nil || q.ProjectID == uuid.Nil {
		return loom.Nullable[api.WorkitemAttachmentResult]{}, api.MakeInvalidParams(errors.New("invalid attachment project"))
	}
	q.IssueID, err = uuid.Parse(item)
	if err != nil || q.IssueID == uuid.Nil {
		return loom.Nullable[api.WorkitemAttachmentResult]{}, api.MakeInvalidParams(errors.New("invalid attachment item"))
	}
	if q.Action != "list" {
		q.AttachmentID, err = uuid.Parse(attachment)
		if err != nil || q.AttachmentID == uuid.Nil {
			return loom.Nullable[api.WorkitemAttachmentResult]{}, api.MakeInvalidParams(errors.New("invalid attachment identifier"))
		}
	}
	ctx, actor, err := readActor(ctx, workspace)
	if err != nil {
		return loom.Nullable[api.WorkitemAttachmentResult]{}, err
	}
	q.ActorID, q.WorkspaceID, q.WorkspaceSlug = actor.UserID, actor.WorkspaceID, actor.WorkspaceSlug
	value, err := s.attachments.Read(ctx, q)
	if err != nil {
		return loom.Nullable[api.WorkitemAttachmentResult]{}, resourceReadError(err, projectmutations.ErrForbidden, assets.ErrNotFound)
	}
	switch q.Action {
	case "list":
		rows, ok := value.([]map[string]any)
		if !ok {
			return loom.Nullable[api.WorkitemAttachmentResult]{}, workitemResultError()
		}
		records := api.AttachmentRecords{}
		for _, row := range rows {
			record, attachmentProjectionErr := attachmentProjection(djangoapi.Timestamps(row, actor.Timezone).(map[string]any))
			if attachmentProjectionErr != nil {
				return loom.Nullable[api.WorkitemAttachmentResult]{}, attachmentProjectionErr
			}
			records = append(records, record)
		}
		return loom.NullableValue(api.NewWorkitemAttachmentResultAttachmentRecords(records)), nil
	case "download_url":
		result := &api.AttachmentDownload{}
		if err = decodeWorkitemProjection(value, result); err != nil {
			return loom.Nullable[api.WorkitemAttachmentResult]{}, err
		}
		return loom.NullableValue(api.NewWorkitemAttachmentResultAttachmentDownload(result)), nil
	default:
		result := &api.AttachmentInline{}
		if err = decodeWorkitemProjection(value, result); err != nil {
			return loom.Nullable[api.WorkitemAttachmentResult]{}, err
		}
		return loom.NullableValue(api.NewWorkitemAttachmentResultAttachmentInline(result)), nil
	}
}
func attachmentProjection(row map[string]any) (*api.AttachmentRecord, error) {
	for _, key := range []string{"id", "created_at", "updated_at", "deleted_at", "entity_type", "entity_identifier", "external_id", "external_source", "created_by", "updated_by", "user", "workspace", "draft_issue", "project", "issue", "comment", "page", "name", "content_type", "attributes", "storage_metadata", "is_deleted", "is_archived", "is_uploaded", "size"} {
		if _, ok := row[key]; !ok {
			row[key] = nil
		}
	}
	size, err := sdkInteger(row["size"])
	if err != nil {
		return nil, workitemResultError()
	}
	row["size"] = size
	result := &api.AttachmentRecord{}
	return result, decodeWorkitemProjection(row, result)
}
