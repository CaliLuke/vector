/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createContext, useContext, useMemo } from "react";
import type { ReactNode } from "react";
import type { TIssue, TPaginationData, TIssuesResponse } from "@plane/types";
import type { IssueListGroupedIds } from "@/queries/issue-lists";
import type { useIssueListQuery } from "@/hooks/use-issue-list-query";

export type IssueListViewData = Readonly<{
  parentGroups?: TIssuesResponse["parent_groups"];
  issuesById: Readonly<Record<string, TIssue>>;
  groupedIssueIds: IssueListGroupedIds;
  countForGroup: (groupId?: string, subGroupId?: string, cumulative?: boolean) => number | undefined;
  paginationForGroup: (groupId?: string, subGroupId?: string) => TPaginationData | undefined;
  isLoadingNextPage: (groupId?: string, subGroupId?: string) => boolean;
  loadNextPage: (groupId?: string, subGroupId?: string) => Promise<void>;
}>;

const IssueListViewContext = createContext<IssueListViewData | undefined>(undefined);

type IssueListReadState = Pick<
  ReturnType<typeof useIssueListQuery>,
  | "response"
  | "issuesById"
  | "groupedIssueIds"
  | "getGroupIssueCount"
  | "getPaginationData"
  | "getIssueLoader"
  | "fetchNextIssues"
>;

export function IssueListViewProvider({ readState, children }: { readState: IssueListReadState; children: ReactNode }) {
  const {
    response,
    issuesById,
    groupedIssueIds,
    getGroupIssueCount,
    getPaginationData,
    getIssueLoader,
    fetchNextIssues,
  } = readState;
  const value = useMemo<IssueListViewData>(
    () => ({
      parentGroups: response?.parent_groups,
      issuesById,
      groupedIssueIds: groupedIssueIds ?? {},
      countForGroup: getGroupIssueCount,
      paginationForGroup: getPaginationData,
      isLoadingNextPage: (groupId, subGroupId) => getIssueLoader(groupId, subGroupId) === "pagination",
      loadNextPage: fetchNextIssues,
    }),
    [response, issuesById, groupedIssueIds, getGroupIssueCount, getPaginationData, getIssueLoader, fetchNextIssues]
  );
  return <IssueListViewContext.Provider value={value}>{children}</IssueListViewContext.Provider>;
}

export function useIssueListView() {
  const value = useContext(IssueListViewContext);
  if (!value) throw new Error("Issue list presentation requires its mounted Query owner.");
  return value;
}

export function useOptionalIssueListView() {
  return useContext(IssueListViewContext);
}
