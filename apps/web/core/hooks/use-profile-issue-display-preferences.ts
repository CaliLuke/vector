/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { useParams } from "@tanstack/react-router";

import { EIssueFilterType, ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@plane/constants";
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
import { useLocalIssuePreferences } from "@/providers/local-issue-preferences";

export function useProfileIssueDisplayPreferences() {
  const { workspaceSlug, userId: routeUserId } = useParams({ strict: false });
  const userId = routeUserId;
  const preferences = useLocalIssuePreferences();
  if (!preferences || preferences.domain !== EIssuesStoreType.PROFILE) {
    throw new Error("Profile issue preferences are not mounted for this route.");
  }

  const { issueFilters, updateFilters } = preferences;
  const activeLayout = issueFilters?.displayFilters?.layout;

  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      if (!workspaceSlug || !userId) return;
      void updateFilters(EIssueFilterType.DISPLAY_FILTERS, { layout });
    },
    [workspaceSlug, updateFilters, userId]
  );

  const handleDisplayFilters = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      if (!workspaceSlug || !userId) return;
      void updateFilters(EIssueFilterType.DISPLAY_FILTERS, updatedDisplayFilter);
    },
    [workspaceSlug, updateFilters, userId]
  );

  const handleDisplayProperties = useCallback(
    (property: Partial<IIssueDisplayProperties>) => {
      if (!workspaceSlug || !userId) return;
      void updateFilters(EIssueFilterType.DISPLAY_PROPERTIES, property);
    },
    [workspaceSlug, updateFilters, userId]
  );

  return {
    activeLayout,
    handleDisplayFilters,
    handleDisplayProperties,
    handleLayoutChange,
    issueFilters,
    layoutDisplayFiltersOptions: activeLayout
      ? ISSUE_DISPLAY_FILTERS_BY_PAGE.profile_issues.layoutOptions[activeLayout]
      : undefined,
    userId,
  };
}
