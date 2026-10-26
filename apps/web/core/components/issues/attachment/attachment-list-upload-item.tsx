/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// ui
import { Tooltip } from "@plane/propel/tooltip";
import { AttachmentUploadProgress } from "./upload-progress";
// components
import { getFileExtension } from "@plane/utils/attachment";
import { getFileIcon } from "@/components/icons";
// helpers
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// types
import type { TAttachmentUploadStatus } from "@/hooks/use-issue-resources";

type Props = {
  uploadStatus: TAttachmentUploadStatus;
};

export function IssueAttachmentsUploadItem(props: Props) {
  // props
  const { uploadStatus } = props;
  // derived values
  const fileName = uploadStatus.name;
  const fileExtension = getFileExtension(uploadStatus.name ?? "");
  const fileIcon = getFileIcon(fileExtension, 18);
  // hooks
  const { isMobile } = usePlatformOS();

  return (
    <div className="pointer-events-none flex h-11 items-center justify-between gap-3 bg-surface-2 pr-2 pl-9">
      <div className="flex items-center gap-3 truncate text-13">
        <div className="flex-shrink-0">{fileIcon}</div>
        <Tooltip tooltipContent={fileName} isMobile={isMobile}>
          <p className="truncate font-medium text-secondary">{fileName}</p>
        </Tooltip>
      </div>
      <AttachmentUploadProgress progress={uploadStatus.progress} />
    </div>
  );
}
