/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";
// plane imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@plane/constants";
import { EIssuesStoreType } from "@plane/types";
// components
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
// hooks
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
import { useLocalIssuePreferences } from "@/providers/local-issue-preferences";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { ArchivedIssueListLayout } from "../list/roots/archived-issue-root";

export function ArchivedIssueLayoutRoot() {
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // hooks
  const preferences = useLocalIssuePreferences();
  if (!preferences || preferences.domain !== EIssuesStoreType.ARCHIVED)
    throw new Error("Archived issue preferences are not mounted for this route.");
  // derived values
  const workItemFilters = preferences.issueFilters;

  if (!workspaceSlug || !projectId || !workItemFilters) return <></>;
  return (
    <ProjectLevelWorkItemFiltersHOC
      entityType={EIssuesStoreType.ARCHIVED}
      entityId={projectId}
      filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.archived_issues.filters}
      initialWorkItemFilters={workItemFilters}
      updateFilters={preferences.updateRichFilters}
      projectId={projectId}
      workspaceSlug={workspaceSlug}
    >
      {({ editor: archivedWorkItemsFilter }) => (
        <>
          {archivedWorkItemsFilter && <WorkItemFiltersRow editor={archivedWorkItemsFilter} />}
          <div className="relative h-full w-full overflow-auto">
            <ArchivedIssueListLayout />
          </div>
          <IssuePeekOverview />
        </>
      )}
    </ProjectLevelWorkItemFiltersHOC>
  );
}
