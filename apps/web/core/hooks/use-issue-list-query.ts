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
import type { QueryClient, QueryKey } from "@tanstack/react-query";
import { isAxiosError } from "axios";
import { ALL_ISSUES } from "@plane/constants";
import { processIssueResponse } from "@/helpers/issue-list/response";
import { getGroupKey } from "@/helpers/issue-list/grouping";
import type { IssueListGroupedIds, IssueListRead, IssueListReadData } from "@/queries/issue-lists";
import {
  issueListGroupOptions,
  issueListGroupPathKey,
  issueListGroups,
  issueListOptions,
  issueListPresentationKey,
} from "@/queries/issue-lists";
import { isAuthorizationError } from "@/queries/authorization-errors";
import { captureParentAccess } from "@/queries/cycle-module-scope";

function isRetryableReadError(error: unknown) {
  const response = isAxiosError(error) ? error.response : error;
  if (response && typeof response === "object" && "status" in response && typeof response.status === "number") {
    return response.status === 429 || (response.status >= 500 && response.status < 600);
  }
  return isAxiosError(error) && ["ERR_NETWORK", "ECONNABORTED", "ETIMEDOUT"].includes(error.code ?? "");
}

function createPageObservers(client: QueryClient, options: ReturnType<typeof issueListGroupOptions>[]) {
  return options.map((groupOptions) => new InfiniteQueryObserver(client, groupOptions));
}

function useIssueListPages(client: QueryClient, groupOptions: ReturnType<typeof issueListGroupOptions>[]) {
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
  return useSyncExternalStore(subscribePageResults, getPageResults, getPageResults);
}

type IssueListProjection = {
  issuesById: Record<string, TIssue>;
  groupedIssueIds: IssueListGroupedIds;
  groupedIssueCount: TGroupedIssueCount;
  paginationByGroup: Record<string, TPaginationData>;
};

function useSettledIssueList(input: {
  client: QueryClient;
  queryKey: QueryKey;
  viewKey: string | undefined;
  readData: IssueListReadData | undefined;
  projection: IssueListProjection;
  access: ReturnType<typeof captureParentAccess> | undefined;
  pending: boolean;
}) {
  const { client, queryKey, viewKey, readData, projection, access, pending } = input;
  const [previous, setPrevious] = useState<{
    viewKey: string;
    data: IssueListReadData;
    projection: IssueListProjection;
    isCurrent: () => boolean;
  }>();
  if (viewKey && readData) {
    if (previous?.viewKey !== viewKey || previous.data !== readData || previous.projection !== projection) {
      const query = client.getQueryCache().find({ queryKey, exact: true });
      const hasRows = Object.keys(projection.issuesById).length > 0;
      setPrevious({
        viewKey,
        data: readData,
        projection,
        isCurrent: () =>
          hasRows &&
          access?.isVerified() === true &&
          query !== undefined &&
          (query.state.status === "success" || isRetryableReadError(query.state.error)) &&
          query.state.data === readData &&
          client.getQueryCache().find({ queryKey, exact: true }) === query,
      });
    }
  } else if (previous) {
    if (!pending || previous.viewKey !== viewKey) setPrevious(undefined);
  }
  if (!pending || previous?.viewKey !== viewKey) return undefined;
  return previous?.isCurrent() ? previous : undefined;
}

function useAuthorizedIssueList(client: QueryClient, read: IssueListRead | undefined) {
  const options = read
    ? issueListOptions(read)
    : queryOptions<IssueListReadData>({ queryKey: ["issue-list", "unavailable"], queryFn: skipToken });
  const list = useQuery(options);
  const access = read ? captureParentAccess(client, read.scope) : undefined;
  const canPresent = read && access?.isVerified() && (read.isPresentationCurrent ?? read.request.isReadCurrent)();
  const isRefreshError = Boolean(canPresent && list.isRefetchError && isRetryableReadError(list.error));
  const readData = canPresent && (list.isSuccess || isRefreshError) ? list.data : undefined;
  const viewKey = canPresent ? issueListPresentationKey(read) : undefined;
  return { list, access, readData, viewKey, isRefreshError, queryKey: options.queryKey };
}

export function useIssueListQuery(read: IssueListRead | undefined) {
  const client = useQueryClient();
  const { list, access, readData, viewKey, isRefreshError, queryKey } = useAuthorizedIssueList(client, read);
  const groups = useMemo(() => (readData ? issueListGroups(readData.response) : []), [readData]);
  const groupOptions = useMemo(
    () => (read && readData ? groups.map((group) => issueListGroupOptions(read, readData, group)) : []),
    [groups, read, readData]
  );
  const pageQueries = useIssueListPages(client, groupOptions);
  const pageAccessError = pageQueries.find((query) => isAuthorizationError(query.error))?.error;
  const currentProjection = useMemo(() => {
    const issuesById: Record<string, TIssue> = {};
    const groupedIssueIds: IssueListGroupedIds = {};
    const paginationByGroup: Record<string, TPaginationData> = {};
    if (!pageAccessError)
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
          paginationByGroup[issueListGroupPathKey(group)] = {
            prevCursor: lastPage.prev_cursor,
            nextCursor: lastPage.next_cursor,
            nextPageResults: query.hasNextPage && !query.isFetchNextPageError,
          };
      });
    const groupedIssueCount: TGroupedIssueCount =
      readData && !pageAccessError ? processIssueResponse(readData.response).groupedIssueCount : {};
    return { issuesById, groupedIssueIds, groupedIssueCount, paginationByGroup };
  }, [groups, pageAccessError, pageQueries, readData]);
  const retained = useSettledIssueList({
    client,
    queryKey,
    viewKey,
    readData: pageAccessError ? undefined : readData,
    projection: currentProjection,
    access,
    pending: list.isPending,
  });
  const projection = retained?.projection ?? currentProjection;
  const fetchNextIssues = useCallback(
    async (groupId?: string, subGroupId?: string) => {
      if (list.isFetching || pageAccessError) return;
      const normalizedGroupId = groupId === ALL_ISSUES ? undefined : groupId;
      const queries =
        normalizedGroupId === undefined && subGroupId === undefined
          ? pageQueries
          : [
              pageQueries[
                groups.findIndex((group) => group.groupId === normalizedGroupId && group.subGroupId === subGroupId)
              ],
            ];
      await Promise.all(
        queries.flatMap((query) =>
          query?.hasNextPage && !query.isFetchingNextPage && !query.isFetchNextPageError
            ? [query.fetchNextPage({ cancelRefetch: false })]
            : []
        )
      );
    },
    [groups, list.isFetching, pageAccessError, pageQueries]
  );
  const retryFailedPages = useCallback(async () => {
    if (list.isFetching || pageAccessError) return;
    await Promise.all(
      pageQueries
        .filter((query) => query.isFetchNextPageError && !query.isFetchingNextPage)
        .map((query) => query.fetchNextPage({ cancelRefetch: false }))
    );
  }, [list.isFetching, pageAccessError, pageQueries]);
  const getPaginationData = useCallback(
    (groupId?: string, subGroupId?: string) => {
      const normalizedGroupId = groupId === ALL_ISSUES ? undefined : groupId;
      return projection.paginationByGroup[issueListGroupPathKey({ groupId: normalizedGroupId, subGroupId })];
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
    response: pageAccessError ? undefined : (readData?.response ?? retained?.data.response),
    isPending: !readData && !retained && !list.isError,
    isUpdatingResults: Boolean(retained),
    isFetching: list.isFetching,
    isError: Boolean(pageAccessError) || (list.isError && !isRefreshError),
    error: pageAccessError ?? list.error,
    isRefreshError,
    retry: list.refetch,
    retryRefresh: list.refetch,
    fetchNextIssues,
    getPaginationData,
    getGroupIssueCount,
    getIssueLoader,
    isFetchingNextPage: pageQueries.some((query) => query.isFetchingNextPage),
    isPageError: !pageAccessError && pageQueries.some((query) => query.isFetchNextPageError),
    retryFailedPages,
  };
}
