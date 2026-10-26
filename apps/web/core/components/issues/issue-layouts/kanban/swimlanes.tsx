/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TypeGroupingStatus } from "@/components/issues/issue-layouts/type-group-columns";
import { useGroupColumns } from "@/components/issues/issue-layouts/use-group-columns";
import { useParams } from "@tanstack/react-router";

// plane imports
import type { GroupByColumnTypes, IGroupByColumn } from "@plane/types";
import { Row } from "@plane/ui";
// hooks
import { useIssueListType } from "@/hooks/use-issue-list-type";
// plane web imports
import { useWorkFlowFDragNDrop } from "@/components/workflow";
// local imports
import { isWorkspaceLevel } from "../utils";
import { KanBan } from "./default";
import type { IKanBanSwimlaneProps } from "./default";
import { HeaderGroupByCard } from "./headers/group-by-card";
import { HeaderSubGroupByCard } from "./headers/sub-group-by-card";

type ISubGroupSwimlaneHeader = Pick<
  IKanBanSwimlaneProps,
  | "collapsedGroups"
  | "getGroupIssueCount"
  | "group_by"
  | "handleCollapsedGroups"
  | "isEpic"
  | "showEmptyGroup"
  | "sub_group_by"
> & {
  list: IGroupByColumn[];
};

const visibilitySubGroupByGroupCount = (subGroupIssueCount: number, showEmptyGroup: boolean): boolean => {
  let subGroupHeaderVisibility = true;

  if (showEmptyGroup) subGroupHeaderVisibility = true;
  else {
    if (subGroupIssueCount > 0) subGroupHeaderVisibility = true;
    else subGroupHeaderVisibility = false;
  }

  return subGroupHeaderVisibility;
};

const SubGroupSwimlaneHeader = function SubGroupSwimlaneHeader({
  collapsedGroups,
  getGroupIssueCount,
  group_by,
  handleCollapsedGroups,
  isEpic = false,
  list,
  showEmptyGroup,
  sub_group_by,
}: ISubGroupSwimlaneHeader) {
  const { getIsWorkflowWorkItemCreationDisabled } = useWorkFlowFDragNDrop(group_by, sub_group_by);

  return (
    <div className="relative flex h-max min-h-full w-full items-center gap-4">
      {list &&
        list.length > 0 &&
        list.map((_list: IGroupByColumn) => {
          const groupCount = getGroupIssueCount(_list?.id, undefined, false) ?? 0;

          const subGroupByVisibilityToggle = visibilitySubGroupByGroupCount(groupCount, showEmptyGroup);

          if (subGroupByVisibilityToggle === false) return <></>;

          return (
            <div key={`${sub_group_by}_${_list.id}`} className="flex w-[350px] flex-shrink-0 flex-col">
              <HeaderGroupByCard
                sub_group_by={sub_group_by}
                group_by={group_by}
                column_id={_list.id}
                icon={_list.icon}
                title={_list.name}
                count={groupCount}
                collapsedGroups={collapsedGroups}
                handleCollapsedGroups={handleCollapsedGroups}
                issuePayload={_list.payload}
                disableIssueCreation={getIsWorkflowWorkItemCreationDisabled(_list.id)}
                isEpic={isEpic}
              />
            </div>
          );
        })}
    </div>
  );
};

type ISubGroupSwimlane = IKanBanSwimlaneProps & Pick<ISubGroupSwimlaneHeader, "list">;

const SubGroupSwimlane = function SubGroupSwimlane(props: ISubGroupSwimlane) {
  const {
    addIssuesToView,
    canEditProperties,
    collapsedGroups,
    disableIssueCreation,
    displayProperties,
    enableQuickIssueCreate,
    getGroupIssueCount,
    group_by,
    groupedIssueIds,
    handleCollapsedGroups,
    handleOnDrop,
    isEpic = false,
    issuesMap,
    list,
    loadMoreIssues,
    orderBy,
    quickActions,
    quickAddCallback,
    scrollableContainerRef,
    showEmptyGroup,
    sub_group_by,
    updateIssue,
  } = props;

  const visibilitySubGroupBy = (
    _list: IGroupByColumn,
    subGroupCount: number
  ): { showGroup: boolean; showIssues: boolean } => {
    const subGroupVisibility = {
      showGroup: true,
      showIssues: true,
    };
    if (showEmptyGroup) subGroupVisibility.showGroup = true;
    else {
      if (subGroupCount > 0) subGroupVisibility.showGroup = true;
      else subGroupVisibility.showGroup = false;
    }
    if (collapsedGroups?.sub_group_by.includes(_list.id)) subGroupVisibility.showIssues = false;
    return subGroupVisibility;
  };

  return (
    <div className="relative h-max min-h-full w-full">
      {list &&
        list.length > 0 &&
        list.map((_list: IGroupByColumn, subGroupIndex) => {
          const issueCount = getGroupIssueCount(undefined, _list.id, true) ?? 0;
          const subGroupByVisibilityToggle = visibilitySubGroupBy(_list, issueCount);
          if (subGroupByVisibilityToggle.showGroup === false) return <></>;
          return (
            <div key={_list.id} className="flex flex-shrink-0 flex-col">
              <div className="sticky top-[50px] z-[3] flex w-full items-center border-y-[0.5px] border-subtle bg-layer-1 py-1">
                <Row className="sticky left-0 flex-shrink-0">
                  <HeaderSubGroupByCard
                    column_id={_list.id}
                    icon={_list.icon}
                    title={_list.name}
                    count={issueCount}
                    collapsedGroups={collapsedGroups}
                    handleCollapsedGroups={handleCollapsedGroups}
                    sub_group_by={sub_group_by}
                  />
                </Row>
              </div>

              {subGroupByVisibilityToggle.showIssues && (
                <div className="relative">
                  <KanBan
                    issuesMap={issuesMap}
                    groupedIssueIds={groupedIssueIds}
                    getGroupIssueCount={getGroupIssueCount}
                    displayProperties={displayProperties}
                    sub_group_by={sub_group_by}
                    group_by={group_by}
                    sub_group_id={_list.id}
                    subGroupIndex={subGroupIndex}
                    updateIssue={updateIssue}
                    quickActions={quickActions}
                    collapsedGroups={collapsedGroups}
                    handleCollapsedGroups={handleCollapsedGroups}
                    showEmptyGroup={showEmptyGroup}
                    enableQuickIssueCreate={enableQuickIssueCreate}
                    disableIssueCreation={disableIssueCreation}
                    canEditProperties={canEditProperties}
                    addIssuesToView={addIssuesToView}
                    quickAddCallback={quickAddCallback}
                    scrollableContainerRef={scrollableContainerRef}
                    loadMoreIssues={loadMoreIssues}
                    handleOnDrop={handleOnDrop}
                    orderBy={orderBy}
                    isDropDisabled={_list.isDropDisabled}
                    dropErrorMessage={_list.dropErrorMessage}
                    isEpic={isEpic}
                  />
                </div>
              )}
            </div>
          );
        })}
    </div>
  );
};

export type IKanBanSwimLanes = IKanBanSwimlaneProps;

export const KanBanSwimLanes = function KanBanSwimLanes(props: IKanBanSwimLanes) {
  const {
    issuesMap,
    groupedIssueIds,
    getGroupIssueCount,
    displayProperties,
    sub_group_by,
    group_by,
    orderBy,
    updateIssue,
    quickActions,
    collapsedGroups,
    handleCollapsedGroups,
    loadMoreIssues,
    showEmptyGroup,
    handleOnDrop,
    disableIssueCreation,
    enableQuickIssueCreate,
    canEditProperties,
    addIssuesToView,
    quickAddCallback,
    scrollableContainerRef,
    isEpic = false,
  } = props;
  // store hooks
  const storeType = useIssueListType();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // derived values
  const groupByList = useGroupColumns({
    workspaceSlug,
    projectId,
    groupBy: group_by as GroupByColumnTypes,
    includeNone: true,
    isWorkspaceLevel: isWorkspaceLevel(storeType),
    isEpic: isEpic,
  });
  const subGroupByList = useGroupColumns({
    workspaceSlug,
    projectId,
    groupBy: sub_group_by as GroupByColumnTypes,
    includeNone: true,
    isWorkspaceLevel: isWorkspaceLevel(storeType),
    isEpic: isEpic,
  });

  if (!groupByList || !subGroupByList)
    return <TypeGroupingStatus active={group_by === "type_id" || sub_group_by === "type_id"} />;

  return (
    <div className="relative">
      <Row className="sticky top-0 z-[4] h-[50px] bg-surface-2">
        <SubGroupSwimlaneHeader
          getGroupIssueCount={getGroupIssueCount}
          group_by={group_by}
          sub_group_by={sub_group_by}
          collapsedGroups={collapsedGroups}
          handleCollapsedGroups={handleCollapsedGroups}
          list={groupByList}
          showEmptyGroup={showEmptyGroup}
          isEpic={isEpic}
        />
      </Row>

      {sub_group_by && (
        <SubGroupSwimlane
          issuesMap={issuesMap}
          list={subGroupByList}
          groupedIssueIds={groupedIssueIds}
          getGroupIssueCount={getGroupIssueCount}
          displayProperties={displayProperties}
          group_by={group_by}
          sub_group_by={sub_group_by}
          orderBy={orderBy}
          updateIssue={updateIssue}
          quickActions={quickActions}
          collapsedGroups={collapsedGroups}
          handleCollapsedGroups={handleCollapsedGroups}
          loadMoreIssues={loadMoreIssues}
          showEmptyGroup={showEmptyGroup}
          handleOnDrop={handleOnDrop}
          disableIssueCreation={disableIssueCreation}
          enableQuickIssueCreate={enableQuickIssueCreate}
          addIssuesToView={addIssuesToView}
          canEditProperties={canEditProperties}
          quickAddCallback={quickAddCallback}
          scrollableContainerRef={scrollableContainerRef}
          isEpic={isEpic}
        />
      )}
    </div>
  );
};
