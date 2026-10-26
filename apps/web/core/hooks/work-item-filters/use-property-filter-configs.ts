import { useMemo } from "react";
import { useWorkItemPropertyCatalog } from "@/hooks/use-work-item-property-catalog";
import { propertyFilterConfig } from "./property-filter-config";

export function usePropertyFilterConfigs(workspaceSlug: string, projectId: string | undefined, allowed: boolean) {
  const { identities, isError, retry } = useWorkItemPropertyCatalog(workspaceSlug, projectId, allowed);
  const configs = useMemo(
    () =>
      allowed && identities?.enabled
        ? identities.fields.filter((field) => field.operators.length > 0).map(propertyFilterConfig)
        : [],
    [allowed, identities]
  );
  return { configs, isError, retry };
}
