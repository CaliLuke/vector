import { queryOptions, skipToken } from "@tanstack/react-query";
import type { TProjectRosterScope } from "./member-rosters";
import { getIdentitySessionGeneration } from "./identity";
import { captureItemReadScope } from "./work-item-read-scope";
import { workItemTypesService } from "@/services/work-item-types.service";

/** The full catalogue identifies saved inactive/unimported types as well as choices. */
type DiscoveryScope = TProjectRosterScope & { generation: number };

export function itemTypeIdentityOptions(scope: DiscoveryScope, canLoad = true) {
  return itemDiscoveryOptions(
    scope,
    "item-type-identity",
    (signal) => workItemTypesService.identities(scope.workspaceSlug!, scope.projectId, signal),
    canLoad
  );
}

export function itemPropertyIdentityOptions(scope: DiscoveryScope, canLoad = true) {
  return itemDiscoveryOptions(
    scope,
    "item-property-identity",
    (signal) => workItemTypesService.propertyIdentities(scope.workspaceSlug!, scope.projectId, signal),
    canLoad
  );
}

function itemDiscoveryOptions<T>(
  scope: DiscoveryScope,
  category: string,
  load: (signal: AbortSignal) => Promise<T>,
  canLoad: boolean
) {
  const queryKey = [
    "workspace",
    scope.workspaceSlug,
    category,
    scope.accountId,
    scope.generation,
    scope.projectId,
    scope.membershipRevision,
    scope.membershipRole,
    scope.projectRole,
  ] as const;
  const enabled = Boolean(
    canLoad &&
    scope.accountId &&
    scope.workspaceSlug &&
    scope.membershipRevision &&
    scope.membershipRole !== undefined &&
    (!scope.projectId || scope.projectRole !== undefined)
  );
  return queryOptions({
    queryKey,
    queryFn: enabled
      ? async ({ client, signal }) => {
          const authorized = captureItemReadScope(client, scope, queryKey);
          const current = () => authorized() && getIdentitySessionGeneration(client) === scope.generation;
          if (!current()) throw new Error("Work item access changed.");
          const catalog = await load(signal);
          if (!current()) throw new Error("Work item access changed.");
          return catalog;
        }
      : skipToken,
    retry: false,
    staleTime: 30_000,
    refetchOnMount: true,
  });
}
