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
import { useCallback, useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";

import { EUserPermissions } from "@plane/constants";
import { EIssuesStoreType } from "@plane/types";
import { EIssueLayoutTypes } from "@plane/types";
import type { TIssue } from "@plane/types";
//hooks
import { KanbanDragStateProvider, useKanbanDragState } from "@/providers/issue-layout-ui-state";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useIssueLayoutActions } from "@/hooks/use-issue-layout-actions";
import { useIssueKanbanCollapse } from "@/hooks/use-issue-kanban-collapse";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useGroupIssuesDragNDrop } from "@/hooks/use-group-dragndrop";
import { useIssueListType } from "@/hooks/use-issue-list-type";

// components
import { DeleteIssueModal } from "../../delete-issue-modal";
import { IssueLayoutHOC } from "../issue-layout-HOC";
import type { IQuickActionProps } from "../list/list-view-types";
import { KanBan } from "./default";
import { KanBanSwimLanes } from "./swimlanes";
// helpers
import { projectGroupedIssueIdsByDisplayOrder } from "../issue-display-ordering";
import { getSourceFromDropPayload, isSubscribedProfileRequest } from "../utils";
import { IssueListViewProvider } from "@/providers/issue-list-view";

export type KanbanStoreType =
  | EIssuesStoreType.PROJECT
  | EIssuesStoreType.MODULE
  | EIssuesStoreType.CYCLE
  | EIssuesStoreType.PROJECT_VIEW
  | EIssuesStoreType.PROFILE
  | EIssuesStoreType.TEAM
  | EIssuesStoreType.TEAM_VIEW
  | EIssuesStoreType.EPIC;

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
  const storeType = useIssueListType() as KanbanStoreType;
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { issueFilters: displayPreferences, requestInput, actions } = useIssueLayoutActions(storeType);
  const { quickAddIssue, updateIssue, removeIssue, updateFilters } = actions;
  const { filters: collapsedGroups, toggle: toggleCollapsedGroup } = useIssueKanbanCollapse(
    storeType,
    workspaceSlug,
    projectId,
    displayPreferences?.kanbanFilters,
    updateFilters
  );

  const deleteAreaRef = useRef<HTMLDivElement | null>(null);
  const [isDragOverDelete, setIsDragOverDelete] = useState(false);
  const { isDragging } = useKanbanDragState();

  const displayFilters = displayPreferences?.displayFilters;
  const displayProperties = displayPreferences?.displayProperties;

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

  const userDisplayFilters = displayFilters || null;

  const KanBanView = sub_group_by ? KanBanSwimLanes : KanBan;

  const enableQuickAdd = storeType !== EIssuesStoreType.PROFILE;
  const isSubscribedProfile = isSubscribedProfileRequest(requestInput);
  const enableIssueCreation = !isSubscribedProfile;
  const enableInlineEditing = true;

  const scrollableContainerRef = useRef<HTMLDivElement | null>(null);

  // states
  const [draggedIssueId, setDraggedIssueId] = useState<string | undefined>(undefined);
  const [deleteIssueModal, setDeleteIssueModal] = useState(false);

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
    (targetProjectId: string | undefined) => {
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && targetProjectId
          ? canEditPropertiesBasedOnProject(targetProjectId)
          : isEditingAllowed;

      return enableInlineEditing && isEditingAllowedBasedOnProject;
    },
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

  // Make the Issue Delete Box a Drop Target
  useEffect(() => {
    const element = deleteAreaRef.current;

    if (!element) return undefined;

    return combine(
      dropTargetForElements({
        element,
        getData: () => ({ columnId: "issue-trash-box", groupId: "issue-trash-box", type: "DELETE" }),
        onDragEnter: () => {
          setIsDragOverDelete(true);
        },
        onDragLeave: () => {
          setIsDragOverDelete(false);
        },
        onDrop: (payload) => {
          setIsDragOverDelete(false);
          const source = getSourceFromDropPayload(payload);

          if (!source) return;

          setDraggedIssueId(source.id);
          setDeleteIssueModal(true);
        },
      })
    );
  }, [setIsDragOverDelete, setDraggedIssueId, setDeleteIssueModal]);

  const renderQuickActions = useIssueLayoutQuickActions({
    QuickActions,
    actions,
    canEditProperties,
    isCompletedCycle,
  });

  const handleDeleteIssue = async () => {
    const draggedIssue = issueMap[draggedIssueId ?? ""];

    if (!draggedIssueId || !draggedIssue) return;

    try {
      await removeIssue(draggedIssue.project_id, draggedIssueId);
      setDeleteIssueModal(false);
      setDraggedIssueId(undefined);
    } catch (_error) {
      setDeleteIssueModal(false);
      setDraggedIssueId(undefined);
    }
  };

  const handleCollapsedGroups = toggleCollapsedGroup;

  return (
    <IssueListViewProvider readState={readState}>
      <>
        <DeleteIssueModal
          data={issueMap[draggedIssueId ?? ""]}
          isOpen={deleteIssueModal}
          handleClose={() => setDeleteIssueModal(false)}
          onSubmit={handleDeleteIssue}
          isEpic={isEpic}
        />
        {/* drag and delete component */}
        <div
          className={`fixed left-1/2 -translate-x-1/2 ${
            isDragging ? "z-40" : ""
          } top-3 mx-3 flex w-72 items-center justify-center`}
          ref={deleteAreaRef}
        >
          <div
            className={`${
              isDragging ? `opacity-100` : `opacity-0`
            } flex w-full items-center justify-center rounded-sm border-2 border-danger-strong/20 bg-surface-1 px-3 py-5 text-11 font-medium text-danger-primary italic ${
              isDragOverDelete ? "bg-danger-primary blur-2xl" : ""
            } transition duration-300`}
          >
            Drop here to delete the work item.
          </div>
        </div>
        <IssueLayoutHOC readState={readState} layout={EIssueLayoutTypes.KANBAN}>
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
                  showEmptyGroup={userDisplayFilters?.show_empty_groups ?? true}
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
