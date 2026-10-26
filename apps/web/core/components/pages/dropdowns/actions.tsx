/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { useMemo, useState } from "react";

import { ArchiveRestoreIcon, LockKeyhole, LockKeyholeOpen } from "lucide-react";
// constants
import { EPageAccess } from "@plane/types";
// plane editor
import { LinkIcon, CopyIcon, LockIcon, NewTabIcon, ArchiveIcon, TrashIcon, GlobeIcon } from "@plane/propel/icons";
// plane ui
import type { TContextMenuItem } from "@plane/ui";
import { ContextMenu, CustomMenu } from "@plane/ui";
// components
import { cn } from "@plane/utils/common";
import { DeletePageModal } from "@/components/pages/modals/delete-page-modal";
// hooks
import { usePageOperations } from "@/hooks/use-page-operations";
import { usePagePermissions } from "@/hooks/use-page-permissions";
// plane web hooks
// store types

export type TPageActions =
  | "full-screen"
  | "sticky-toolbar"
  | "copy-markdown"
  | "toggle-lock"
  | "toggle-access"
  | "open-in-new-tab"
  | "copy-link"
  | "make-a-copy"
  | "archive-restore"
  | "delete"
  | "version-history"
  | "export";

type Props = {
  extraOptions?: (TContextMenuItem & { key: TPageActions })[];
  optionsOrder: TPageActions[];
  page: TPage;
  parentRef?: React.RefObject<HTMLElement | null>;
  projectId?: string;
  workspaceSlug: string;
};

export function PageActions(props: Props) {
  const { extraOptions, optionsOrder, page, parentRef, projectId, workspaceSlug } = props;
  // states
  const [deletePageModal, setDeletePageModal] = useState(false);
  // params
  const permissions = usePagePermissions(page, workspaceSlug);
  // page operations
  const { pageOperations } = usePageOperations({
    page,
    projectId,
    workspaceSlug,
  });
  // derived values
  const { access, archived_at, is_locked } = page;
  const {
    canCurrentUserArchivePage,
    canCurrentUserChangeAccess,
    canCurrentUserDeletePage,
    canCurrentUserDuplicatePage,
    canCurrentUserLockPage,
  } = permissions;
  // menu items
  const MENU_ITEMS = useMemo(
    function MENU_ITEMS() {
      const menuItems: (TContextMenuItem & { key: TPageActions })[] = [
        {
          key: "toggle-lock",
          action: () => {
            void pageOperations.toggleLock();
          },
          title: is_locked ? "Unlock" : "Lock",
          icon: is_locked ? LockKeyholeOpen : LockKeyhole,
          shouldRender: canCurrentUserLockPage,
        },
        {
          key: "toggle-access",
          action: () => {
            void pageOperations.toggleAccess();
          },
          title: access === EPageAccess.PUBLIC ? "Make private" : "Make public",
          icon: access === EPageAccess.PUBLIC ? LockIcon : GlobeIcon,
          shouldRender: canCurrentUserChangeAccess && !archived_at,
        },
        {
          key: "open-in-new-tab",
          action: pageOperations.openInNewTab,
          title: "Open in new tab",
          icon: NewTabIcon,
          shouldRender: true,
        },
        {
          key: "copy-link",
          action: () => void pageOperations.copyLink(),
          title: "Copy link",
          icon: LinkIcon,
          shouldRender: true,
        },
        {
          key: "make-a-copy",
          action: () => {
            void pageOperations.duplicate();
          },
          title: "Make a copy",
          icon: CopyIcon,
          shouldRender: canCurrentUserDuplicatePage,
        },
        {
          key: "archive-restore",
          action: () => {
            void pageOperations.toggleArchive();
          },
          title: archived_at ? "Restore" : "Archive",
          icon: archived_at ? ArchiveRestoreIcon : ArchiveIcon,
          shouldRender: canCurrentUserArchivePage,
        },
        {
          key: "delete",
          action: () => {
            setDeletePageModal(true);
          },
          title: "Delete",
          icon: TrashIcon,
          shouldRender: canCurrentUserDeletePage && !!archived_at,
        },
      ];
      if (extraOptions) {
        menuItems.push(...extraOptions);
      }
      return menuItems;
    },
    [
      extraOptions,
      is_locked,
      canCurrentUserLockPage,
      access,
      canCurrentUserChangeAccess,
      archived_at,
      canCurrentUserDuplicatePage,
      canCurrentUserArchivePage,
      canCurrentUserDeletePage,
      pageOperations,
    ]
  );
  // arrange options
  const arrangedOptions = useMemo<(TContextMenuItem & { key: TPageActions })[]>(
    () =>
      optionsOrder
        .map((key) => MENU_ITEMS.find((item) => item.key === key))
        .filter((item): item is TContextMenuItem & { key: TPageActions } => !!item),
    [optionsOrder, MENU_ITEMS]
  );

  return (
    <>
      <DeletePageModal
        isOpen={deletePageModal}
        onClose={() => setDeletePageModal(false)}
        page={page}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      />
      {parentRef && <ContextMenu parentRef={parentRef} items={arrangedOptions} />}
      <CustomMenu placement="bottom-end" optionsClassName="max-h-[90vh]" ellipsis closeOnSelect>
        {arrangedOptions.map((item) => {
          if (item.shouldRender === false) return null;
          return (
            <CustomMenu.MenuItem
              key={item.key}
              onClick={() => {
                item.action?.();
              }}
              className={cn("flex items-center gap-2", item.className)}
              disabled={item.disabled}
            >
              {item.customContent ?? (
                <>
                  {item.icon && <item.icon className="size-3" />}
                  {item.title}
                </>
              )}
            </CustomMenu.MenuItem>
          );
        })}
      </CustomMenu>
    </>
  );
}
