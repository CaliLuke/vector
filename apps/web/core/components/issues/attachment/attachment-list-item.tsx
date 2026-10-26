import { useParams } from "@tanstack/react-router";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { TrashIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssueAttachment } from "@plane/types";
// ui
import { CustomMenu } from "@plane/ui";
import { convertBytesToSize, getFileExtension, getFileName } from "@plane/utils/attachment";
import { getFileURL } from "@plane/utils/file";
import { renderFormattedDate } from "@plane/utils/datetime";
// components
//
import { ButtonAvatars } from "@/components/dropdowns/member/avatar";
import { getFileIcon } from "@/components/icons";
// helpers
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";

type TIssueAttachmentsListItem = {
  disabled?: boolean;
  attachment: TIssueAttachment;
  onDelete: (attachmentId: string) => void;
};

export function IssueAttachmentsListItem(props: TIssueAttachmentsListItem) {
  const { t } = useTranslation();
  // props
  const { attachment, disabled, onDelete } = props;
  // store hooks
  const { workspaceSlug } = useParams({ strict: false });
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const getUserDetails = (userId: string) => membersById?.get(userId)?.member;
  // derived values
  const fileName = getFileName(attachment.attributes.name ?? "");
  const fileExtension = getFileExtension(attachment.attributes.name ?? "");
  const fileIcon = getFileIcon(fileExtension, 18);
  const fileURL = getFileURL(attachment.asset_url ?? "");
  // hooks
  const { isMobile } = usePlatformOS();

  return (
    <>
      <div className="group flex h-11 items-center justify-between gap-3 pr-2 pl-9 hover:bg-surface-2">
        <button
          type="button"
          className="flex min-w-0 flex-1 items-center gap-3 truncate text-left text-13"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            window.open(fileURL, "_blank");
          }}
        >
          <div className="flex items-center gap-3">{fileIcon}</div>
          <Tooltip tooltipContent={`${fileName}.${fileExtension}`} isMobile={isMobile}>
            <p className="truncate font-medium text-secondary">{`${fileName}.${fileExtension}`}</p>
          </Tooltip>
          <span className="flex size-1.5 rounded-full bg-layer-1" />
          <span className="flex-shrink-0 text-placeholder">{convertBytesToSize(attachment.attributes.size)}</span>
        </button>

        <div className="flex items-center gap-3">
          {attachment?.created_by && (
            <Tooltip
              isMobile={isMobile}
              tooltipContent={`${getUserDetails(attachment?.created_by)?.display_name ?? ""} uploaded on ${renderFormattedDate(attachment.updated_at)}`}
            >
              <div className="flex items-center justify-center">
                <ButtonAvatars showTooltip userIds={attachment?.created_by} workspaceSlug={workspaceSlug} />
              </div>
            </Tooltip>
          )}

          <CustomMenu ellipsis closeOnSelect placement="bottom-end" disabled={disabled}>
            <CustomMenu.MenuItem
              onClick={() => {
                onDelete(attachment.id);
              }}
            >
              <div className="flex items-center gap-2">
                <TrashIcon className="h-3.5 w-3.5" strokeWidth={2} />
                <span>{t("common.actions.delete")}</span>
              </div>
            </CustomMenu.MenuItem>
          </CustomMenu>
        </div>
      </div>
    </>
  );
}
