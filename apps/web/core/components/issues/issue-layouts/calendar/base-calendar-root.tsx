/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";
import { useIssueListPaginationOptions } from "@/hooks/use-issue-list-pagination-options";
import { useParams } from "@tanstack/react-router";
import type { FC } from "react";
import { useCallback, useMemo } from "react";

// plane imports
import { EUserPermissions } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IIssueDisplayFilterOptions, TIssue } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
// hooks
import { useCalendarState } from "@/providers/issue-layout-ui-state";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useIssueLayoutActions } from "@/hooks/use-issue-layout-actions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { useCurrentUserStartOfWeek } from "@/hooks/use-current-user-profile";
import { useCalendarDisplayFilterUpdate } from "@/hooks/use-calendar-display-filter-update";
// types
import type { IQuickActionProps, TRenderQuickActions } from "../list/list-view-types";
import { IssueLayoutQuickActions } from "../issue-layout-quick-actions";
import { CalendarChart } from "./calendar";
import { getCalendarState } from "./calendar-state";
import { handleDragDrop } from "./utils";
import { projectGroupedIssueIdsByDisplayOrder } from "../issue-display-ordering";
import { IssueListViewProvider } from "@/providers/issue-list-view";

interface IBaseCalendarRoot {
  QuickActions: FC<IQuickActionProps>;
  addIssuesToView?: (issueIds: string[]) => Promise<unknown>;
  isCompletedCycle?: boolean;
  viewId?: string | undefined;
  isEpic?: boolean;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
}

async function updateCalendarIssueDate(
  issueId: string | undefined,
  issueProjectId: string | undefined,
  sourceDate: string | undefined,
  destinationDate: string | undefined,
  workspaceSlug: string,
  updateIssue?: (projectId: string, issueId: string, data: Partial<TIssue>) => Promise<unknown>
) {
  if (!issueId || !destinationDate || !sourceDate || !issueProjectId) return;

  try {
    await handleDragDrop(issueId, sourceDate, destinationDate, workspaceSlug, issueProjectId, updateIssue);
  } catch (error) {
    setToast({
      title: "Error!",
      type: TOAST_TYPE.ERROR,
      message: error instanceof Error ? error.message : "Failed to perform this action",
    });
  }
}

function isCalendarIssuePropertyEditable(
  targetProjectId: string | undefined,
  canEditForProject: ((projectId: string) => boolean) | undefined,
  inlineEditingEnabled: boolean | undefined,
  isEditingAllowed: boolean
) {
  const isEditingAllowedForIssue =
    canEditForProject && targetProjectId ? canEditForProject(targetProjectId) : isEditingAllowed;

  return !!inlineEditingEnabled && isEditingAllowedForIssue;
}

export function BaseCalendarRoot(props: IBaseCalendarRoot) {
  const {
    QuickActions,
    addIssuesToView,
    isCompletedCycle = false,
    isEpic = false,
    canEditPropertiesBasedOnProject,
  } = props;

  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { projectId } = useParams({ strict: false });
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);

  // hooks
  const fallbackStoreType = useIssueListType();
  const storeType = isEpic ? EIssuesStoreType.EPIC : fallbackStoreType;
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { projectPreferences, issueFilters: displayPreferences, actions } = useIssueLayoutActions(storeType);
  const { quickAddIssue, updateIssue, removeIssue, removeIssueFromView, archiveIssue, restoreIssue, updateFilters } =
    actions;

  const { activeMonthDate, activeWeekDate, updateCalendarDates } = useCalendarState();
  const startOfWeek = useCurrentUserStartOfWeek();
  const calendarState = useMemo(
    () => getCalendarState(activeMonthDate, activeWeekDate, startOfWeek),
    [activeMonthDate, activeWeekDate, startOfWeek]
  );

  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const enableInlineEditing = true;

  const displayFilters: IIssueDisplayFilterOptions = displayPreferences?.displayFilters ?? {};
  const handleDisplayFilterUpdate = useCalendarDisplayFilterUpdate(
    storeType,
    projectPreferences,
    displayFilters.calendar,
    updateFilters
  );

  const calendarFilters = displayFilters.calendar;
  const layout = calendarFilters?.layout ?? "month";
  const showWeekends = calendarFilters?.show_weekends ?? false;
  const paginationOptions = useIssueListPaginationOptions(storeType, displayFilters, EIssueLayoutTypes.CALENDAR);
  const readState = useIssueListPresentation(storeType, paginationOptions, displayFilters);
  const { fetchNextIssues, getPaginationData: readPagination, getGroupIssueCount: readGroupCount } = readState;
  const issueMap = readState.issuesById;
  const { labelsById, estimateProjection, stateProjection, parentProjection } = readState;
  const groupedIssueIds = readState.groupedIssueIds ?? {};
  const visibleGroupedIssueIds = projectGroupedIssueIdsByDisplayOrder(
    groupedIssueIds,
    (issueId) => issueMap[issueId],
    membersById,
    displayFilters.order_by,
    labelsById,
    estimateProjection,
    stateProjection,
    parentProjection
  );

  const handleDragAndDrop = useCallback(
    (
      issueId: string | undefined,
      issueProjectId: string | undefined,
      sourceDate: string | undefined,
      destinationDate: string | undefined
    ) => updateCalendarIssueDate(issueId, issueProjectId, sourceDate, destinationDate, workspaceSlug, updateIssue),
    [updateIssue, workspaceSlug]
  );

  const loadMoreIssues = useCallback(
    (dateString: string) => {
      void fetchNextIssues(dateString);
    },
    [fetchNextIssues]
  );

  const getPaginationData = useCallback(
    (groupId: string | undefined) => readPagination(groupId, undefined),
    [readPagination]
  );

  const getGroupIssueCount = useCallback(
    (groupId: string | undefined) => readGroupCount(groupId, undefined, false),
    [readGroupCount]
  );

  const canEditProperties = useCallback(
    (targetProjectId: string | undefined) =>
      isCalendarIssuePropertyEditable(
        targetProjectId,
        canEditPropertiesBasedOnProject,
        enableInlineEditing,
        isEditingAllowed
      ),
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed]
  );

  const renderQuickActions = useCallback<TRenderQuickActions>(
    (actionProps) => (
      <IssueLayoutQuickActions
        {...actionProps}
        QuickActions={QuickActions}
        readOnly={!canEditProperties(actionProps.issue.project_id ?? undefined) || isCompletedCycle}
        actions={{ removeIssue, updateIssue, removeIssueFromView, archiveIssue, restoreIssue }}
      />
    ),
    [
      QuickActions,
      isCompletedCycle,
      canEditProperties,
      removeIssue,
      updateIssue,
      removeIssueFromView,
      archiveIssue,
      restoreIssue,
    ]
  );

  return (
    <IssueListViewProvider readState={readState}>
      <div className="h-full w-full overflow-hidden bg-surface-1 pt-4">
        <CalendarChart
          displayFilters={displayFilters}
          issues={issueMap}
          groupedIssueIds={visibleGroupedIssueIds ?? {}}
          layout={layout}
          showWeekends={showWeekends}
          calendarState={calendarState}
          activeMonthDate={activeMonthDate}
          activeWeekDate={activeWeekDate}
          updateCalendarDates={updateCalendarDates}
          quickActions={renderQuickActions}
          loadMoreIssues={loadMoreIssues}
          getPaginationData={getPaginationData}
          getGroupIssueCount={getGroupIssueCount}
          addIssuesToView={addIssuesToView}
          quickAddCallback={quickAddIssue}
          readOnly={isCompletedCycle}
          updateFilters={handleDisplayFilterUpdate}
          handleDragAndDrop={handleDragAndDrop}
          canEditProperties={canEditProperties}
          isEpic={isEpic}
          readState={readState}
        />
      </div>
    </IssueListViewProvider>
  );
}
