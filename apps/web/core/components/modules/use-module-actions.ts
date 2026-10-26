/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { MouseEvent } from "react";
import { IS_FAVORITE_MENU_OPEN } from "@plane/constants";
import { useLocalStorage } from "@plane/hooks";
import { TOAST_TYPE, setPromiseToast, setToast } from "@plane/propel/toast";
import type { IModule } from "@plane/types";
import { EUserPermissions } from "@plane/constants";
import { useProjectModules } from "@/hooks/use-modules";
import { useQueryClient } from "@tanstack/react-query";
import { captureParentAccess } from "@/queries/cycle-module-scope";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useFavorites } from "@/hooks/use-favorites";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";

export type ModuleActionsProps = {
  moduleId: string;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
};

export function useModuleActions(props: ModuleActionsProps) {
  const { moduleId, workspaceSlug, projectId } = props;
  const catalogue = useProjectModules(workspaceSlug, projectId),
    client = useQueryClient();
  const current = useCommittedScope(JSON.stringify([catalogue.scope, moduleId]));
  const access = captureParentAccess(client, catalogue.scope);
  const { favorites, favoritesByEntity, toggleFavorite } = useFavorites(workspaceSlug);
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const isFavorite = favoritesByEntity?.has(`module:${moduleId}`) ?? false;
  const getUserDetails = (userId: string) => membersById?.get(userId)?.member;
  const { setValue: toggleFavoriteMenu, storedValue } = useLocalStorage<boolean>(IS_FAVORITE_MENU_OPEN, false);

  const handleAddToFavorites = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    event.preventDefault();
    if (!workspaceSlug || !projectId) return;

    const addToFavoritePromise = toggleFavorite("module", moduleId, projectId).then((result) => {
      if (!storedValue) toggleFavoriteMenu(true);
      return result;
    });

    setPromiseToast(addToFavoritePromise, {
      loading: "Adding module to favorites...",
      success: {
        title: "Success!",
        message: () => "Module added to favorites.",
      },
      error: {
        title: "Error!",
        message: () => "Couldn't add the module to favorites. Please try again.",
      },
    });
  };

  const handleRemoveFromFavorites = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    event.preventDefault();
    if (!workspaceSlug || !projectId) return;

    const removeFromFavoritePromise = toggleFavorite("module", moduleId, projectId);

    setPromiseToast(removeFromFavoritePromise, {
      loading: "Removing module from favorites...",
      success: {
        title: "Success!",
        message: () => "Module removed from favorites.",
      },
      error: {
        title: "Error!",
        message: () => "Couldn't remove the module from favorites. Please try again.",
      },
    });
  };

  const handleModuleDetailsChange = async (payload: Partial<IModule>) => {
    if (!workspaceSlug || !projectId) return;

    try {
      const result = await catalogue.mutation.mutateAsync({ type: "patch", id: moduleId, data: payload });
      if (!current() || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Module updated successfully.",
      });
    } catch (error: unknown) {
      const detail =
        typeof error === "object" && error !== null && "detail" in error && typeof error.detail === "string"
          ? error.detail
          : "Module could not be updated. Please try again.";

      if (current() && access.isCurrent())
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: detail,
        });
    }
  };

  return {
    handleAddToFavorites,
    handleRemoveFromFavorites,
    handleModuleDetailsChange,
    isFavorite,
    isFavoritesReady: favorites !== undefined,
    isEditingAllowed: canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]),
    getUserDetails,
  };
}
