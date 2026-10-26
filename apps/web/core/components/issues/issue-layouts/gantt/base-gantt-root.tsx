/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";
import { useIssueListPaginationOptions } from "@/hooks/use-issue-list-pagination-options";
import { useParams } from "@tanstack/react-router";
import React, { useCallback, useState } from "react";
import { addDays } from "date-fns/addDays";

// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IBlockUpdateData, TIssue } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType, GANTT_TIMELINE_TYPE } from "@plane/types";
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
// components
import { TimeLineTypeContext } from "@/components/gantt-chart/contexts";
import { GanttChartRoot } from "@/components/gantt-chart/root";
import { IssueGanttSidebar } from "@/components/gantt-chart/sidebar/issues/sidebar";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useIssueLayoutActions } from "@/hooks/use-issue-layout-actions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useIssueListType } from "@/hooks/use-issue-list-type";

import { useBulkOperationStatus } from "@/hooks/use-bulk-operation-status";
// local imports
import { IssueLayoutHOC } from "../issue-layout-HOC";
import { GanttQuickAddIssueButton, QuickAddIssueRoot } from "../quick-add";
import { IssueGanttBlock } from "./blocks";
import { projectAllIssueIdsByDisplayOrder } from "../issue-display-ordering";

interface IBaseGanttRoot {
  viewId?: string | undefined;
  isCompletedCycle?: boolean;
  isEpic?: boolean;
}

export type GanttStoreType =
  | EIssuesStoreType.PROJECT
  | EIssuesStoreType.MODULE
  | EIssuesStoreType.CYCLE
  | EIssuesStoreType.PROJECT_VIEW
  | EIssuesStoreType.EPIC;

function isGanttStoreType(storeType: EIssuesStoreType): storeType is GanttStoreType {
  return (
    storeType === EIssuesStoreType.PROJECT ||
    storeType === EIssuesStoreType.MODULE ||
    storeType === EIssuesStoreType.CYCLE ||
    storeType === EIssuesStoreType.PROJECT_VIEW ||
    storeType === EIssuesStoreType.EPIC
  );
}

export function BaseGanttRoot(props: IBaseGanttRoot) {
  const { isCompletedCycle = false, isEpic = false } = props;
  const { t } = useTranslation();
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { projectId } = useParams({ strict: false });

  const issueListType = useIssueListType();
  if (!isGanttStoreType(issueListType)) throw new Error("Gantt layouts require a project-scoped issue list.");
  const storeType = issueListType;
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const { issueFilters: displayPreferences, actions } = useIssueLayoutActions(storeType);
  const { updateIssue, quickAddIssue, updateIssueDates } = actions;
  // store hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const appliedDisplayFilters = displayPreferences?.displayFilters;
  // plane web hooks
  const isBulkOperationsEnabled = useBulkOperationStatus();
  // derived values
  const [today] = useState(() => new Date());
  const targetDate = addDays(today, 1);

  const paginationOptions = useIssueListPaginationOptions(storeType, appliedDisplayFilters, EIssueLayoutTypes.GANTT);
  const readState = useIssueListPresentation(storeType, paginationOptions, appliedDisplayFilters);
  const { fetchNextIssues, getPaginationData } = readState;
  const issueMap = readState.issuesById;
  const { labelsById, estimateProjection, stateProjection, parentProjection } = readState;

  const issuesIds =
    projectAllIssueIdsByDisplayOrder(
      readState.groupedIssueIds,
      (issueId) => issueMap[issueId],
      membersById,
      appliedDisplayFilters?.order_by,
      labelsById,
      estimateProjection,
      stateProjection,
      parentProjection
    ) ?? [];
  const nextPageResults = getPaginationData(undefined, undefined)?.nextPageResults;

  const loadMoreIssues = useCallback(() => {
    void fetchNextIssues();
  }, [fetchNextIssues]);

  const updateIssueBlockStructure = async (issue: TIssue, data: IBlockUpdateData) => {
    if (!workspaceSlug) return;

    const payload: Partial<TIssue> = {
      start_date: data.start_date,
      target_date: data.target_date,
    };
    if (data.sort_order !== undefined) payload.sort_order = data.sort_order.newSortOrder;

    if (updateIssue) await updateIssue(issue.project_id, issue.id, payload);
  };

  const isAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  const updateBlockDates = useCallback(
    (updates: { id: string; start_date?: string; target_date?: string }[]) =>
      (updateIssueDates ? updateIssueDates(projectId, updates) : Promise.resolve()).catch(() => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("toast.error"),
          message: "Error while updating work item dates, Please try again Later",
        });
      }),
    [projectId, t, updateIssueDates]
  );

  const quickAdd =
    isAllowed && !isCompletedCycle ? (
      <QuickAddIssueRoot
        layout={EIssueLayoutTypes.GANTT}
        QuickAddButton={GanttQuickAddIssueButton}
        containerClassName="sticky bottom-0 z-[1]"
        prePopulatedData={{
          start_date: renderFormattedPayloadDate(today),
          target_date: renderFormattedPayloadDate(targetDate),
        }}
        quickAddCallback={quickAddIssue}
        isEpic={isEpic}
      />
    ) : undefined;

  const renderSidebar = useCallback(
    (sidebarProps: Parameters<typeof IssueGanttSidebar>[0]) => (
      <IssueGanttSidebar {...sidebarProps} showAllBlocks isEpic={isEpic} isPaginating={readState.isFetchingNextPage} />
    ),
    [isEpic, readState.isFetchingNextPage]
  );

  return (
    <IssueLayoutHOC readState={readState} layout={EIssueLayoutTypes.GANTT}>
      <TimeLineTypeContext.Provider value={GANTT_TIMELINE_TYPE.ISSUE}>
        <div className="h-full w-full">
          <GanttChartRoot
            border={false}
            title={isEpic ? t("epic.label", { count: 2 }) : t("issue.label", { count: 2 })}
            loaderTitle={isEpic ? t("epic.label", { count: 2 }) : t("issue.label", { count: 2 })}
            blockIds={issuesIds}
            blockItems={issuesIds.flatMap((id) => (issueMap[id] ? [issueMap[id]] : []))}
            blockUpdateHandler={(issue, data) => {
              void updateIssueBlockStructure(issue, data);
            }}
            blockToRender={(data: TIssue) => <IssueGanttBlock issue={data} isEpic={isEpic} />}
            sidebarToRender={renderSidebar}
            enableBlockLeftResize={isAllowed}
            enableBlockRightResize={isAllowed}
            enableBlockMove={isAllowed}
            enableReorder={appliedDisplayFilters?.order_by === "sort_order" && isAllowed}
            enableAddBlock={isAllowed}
            enableSelection={isBulkOperationsEnabled && isAllowed}
            quickAdd={quickAdd}
            loadMoreBlocks={loadMoreIssues}
            canLoadMoreBlocks={nextPageResults}
            updateBlockDates={updateBlockDates}
            showAllBlocks
            enableDependency
            isEpic={isEpic}
          />
        </div>
      </TimeLineTypeContext.Provider>
    </IssueLayoutHOC>
  );
}
