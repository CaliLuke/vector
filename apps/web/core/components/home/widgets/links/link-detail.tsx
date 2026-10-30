/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";

import { useTranslation } from "@plane/i18n";
import { LinkIcon, NewTabIcon, EditIcon, TrashIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TContextMenuItem } from "@plane/ui";
import { LinkItemBlock } from "@plane/ui";
// plane utils
import { copyTextToClipboard } from "@plane/utils/string";
// hooks
import type { TLink } from "@plane/types";
// types
import type { TLinkOperations } from "./use-links";

export type TProjectLinkDetail = {
  linkDetail: TLink;
  onEdit: (link: TLink) => void;
  linkOperations: TLinkOperations;
};

export const ProjectLinkDetail = function ProjectLinkDetail(props: TProjectLinkDetail) {
  // props
  const { linkDetail, linkOperations, onEdit } = props;
  // hooks
  const { t } = useTranslation();
  // derived values
  const linkId = linkDetail.id;
  const linkUrl = linkDetail?.url;

  // handlers
  const handleEdit = useCallback(() => onEdit(linkDetail), [linkDetail, onEdit]);

  const handleCopyText = useCallback(() => {
    if (!linkUrl) return;
    copyTextToClipboard(linkUrl)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("link_copied"),
          message: t("view_link_copied_to_clipboard"),
        });
      })
      .catch(() => {
        setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy to clipboard", message: "Please try again." });
      });
  }, [linkUrl, t]);

  const handleOpenInNewTab = useCallback(() => {
    window.open(linkUrl, "_blank", "noopener,noreferrer");
  }, [linkUrl]);

  const handleDelete = useCallback(() => {
    void linkOperations.remove(linkId).catch(console.error);
  }, [linkId, linkOperations]);

  // derived values
  const menuItems = useMemo<TContextMenuItem[]>(
    () => [
      {
        key: "edit",
        action: handleEdit,
        title: t("edit"),
        icon: EditIcon,
      },
      {
        key: "open-new-tab",
        action: handleOpenInNewTab,
        title: t("open_in_new_tab"),
        icon: NewTabIcon,
      },
      {
        key: "copy-link",
        action: handleCopyText,
        title: t("copy_link"),
        icon: LinkIcon,
      },
      {
        key: "delete",
        action: handleDelete,
        title: t("delete"),
        icon: TrashIcon,
      },
    ],
    [handleEdit, handleOpenInNewTab, handleCopyText, handleDelete, t]
  );

  if (!linkDetail) return null;

  return (
    <LinkItemBlock
      title={linkDetail.title || linkDetail.url}
      url={linkDetail.url}
      createdAt={linkDetail.created_at}
      menuItems={menuItems}
      onClick={handleOpenInNewTab}
    />
  );
};
