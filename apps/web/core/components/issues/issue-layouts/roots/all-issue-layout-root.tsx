/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation, useRouter } from "@tanstack/react-router";
import React, { useMemo } from "react";

// plane imports
import { GLOBAL_VIEW_TRACKER_ELEMENTS, ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@plane/constants";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import type { EIssueLayoutTypes } from "@plane/types";
import { EIssuesStoreType, STATIC_VIEW_TYPES } from "@plane/types";
// assets
// components
import { IssuePeekOverview } from "@/components/issues/peek-overview";
import { WorkspaceActiveLayout } from "@/components/views/helper";
import { WorkspaceLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/workspace-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useIssueListPaginationOptions } from "@/hooks/use-issue-list-pagination-options";
import { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";
import { useWorkspaceIssuePreferences } from "@/providers/workspace-issue-preferences";
import { getWorkspaceViewsLoadState, useWorkspaceView } from "@/hooks/use-workspace-views";

import { useWorkspaceIssueProperties } from "@/hooks/use-workspace-issue-properties";

type Props = {
  isDefaultView: boolean;
};

export function AllIssueLayoutRoot(props: Props) {
  const { isDefaultView } = props;
  // router
  const router = useRouter();
  const { workspaceSlug, globalViewId } = useParams({ strict: false });
  // search params
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  // store hooks
  const viewsQuery = useWorkspaceView(workspaceSlug, globalViewId);
  const preferences = useWorkspaceIssuePreferences();
  // Derived values
  const viewDetails = viewsQuery.view;
  const workItemFilters = preferences?.issueFilters;
  const activeLayout: EIssueLayoutTypes | undefined = workItemFilters?.displayFilters?.layout;
  // Determine initial work item filters based on view type and availability
  const initialWorkItemFilters = useMemo(() => {
    if (!globalViewId) return undefined;

    const isStaticView = STATIC_VIEW_TYPES.includes(globalViewId);
    const hasViewDetails = Boolean(viewDetails);

    if (!isStaticView && !hasViewDetails) return undefined;

    return {
      displayFilters: workItemFilters?.displayFilters,
      displayProperties: workItemFilters?.displayProperties,
      kanbanFilters: workItemFilters?.kanbanFilters,
      richFilters: workItemFilters?.richFilters ?? {},
    };
  }, [globalViewId, viewDetails, workItemFilters]);

  // Custom hooks
  useWorkspaceIssueProperties(workspaceSlug);

  // Route filters
  const routeFilters: { [key: string]: string } = {};
  searchParams.forEach((value: string, key: string) => {
    routeFilters[key] = value;
  });

  // Fetch next pages callback
  // Canonical saved views and the scoped React draft supply the current filter inputs.
  const { isLoading: globalViewsLoading, isError: viewLoadFailed } = getWorkspaceViewsLoadState(
    viewsQuery,
    !isDefaultView
  );

  const paginationOptions = useIssueListPaginationOptions(EIssuesStoreType.GLOBAL, workItemFilters?.displayFilters);
  const issueRead = useIssueListPresentation(
    EIssuesStoreType.GLOBAL,
    paginationOptions,
    workItemFilters?.displayFilters
  );
  const issuesLoading = issueRead.isPending;

  if (viewLoadFailed) {
    return (
      <EmptyStateDetailed
        title="Workspace views are unavailable"
        description="We couldn't verify workspace access or load this view. Try again."
        assetKey="view"
        actions={[{ label: "Retry", onClick: () => void viewsQuery.retry(), variant: "primary" }]}
      />
    );
  }

  // Empty state
  if (!globalViewsLoading && !viewDetails && !isDefaultView) {
    return (
      <EmptyStateDetailed
        title="View does not exist"
        description="The view you are looking for does not exist or you don't have permission to view it."
        assetKey="view"
        actions={[
          {
            label: "Go to All work items",
            onClick: () => void router.navigate({ href: `/${workspaceSlug}/workspace-views/all-issues` }),
            variant: "primary",
          },
        ]}
      />
    );
  }

  if (issueRead.isError)
    return (
      <EmptyStateDetailed
        title="Work items are unavailable"
        description="Work items could not be loaded. Try again."
        assetKey="view"
        actions={[{ label: "Retry", onClick: () => void issueRead.retry(), variant: "primary" }]}
      />
    );

  if (!workspaceSlug || !globalViewId) return null;
  return (
    <WorkspaceLevelWorkItemFiltersHOC
      enableSaveView
      saveViewOptions={{
        label: "Save as",
      }}
      enableUpdateView
      entityId={globalViewId}
      entityType={EIssuesStoreType.GLOBAL}
      filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.my_issues.filters}
      initialWorkItemFilters={initialWorkItemFilters}
      updateFilters={(expression) => preferences?.updateExpression(expression)}
      workspaceSlug={workspaceSlug}
    >
      {({ editor: globalWorkItemsFilter }) => (
        <div className="h-full overflow-hidden bg-surface-1">
          <div className="flex h-full w-full flex-col border-b border-strong">
            {globalWorkItemsFilter && (
              <WorkItemFiltersRow
                editor={globalWorkItemsFilter}
                trackerElements={{
                  saveView: GLOBAL_VIEW_TRACKER_ELEMENTS.HEADER_SAVE_VIEW_BUTTON,
                }}
              />
            )}
            <WorkspaceActiveLayout
              activeLayout={activeLayout}
              isDefaultView={isDefaultView}
              isLoading={issuesLoading}
              workspaceSlug={workspaceSlug}
              globalViewId={globalViewId}
              routeFilters={routeFilters}
              readState={issueRead}
              globalViewsLoading={globalViewsLoading}
            />
          </div>
          {/* peek overview */}
          <IssuePeekOverview />
        </div>
      )}
    </WorkspaceLevelWorkItemFiltersHOC>
  );
}
