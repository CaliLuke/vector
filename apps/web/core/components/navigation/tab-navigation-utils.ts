/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Tab preferences type
export type TTabPreferences = {
  defaultTab: string;
  hiddenTabs: string[];
};

export const DEFAULT_TAB_KEY = "work_items";

/**
 * Map tab keys to their corresponding URLs
 * @param workspaceSlug - The workspace slug
 * @param projectId - The project ID
 * @param tabKey - The tab key to map
 * @returns Full URL path for the tab
 */
export const getTabUrl = (workspaceSlug: string, projectId: string, tabKey: string): string => {
  const baseUrl = `/${workspaceSlug}/projects/${projectId}`;
  const tabUrlMap: Record<string, string> = {
    work_items: `${baseUrl}/issues`,
    cycles: `${baseUrl}/cycles`,
    modules: `${baseUrl}/modules`,
    views: `${baseUrl}/views`,
    pages: `${baseUrl}/pages`,
    intake: `${baseUrl}/intake`,
    overview: `${baseUrl}/overview`,
    epics: `${baseUrl}/epics`,
  };
  return tabUrlMap[tabKey] || `${baseUrl}/issues`; // fallback to issues
};
