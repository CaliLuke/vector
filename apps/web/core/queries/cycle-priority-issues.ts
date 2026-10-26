/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { infiniteQueryOptions, skipToken } from "@tanstack/react-query";
import type { TIssueOrderByOptions } from "@plane/types";
import type { ProjectCatalogueScope } from "./catalogue-scope";
import { captureParentAccess, parentScopeReady } from "./cycle-module-scope";
import { parentCatalogueKeys } from "./cycle-modules";
import { CycleService } from "@/services/cycle.service";
import { processIssueResponse } from "@/helpers/issue-list/response";
const service = new CycleService();
export function cyclePriorityIssueOptions(
  scope: ProjectCatalogueScope,
  cycleId: string | undefined,
  orderBy: TIssueOrderByOptions | undefined
) {
  const queryKey = [...parentCatalogueKeys.scope("cycle", scope), "priority-issues", cycleId, orderBy] as const;
  return infiniteQueryOptions({
    queryKey,
    retry: false,
    initialPageParam: "30:0:0",
    queryFn:
      parentScopeReady(scope) && cycleId
        ? async ({ client, signal, pageParam }) => {
            if (!scope.workspaceSlug || !scope.projectId || !cycleId) throw new Error("An active cycle is required.");
            const access = captureParentAccess(client, scope),
              query = client.getQueryCache().find({ queryKey, exact: true });
            const check = () => {
              if (
                !access.isVerified() ||
                signal.aborted ||
                !query ||
                client.getQueryCache().find({ queryKey, exact: true }) !== query
              )
                throw new Error("Active cycle access changed.");
            };
            check();
            const response = await service.getCycleIssues(
              scope.workspaceSlug,
              scope.projectId,
              cycleId,
              { priority: "urgent,high", cursor: pageParam, per_page: 30, ...(orderBy ? { order_by: orderBy } : {}) },
              { signal }
            );
            check();
            if (processIssueResponse(response).issueList.some((issue) => issue.project_id !== scope.projectId))
              throw new Error("An active cycle response contained another project's work items.");
            return response;
          }
        : skipToken,
    getNextPageParam: (page) => (page.next_page_results ? page.next_cursor : undefined),
  });
}
