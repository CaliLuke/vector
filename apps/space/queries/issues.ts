/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { infiniteQueryOptions, queryOptions, skipToken } from "@tanstack/react-query";
import { SitesIssueService } from "@plane/services";
import type { IPublicIssue, TPublicIssuesResponse } from "@plane/types";

const service = new SitesIssueService();
export type PublicIssueFilters = { state: string[]; priority: string[]; labels: string[] };
export type IssuePage = { issues: IPublicIssue[]; count: number; nextCursor: string | undefined };
const pageSize = 50;
const filterParams = (filters: PublicIssueFilters) =>
  Object.fromEntries(
    Object.entries(filters)
      .filter(([, values]) => values.length)
      .map(([key, values]) => [key, values.join(",")])
  );
export function publicIssuesOptions(anchor: string, filters: PublicIssueFilters) {
  return queryOptions({
    queryKey: ["public", anchor, "issues", "grouped", filters],
    queryFn: ({ signal }) =>
      service.list(
        anchor,
        {
          ...filterParams(filters),
          group_by: "state_id",
          per_page: pageSize,
          cursor: `${pageSize}:0:0`,
        },
        signal
      ),
    staleTime: 30_000,
  });
}
export function firstStatePage(response: TPublicIssuesResponse, stateId: string): IssuePage {
  if (Array.isArray(response.results)) throw new Error("Expected issues grouped by state");
  const group = response.results[stateId];
  if (!group) return { issues: [], count: 0, nextCursor: undefined };
  if (!Array.isArray(group.results)) throw new Error("Unexpected nested issue grouping");
  return {
    issues: group.results,
    count: group.total_results,
    nextCursor: group.results.length < group.total_results ? `${pageSize}:1:0` : undefined,
  };
}
export function stateIssuesOptions(anchor: string, filters: PublicIssueFilters, stateId: string) {
  return infiniteQueryOptions({
    queryKey: ["public", anchor, "issues", "state", filters, stateId],
    initialPageParam: `${pageSize}:0:0`,
    queryFn: async ({ pageParam, signal }): Promise<IssuePage> => {
      const response = await service.list(
        anchor,
        {
          ...filterParams(filters),
          state: stateId,
          per_page: pageSize,
          cursor: pageParam,
        },
        signal
      );
      if (!Array.isArray(response.results)) throw new Error("Expected a page of issues");
      return {
        issues: response.results,
        count: response.total_count,
        nextCursor: response.next_page_results ? response.next_cursor : undefined,
      };
    },
    getNextPageParam: (page) => page.nextCursor,
    staleTime: 30_000,
  });
}
export function issueOptions(anchor: string, issueId: string | undefined) {
  return queryOptions({
    queryKey: ["public", anchor, "issue", issueId],
    queryFn: issueId ? ({ signal }) => service.retrieve(anchor, issueId, signal) : skipToken,
    staleTime: 30_000,
  });
}
