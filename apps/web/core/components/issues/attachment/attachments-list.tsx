/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
// types
import type { TAttachmentHelpers } from "../issue-detail-widgets/attachments/helper";
// components
import { IssueAttachmentsDetail } from "./attachment-detail";
import { IssueAttachmentsUploadDetails } from "./attachment-upload-details";

type TIssueAttachmentsList = {
  workspaceSlug: string;
  attachmentHelpers: TAttachmentHelpers;
  disabled?: boolean;
};

export function IssueAttachmentsList(props: TIssueAttachmentsList) {
  const { workspaceSlug, attachmentHelpers, disabled } = props;
  // derived values
  const { snapshot: attachmentSnapshot } = attachmentHelpers;
  const { uploadStatus } = attachmentSnapshot;
  const issueAttachments = attachmentSnapshot.attachments;

  return (
    <>
      {uploadStatus?.map((status) => (
        <IssueAttachmentsUploadDetails key={status.id} uploadStatus={status} />
      ))}
      {attachmentSnapshot.isError && (
        <button type="button" className="text-11 text-danger-primary" onClick={() => void attachmentSnapshot.retry()}>
          Retry
        </button>
      )}
      {issueAttachments?.map((attachment) => (
        <IssueAttachmentsDetail
          key={attachment.id}
          attachment={attachment}
          disabled={disabled}
          attachmentHelpers={attachmentHelpers}
          workspaceSlug={workspaceSlug}
        />
      ))}
    </>
  );
}
