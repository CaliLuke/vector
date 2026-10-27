import { EIssueGroupByToServerOptions } from "@plane/constants";
import type { TIssueGroupByOptions } from "@plane/types";

export function isCustomGrouping(axis: unknown): axis is `customproperty_${string}` {
  return typeof axis === "string" && axis.startsWith("customproperty_");
}

export function groupingServerAxis(axis: TIssueGroupByOptions | undefined) {
  if (!axis) return undefined;
  return isCustomGrouping(axis) || isParentTypeGrouping(axis) ? axis : EIssueGroupByToServerOptions[axis];
}

export function isDiscoveryGrouping(
  axis: unknown
): axis is "type_id" | `customproperty_${string}` | `parent_type_${string}` {
  return axis === "type_id" || isCustomGrouping(axis) || isParentTypeGrouping(axis);
}

export function isParentTypeGrouping(axis: unknown): axis is `parent_type_${string}` {
  return typeof axis === "string" && axis.startsWith("parent_type_");
}

export function showEmptyDiscoveryGroups(axis: unknown, showEmpty: boolean | undefined = true) {
  return isDiscoveryGrouping(axis) && showEmpty;
}
