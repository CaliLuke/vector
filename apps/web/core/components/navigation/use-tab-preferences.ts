/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { EUserPermissions } from "@plane/constants";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { usePatchProjectUserProperties, useProjectUserProperties } from "@/hooks/use-project-user-properties";
import type { ProjectUserPropertiesPatch } from "@/queries/project-user-properties";
import { DEFAULT_TAB_KEY } from "./tab-navigation-utils";
import type { TTabPreferences } from "./tab-navigation-utils";

const EMPTY_HIDDEN_TABS: string[] = [];

export type TTabPreferencesHook = {
  tabPreferences: TTabPreferences;
  handleToggleDefaultTab: (tabKey: string) => void;
  handleHideTab: (tabKey: string) => void;
  handleShowTab: (tabKey: string) => void;
};

/**
 * Custom hook to manage tab preferences for a project
 * Reads the native project-preferences Query and persists through its mutation.
 *
 * @param workspaceSlug - The workspace slug
 * @param projectId - The project ID
 * @returns Tab preferences state and handlers
 */
export const useTabPreferences = (workspaceSlug: string, projectId: string): TTabPreferencesHook => {
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const canAccess = permissions.can([EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST]);
  const propertiesQuery = useProjectUserProperties(workspaceSlug, projectId, {
    canAccess,
    fetchEnabled: false,
  });
  const canWrite = canAccess && propertiesQuery.data !== undefined && !propertiesQuery.isError;
  const { mutateAsync } = usePatchProjectUserProperties(workspaceSlug, projectId, canWrite);
  const navigationPreferences = propertiesQuery.data?.preferences?.navigation;
  const defaultTab = navigationPreferences?.default_tab || DEFAULT_TAB_KEY;
  const hideInMoreMenu = navigationPreferences?.hide_in_more_menu || EMPTY_HIDDEN_TABS;

  // Convert native preferences to the navigation component shape.
  const tabPreferences: TTabPreferences = useMemo(() => {
    return {
      defaultTab,
      hiddenTabs: hideInMoreMenu,
    };
  }, [defaultTab, hideInMoreMenu]);

  const updatePreferences = (patch: ProjectUserPropertiesPatch) => {
    if (!canWrite) return Promise.reject(new Error("Project preferences are not ready to update."));
    return mutateAsync(patch);
  };

  /**
   * Toggle default tab setting
   * If tab is already default, resets to work_items; otherwise sets as default
   */
  const handleToggleDefaultTab = (tabKey: string) => {
    const newDefaultTab = tabKey === tabPreferences.defaultTab ? DEFAULT_TAB_KEY : tabKey;
    updatePreferences({ preferences: { navigation: { default_tab: newDefaultTab } } })
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Default tab updated successfully.",
        });
        return;
      })
      .catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Failed to update default tab. Please try again later.",
        });
      });
  };

  /**
   * Hide a tab (moves to overflow menu with "Show" option)
   */
  const handleHideTab = (tabKey: string) => {
    void updatePreferences({ navigationAction: { type: "hide-tab", tabKey } }).catch((error) => {
      console.error("Error hiding tab:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to hide tab. Please try again later.",
      });
    });
  };

  /**
   * Show a previously hidden tab (returns to visible pool)
   */
  const handleShowTab = (tabKey: string) => {
    void updatePreferences({ navigationAction: { type: "show-tab", tabKey } }).catch((error) => {
      console.error("Error showing tab:", error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Something went wrong. Please try again later.",
      });
    });
  };

  return {
    tabPreferences,
    handleToggleDefaultTab,
    handleHideTab,
    handleShowTab,
  };
};
