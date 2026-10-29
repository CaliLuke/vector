/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { LinkIcon, CopyIcon, EditIcon, TrashIcon } from "@plane/propel/icons";
import { useIssueAction } from "@/hooks/use-issue-action";
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssueLink, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// ui
import { CustomMenu } from "@plane/ui";
import { calculateTimeAgo } from "@plane/utils/datetime";
import { copyTextToClipboard } from "@plane/utils/string";
// helpers
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { usePlatformOS } from "@/hooks/use-platform-os";
import type { TLinkOperations } from "@/hooks/use-issue-link-operations";

type TIssueLinkItem = {
  link: TIssueLink;
  linkOperations: Pick<TLinkOperations, "remove" | "pending">;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isNotAllowed: boolean;
  issueServiceType?: TIssueServiceType;
};

export function IssueLinkItem(props: TIssueLinkItem) {
  // props
  const {
    link,
    linkOperations,
    isNotAllowed,
    workspaceSlug,
    projectId,
    issueId,
    issueServiceType = EIssueServiceType.ISSUES,
  } = props;
  // hooks
  const { t } = useTranslation();
  const { toggleIssueLinkModal: toggleIssueLinkModalStore, setIssueLinkData } = useIssueDetailUiState(issueServiceType);
  const { isMobile } = usePlatformOS();
  const linkDetail = link;

  const copy = useIssueAction(
    { workspaceSlug, projectId, issueId, context: `link-copy:${link.id}:${link.url}` },
    Boolean(workspaceSlug && projectId && issueId),
    {
      successTitle: t("common.link_copied"),
      success: t("common.link_copied_to_clipboard"),
      error: t("power_k.contextual_actions.work_item.copy_url_toast_error"),
    }
  );
  const faviconUrl: string | undefined = linkDetail.metadata?.favicon;
  const linkTitle: string | undefined = linkDetail.metadata?.title;

  const toggleIssueLinkModal = (modalToggle: boolean) => {
    toggleIssueLinkModalStore(modalToggle);
    setIssueLinkData(linkDetail);
  };
  return (
    <>
      <div
        key={link.id}
        className="group 3xl:col-span-2 col-span-12 flex h-10 flex-shrink-0 items-center justify-between gap-3 rounded-sm border-[0.5px] border-subtle bg-surface-2 px-3 hover:bg-layer-1 lg:col-span-6 xl:col-span-4 2xl:col-span-3"
      >
        <div className="flex min-w-0 flex-1 items-center gap-2.5">
          {faviconUrl ? (
            <img src={faviconUrl} alt="favicon" className="size-4 flex-shrink-0" />
          ) : (
            <LinkIcon className="size-4 flex-shrink-0 text-tertiary group-hover:text-primary" />
          )}
          <Tooltip tooltipContent={linkDetail.url} isMobile={isMobile}>
            <a
              href={linkDetail.url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex w-0 flex-1 cursor-pointer items-center text-body-xs-regular"
            >
              <span className="w-0 flex-1 truncate">
                {linkDetail.title && linkDetail.title !== "" ? linkDetail.title : linkDetail.url}
                {linkTitle && linkTitle !== "" && (
                  <span className="text-caption-sm-regular text-placeholder"> {linkTitle}</span>
                )}
              </span>
            </a>
          </Tooltip>
        </div>
        <div className="flex flex-shrink-0 items-center gap-1">
          <p className="group-hover-text-secondary p-1 align-bottom text-caption-sm-regular leading-5 text-placeholder">
            {calculateTimeAgo(linkDetail.created_at)}
          </p>
          <button
            type="button"
            aria-label={t("common.actions.copy_link")}
            disabled={copy.disabled}
            onClick={() => copy.run(() => copyTextToClipboard(linkDetail.url))}
            className="relative grid cursor-pointer place-items-center rounded-sm p-1 text-placeholder outline-none group-hover:text-secondary hover:bg-layer-1"
          >
            <CopyIcon className="h-3.5 w-3.5 stroke-[1.5]" />
          </button>
          <CustomMenu
            ellipsis
            buttonClassName="text-placeholder group-hover:text-secondary"
            placement="bottom-end"
            closeOnSelect
            disabled={isNotAllowed || linkOperations.pending}
          >
            <CustomMenu.MenuItem
              className="flex items-center gap-2"
              onClick={() => {
                toggleIssueLinkModal(true);
              }}
            >
              <EditIcon className="h-3 w-3 stroke-[1.5] text-secondary" />
              {t("common.actions.edit")}
            </CustomMenu.MenuItem>
            <CustomMenu.MenuItem
              className="flex items-center gap-2"
              onClick={() => {
                void linkOperations.remove(linkDetail.id).catch(() => undefined);
              }}
            >
              <TrashIcon className="h-3 w-3" />
              {t("common.actions.delete")}
            </CustomMenu.MenuItem>
          </CustomMenu>
        </div>
      </div>
    </>
  );
}
