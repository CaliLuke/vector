import { useWorkItemDiscoveryScope } from "./use-work-item-discovery-scope";
import { useWorkItemTypeCatalog } from "./use-work-item-type-catalog";

export function useWorkItemGroupingCatalog(enabled = true, location?: Parameters<typeof useWorkItemDiscoveryScope>[1]) {
  const scope = useWorkItemDiscoveryScope(enabled, location);
  const catalog = useWorkItemTypeCatalog(scope.workspaceSlug, scope.projectId, scope.canLoad);
  const denied =
    !catalog.permissions.isPending &&
    (catalog.permissions.workspaceRole === undefined ||
      (!!scope.projectId && catalog.permissions.projectRole === undefined));
  return { ...catalog, supported: scope.supported, denied };
}
