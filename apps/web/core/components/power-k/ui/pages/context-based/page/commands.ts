/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";

import { ArchiveIcon, ArchiveRestoreIcon, LockKeyhole, LockKeyholeOpen, Star, StarOff } from "lucide-react";
import { useTranslation } from "@plane/i18n";
// plane imports
import { LinkIcon, GlobeIcon, LockIcon } from "@plane/propel/icons";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { EPageAccess } from "@plane/types";
import { copyTextToClipboard } from "@plane/utils/string";
// components
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
// plane web imports
import { useParams } from "@tanstack/react-router";
import { usePagePermissions } from "@/hooks/use-page-permissions";
import { useFavorites } from "@/hooks/use-favorites";
import { useProjectPage } from "@/hooks/use-project-pages";
import { useCollaborativePageActions } from "@/hooks/use-collaborative-page-actions";

function createFavoriteCommand(isFavorite: boolean, canFavorite: boolean, action: () => void): TPowerKCommandConfig {
  return {
    id: "toggle_page_favorite",
    i18n_title: isFavorite
      ? "power_k.contextual_actions.page.remove_from_favorites"
      : "power_k.contextual_actions.page.add_to_favorites",
    icon: isFavorite ? StarOff : Star,
    group: "contextual",
    contextType: "page",
    type: "action",
    action,
    modifierShortcut: "shift+f",
    isEnabled: () => canFavorite,
    isVisible: () => canFavorite,
    closeOnSelect: true,
  };
}

function createLockCommand(isLocked: boolean, canLock: boolean, action: () => void): TPowerKCommandConfig {
  return {
    id: "toggle_page_lock",
    i18n_title: isLocked ? "power_k.contextual_actions.page.unlock" : "power_k.contextual_actions.page.lock",
    icon: isLocked ? LockKeyholeOpen : LockKeyhole,
    group: "contextual",
    contextType: "page",
    type: "action",
    action: () => {
      action();
    },
    modifierShortcut: "shift+l",
    isEnabled: () => canLock,
    isVisible: () => canLock,
    closeOnSelect: true,
  };
}

function createAccessCommand(
  access: EPageAccess | undefined,
  canChangeAccess: boolean,
  action: () => void
): TPowerKCommandConfig {
  return {
    id: "toggle_page_access",
    i18n_title:
      access === EPageAccess.PUBLIC
        ? "power_k.contextual_actions.page.make_private"
        : "power_k.contextual_actions.page.make_public",
    icon: access === EPageAccess.PUBLIC ? LockIcon : GlobeIcon,
    group: "contextual",
    contextType: "page",
    type: "action",
    action: () => {
      action();
    },
    modifierShortcut: "shift+a",
    isEnabled: () => canChangeAccess,
    isVisible: () => canChangeAccess,
    closeOnSelect: true,
  };
}

export const usePowerKPageContextBasedActions = (): TPowerKCommandConfig[] => {
  // navigation
  const { pageId, workspaceSlug, projectId } = useParams({ strict: false });
  const pageQuery = useProjectPage(workspaceSlug, projectId, pageId, false);
  const page = pageQuery.data;
  const permissions = usePagePermissions(page, workspaceSlug);
  const { access, archived_at } = page ?? {};
  const { canCurrentUserArchivePage, canCurrentUserChangeAccess, canCurrentUserFavoritePage, canCurrentUserLockPage } =
    permissions;
  const { favoritesByEntity, toggleFavorite } = useFavorites(workspaceSlug?.toString());
  const isFavorite = favoritesByEntity?.has(`page:${page?.id}`) ?? false;
  const isLocked = !!page?.is_locked;
  const { executeCollaborativeAction } = useCollaborativePageActions({
    page,
    workspaceSlug: workspaceSlug ?? "",
    projectId,
  });
  const performPageAction = useCallback(
    async (message: "lock" | "unlock" | "archive" | "unarchive" | "make-public" | "make-private") => {
      if (!page) return;
      try {
        await executeCollaborativeAction({ type: "sendMessageToServer", message });
      } catch {
        setToast({ type: TOAST_TYPE.ERROR, title: "Some error occurred" });
      }
    },
    [executeCollaborativeAction, page]
  );
  const lockAction = useCallback(() => {
    void performPageAction(isLocked ? "unlock" : "lock");
  }, [isLocked, performPageAction]);
  const accessAction = useCallback(() => {
    void performPageAction(access === EPageAccess.PUBLIC ? "make-private" : "make-public");
  }, [access, performPageAction]);
  const archiveAction = useCallback(() => {
    void performPageAction(archived_at ? "unarchive" : "archive");
  }, [archived_at, performPageAction]);
  // translation
  const { t } = useTranslation();

  const togglePageFavorite = useCallback(() => {
    if (!page?.id) return;
    void toggleFavorite("page", page.id, page.project_ids?.[0] ?? null).catch(() => {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Some error occurred",
      });
    });
  }, [page?.id, page?.project_ids, toggleFavorite]);

  const copyPageUrlToClipboard = useCallback(() => {
    const url = new URL(window.location.href);
    copyTextToClipboard(url.href)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("power_k.contextual_actions.page.copy_url_toast_success"),
        });
        return undefined;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("power_k.contextual_actions.page.copy_url_toast_error"),
        });
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return [
    createLockCommand(isLocked, canCurrentUserLockPage, lockAction),
    createAccessCommand(access, canCurrentUserChangeAccess, accessAction),
    {
      id: "toggle_page_archive",
      i18n_title: archived_at ? "power_k.contextual_actions.page.restore" : "power_k.contextual_actions.page.archive",
      icon: archived_at ? ArchiveRestoreIcon : ArchiveIcon,
      group: "contextual",
      contextType: "page",
      type: "action",
      action: () => {
        archiveAction();
      },
      modifierShortcut: "shift+r",
      isEnabled: () => canCurrentUserArchivePage,
      isVisible: () => canCurrentUserArchivePage,
      closeOnSelect: true,
    },
    createFavoriteCommand(isFavorite, canCurrentUserFavoritePage, togglePageFavorite),
    {
      id: "copy_page_url",
      i18n_title: "power_k.contextual_actions.page.copy_url",
      icon: LinkIcon,
      group: "contextual",
      contextType: "page",
      type: "action",
      action: copyPageUrlToClipboard,
      modifierShortcut: "cmd+shift+,",
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
  ];
};
