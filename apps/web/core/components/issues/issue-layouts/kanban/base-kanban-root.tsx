import { showEmptyDiscoveryGroups } from "@/helpers/issue-list/grouping-axis";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useIssueLayoutQuickActions } from "../issue-layout-quick-actions";

import { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";
import { useIssueListPaginationOptions } from "@/hooks/use-issue-list-pagination-options";
import { useParams } from "@tanstack/react-router";
import type { FC } from "react";
import { useCallback, useEffect, useRef } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";

import { EUserPermissions } from "@plane/constants";
import { EIssueLayoutTypes } from "@plane/types";
import type { TIssue } from "@plane/types";
//hooks
import { KanbanDragStateProvider } from "@/providers/issue-layout-ui-state";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useIssueLayoutActions } from "@/hooks/use-issue-layout-actions";
import { useIssueKanbanCollapse } from "@/hooks/use-issue-kanban-collapse";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useGroupIssuesDragNDrop } from "@/hooks/use-group-dragndrop";
import { useIssueListType } from "@/hooks/use-issue-list-type";

// components
import { KanbanDragDelete } from "./drag-delete";
import { IssueLayoutHOC } from "../issue-layout-HOC";
import type { IQuickActionProps } from "../list/list-view-types";
import { KanBan } from "./default";
import { KanBanSwimLanes } from "./swimlanes";
// helpers
import { projectGroupedIssueIdsByDisplayOrder } from "../issue-display-ordering";
import { issueLayoutCapabilities, isIssuePropertyEditable } from "../utils";
import { IssueListViewProvider } from "@/providers/issue-list-view";

export interface IBaseKanBanLayout {
  QuickActions: FC<IQuickActionProps>;
  addIssuesToView?: (issueIds: string[]) => Promise<void | TIssue>;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
  isCompletedCycle?: boolean;
  viewId?: string | undefined;
  isEpic?: boolean;
}

export function BaseKanBanRoot(props: IBaseKanBanLayout) {
  return (
    <KanbanDragStateProvider>
      <BaseKanBanRootContent {...props} />
    </KanbanDragStateProvider>
  );
}

function BaseKanBanRootContent(props: IBaseKanBanLayout) {
  const {
    QuickActions,
    addIssuesToView,
    canEditPropertiesBasedOnProject,
    isCompletedCycle = false,
    isEpic = false,
  } = props;
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  // store hooks
  const storeType = useIssueListType();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { issueFilters: displayPreferences, requestInput, actions } = useIssueLayoutActions(storeType);
  const { quickAddIssue, updateIssue, removeIssue, updateFilters } = actions;
  const { displayFilters, displayProperties, kanbanFilters } = displayPreferences ?? {};
  const { filters: collapsedGroups, toggle: toggleCollapsedGroup } = useIssueKanbanCollapse(
    storeType,
    workspaceSlug,
    projectId,
    kanbanFilters,
    updateFilters
  );

  const sub_group_by = displayFilters?.sub_group_by;
  const group_by = displayFilters?.group_by;

  const orderBy = displayFilters?.order_by;

  const paginationOptions = useIssueListPaginationOptions(storeType, displayFilters, EIssueLayoutTypes.KANBAN);
  const readState = useIssueListPresentation(storeType, paginationOptions, displayFilters);
  const { fetchNextIssues } = readState;
  const issueMap = readState.issuesById;
  const { labelsById, estimateProjection, stateProjection, parentProjection } = readState;

  const fetchMoreIssues = useCallback(
    (groupId?: string, subgroupId?: string) => void fetchNextIssues(groupId, subgroupId),
    [fetchNextIssues]
  );

  const groupedIssueIds = readState.groupedIssueIds;
  const visibleGroupedIssueIds = projectGroupedIssueIdsByDisplayOrder(
    groupedIssueIds,
    (issueId) => issueMap[issueId],
    membersById,
    orderBy,
    labelsById,
    estimateProjection,
    stateProjection,
    parentProjection
  );

  const showEmptyGroup = displayFilters?.show_empty_groups ?? true;

  const KanBanView = sub_group_by ? KanBanSwimLanes : KanBan;

  const { enableQuickAdd, enableIssueCreation, enableInlineEditing } = issueLayoutCapabilities(
    "kanban",
    storeType,
    requestInput
  );

  const scrollableContainerRef = useRef<HTMLDivElement | null>(null);

  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const handleOnDrop = useGroupIssuesDragNDrop(
    storeType,
    orderBy,
    group_by,
    sub_group_by,
    visibleGroupedIssueIds,
    issueMap,
    groupedIssueIds
  );

  const canEditProperties = useCallback(
    (targetProjectId: string | undefined) =>
      isIssuePropertyEditable(targetProjectId, canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed),
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed]
  );

  // Enable Auto Scroll for Main Kanban
  useEffect(() => {
    const element = scrollableContainerRef.current;

    if (!element) return undefined;

    return combine(
      autoScrollForElements({
        element,
      })
    );
  }, []);

  const renderQuickActions = useIssueLayoutQuickActions({
    QuickActions,
    actions,
    canEditProperties,
    isCompletedCycle,
  });

  const handleCollapsedGroups = toggleCollapsedGroup;

  return (
    <IssueListViewProvider readState={readState}>
      <>
        <KanbanDragDelete issuesById={issueMap} removeIssue={removeIssue} isEpic={isEpic} />
        <IssueLayoutHOC
          allowEmptyGroups={showEmptyDiscoveryGroups(group_by, showEmptyGroup)}
          readState={readState}
          layout={EIssueLayoutTypes.KANBAN}
        >
          <div
            className={`horizontal-scrollbar relative flex scrollbar-lg h-full w-full bg-surface-2 ${sub_group_by ? "vertical-scrollbar overflow-y-auto" : "overflow-x-auto overflow-y-hidden"}`}
            ref={scrollableContainerRef}
          >
            <div className="relative h-full w-max min-w-full bg-surface-2">
              <div className="h-full w-max">
                <KanBanView
                  issuesMap={issueMap}
                  groupedIssueIds={visibleGroupedIssueIds ?? {}}
                  getGroupIssueCount={readState.getGroupIssueCount}
                  displayProperties={displayProperties}
                  sub_group_by={sub_group_by}
                  group_by={group_by}
                  orderBy={orderBy}
                  updateIssue={updateIssue}
                  quickActions={renderQuickActions}
                  handleCollapsedGroups={handleCollapsedGroups}
                  collapsedGroups={collapsedGroups}
                  enableQuickIssueCreate={enableQuickAdd}
                  showEmptyGroup={showEmptyGroup}
                  quickAddCallback={quickAddIssue}
                  disableIssueCreation={!enableIssueCreation || !isEditingAllowed || isCompletedCycle}
                  canEditProperties={canEditProperties}
                  addIssuesToView={addIssuesToView}
                  scrollableContainerRef={scrollableContainerRef}
                  handleOnDrop={handleOnDrop}
                  loadMoreIssues={fetchMoreIssues}
                  isEpic={isEpic}
                />
              </div>
            </div>
          </div>
        </IssueLayoutHOC>
      </>
    </IssueListViewProvider>
  );
}
