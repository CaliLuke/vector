/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";

import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
import { useIssueFilterPreferences } from "@/hooks/use-issue-filter-preferences";

type Scope = {
  workspaceSlug: string | undefined;
  projectId: string | undefined;
};

export function useProjectIssueDisplayPreferences({ workspaceSlug, projectId }: Scope) {
  const { issueFilters, projectPreferences } = useIssueFilterPreferences(EIssuesStoreType.PROJECT);
  const activeLayout = issueFilters?.displayFilters?.layout;

  const handleLayoutChange = useCallback(
    (layout: EIssueLayoutTypes) => {
      if (!workspaceSlug || !projectId) return;
      void projectPreferences?.save({ display_filters: { layout } });
    },
    [workspaceSlug, projectId, projectPreferences]
  );

  const handleDisplayFilters = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      if (!workspaceSlug || !projectId) return;
      void projectPreferences?.save({ display_filters: updatedDisplayFilter });
    },
    [workspaceSlug, projectId, projectPreferences]
  );

  const handleDisplayProperties = useCallback(
    (property: Partial<IIssueDisplayProperties>) => {
      if (!workspaceSlug || !projectId) return;
      void projectPreferences?.save({ display_properties: property });
    },
    [workspaceSlug, projectId, projectPreferences]
  );

  return {
    activeLayout,
    handleDisplayFilters,
    handleDisplayProperties,
    handleLayoutChange,
    issueFilters,
  };
}
