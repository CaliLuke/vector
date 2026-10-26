/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { IssueLayoutQuickActions } from "../issue-layout-quick-actions";

import { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";
import { useIssueListPaginationOptions } from "@/hooks/use-issue-list-pagination-options";
import { useParams } from "@tanstack/react-router";
import type { FC } from "react";
import { useCallback } from "react";

// plane imports
import { ALL_ISSUES, EIssueFilterType, EUserPermissions } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IIssueDisplayFilterOptions } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useIssueLayoutActions } from "@/hooks/use-issue-layout-actions";
import { useIssueListType } from "@/hooks/use-issue-list-type";

// local imports
import { IssueLayoutHOC } from "../issue-layout-HOC";
import type { IQuickActionProps, TRenderQuickActions } from "../list/list-view-types";
import { SpreadsheetView } from "./spreadsheet-view";
import { projectAllIssueIdsByDisplayOrder } from "../issue-display-ordering";
import { IssueListViewProvider } from "@/providers/issue-list-view";

export type SpreadsheetStoreType =
  | EIssuesStoreType.PROJECT
  | EIssuesStoreType.MODULE
  | EIssuesStoreType.CYCLE
  | EIssuesStoreType.PROJECT_VIEW
  | EIssuesStoreType.TEAM
  | EIssuesStoreType.TEAM_VIEW
  | EIssuesStoreType.EPIC;

function isSpreadsheetStoreType(storeType: EIssuesStoreType): storeType is SpreadsheetStoreType {
  return (
    storeType === EIssuesStoreType.PROJECT ||
    storeType === EIssuesStoreType.MODULE ||
    storeType === EIssuesStoreType.CYCLE ||
    storeType === EIssuesStoreType.PROJECT_VIEW ||
    storeType === EIssuesStoreType.TEAM ||
    storeType === EIssuesStoreType.TEAM_VIEW ||
    storeType === EIssuesStoreType.EPIC
  );
}

interface IBaseSpreadsheetRoot {
  QuickActions: FC<IQuickActionProps>;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
  isCompletedCycle?: boolean;
  viewId?: string;
  isEpic?: boolean;
}

export function BaseSpreadsheetRoot(props: IBaseSpreadsheetRoot) {
  const { QuickActions, canEditPropertiesBasedOnProject, isCompletedCycle = false, isEpic = false } = props;
  // router
  const { projectId } = useParams({ strict: false });
  // store hooks
  const issueListType = useIssueListType();
  if (!isSpreadsheetStoreType(issueListType)) throw new Error("Spreadsheet layouts require a supported issue list.");
  const storeType = issueListType;
  const { workspaceSlug } = useParams({ strict: false });
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { issueFilters: displayPreferences, projectPreferences, actions } = useIssueLayoutActions(storeType);
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const { quickAddIssue, updateIssue, removeIssue, removeIssueFromView, archiveIssue, restoreIssue, updateFilters } =
    actions;
  // derived values
  const enableQuickAdd = true;
  const enableIssueCreation = true;
  const enableInlineEditing = true;
  const displayFilters = displayPreferences?.displayFilters ?? {};
  const displayProperties = displayPreferences?.displayProperties ?? {};
  // user role validation
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const paginationOptions = useIssueListPaginationOptions(storeType, displayFilters, EIssueLayoutTypes.SPREADSHEET);
  const readState = useIssueListPresentation(storeType, paginationOptions, displayFilters);
  const { labelsById, estimateProjection, stateProjection, parentProjection } = readState;

  const canEditProperties = useCallback(
    (targetProjectId: string | undefined) => {
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && targetProjectId
          ? canEditPropertiesBasedOnProject(targetProjectId)
          : isEditingAllowed;

      return enableInlineEditing && isEditingAllowedBasedOnProject;
    },
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed]
  );

  const issueIds =
    projectAllIssueIdsByDisplayOrder(
      readState.groupedIssueIds,
      (issueId) => readState.issuesById[issueId],
      membersById,
      displayFilters.order_by,
      labelsById,
      estimateProjection,
      stateProjection,
      parentProjection
    ) ?? [];
  const nextPageResults = readState.getPaginationData(ALL_ISSUES, undefined)?.nextPageResults;

  const saveDisplayFilters = useCallback(
    async (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      try {
        if (storeType === EIssuesStoreType.PROJECT) {
          if (!projectPreferences) throw new Error("Project preferences are unavailable for the active issue list.");
          await projectPreferences.save({ display_filters: updatedDisplayFilter });
          return;
        }
        if (!updateFilters) throw new Error("Issue display filters are unavailable for this list.");
        await updateFilters(projectId ?? "", EIssueFilterType.DISPLAY_FILTERS, updatedDisplayFilter);
      } catch (error) {
        setToast({
          title: "Error!",
          type: TOAST_TYPE.ERROR,
          message: error instanceof Error ? error.message : "Failed to update issue display filters.",
        });
      }
    },
    [projectId, projectPreferences, storeType, updateFilters]
  );
  const handleDisplayFiltersUpdate = useCallback(
    (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
      void saveDisplayFilters(updatedDisplayFilter);
    },
    [saveDisplayFilters]
  );

  const renderQuickActions: TRenderQuickActions = useCallback(
    ({ issue, parentRef, customActionButton, placement, portalElement }) => (
      <IssueLayoutQuickActions
        QuickActions={QuickActions}
        parentRef={parentRef}
        customActionButton={customActionButton}
        issue={issue}
        actions={{ removeIssue, updateIssue, removeIssueFromView, archiveIssue, restoreIssue }}
        portalElement={portalElement}
        readOnly={!canEditProperties(issue.project_id ?? undefined) || isCompletedCycle}
        placement={placement}
      />
    ),
    [
      isCompletedCycle,
      canEditProperties,
      removeIssue,
      QuickActions,
      updateIssue,
      removeIssueFromView,
      archiveIssue,
      restoreIssue,
    ]
  );

  return (
    <IssueListViewProvider readState={readState}>
      <IssueLayoutHOC readState={readState} layout={EIssueLayoutTypes.SPREADSHEET}>
        <SpreadsheetView
          displayProperties={displayProperties}
          displayFilters={displayFilters}
          handleDisplayFilterUpdate={handleDisplayFiltersUpdate}
          issueIds={issueIds}
          quickActions={renderQuickActions}
          updateIssue={updateIssue}
          canEditProperties={canEditProperties}
          quickAddCallback={quickAddIssue}
          enableQuickCreateIssue={enableQuickAdd}
          disableIssueCreation={!enableIssueCreation || !isEditingAllowed || isCompletedCycle}
          canLoadMoreIssues={nextPageResults}
          loadMoreIssues={() => void readState.fetchNextIssues()}
          isEpic={isEpic}
        />
      </IssueLayoutHOC>
    </IssueListViewProvider>
  );
}
