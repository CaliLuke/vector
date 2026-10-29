/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { CommentActions } from "@/components/comments/types";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useRef, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
// plane imports
import { EIssueCommentAccessSpecifier } from "@plane/constants";
import { EFileAssetType } from "@plane/types";
import type { EditorRefApi } from "@plane/editor";

import { cn } from "@plane/utils/common";
import { isCommentEmpty } from "@plane/utils/string";
import { FormErrors } from "@plane/ui/form-fields";
// components
import { LiteTextEditor } from "@/components/editor/lite-text";
// hooks
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
// services
import { FileService } from "@/services/file.service";

type TCommentCreate = {
  canSubmit?: boolean;
  entityId: string;
  workspaceSlug: string;
  activityOperations: CommentActions;
  showToolbarInitially?: boolean;
  projectId?: string;
  onSubmitCallback?: (elementId: string) => void;
};

// services
const fileService = new FileService();

export const CommentCreate = function CommentCreate(props: TCommentCreate) {
  const {
    canSubmit = true,
    workspaceSlug,
    entityId,
    activityOperations,
    showToolbarInitially = false,
    projectId,
    onSubmitCallback,
  } = props;
  const scopeIsCurrent = useCommittedScope(JSON.stringify([workspaceSlug, projectId, entityId]));
  // states
  const [uploadedAssetIds, setUploadedAssetIds] = useState<string[]>([]);
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  // store hooks
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  // derived values
  const workspaceId = workspaceDetails?.id ?? "";
  const attachmentRequest = (assetIds: string[]) => ({
    assetIds,
    workspaceSlug,
    projectId,
    entityId,
    isVerified: activityOperations.isVerified,
    isCurrent: () => scopeIsCurrent() && activityOperations.isCurrent(),
  });
  const attachments = useMutation({
    mutationFn: async (request: ReturnType<typeof attachmentRequest>) => {
      if (!request.isVerified()) throw new Error("Reload your access before saving comment attachments.");
      if (request.projectId)
        await fileService.updateBulkProjectAssetsUploadStatus(
          request.workspaceSlug,
          request.projectId,
          request.entityId,
          { asset_ids: request.assetIds }
        );
      else
        await fileService.updateBulkWorkspaceAssetsUploadStatus(request.workspaceSlug, request.entityId, {
          asset_ids: request.assetIds,
        });
      return request;
    },
    onSuccess: (request) => {
      if (request.isCurrent()) setUploadedAssetIds((current) => current.filter((id) => !request.assetIds.includes(id)));
    },
  });
  const retryAttachments = () => {
    if (attachments.variables) attachments.mutate(attachmentRequest(attachments.variables.assetIds));
  };
  const form = useForm({
    defaultValues: { comment_html: "<p></p>", access: EIssueCommentAccessSpecifier.INTERNAL },
    onSubmit: async ({ value, formApi }) => {
      if (!canSubmit || !editorRef.current?.isEditorReadyToDiscard() || attachments.isPending || attachments.isError)
        throw new Error("Finish the comment editor and attachment operation before sending.");
      const comment = await activityOperations.createComment(value);
      if (!comment?.id) throw new Error("Comment creation did not return a saved comment.");
      if (!scopeIsCurrent() || !activityOperations.isCurrent()) return;
      formApi.reset();
      editorRef.current?.clearEditor();
      onSubmitCallback?.(comment.id);
      if (uploadedAssetIds.length) attachments.mutate(attachmentRequest([...uploadedAssetIds]));
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);

  return (
    <div className={cn("sticky bottom-0 z-[4] bg-surface-1 sm:static")}>
      {attachments.isError && (
        <div role="alert">
          Your comment was posted, but its attachments could not be saved.
          <button type="button" onClick={retryAttachments}>
            Retry attachments
          </button>
        </div>
      )}
      <form.Field name="access">
        {(access) => (
          <form.Field
            name="comment_html"
            validators={{
              onSubmit: ({ value }) => {
                if (!canSubmit) return "Reload the discussion before sending.";
                if (isCommentEmpty(value)) return "Enter a comment before sending.";
                if (!editorRef.current?.isEditorReadyToDiscard()) return "Comment editor is still loading.";
                if (attachments.isPending || attachments.isError)
                  return "Finish the comment attachment operation first.";
                return undefined;
              },
            }}
          >
            {(field) => (
              <>
                <LiteTextEditor
                  editable
                  workspaceId={workspaceId}
                  id={"add_comment_" + entityId}
                  value={"<p></p>"}
                  workspaceSlug={workspaceSlug}
                  projectId={projectId}
                  onEnterKeyPress={(e) => {
                    e?.preventDefault();
                    form.setErrorMap({ onSubmit: undefined });
                    void form.handleSubmit().catch(() => {});
                  }}
                  ref={editorRef}
                  initialValue={field.state.value}
                  containerClassName="min-h-min"
                  onChange={(_comment_json, comment_html) => field.handleChange(comment_html)}
                  accessSpecifier={access.state.value}
                  handleAccessChange={access.handleChange}
                  isSubmitting={!canSubmit || isSubmitting || attachments.isPending || attachments.isError}
                  fileAsset={{ entity_identifier: "", entity_type: EFileAssetType.COMMENT_DESCRIPTION }}
                  onAssetUpload={(assetId) => setUploadedAssetIds((prev) => [...prev, assetId])}

                  showToolbarInitially={showToolbarInitially}
                  parentClassName="p-2"
                  displayConfig={{
                    fontSize: "small-font",
                  }}
                />
                <FormErrors errors={field.state.meta.errors} />
              </>
            )}
          </form.Field>
        )}
      </form.Field>
    </div>
  );
};
