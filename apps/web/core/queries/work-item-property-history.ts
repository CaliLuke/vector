import { queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { TProjectRosterScope } from "./member-rosters";
import { captureItemReadScope } from "./work-item-read-scope";
import { readVisibleItem } from "./work-item-access";
import { workItemPropertyHistoryService } from "@/services/work-item-property-history.service";
import type { PropertyHistoryTarget } from "@/services/work-item-property-history.service";

export type PropertyHistoryScope = TProjectRosterScope & PropertyHistoryTarget & { generation: number };

function historyQuery<T>(
  scope: PropertyHistoryScope,
  selection: readonly unknown[],
  read: (signal: AbortSignal) => Promise<T>,
  enabled = true
) {
  const queryKey = [
    "workspace",
    scope.workspaceSlug,
    "property-history",
    scope.accountId,
    scope.generation,
    scope.projectId,
    scope.issueId,
    scope.membershipRole,
    scope.membershipRevision,
    scope.projectRole,
    ...selection,
  ] as const;
  return queryOptions({
    queryKey,
    queryFn:
      enabled && scope.accountId && scope.membershipRevision && scope.projectRole !== undefined
        ? async ({ client, signal }) => {
            const current = captureItemReadScope(client, scope, queryKey);
            return readVisibleItem(client, scope, current, () => read(signal));
          }
        : skipToken,
    retry: false,
    staleTime: 0,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
  });
}
export function propertyHistoryCatalogOptions(scope: PropertyHistoryScope) {
  return historyQuery(scope, ["catalog"], (signal) => workItemPropertyHistoryService.catalog(scope, signal));
}
export function propertyVersionPageOptions(scope: PropertyHistoryScope, propertyId: string, before: string | null) {
  return historyQuery(
    scope,
    ["versions", propertyId, before],
    (signal) => workItemPropertyHistoryService.versions(scope, propertyId, before, signal),
    !!propertyId
  );
}
export function propertyVersionDetailOptions(scope: PropertyHistoryScope, propertyId: string, versionId: string) {
  return historyQuery(
    scope,
    ["detail", propertyId, versionId],
    (signal) => workItemPropertyHistoryService.detail(scope, propertyId, versionId, signal),
    !!propertyId && !!versionId
  );
}
export function refreshPropertyHistory(
  client: QueryClient,
  scope: PropertyHistoryTarget,
  options?: { throwOnError?: boolean }
) {
  return client.invalidateQueries(
    {
      predicate: ({ queryKey: key }) =>
        key[0] === "workspace" &&
        key[1] === scope.workspaceSlug &&
        key[2] === "property-history" &&
        key[5] === scope.projectId &&
        key[6] === scope.issueId,
    },
    options
  );
}

export { isAuthorizationError as isPropertyHistoryAccessDenied } from "./authorization-errors";
