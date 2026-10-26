/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { AttachmentUploadError } from "./upload-error";
import { useAttachmentDropzone } from "./use-attachment-dropzone";
// types
import type { TAttachmentOperations } from "../issue-detail-widgets/attachments/helper";

type TAttachmentOperationsModal = Pick<TAttachmentOperations, "create">;

type Props = {
  workspaceSlug: string;
  disabled?: boolean;
  attachmentOperations: TAttachmentOperationsModal;
};

export const IssueAttachmentUpload = function IssueAttachmentUpload(props: Props) {
  const { workspaceSlug, disabled = false, attachmentOperations } = props;
  const { getRootProps, getInputProps, isDragActive, isDragReject, isUploading, error, canRetry, retry } =
    useAttachmentDropzone({
      workspaceSlug,
      create: attachmentOperations.create,
      disabled,
    });

  return (
    <div>
      <div
        {...getRootProps()}
        className={`flex h-[60px] items-center justify-center rounded-md border-2 border-dashed bg-accent-primary/5 px-4 text-11 text-accent-primary ${
          isDragActive ? "border-accent-strong bg-accent-primary/10" : "border-subtle"
        } ${isDragReject ? "bg-danger-subtle" : ""} ${disabled ? "cursor-not-allowed" : "cursor-pointer"}`}
      >
        <input {...getInputProps()} />
        <span className="flex items-center gap-2">
          {isDragActive ? (
            <p>Drop here...</p>
          ) : isUploading ? (
            <p className="text-center">Uploading...</p>
          ) : (
            <p className="text-center">Click or drag a file here</p>
          )}
        </span>
      </div>
      <AttachmentUploadError error={error} canRetry={canRetry} retry={retry} disabled={disabled || isUploading} />
    </div>
  );
};
