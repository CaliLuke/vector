import { useQuery } from "@tanstack/react-query";
import { useWorkItemCatalogAccess } from "@/hooks/use-work-item-catalog-access";
import { itemTypeIdentityOptions } from "@/queries/work-item-type-identity";

/** Usage-free identities shared by badges and filters, scoped to current membership. */
export function useWorkItemTypeCatalog(workspaceSlug: string, projectId: string | undefined, canLoad = true) {
  const access = useWorkItemCatalogAccess(workspaceSlug, projectId, canLoad);
  const catalog = useQuery(itemTypeIdentityOptions(access.scope, canLoad));
  return {
    catalog,
    permissions: access.permissions,
    identities: access.allowed && catalog.isSuccess ? catalog.data : undefined,
    retry: () => access.retry(catalog.refetch),
    isError: canLoad && (access.permissionError || catalog.isError),
  };
}
