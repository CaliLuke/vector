/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { UploadCloud } from "lucide-react";
import { useTranslation } from "@plane/i18n";
import type { TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
// plane web hooks
// types
import type { TAttachmentHelpers } from "../issue-detail-widgets/attachments/helper";
// components
import { AttachmentUploadError } from "./upload-error";
import { useAttachmentDropzone } from "./use-attachment-dropzone";
import { IssueAttachmentsListItem } from "./attachment-list-item";
import { IssueAttachmentsUploadItem } from "./attachment-list-upload-item";
// types
import { IssueAttachmentDeleteModal } from "./delete-attachment-modal";

type TIssueAttachmentItemList = {
  workspaceSlug: string;
  attachmentHelpers: TAttachmentHelpers;
  disabled?: boolean;
  issueServiceType?: TIssueServiceType;
};

export function IssueAttachmentItemList(props: TIssueAttachmentItemList) {
  return <IssueAttachmentItemListContent key={props.attachmentHelpers.uploadScopeKey} {...props} />;
}

const IssueAttachmentItemListContent = function IssueAttachmentItemListContent(props: TIssueAttachmentItemList) {
  const { workspaceSlug, attachmentHelpers, disabled, issueServiceType = EIssueServiceType.ISSUES } = props;
  const { t } = useTranslation();
  // states
  // store hooks
  const { attachmentDeleteModalId, toggleDeleteAttachmentModal } = useIssueDetailUiState(issueServiceType);
  const { operations: attachmentOperations, snapshot: attachmentSnapshot } = attachmentHelpers;
  const { create: createAttachment } = attachmentOperations;
  const { uploadStatus } = attachmentSnapshot;
  // file size
  // derived values
  const issueAttachments = attachmentSnapshot.attachments;

  const { getRootProps, getInputProps, isDragActive, isUploading, error, canRetry, retry } = useAttachmentDropzone({
    workspaceSlug,
    create: createAttachment,
    disabled,
  });

  return (
    <>
      {uploadStatus?.map((status) => (
        <IssueAttachmentsUploadItem key={status.id} uploadStatus={status} />
      ))}
      {attachmentSnapshot.isError && (
        <button
          type="button"
          className="px-2 text-11 text-danger-primary"
          onClick={() => void attachmentSnapshot.retry()}
        >
          {t("common.retry")}
        </button>
      )}
      <AttachmentUploadError error={error} canRetry={canRetry} retry={retry} disabled={!!disabled || isUploading} />
      {issueAttachments && (
        <>
          {attachmentDeleteModalId && (
            <IssueAttachmentDeleteModal
              isOpen={Boolean(attachmentDeleteModalId)}
              onClose={() => toggleDeleteAttachmentModal(null)}
              attachmentOperations={attachmentOperations}
              attachment={issueAttachments.find((attachment) => attachment.id === attachmentDeleteModalId)}
            />
          )}
          <div
            {...getRootProps()}
            className={`relative flex flex-col ${isDragActive && issueAttachments.length < 3 ? "min-h-[200px]" : ""} ${disabled ? "cursor-not-allowed" : "cursor-pointer"}`}
          >
            <input {...getInputProps()} />
            {isDragActive && (
              <div className="absolute top-0 left-0 z-30 flex h-full w-full items-center justify-center bg-surface-2/75">
                <div className="flex items-center justify-center rounded-md bg-surface-1 p-1">
                  <div className="flex flex-col items-center justify-center rounded-md border border-dashed border-strong px-5 py-6">
                    <UploadCloud className="size-7" />
                    <span className="text-13 text-tertiary">{t("attachment.drag_and_drop")}</span>
                  </div>
                </div>
              </div>
            )}
            {issueAttachments?.map((attachment) => (
              <IssueAttachmentsListItem
                key={attachment.id}
                attachment={attachment}
                disabled={disabled}
                onDelete={toggleDeleteAttachmentModal}
              />
            ))}
          </div>
        </>
      )}
    </>
  );
};
