import type { TIssueGroupByOptions, TIssueLayouts } from "@plane/types";
import { useWorkItemDiscoveryScope } from "./use-work-item-discovery-scope";
import { useWorkItemTypeCatalog } from "./use-work-item-type-catalog";
import { useWorkItemPropertyCatalog } from "./use-work-item-property-catalog";
import { isCustomGrouping } from "@/helpers/issue-list/grouping-axis";
import type { IIssueDisplayFilterOptions } from "@plane/types";

export function usePropertyGroupingRevision(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  filters: IIssueDisplayFilterOptions | undefined
) {
  const enabled = isCustomGrouping(filters?.group_by) || isCustomGrouping(filters?.sub_group_by);
  const catalog = useWorkItemPropertyCatalog(workspaceSlug ?? "", projectId, enabled);
  return catalog.identities?.revision;
}

export function useWorkItemGroupingCatalog(enabled = true, location?: Parameters<typeof useWorkItemDiscoveryScope>[1]) {
  const scope = useWorkItemDiscoveryScope(enabled, location);
  const catalog = useWorkItemTypeCatalog(scope.workspaceSlug, scope.projectId, scope.canLoad);
  const denied =
    !catalog.permissions.isPending &&
    (catalog.permissions.workspaceRole === undefined ||
      (!!scope.projectId && catalog.permissions.projectRole === undefined));
  return { ...catalog, supported: scope.supported, denied };
}

/** Keep unsupported routes, disabled workspaces and epic views out of the picker. */
export function useTypeGroupingOptions(layout: TIssueLayouts | undefined, isEpic: boolean) {
  const eligible = !isEpic && (layout === "list" || layout === "kanban");
  const { identities, supported } = useWorkItemGroupingCatalog(eligible);
  const available = eligible && supported && identities?.enabled === true;
  return (options: TIssueGroupByOptions[]): TIssueGroupByOptions[] =>
    available ? [...new Set([...options, "type_id" as const])] : options.filter((option) => option !== "type_id");
}
