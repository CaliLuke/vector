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
import { useModule } from "@/hooks/store/use-module";

type ModuleActionsProps = {
  moduleId: string;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
};

export function useModuleActions(props: ModuleActionsProps) {
  const { moduleId, workspaceSlug, projectId } = props;
  const { addModuleToFavorites, removeModuleFromFavorites, updateModuleDetails } = useModule();
  const { setValue: toggleFavoriteMenu, storedValue } = useLocalStorage<boolean>(IS_FAVORITE_MENU_OPEN, false);

  const handleAddToFavorites = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    event.preventDefault();
    if (!workspaceSlug || !projectId) return;

    const addToFavoritePromise = addModuleToFavorites(workspaceSlug, projectId, moduleId).then((result) => {
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

    const removeFromFavoritePromise = removeModuleFromFavorites(workspaceSlug, projectId, moduleId);

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
      await updateModuleDetails(workspaceSlug, projectId, moduleId, payload);
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

      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: detail,
      });
    }
  };

  return { handleAddToFavorites, handleRemoveFromFavorites, handleModuleDetailsChange };
}
