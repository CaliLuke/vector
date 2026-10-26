/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TypeGroupingStatus } from "@/components/issues/issue-layouts/type-group-columns";
import { useGroupColumns } from "@/components/issues/issue-layouts/use-group-columns";
import { useParams } from "@tanstack/react-router";
import type { MutableRefObject } from "react";

import type {
  GroupByColumnTypes,
  IGroupByColumn,
  TIssue,
  IIssueDisplayProperties,
  IIssueMap,
  TIssueKanbanFilters,
  TIssueGroupByOptions,
  TIssueOrderByOptions,
} from "@plane/types";
// constants
import { ContentWrapper } from "@plane/ui";
// components
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { KanbanColumnLoader } from "@/components/ui/loader/layouts/kanban-layout-loader";
// hooks
import { canDragDropForGrouping } from "@/providers/issue-layout-ui-state";
import { useIssueListType } from "@/hooks/use-issue-list-type";
// types
// parent components
import { useWorkFlowFDragNDrop } from "@/components/workflow";
import type { TRenderQuickActions } from "../list/list-view-types";
import type { GroupDropLocation } from "../utils";
import { isWorkspaceLevel, getApproximateCardHeight } from "../utils";
// components
import { HeaderGroupByCard } from "./headers/group-by-card";
import { KanbanGroup } from "./kanban-group";
import { getGroupedIssueIds } from "../issue-display-ordering";
import type { IssueListGroupedIds } from "@/queries/issue-lists";

export interface IKanBan {
  issuesMap: IIssueMap;
  groupedIssueIds: IssueListGroupedIds;
  getGroupIssueCount: (
    groupId: string | undefined,
    subGroupId: string | undefined,
    isSubGroupCumulative: boolean
  ) => number | undefined;
  displayProperties: IIssueDisplayProperties | undefined;
  sub_group_by: TIssueGroupByOptions | undefined;
  group_by: TIssueGroupByOptions | undefined;
  orderBy: TIssueOrderByOptions | undefined;
  isDropDisabled?: boolean;
  dropErrorMessage?: string | undefined;
  sub_group_id?: string;
  sub_group_index?: number;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  quickActions: TRenderQuickActions;
  collapsedGroups: TIssueKanbanFilters;
  handleCollapsedGroups: (toggle: "group_by" | "sub_group_by", value: string) => void;
  loadMoreIssues: (groupId?: string, subGroupId?: string) => void;
  enableQuickIssueCreate?: boolean;
  quickAddCallback?: (projectId: string | null | undefined, data: TIssue) => Promise<TIssue | undefined>;
  disableIssueCreation?: boolean;
  addIssuesToView?: (issueIds: string[]) => Promise<void | TIssue>;
  canEditProperties: (projectId: string | undefined) => boolean;
  scrollableContainerRef?: MutableRefObject<HTMLDivElement | null>;
  handleOnDrop: (source: GroupDropLocation, destination: GroupDropLocation) => Promise<void>;
  showEmptyGroup?: boolean;
  subGroupIndex?: number;
  isEpic?: boolean;
}

export type IKanBanSwimlaneProps = Omit<
  IKanBan,
  | "dropErrorMessage"
  | "enableQuickIssueCreate"
  | "isDropDisabled"
  | "showEmptyGroup"
  | "sub_group_id"
  | "sub_group_index"
> &
  Required<Pick<IKanBan, "enableQuickIssueCreate" | "showEmptyGroup">>;

export const KanBan = function KanBan(props: IKanBan) {
  const {
    issuesMap,
    groupedIssueIds,
    getGroupIssueCount,
    displayProperties,
    sub_group_by,
    group_by,
    sub_group_id = "null",
    updateIssue,
    quickActions,
    collapsedGroups,
    handleCollapsedGroups,
    enableQuickIssueCreate,
    quickAddCallback,
    loadMoreIssues,
    disableIssueCreation,
    addIssuesToView,
    canEditProperties,
    scrollableContainerRef,
    handleOnDrop,
    showEmptyGroup = true,
    orderBy,
    isDropDisabled,
    dropErrorMessage,
    subGroupIndex = 0,
    isEpic = false,
  } = props;
  // i18n
  // store hooks
  const storeType = useIssueListType();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // derived values
  const isDragDisabled = !canDragDropForGrouping(group_by, sub_group_by);

  const { getIsWorkflowWorkItemCreationDisabled } = useWorkFlowFDragNDrop(group_by, sub_group_by);

  const list = useGroupColumns({
    workspaceSlug,
    projectId,
    groupBy: group_by as GroupByColumnTypes,
    includeNone: true,
    isWorkspaceLevel: isWorkspaceLevel(storeType),
    isEpic: isEpic,
  });

  if (!list) return <TypeGroupingStatus active={group_by === "type_id"} />;

  const visibilityGroupBy = (_list: IGroupByColumn): { showGroup: boolean; showIssues: boolean } => {
    if (sub_group_by) {
      const groupVisibility = {
        showGroup: true,
        showIssues: true,
      };
      if (!showEmptyGroup) {
        groupVisibility.showGroup = (getGroupIssueCount(_list.id, undefined, false) ?? 0) > 0;
      }
      return groupVisibility;
    } else {
      const groupVisibility = {
        showGroup: true,
        showIssues: true,
      };
      if (!showEmptyGroup) {
        if ((getGroupIssueCount(_list.id, undefined, false) ?? 0) > 0) groupVisibility.showGroup = true;
        else groupVisibility.showGroup = false;
      }
      if (collapsedGroups?.group_by.includes(_list.id)) groupVisibility.showIssues = false;
      return groupVisibility;
    }
  };

  const isGroupByCreatedBy = group_by === "created_by";
  const approximateCardHeight = getApproximateCardHeight(displayProperties);
  const isSubGroup = !!sub_group_id && sub_group_id !== "null";

  return (
    <ContentWrapper className={`relative flex-row gap-4 !pt-2 !pb-0`}>
      {list &&
        list.length > 0 &&
        list.map((subList: IGroupByColumn, groupIndex) => {
          const groupByVisibilityToggle = visibilityGroupBy(subList);

          if (groupByVisibilityToggle.showGroup === false) return <></>;

          const issueIds = getGroupedIssueIds(groupedIssueIds, subList.id, isSubGroup ? sub_group_id : undefined) ?? [];
          const issueLength = issueIds?.length;
          const groupHeight = issueLength * approximateCardHeight;

          return (
            <div
              key={subList.id}
              className={`group relative flex flex-shrink-0 flex-col ${
                groupByVisibilityToggle.showIssues ? `w-[350px]` : ``
              } `}
            >
              {sub_group_by === null && (
                <div className="sticky top-0 z-[2] w-full flex-shrink-0 bg-surface-2 py-1">
                  <HeaderGroupByCard
                    sub_group_by={sub_group_by}
                    group_by={group_by}
                    column_id={subList.id}
                    icon={subList.icon}
                    title={subList.name}
                    count={getGroupIssueCount(subList.id, undefined, false) ?? 0}
                    issuePayload={subList.payload}
                    disableIssueCreation={
                      disableIssueCreation ||
                      isGroupByCreatedBy ||
                      getIsWorkflowWorkItemCreationDisabled(subList.id, sub_group_id)
                    }
                    addIssuesToView={addIssuesToView}
                    collapsedGroups={collapsedGroups}
                    handleCollapsedGroups={handleCollapsedGroups}
                    isEpic={isEpic}
                  />
                </div>
              )}

              {groupByVisibilityToggle.showIssues && (
                <RenderIfVisible
                  verticalOffset={100}
                  horizontalOffset={100}
                  root={scrollableContainerRef}
                  classNames="h-full min-h-[120px]"
                  defaultHeight={`${groupHeight}px`}
                  placeholderChildren={
                    <KanbanColumnLoader
                      ignoreHeader
                      cardHeight={approximateCardHeight}
                      cardsInColumn={issueLength !== undefined && issueLength < 3 ? issueLength : 3}
                      shouldAnimate={false}
                    />
                  }
                  defaultValue={groupIndex < 5 && subGroupIndex < 2}
                  useIdleTime
                >
                  <KanbanGroup
                    groupId={subList.id}
                    issuesMap={issuesMap}
                    groupedIssueIds={groupedIssueIds}
                    displayProperties={displayProperties}
                    sub_group_by={sub_group_by}
                    group_by={group_by}
                    orderBy={orderBy}
                    sub_group_id={sub_group_id}
                    isDragDisabled={isDragDisabled}
                    isDropDisabled={!!subList.isDropDisabled || !!isDropDisabled}
                    dropErrorMessage={subList.dropErrorMessage ?? dropErrorMessage}
                    updateIssue={updateIssue}
                    quickActions={quickActions}
                    enableQuickIssueCreate={enableQuickIssueCreate}
                    quickAddCallback={quickAddCallback}
                    disableIssueCreation={disableIssueCreation}
                    canEditProperties={canEditProperties}
                    scrollableContainerRef={scrollableContainerRef}
                    loadMoreIssues={loadMoreIssues}
                    handleOnDrop={handleOnDrop}
                    isEpic={isEpic}
                  />
                </RenderIfVisible>
              )}
            </div>
          );
        })}
    </ContentWrapper>
  );
};
