/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { NewTabIcon, EditIcon, TrashIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import { getIconForLink } from "@plane/utils/get-icon-for-link";
import { copyTextToClipboard } from "@plane/utils/string";
import { calculateTimeAgo } from "@plane/utils/datetime";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
// types
import type { TLinkOperationsModal } from "./create-update-link-modal";
import type { TIssueLink } from "@plane/types";

export type TIssueLinkDetail = {
  link: TIssueLink;
  linkOperations: TLinkOperationsModal;
  isNotAllowed: boolean;
  workspaceSlug: string;
};

export function IssueLinkDetail(props: TIssueLinkDetail) {
  // props
  const { link, linkOperations, isNotAllowed, workspaceSlug } = props;
  // hooks
  const { toggleIssueLinkModal: toggleIssueLinkModalStore, setIssueLinkData } = useIssueDetailUiState();
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const { isMobile } = usePlatformOS();
  const linkDetail = link;

  const Icon = getIconForLink(linkDetail.url);

  const toggleIssueLinkModal = (modalToggle: boolean) => {
    toggleIssueLinkModalStore(modalToggle);
    setIssueLinkData(linkDetail);
  };

  const createdByDetails = memberById(linkDetail.created_by_id)?.member;

  return (
    <div key={link.id}>
      <div className="relative flex flex-col rounded-md bg-surface-2 p-2.5">
        <div
          className="flex w-full cursor-pointer items-start justify-between gap-2"
          onClick={() => {
            copyTextToClipboard(linkDetail.url);
            setToast({
              type: TOAST_TYPE.SUCCESS,
              title: "Link copied!",
              message: "Link copied to clipboard",
            });
          }}
        >
          <div className="flex items-start gap-2 truncate">
            <span className="py-1">
              <Icon className="size-3 flex-shrink-0 stroke-2 text-tertiary group-hover:text-primary" />
            </span>
            <Tooltip
              tooltipContent={linkDetail.title && linkDetail.title !== "" ? linkDetail.title : linkDetail.url}
              isMobile={isMobile}
            >
              <span className="truncate text-11">
                {linkDetail.title && linkDetail.title !== "" ? linkDetail.title : linkDetail.url}
              </span>
            </Tooltip>
          </div>

          {!isNotAllowed && (
            <div className="z-[1] flex flex-shrink-0 items-center gap-2">
              <button
                type="button"
                className="flex items-center justify-center p-1 hover:bg-layer-1"
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  toggleIssueLinkModal(true);
                }}
              >
                <EditIcon className="h-3 w-3 stroke-[1.5] text-secondary" />
              </button>
              <a
                href={linkDetail.url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-center p-1 hover:bg-layer-1"
              >
                <NewTabIcon className="h-3 w-3 stroke-[1.5] text-secondary" />
              </a>
              <button
                type="button"
                className="flex items-center justify-center p-1 hover:bg-layer-1"
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  linkOperations.remove(linkDetail.id);
                }}
              >
                <TrashIcon className="h-3 w-3" />
              </button>
            </div>
          )}
        </div>

        <div className="px-5">
          <p className="mt-0.5 stroke-[1.5] text-11 text-tertiary">
            Added {calculateTimeAgo(linkDetail.created_at)}
            <br />
            {createdByDetails && (
              <>
                by {createdByDetails?.is_bot ? createdByDetails?.first_name + " Bot" : createdByDetails?.display_name}
              </>
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
