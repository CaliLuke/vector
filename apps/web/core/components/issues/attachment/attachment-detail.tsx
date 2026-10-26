/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";
import { useState } from "react";

import { AlertCircle } from "lucide-react";
import { CloseIcon } from "@plane/propel/icons";
// ui
import { Tooltip } from "@plane/propel/tooltip";
import { convertBytesToSize, getFileExtension, getFileName } from "@plane/utils/attachment";
import { getFileURL } from "@plane/utils/file";
import { renderFormattedDate } from "@plane/utils/datetime";
import { truncateText } from "@plane/utils/string";
// icons
//
import { getFileIcon } from "@/components/icons";
// components
import { IssueAttachmentDeleteModal } from "@/components/issues/attachment/delete-attachment-modal";
// helpers
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
// types
import type { TIssueAttachment } from "@plane/types";
import type { TAttachmentHelpers } from "../issue-detail-widgets/attachments/helper";

type TIssueAttachmentsDetail = {
  attachment: TIssueAttachment;
  attachmentHelpers: TAttachmentHelpers;
  disabled?: boolean;
  workspaceSlug: string;
};

export const IssueAttachmentsDetail = function IssueAttachmentsDetail(props: TIssueAttachmentsDetail) {
  // props
  const { attachment, attachmentHelpers, disabled, workspaceSlug } = props;
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  // state
  const [isDeleteIssueAttachmentModalOpen, setIsDeleteIssueAttachmentModalOpen] = useState(false);
  // derived values
  const fileName = getFileName(attachment.attributes.name ?? "");
  const fileExtension = getFileExtension(attachment.asset_url ?? "");
  const fileIcon = getFileIcon(fileExtension, 28);
  const fileURL = getFileURL(attachment.asset_url ?? "");
  // hooks
  const { isMobile } = usePlatformOS();

  return (
    <>
      {isDeleteIssueAttachmentModalOpen && (
        <IssueAttachmentDeleteModal
          isOpen={isDeleteIssueAttachmentModalOpen}
          onClose={() => setIsDeleteIssueAttachmentModalOpen(false)}
          attachmentOperations={attachmentHelpers.operations}
          attachment={attachment}
        />
      )}
      <div className="flex h-[60px] items-center justify-between gap-1 rounded-md border-[2px] border-subtle bg-surface-1 px-4 py-2 text-13">
        <Link to={fileURL ?? ""} target="_blank" rel="noopener noreferrer">
          <div className="flex items-center gap-3">
            <div className="h-7 w-7">{fileIcon}</div>
            <div className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <Tooltip tooltipContent={fileName} isMobile={isMobile}>
                  <span className="text-13">{truncateText(`${fileName}`, 10)}</span>
                </Tooltip>
                <Tooltip
                  isMobile={isMobile}
                  tooltipContent={`${
                    memberById(attachment.updated_by)?.member.display_name ?? ""
                  } uploaded on ${renderFormattedDate(attachment.updated_at)}`}
                >
                  <span>
                    <AlertCircle className="h-3 w-3" />
                  </span>
                </Tooltip>
              </div>

              <div className="flex items-center gap-3 text-11 text-secondary">
                <span>{fileExtension.toUpperCase()}</span>
                <span>{convertBytesToSize(attachment.attributes.size)}</span>
              </div>
            </div>
          </div>
        </Link>

        {!disabled && (
          <button type="button" onClick={() => setIsDeleteIssueAttachmentModalOpen(true)}>
            <CloseIcon className="h-4 w-4 text-secondary hover:text-primary" />
          </button>
        )}
      </div>
    </>
  );
};
