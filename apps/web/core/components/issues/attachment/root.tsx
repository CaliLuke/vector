/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
import { useAttachmentOperations } from "../issue-detail-widgets/attachments/helper";
// components
import { IssueAttachmentUpload } from "./attachment-upload";
import { IssueAttachmentsList } from "./attachments-list";

export type TIssueAttachmentRoot = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled?: boolean;
};

export function IssueAttachmentRoot(props: TIssueAttachmentRoot) {
  // props
  const { workspaceSlug, projectId, issueId, disabled = false } = props;
  // hooks
  const attachmentHelpers = useAttachmentOperations(workspaceSlug, projectId, issueId);

  return (
    <div className="relative space-y-3">
      <h3 className="text-body-sm-medium">Attachments</h3>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
        <IssueAttachmentUpload
          key={attachmentHelpers.uploadScopeKey}
          workspaceSlug={workspaceSlug}
          disabled={disabled}
          attachmentOperations={attachmentHelpers.operations}
        />
        <IssueAttachmentsList workspaceSlug={workspaceSlug} disabled={disabled} attachmentHelpers={attachmentHelpers} />
      </div>
    </div>
  );
}
