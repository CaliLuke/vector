/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { IIssueFiltersResponse } from "@plane/types";
import { IssueFiltersService } from "@/services/issue_filter.service";
import { isAuthorizationError } from "./authorization-errors";
import { captureParentAccess, parentScopeReady, parentScopeKey } from "./cycle-module-scope";
import type { ProjectCatalogueScope } from "./catalogue-scope";
import type { ParentDomain } from "./cycle-modules";
export type ParentIssuePreferencesPatch = Partial<
  Pick<IIssueFiltersResponse, "rich_filters" | "display_filters" | "display_properties">
>;
export const parentIssuePreferenceKey = (
  domain: ParentDomain,
  scope: ProjectCatalogueScope,
  parentId: string | undefined
) => ["parent-issue-preferences", domain, ...parentScopeKey(scope), parentId] as const;
const service = new IssueFiltersService();
export function parentIssuePreferenceOptions(
  domain: ParentDomain,
  scope: ProjectCatalogueScope,
  parentId: string | undefined
) {
  const queryKey = parentIssuePreferenceKey(domain, scope, parentId);
  return queryOptions({
    queryKey,
    retry: false,
    queryFn:
      parentScopeReady(scope) && parentId
        ? async ({ client, signal }) => {
            if (!scope.workspaceSlug || !scope.projectId || !parentId)
              throw new Error("An issue list parent is required.");
            const access = captureParentAccess(client, scope),
              query = client.getQueryCache().find({ queryKey, exact: true });
            const check = () => {
              if (
                signal.aborted ||
                !access.isVerified() ||
                !query ||
                client.getQueryCache().find({ queryKey, exact: true }) !== query
              )
                throw new Error("Issue preference access changed.");
            };
            check();
            const response = await (domain === "cycle"
              ? service.fetchCycleIssueFilters(scope.workspaceSlug, scope.projectId, parentId, signal)
              : service.fetchModuleIssueFilters(scope.workspaceSlug, scope.projectId, parentId, signal));
            check();
            return response;
          }
        : skipToken,
  });
}
export function patchParentIssuePreferenceOptions(
  client: QueryClient,
  domain: ParentDomain,
  scope: ProjectCatalogueScope,
  parentId: string | undefined
) {
  const key = parentIssuePreferenceKey(domain, scope, parentId);
  return mutationOptions({
    mutationKey: [...key, "write"],
    scope: {
      id: JSON.stringify([
        domain,
        scope.accountId,
        scope.sessionGeneration,
        scope.workspaceSlug,
        scope.projectId,
        parentId,
        "preferences",
      ]),
    },
    retry: false,
    mutationFn: async (patch: ParentIssuePreferencesPatch) => {
      const access = captureParentAccess(client, scope),
        query = client.getQueryCache().find({ queryKey: key, exact: true });
      if (
        !scope.workspaceSlug ||
        !scope.projectId ||
        !parentId ||
        !query ||
        query.state.status !== "success" ||
        !access.isVerified()
      )
        throw new Error("Issue preference access is unavailable.");
      const isCurrent = () =>
        access.isCurrent() &&
        !isAuthorizationError(query.state.error) &&
        client.getQueryCache().find({ queryKey: key, exact: true }) === query;
      const response = await (domain === "cycle"
        ? service.patchCycleIssueFilters(scope.workspaceSlug, scope.projectId, parentId, patch)
        : service.patchModuleIssueFilters(scope.workspaceSlug, scope.projectId, parentId, patch));
      if (isCurrent()) {
        await client.cancelQueries({ queryKey: key, exact: true });
        if (isCurrent()) client.setQueryData(key, response);
      }
      return { preferences: response, captured: { isCurrent } };
    },
  });
}
