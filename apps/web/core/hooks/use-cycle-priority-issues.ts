/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useMemo } from "react";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import type { TIssueOrderByOptions } from "@plane/types";
import { useParentCatalogueScope } from "./use-parent-catalogue-scope";
import { cyclePriorityIssueOptions } from "@/queries/cycle-priority-issues";
import { currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import { processIssueResponse } from "@/helpers/issue-list/response";
export function useCyclePriorityIssues(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  cycleId: string | undefined,
  orderBy?: TIssueOrderByOptions
) {
  const access = useParentCatalogueScope(workspaceSlug, projectId),
    client = useQueryClient();
  const query = useInfiniteQuery(cyclePriorityIssueOptions(access.projectScope, cycleId, orderBy));
  const issues = useMemo(
    () =>
      access.isReady && query.isSuccess
        ? Array.from(
            new Map(
              query.data.pages.flatMap((page) => processIssueResponse(page).issueList).map((issue) => [issue.id, issue])
            ).values()
          )
        : undefined,
    [access.isReady, query.isSuccess, query.data]
  );
  return {
    query,
    issues,
    isPending: access.isPending || query.isPending,
    isError: access.isError || query.isError,
    isAccessDenied: access.isAccessDenied,
    retry: async () => {
      if (await access.retry())
        await client.fetchInfiniteQuery({
          ...cyclePriorityIssueOptions(
            currentParentCatalogueScope(client, workspaceSlug, projectId).project,
            cycleId,
            orderBy
          ),
          staleTime: 0,
        });
    },
  };
}
