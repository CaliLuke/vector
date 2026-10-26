/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { useParams } from "@tanstack/react-router";
import { IS_FAVORITE_MENU_OPEN } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EPageAccess } from "@plane/types";
import type { TPage } from "@plane/types";
import { copyUrlToClipboard } from "@plane/utils/string";
import { pageRedirectLink } from "@/helpers/page-permissions";
import { useCollaborativePageActions } from "@/hooks/use-collaborative-page-actions";
import { useDuplicateProjectPage } from "@/hooks/use-project-pages";
import { useFavorites } from "@/hooks/use-favorites";
import useLocalStorage from "./use-local-storage";

type TPageOperations = {
  toggleLock: () => Promise<void>;
  toggleAccess: () => Promise<void>;
  toggleFavorite: () => Promise<void>;
  openInNewTab: () => void;
  copyLink: () => Promise<boolean>;
  duplicate: () => Promise<void>;
  toggleArchive: () => Promise<void>;
};

type Props = { page: TPage; projectId?: string; workspaceSlug?: string };

function showOperationError(message: string) {
  setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message });
}

export const usePageOperations = ({
  page,
  projectId: explicitProjectId,
  workspaceSlug: explicitWorkspaceSlug,
}: Props) => {
  const routeParams = useParams({ strict: false });
  const workspaceSlug = explicitWorkspaceSlug ?? routeParams.workspaceSlug;
  const projectId = explicitProjectId ?? routeParams.projectId ?? page.project_ids?.[0];
  const pageId = page.id;
  const pageLink = workspaceSlug && projectId ? pageRedirectLink(workspaceSlug, projectId, pageId) : "";
  const favoritesState = useFavorites(workspaceSlug);
  const { favorites, favoritesByEntity, toggleFavorite } = favoritesState;
  const isFavorite = favoritesByEntity?.has(`page:${pageId}`) ?? false;
  const duplicate = useDuplicateProjectPage(workspaceSlug, projectId, pageId);
  const { executeCollaborativeAction } = useCollaborativePageActions({
    page,
    projectId,
    workspaceSlug: workspaceSlug ?? "",
  });
  const { setValue: toggleFavoriteMenu, storedValue: isFavoriteMenuOpen } = useLocalStorage<boolean>(
    IS_FAVORITE_MENU_OPEN,
    false
  );

  const pageOperations: TPageOperations = useMemo(
    () => ({
      copyLink: async () => {
        try {
          await copyUrlToClipboard(pageLink);
          setToast({ type: TOAST_TYPE.SUCCESS, title: "Link Copied!", message: "Page link copied to clipboard." });
          return true;
        } catch {
          showOperationError("Page link could not be copied. Please try again.");
          return false;
        }
      },
      duplicate: async () => {
        try {
          await duplicate.mutateAsync();
          setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Page duplicated successfully." });
        } catch {
          showOperationError("Page could not be duplicated. Please try again later.");
        }
      },
      openInNewTab: () => window.open(pageLink, "_blank"),
      toggleAccess: async () => {
        const isPublic = page.access === EPageAccess.PUBLIC;
        try {
          await executeCollaborativeAction({
            type: "sendMessageToServer",
            message: isPublic ? "make-private" : "make-public",
          });
          const changedPageType = isPublic ? "private" : "public";
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: "Success!",
            message: `The page has been marked ${changedPageType} and moved to the ${changedPageType} section.`,
          });
        } catch {
          showOperationError(`The page couldn't be marked ${isPublic ? "private" : "public"}. Please try again.`);
        }
      },
      toggleArchive: async () => {
        const archived = Boolean(page.archived_at);
        try {
          await executeCollaborativeAction({
            type: "sendMessageToServer",
            message: archived ? "unarchive" : "archive",
          });
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: "Success!",
            message: archived ? "Page restored successfully." : "Page archived successfully.",
          });
        } catch {
          showOperationError(`Page could not be ${archived ? "restored" : "archived"}. Please try again later.`);
        }
      },
      toggleFavorite: async () => {
        if (!pageId || !favorites) return;
        try {
          await toggleFavorite("page", pageId, projectId ?? null);
          if (!isFavorite && !isFavoriteMenuOpen) toggleFavoriteMenu(true);
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: "Success!",
            message: isFavorite ? "Page removed from favorites." : "Page added to favorites.",
          });
        } catch {
          showOperationError(
            isFavorite
              ? "Page could not be removed from favorites. Please try again later."
              : "Page could not be added to favorites. Please try again later."
          );
        }
      },
      toggleLock: async () => {
        const locked = page.is_locked;
        try {
          await executeCollaborativeAction({
            type: "sendMessageToServer",
            message: locked ? "unlock" : "lock",
          });
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: "Success!",
            message: locked ? "Page unlocked successfully." : "Page locked successfully.",
          });
        } catch {
          showOperationError(`Page could not be ${locked ? "unlocked" : "locked"}. Please try again later.`);
        }
      },
    }),
    [
      duplicate,
      executeCollaborativeAction,
      favorites,
      isFavorite,
      isFavoriteMenuOpen,
      page.access,
      page.archived_at,
      page.is_locked,
      pageId,
      pageLink,
      projectId,
      toggleFavorite,
      toggleFavoriteMenu,
    ]
  );

  return { pageOperations, isFavorite, isFavoriteReady: favorites !== undefined };
};
