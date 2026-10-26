/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Tooltip } from "@plane/propel/tooltip";
import { AttachmentUploadProgress } from "./upload-progress";
import { getFileExtension } from "@plane/utils/attachment";
import { truncateText } from "@plane/utils/string";
// ui
// icons
import { getFileIcon } from "@/components/icons";
// helpers
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// types
import type { TAttachmentUploadStatus } from "@/hooks/use-issue-resources";

type Props = {
  uploadStatus: TAttachmentUploadStatus;
};

export function IssueAttachmentsUploadDetails(props: Props) {
  // props
  const { uploadStatus } = props;
  // derived values
  const fileName = uploadStatus.name;
  const fileExtension = getFileExtension(uploadStatus.name ?? "");
  const fileIcon = getFileIcon(fileExtension, 28);
  // hooks
  const { isMobile } = usePlatformOS();

  return (
    <div className="pointer-events-none flex h-[60px] items-center justify-between gap-1 rounded-md border-[2px] border-subtle bg-surface-2 px-4 py-2 text-13">
      <div className="flex flex-shrink-0 items-center gap-3">
        <div className="h-7 w-7">{fileIcon}</div>
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <Tooltip tooltipContent={fileName} isMobile={isMobile}>
              <span className="text-13">{truncateText(`${fileName}`, 10)}</span>
            </Tooltip>
          </div>

          <div className="flex items-center gap-3 text-11 text-secondary">
            <span>{fileExtension.toUpperCase()}</span>
          </div>
        </div>
      </div>
      <AttachmentUploadProgress progress={uploadStatus.progress} />
    </div>
  );
}
