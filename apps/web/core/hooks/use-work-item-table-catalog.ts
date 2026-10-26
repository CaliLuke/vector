import { useWorkItemDiscoveryScope } from "./use-work-item-discovery-scope";
import { useWorkItemPropertyCatalog } from "./use-work-item-property-catalog";

/** Only list owners whose authenticated readers provide property projections. */
export function useWorkItemTableCatalog(enabled = true) {
  const scope = useWorkItemDiscoveryScope(enabled);
  const catalog = useWorkItemPropertyCatalog(scope.workspaceSlug, scope.projectId, scope.canLoad);
  return {
    ...catalog,
    supported: scope.supported,
    fields: catalog.identities?.enabled ? catalog.identities.fields : [],
  };
}
