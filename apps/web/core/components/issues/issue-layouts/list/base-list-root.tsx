import { showEmptyDiscoveryGroups } from "@/helpers/issue-list/grouping-axis";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useIssueLayoutQuickActions } from "../issue-layout-quick-actions";

import { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";
import { useIssueListPaginationOptions } from "@/hooks/use-issue-list-pagination-options";
import { IssueListViewProvider } from "@/providers/issue-list-view";
import { useParams } from "@tanstack/react-router";
import type { FC } from "react";
import { useCallback } from "react";

// plane constants
import { EUserPermissions } from "@plane/constants";
// types
import type { GroupByColumnTypes, TIssue } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useIssueLayoutActions } from "@/hooks/use-issue-layout-actions";
import { useIssueKanbanCollapse } from "@/hooks/use-issue-kanban-collapse";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
// hooks
import { useGroupIssuesDragNDrop } from "@/hooks/use-group-dragndrop";
import { useIssueListType } from "@/hooks/use-issue-list-type";

import { projectGroupedIssueIdsByDisplayOrder } from "../issue-display-ordering";
import { isSubscribedProfileRequest } from "../utils";
// components
import { IssueLayoutHOC } from "../issue-layout-HOC";
import { List } from "./default";
// types
import type { IQuickActionProps } from "./list-view-types";

type ListStoreType =
  | EIssuesStoreType.PROJECT
  | EIssuesStoreType.MODULE
  | EIssuesStoreType.CYCLE
  | EIssuesStoreType.PROJECT_VIEW
  | EIssuesStoreType.PROFILE
  | EIssuesStoreType.ARCHIVED
  | EIssuesStoreType.TEAM
  | EIssuesStoreType.TEAM_VIEW
  | EIssuesStoreType.EPIC;

interface IBaseListRoot {
  QuickActions: FC<IQuickActionProps>;
  addIssuesToView?: (issueIds: string[]) => Promise<void | TIssue>;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
  viewId?: string | undefined;
  isCompletedCycle?: boolean;
  isEpic?: boolean;
}
export function BaseListRoot(props: IBaseListRoot) {
  const {
    QuickActions,
    addIssuesToView,
    canEditPropertiesBasedOnProject,
    isCompletedCycle = false,
    isEpic = false,
  } = props;
  // router
  const storeType = useIssueListType() as ListStoreType;
  const { issueFilters: displayPreferences, requestInput, actions } = useIssueLayoutActions(storeType);
  const { quickAddIssue, updateIssue, updateFilters } = actions;
  const displayFilters = displayPreferences?.displayFilters;
  const displayProperties = displayPreferences?.displayProperties;
  const orderBy = displayFilters?.order_by || undefined;

  const group_by = (displayFilters?.group_by || null) as GroupByColumnTypes | null;
  const showEmptyGroup = displayFilters?.show_empty_groups ?? false;

  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { filters: collapsedGroups, toggle: toggleCollapsedGroup } = useIssueKanbanCollapse(
    storeType,
    workspaceSlug,
    projectId,
    displayPreferences?.kanbanFilters,
    updateFilters
  );

  const paginationOptions = useIssueListPaginationOptions(storeType, displayFilters, EIssueLayoutTypes.LIST);
  const readState = useIssueListPresentation(storeType, paginationOptions, displayFilters);
  const { fetchNextIssues } = readState;
  const { labelsById, estimateProjection, stateProjection, parentProjection } = readState;

  const issueMap = readState.issuesById;
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
  // auth
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  const enableQuickAdd = storeType !== EIssuesStoreType.PROFILE && storeType !== EIssuesStoreType.ARCHIVED;
  const isSubscribedProfile = isSubscribedProfileRequest(requestInput);
  const enableIssueCreation = storeType !== EIssuesStoreType.ARCHIVED && !isSubscribedProfile;
  const enableInlineEditing = storeType !== EIssuesStoreType.ARCHIVED;

  const canEditProperties = useCallback(
    (issueProjectId: string | undefined) => {
      const isEditingAllowedBasedOnProject =
        canEditPropertiesBasedOnProject && issueProjectId
          ? canEditPropertiesBasedOnProject(issueProjectId)
          : isEditingAllowed;

      return !!enableInlineEditing && isEditingAllowedBasedOnProject;
    },
    [canEditPropertiesBasedOnProject, enableInlineEditing, isEditingAllowed]
  );

  const handleOnDrop = useGroupIssuesDragNDrop(
    storeType,
    orderBy,
    group_by,
    undefined,
    visibleGroupedIssueIds,
    issueMap,
    groupedIssueIds
  );

  const renderQuickActions = useIssueLayoutQuickActions({
    QuickActions,
    actions,
    canEditProperties,
    isCompletedCycle,
  });

  const loadMoreIssues = useCallback((groupId?: string) => void fetchNextIssues(groupId), [fetchNextIssues]);

  const handleCollapsedGroups = useCallback(
    (value: string) => toggleCollapsedGroup("group_by", value),
    [toggleCollapsedGroup]
  );

  return (
    <IssueListViewProvider readState={readState}>
      <IssueLayoutHOC
        allowEmptyGroups={showEmptyDiscoveryGroups(group_by, showEmptyGroup)}
        readState={readState}
        layout={EIssueLayoutTypes.LIST}
      >
        <div className={`relative size-full bg-surface-2`}>
          <List
            issuesMap={issueMap}
            displayProperties={displayProperties}
            group_by={group_by}
            orderBy={orderBy}
            updateIssue={updateIssue}
            quickActions={renderQuickActions}
            groupedIssueIds={visibleGroupedIssueIds ?? {}}
            loadMoreIssues={loadMoreIssues}
            showEmptyGroup={showEmptyGroup}
            quickAddCallback={quickAddIssue}
            enableIssueQuickAdd={!!enableQuickAdd}
            canEditProperties={canEditProperties}
            disableIssueCreation={!enableIssueCreation || !isEditingAllowed}
            addIssuesToView={addIssuesToView}
            isCompletedCycle={isCompletedCycle}
            handleOnDrop={handleOnDrop}
            handleCollapsedGroups={handleCollapsedGroups}
            collapsedGroups={collapsedGroups}
            isEpic={isEpic}
            getGroupIssueCount={readState.getGroupIssueCount}
            getPaginationData={readState.getPaginationData}
            getIssueLoader={readState.getIssueLoader}
          />
        </div>
      </IssueLayoutHOC>
    </IssueListViewProvider>
  );
}
