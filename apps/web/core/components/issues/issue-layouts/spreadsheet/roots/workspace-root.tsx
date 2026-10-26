/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { IssueLayoutQuickActions } from "../../issue-layout-quick-actions";

import { EstimateCatalogueBoundary } from "@/components/estimates/catalogue-boundary";
import { useIssueEstimateOrder } from "@/hooks/use-issue-estimate-projection";
import type { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";
import React, { useCallback } from "react";
// plane constants
import { ALL_ISSUES, EIssueFilterType, EUserPermissions } from "@plane/constants";
import type { IIssueDisplayFilterOptions } from "@plane/types";
import { EIssuesStoreType, EIssueLayoutTypes } from "@plane/types";
// components
import { AllIssueQuickActions } from "@/components/issues/issue-layouts/quick-action-dropdowns";
import { SpreadsheetLayoutLoader } from "@/components/ui/loader/layouts/spreadsheet-layout-loader";
// hooks
import { useWorkspaceIssuePreferences } from "@/providers/workspace-issue-preferences";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useWorkspaceIssueProperties } from "@/hooks/use-workspace-issue-properties";
// store
import { IssueLayoutHOC } from "../../issue-layout-HOC";
import type { TRenderQuickActions } from "../../list/list-view-types";
import { SpreadsheetView } from "../spreadsheet-view";
import { projectAllIssueIdsByDisplayOrder } from "../../issue-display-ordering";
import { IssueListViewProvider } from "@/providers/issue-list-view";

type Props = {
  isDefaultView: boolean;
  isLoading?: boolean;
  workspaceSlug: string;
  globalViewId: string;
  routeFilters: {
    [key: string]: string;
  };
  readState: ReturnType<typeof useIssueListPresentation>;
  globalViewsLoading: boolean;
};

export function WorkspaceSpreadsheetRoot(props: Props) {
  const { isLoading = false, workspaceSlug, globalViewId, readState } = props;

  // Custom hooks
  useWorkspaceIssueProperties(workspaceSlug);

  // Store hooks
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const { updateIssue, removeIssue, archiveIssue } = useIssuesActions(EIssuesStoreType.GLOBAL);
  const { canProject } = useWorkspaceProjectRoles(workspaceSlug);

  // Derived values
  const preferences = useWorkspaceIssuePreferences();
  const issueFilters = preferences?.issueFilters;
  const displayFilters: IIssueDisplayFilterOptions = issueFilters?.displayFilters ?? {};
  const displayProperties = issueFilters?.displayProperties ?? {};
  const estimateOrder = useIssueEstimateOrder(workspaceSlug, undefined, true, displayFilters.order_by);
  const issueMap = readState.issuesById;
  const groupedIssueIds = readState.groupedIssueIds;
  const estimateProjection = estimateOrder.projection;

  // Permission checker
  const canEditProperties = useCallback(
    (projectId: string | undefined) => {
      if (!projectId) return false;
      return canProject(projectId, [EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
    },
    [canProject]
  );

  // Display filters handler
  const handleDisplayFiltersUpdate = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      if (!workspaceSlug || !globalViewId) return;

      preferences?.updateFilters(EIssueFilterType.DISPLAY_FILTERS, updatedDisplayFilter);
    },
    [preferences, workspaceSlug, globalViewId]
  );

  // Quick actions renderer
  const renderQuickActions: TRenderQuickActions = useCallback(
    ({ issue, parentRef, customActionButton, placement, portalElement }) => (
      <IssueLayoutQuickActions
        QuickActions={AllIssueQuickActions}
        parentRef={parentRef}
        customActionButton={customActionButton}
        issue={issue}
        actions={{ removeIssue, updateIssue, archiveIssue }}
        portalElement={portalElement}
        readOnly={!canEditProperties(issue.project_id ?? undefined)}
        placement={placement}
      />
    ),
    [canEditProperties, removeIssue, updateIssue, archiveIssue]
  );

  const loading = isLoading || readState.isPending || !globalViewId || !groupedIssueIds;

  // Computed values
  const issueIds =
    projectAllIssueIdsByDisplayOrder(
      groupedIssueIds,
      (issueId) => issueMap[issueId],
      membersById,
      displayFilters.order_by,
      readState.labelsById,
      estimateProjection,
      readState.stateProjection,
      readState.parentProjection
    ) ?? [];
  const nextPageResults = readState.getPaginationData(ALL_ISSUES, undefined)?.nextPageResults;

  // Render spreadsheet
  return (
    <EstimateCatalogueBoundary state={estimateOrder} loading={<SpreadsheetLayoutLoader />}>
      {loading ? (
        <SpreadsheetLayoutLoader />
      ) : (
        <IssueListViewProvider readState={readState}>
          <IssueLayoutHOC
            layout={EIssueLayoutTypes.SPREADSHEET}
            readState={{
              isPending: readState.isPending,
              isError: readState.isError,
              retry: async () => {
                await Promise.all([readState.retry(), estimateOrder.retry()]);
              },
              issueCount: readState.issueCount,
            }}
          >
            <SpreadsheetView
              displayProperties={displayProperties}
              displayFilters={displayFilters}
              handleDisplayFilterUpdate={handleDisplayFiltersUpdate}
              issueIds={issueIds}
              quickActions={renderQuickActions}
              updateIssue={updateIssue}
              canEditProperties={canEditProperties}
              canLoadMoreIssues={nextPageResults}
              loadMoreIssues={() => void readState.fetchNextIssues()}
              isWorkspaceLevel
            />
          </IssueLayoutHOC>
        </IssueListViewProvider>
      )}
    </EstimateCatalogueBoundary>
  );
}
