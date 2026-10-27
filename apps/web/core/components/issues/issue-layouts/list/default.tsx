/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { GroupingStatus, propertyGroupingIds } from "@/components/issues/issue-layouts/property-group-columns";
import { useGroupColumns } from "@/components/issues/issue-layouts/use-group-columns";
import { useParams } from "@tanstack/react-router";
import { useEffect, useRef } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
// plane constants
import { ALL_ISSUES } from "@plane/constants";
// types
import type {
  GroupByColumnTypes,
  TIssue,
  IIssueDisplayProperties,
  TIssueMap,
  TIssueGroupByOptions,
  TIssueOrderByOptions,
  IGroupByColumn,
  TIssueKanbanFilters,
  TPaginationData,
  TLoader,
} from "@plane/types";
// components
import { MultipleSelectGroup } from "@/components/core/multiple-select";
// hooks
import { useIssueListType } from "@/hooks/use-issue-list-type";
// plane web components
import { IssueBulkOperationsRoot } from "@/components/issues/bulk-operations";
// plane web hooks
import { useBulkOperationStatus } from "@/hooks/use-bulk-operation-status";
// utils
import type { GroupDropLocation } from "../utils";
import { isWorkspaceLevel } from "../utils";
import { getGroupedIssueIds } from "../issue-display-ordering";
import type { IssueListGroupedIds } from "@/queries/issue-lists";
import { ListGroup } from "./list-group";
import type { TRenderQuickActions } from "./list-view-types";

export interface IList {
  groupedIssueIds: IssueListGroupedIds;
  issuesMap: TIssueMap;
  group_by: TIssueGroupByOptions | null;
  orderBy: TIssueOrderByOptions | undefined;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  quickActions: TRenderQuickActions;
  displayProperties: IIssueDisplayProperties | undefined;
  enableIssueQuickAdd: boolean;
  showEmptyGroup?: boolean;
  canEditProperties: (projectId: string | undefined) => boolean;
  quickAddCallback?: (projectId: string | null | undefined, data: TIssue) => Promise<TIssue | undefined>;
  disableIssueCreation?: boolean;
  handleOnDrop: (source: GroupDropLocation, destination: GroupDropLocation) => Promise<void>;
  addIssuesToView?: (issueIds: string[]) => Promise<void | TIssue>;
  isCompletedCycle?: boolean;
  loadMoreIssues: (groupId?: string) => void;
  handleCollapsedGroups: (value: string) => void;
  collapsedGroups: TIssueKanbanFilters;
  isEpic?: boolean;
  getGroupIssueCount: (groupId?: string, subGroupId?: string, cumulative?: boolean) => number | undefined;
  getPaginationData: (groupId?: string, subGroupId?: string) => TPaginationData | undefined;
  getIssueLoader: (groupId?: string, subGroupId?: string) => TLoader;
}

export function List(props: IList) {
  const {
    groupedIssueIds,
    issuesMap,
    group_by,
    orderBy,
    updateIssue,
    quickActions,
    displayProperties,
    enableIssueQuickAdd,
    showEmptyGroup,
    canEditProperties,
    quickAddCallback,
    disableIssueCreation,
    handleOnDrop,
    addIssuesToView,
    isCompletedCycle = false,
    loadMoreIssues,
    handleCollapsedGroups,
    collapsedGroups,
    isEpic = false,
    getGroupIssueCount,
    getPaginationData,
    getIssueLoader,
  } = props;

  const storeType = useIssueListType();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // plane web hooks
  const isBulkOperationsEnabled = useBulkOperationStatus();

  const containerRef = useRef<HTMLDivElement | null>(null);

  const groups = useGroupColumns({
    workspaceSlug,
    projectId,
    groupBy: group_by as GroupByColumnTypes,
    observedIds: propertyGroupingIds(groupedIssueIds),
    includeNone: true,
    isWorkspaceLevel: isWorkspaceLevel(storeType),
    isEpic: isEpic,
  });

  // Enable Auto Scroll for Main Kanban
  useEffect(() => {
    const element = containerRef.current;

    if (!element) return;

    return combine(
      autoScrollForElements({
        element,
      })
    );
  }, [containerRef]);

  if (!groups) return <GroupingStatus groupBy={group_by} />;

  const getGroupIndex = (groupId: string | undefined) => groups.findIndex(({ id }) => id === groupId);

  const is_list = group_by === null;

  // create groupIds array and entities object for bulk ops
  const groupIds = groups.map((g) => g.id);
  const orderedGroups: Record<string, string[]> = {};
  groupIds.forEach((gID) => {
    orderedGroups[gID] = [];
  });
  let entities: Record<string, string[]> = {};

  if (is_list) {
    entities = Object.assign(orderedGroups, {
      [groupIds[0]]: getGroupedIssueIds(groupedIssueIds, ALL_ISSUES) ?? [],
    });
  } else {
    entities = { ...orderedGroups };
    for (const groupId of groupIds) {
      const issueIds = getGroupedIssueIds(groupedIssueIds, groupId);
      if (issueIds) entities[groupId] = issueIds;
    }
  }
  return (
    <div className="relative flex size-full flex-col">
      {groups && (
        <MultipleSelectGroup
          containerRef={containerRef}
          entities={entities}
          disabled={!isBulkOperationsEnabled || isEpic}
        >
          {(helpers) => (
            <>
              <div
                ref={containerRef}
                className="vertical-scrollbar relative scrollbar-lg size-full overflow-auto bg-surface-1"
              >
                {groups.map((group: IGroupByColumn) => (
                  <ListGroup
                    key={group.id}
                    groupIssueIds={getGroupedIssueIds(groupedIssueIds, group.id)}
                    issuesMap={issuesMap}
                    group_by={group_by}
                    group={group}
                    updateIssue={updateIssue}
                    quickActions={quickActions}
                    orderBy={orderBy}
                    getGroupIndex={getGroupIndex}
                    handleOnDrop={handleOnDrop}
                    displayProperties={displayProperties}
                    enableIssueQuickAdd={enableIssueQuickAdd}
                    showEmptyGroup={showEmptyGroup}
                    canEditProperties={canEditProperties}
                    quickAddCallback={quickAddCallback}
                    disableIssueCreation={disableIssueCreation}
                    addIssuesToView={addIssuesToView}
                    isCompletedCycle={isCompletedCycle}
                    loadMoreIssues={loadMoreIssues}
                    containerRef={containerRef}
                    selectionHelpers={helpers}
                    handleCollapsedGroups={handleCollapsedGroups}
                    collapsedGroups={collapsedGroups}
                    isEpic={isEpic}
                    getGroupIssueCount={getGroupIssueCount}
                    getPaginationData={getPaginationData}
                    getIssueLoader={getIssueLoader}
                  />
                ))}
              </div>

              <IssueBulkOperationsRoot selectionHelpers={helpers} />
            </>
          )}
        </MultipleSelectGroup>
      )}
    </div>
  );
}
