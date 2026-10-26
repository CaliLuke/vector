/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/session";
import { publishSettingsOptions } from "@/queries/publishing";
import React, { useRef, useState } from "react";

import { useForm, useStore } from "@tanstack/react-form";
// plane imports
import type { EditorRefApi } from "@plane/editor";
import { SitesFileService } from "@plane/services";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
// editor components
import { LiteTextEditor } from "@/components/editor/lite-text-editor";
// hooks
import { commentMutationOptions } from "@/queries/issue-actions";
import { EFileAssetType } from "@plane/types";
// services
const fileService = new SitesFileService();

const defaultValues = {
  comment_html: "",
};

type Props = {
  anchor: string;
  issueId: string;
  disabled?: boolean;
};

export const AddComment = function AddComment(props: Props) {
  const { anchor, issueId } = props;
  // states
  const [uploadedAssetIds, setUploadAssetIds] = useState<string[]>([]);
  // refs
  const editorRef = useRef<EditorRefApi>(null);
  const mutation = useMutation(commentMutationOptions(useQueryClient(), anchor, issueId));
  const { data: currentUser } = useQuery(currentUserOptions);
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));
  const { workspace: workspaceID } = publishSettings ?? {};
  const attachmentMutation = useMutation({
    mutationFn: ({ commentId, assetIds }: { commentId: string; assetIds: string[] }) =>
      fileService.updateBulkAssetsUploadStatus(anchor, commentId, { asset_ids: assetIds }),
    onSuccess: () => setUploadAssetIds([]),
    onError: () =>
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Attachments could not be saved",
        message: "Your comment was posted. Retry saving its attachments.",
      }),
  });

  const form = useForm({
    defaultValues,
    onSubmit: async ({ value, formApi }) => {
      if (!currentUser || attachmentMutation.isPending || attachmentMutation.isError || !value.comment_html) return;
      const comment = await mutation.mutateAsync({ kind: "add", data: value });
      formApi.reset();
      editorRef.current?.clearEditor();
      if (comment && uploadedAssetIds.length > 0) {
        attachmentMutation.mutate({ commentId: comment.id, assetIds: uploadedAssetIds });
      }
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);

  return (
    <div>
      {attachmentMutation.isError && (
        <div role="alert">
          Your comment was posted, but its attachments could not be saved.
          <button onClick={() => attachmentMutation.mutate(attachmentMutation.variables)}>Retry attachments</button>
        </div>
      )}
      <div className="issue-comments-section">
        <form.Field name="comment_html">
          {(field) => (
            <LiteTextEditor
              editable
              onEnterKeyPress={(e) => {
                e.preventDefault();
                if (currentUser)
                  void form.handleSubmit().catch(() => {
                    // The mutation already reported the failed comment save.
                  });
              }}
              anchor={anchor}
              workspaceId={workspaceID?.toString() ?? ""}
              ref={editorRef}
              id="peek-overview-add-comment"
              initialValue={field.state.value}
              onChange={(comment_json, comment_html) => field.handleChange(comment_html)}
              isSubmitting={isSubmitting || attachmentMutation.isPending || attachmentMutation.isError}
              placeholder="Add comment..."
              uploadFile={async (blockId, file) => {
                const { asset_id } = await fileService.uploadAsset(
                  anchor,
                  { entity_identifier: "", entity_type: EFileAssetType.COMMENT_DESCRIPTION },
                  file
                );
                setUploadAssetIds((prev) => [...prev, asset_id]);
                return asset_id;
              }}
              displayConfig={{
                fontSize: "small-font",
              }}
            />
          )}
        </form.Field>
      </div>
    </div>
  );
};
