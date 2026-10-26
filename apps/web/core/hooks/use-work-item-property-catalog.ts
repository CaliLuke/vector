import { useQuery } from "@tanstack/react-query";
import { useWorkItemCatalogAccess } from "@/hooks/use-work-item-catalog-access";
import { itemPropertyIdentityOptions } from "@/queries/work-item-type-identity";

export function useWorkItemPropertyCatalog(workspaceSlug: string, projectId: string | undefined, canLoad = true) {
  const access = useWorkItemCatalogAccess(workspaceSlug, projectId, canLoad);
  const catalog = useQuery(itemPropertyIdentityOptions(access.scope, canLoad));
  return {
    catalog,
    identities: access.allowed && catalog.isSuccess ? catalog.data : undefined,
    retry: () => access.retry(catalog.refetch),
    isError: canLoad && (access.permissionError || catalog.isError),
  };
}
