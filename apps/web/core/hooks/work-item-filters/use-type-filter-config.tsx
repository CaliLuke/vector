import { useMemo } from "react";
import { Shapes } from "lucide-react";
import type { TFilterConfig, TWorkItemFilterProperty } from "@plane/types";
import { COLLECTION_OPERATOR, EQUALITY_OPERATOR } from "@plane/types";
import { createFilterConfig, getMultiSelectConfig, getSingleSelectConfig } from "@plane/utils/rich-filters";
import { TypeIcon } from "@/components/work-item-types/controls";
import { useWorkItemTypeCatalog } from "@/hooks/use-work-item-type-catalog";
import type { WorkItemTypeIdentity } from "@/services/work-item-types.service";

export function useTypeFilterConfig(workspaceSlug: string, projectId: string | undefined, allowed: boolean) {
  const { identities, isError, retry } = useWorkItemTypeCatalog(workspaceSlug, projectId, allowed);
  const config = useMemo((): TFilterConfig<TWorkItemFilterProperty> => {
    const enabled = allowed && identities?.enabled === true;
    const transforms = {
      items: enabled ? identities.types : [],
      getId: (type: WorkItemTypeIdentity) => type.id,
      getLabel: (type: WorkItemTypeIdentity) => type.name,
      getValue: (type: WorkItemTypeIdentity) => type.id,
      getIconData: (type: WorkItemTypeIdentity) => type,
    };
    const options = { isOperatorEnabled: enabled, allowNegative: true };
    const icons = { getOptionIcon: (type: WorkItemTypeIdentity) => <TypeIcon type={type} size="sm" /> };
    return createFilterConfig({
      id: "type_id",
      label: "Work item type",
      icon: Shapes,
      isEnabled: enabled,
      supportedOperatorConfigsMap: new Map([
        [EQUALITY_OPERATOR.EXACT, getSingleSelectConfig(transforms, options, icons)],
        [
          COLLECTION_OPERATOR.IN,
          getMultiSelectConfig(transforms, { ...options, singleValueOperator: COLLECTION_OPERATOR.IN }, icons),
        ],
      ]),
    });
  }, [allowed, identities]);
  return { config, isError, retry };
}
