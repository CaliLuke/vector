/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import {
  hashKey,
  InfiniteQueryObserver,
  queryOptions,
  skipToken,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import type { TGroupedIssueCount, TIssue, TLoader, TPaginationData } from "@plane/types";
import type { QueryClient } from "@tanstack/react-query";
import { ALL_ISSUES } from "@plane/constants";
import { processIssueResponse } from "@/helpers/issue-list/response";
import { getGroupKey } from "@/helpers/issue-list/grouping";
import type { IssueListGroup, IssueListGroupedIds, IssueListRead, IssueListReadData } from "@/queries/issue-lists";
import { issueListGroupOptions, issueListGroups, issueListOptions } from "@/queries/issue-lists";
import { captureParentAccess, parentScopeReady } from "@/queries/cycle-module-scope";

function paginationGroupKey(group: IssueListGroup) {
  return JSON.stringify([group.groupId ?? null, group.subGroupId ?? null]);
}

function createPageObservers(client: QueryClient, options: ReturnType<typeof issueListGroupOptions>[]) {
  return options.map((groupOptions) => new InfiniteQueryObserver(client, groupOptions));
}

export function useIssueListQuery(read: IssueListRead | undefined) {
  const client = useQueryClient();
  const list = useQuery(
    read
      ? issueListOptions(read)
      : queryOptions<IssueListReadData>({ queryKey: ["issue-list", "unavailable"], queryFn: skipToken })
  );
  const isReadCurrent = Boolean(
    read &&
    parentScopeReady(read.scope) &&
    captureParentAccess(client, read.scope).isVerified() &&
    read.request.isReadCurrent()
  );
  const readData = isReadCurrent && list.isSuccess ? list.data : undefined;
  const groups = useMemo(() => (readData ? issueListGroups(readData.response) : []), [readData]);
  const groupOptions = useMemo(
    () => (read && readData ? groups.map((group) => issueListGroupOptions(read, readData, group)) : []),
    [groups, read, readData]
  );
  const groupObserverIdentity = hashKey(groupOptions.map((options) => options.queryKey));
  const [storedPageObservers, setStoredPageObservers] = useState(() => ({
    client,
    identity: groupObserverIdentity,
    observers: createPageObservers(client, groupOptions),
  }));
  let currentPageObservers = storedPageObservers;
  if (storedPageObservers.client !== client || storedPageObservers.identity !== groupObserverIdentity) {
    currentPageObservers = {
      client,
      identity: groupObserverIdentity,
      observers: createPageObservers(client, groupOptions),
    };
    setStoredPageObservers(currentPageObservers);
  }
  const pageObservers = currentPageObservers.observers;
  useEffect(() => {
    pageObservers.forEach((observer, index) => {
      const options = groupOptions[index];
      if (options) observer.setOptions(options);
    });
  }, [groupOptions, pageObservers]);
  const pageResultsRef = useRef<
    | {
        observers: typeof pageObservers;
        results: ReturnType<(typeof pageObservers)[number]["getCurrentResult"]>[];
      }
    | undefined
  >(undefined);
  const getPageResults = useCallback(() => {
    const results = pageObservers.map((observer) => observer.getCurrentResult());
    const previous = pageResultsRef.current;
    if (
      previous?.observers === pageObservers &&
      previous.results.length === results.length &&
      previous.results.every((result, index) => result === results[index])
    )
      return previous.results;
    pageResultsRef.current = { observers: pageObservers, results };
    return results;
  }, [pageObservers]);
  const subscribePageResults = useCallback(
    (onStoreChange: () => void) => {
      const unsubscribers = pageObservers.map((observer) => observer.subscribe(onStoreChange));
      return () => unsubscribers.forEach((unsubscribe) => unsubscribe());
    },
    [pageObservers]
  );
  const pageQueries = useSyncExternalStore(subscribePageResults, getPageResults, getPageResults);
  const projection = useMemo(() => {
    const issuesById: Record<string, TIssue> = {};
    const groupedIssueIds: IssueListGroupedIds = {};
    const paginationByGroup: Record<string, TPaginationData> = {};
    pageQueries.forEach((query, index) => {
      const group = groups[index];
      if (!group || !query.data) return;
      const rows = query.data.pages.flatMap((page) => processIssueResponse(page).issueList);
      rows.forEach((issue) => {
        issuesById[issue.id] = issue;
      });
      const ids = rows.map((issue) => issue.id);
      if (group.subGroupId && group.groupId) {
        const currentGroup = groupedIssueIds[group.groupId];
        const subGroups = Array.isArray(currentGroup) ? {} : (currentGroup ?? {});
        groupedIssueIds[group.groupId] = { ...subGroups, [group.subGroupId]: ids };
      } else if (group.groupId) {
        groupedIssueIds[group.groupId] = ids;
      } else {
        groupedIssueIds[ALL_ISSUES] = ids;
      }
      const lastPage = query.data.pages.at(-1);
      if (lastPage)
        paginationByGroup[paginationGroupKey(group)] = {
          prevCursor: lastPage.prev_cursor,
          nextCursor: lastPage.next_cursor,
          nextPageResults: lastPage.next_page_results,
        };
    });
    const groupedIssueCount: TGroupedIssueCount = readData
      ? processIssueResponse(readData.response).groupedIssueCount
      : {};
    return { issuesById, groupedIssueIds, groupedIssueCount, paginationByGroup };
  }, [groups, pageQueries, readData]);
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      const normalizedGroupId = groupId === ALL_ISSUES ? undefined : groupId;
      const queries =
        normalizedGroupId === undefined && subGroupId === undefined
          ? pageQueries.filter((query) => query.hasNextPage && !query.isFetchingNextPage)
          : [
              pageQueries[
                groups.findIndex((group) => group.groupId === normalizedGroupId && group.subGroupId === subGroupId)
              ],
            ];
      await Promise.all(
        queries.flatMap((query) => (query?.hasNextPage && !query.isFetchingNextPage ? [query.fetchNextPage()] : []))
      );
    },
    [groups, pageQueries]
  );
  const getPaginationData = useCallback(
    (groupId?: string, subGroupId?: string) => {
      const normalizedGroupId = groupId === ALL_ISSUES ? undefined : groupId;
      return projection.paginationByGroup[paginationGroupKey({ groupId: normalizedGroupId, subGroupId })];
    },
    [projection.paginationByGroup]
  );
  const getGroupIssueCount = useCallback(
    (groupId?: string, subGroupId?: string, cumulative = false) => {
      if (cumulative && subGroupId) {
        return Object.entries(projection.groupedIssueCount)
          .filter(([key]) => key.endsWith(`_${subGroupId}`))
          .reduce((total, [, count]) => total + count, 0);
      }
      return projection.groupedIssueCount[getGroupKey(groupId, subGroupId)];
    },
    [projection.groupedIssueCount]
  );
  const getIssueLoader = useCallback(
    (groupId?: string, subGroupId?: string): TLoader => {
      const normalizedGroupId = groupId === ALL_ISSUES ? undefined : groupId;
      const index = groups.findIndex((group) => group.groupId === normalizedGroupId && group.subGroupId === subGroupId);
      return index >= 0 && pageQueries[index]?.isFetchingNextPage ? "pagination" : undefined;
    },
    [groups, pageQueries]
  );
  return {
    ...projection,
    response: readData?.response,
    isPending: list.isPending,
    isFetching: list.isFetching,
    isError: list.isError,
    error: list.error,
    retry: list.refetch,
    fetchNextIssues,
    getPaginationData,
    getGroupIssueCount,
    getIssueLoader,
    isFetchingNextPage: pageQueries.some((query) => query.isFetchingNextPage),
    isPageError: pageQueries.some((query) => query.isError),
  };
}
