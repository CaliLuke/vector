/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { isDiscoveryGrouping } from "@/helpers/issue-list/grouping-axis";
import type { TIssueGroupByOptions } from "@plane/types";

/** Type groups require the full form's type validation before creating a work item. */
export function canQuickAddToGroup(
  enabled: boolean | undefined,
  groupBy: TIssueGroupByOptions | null | undefined,
  subGroupBy?: TIssueGroupByOptions | null
) {
  return Boolean(enabled && !isDiscoveryGrouping(groupBy) && !isDiscoveryGrouping(subGroupBy));
}

/** Apply the outer group first, then let the subgroup override the same field. */
export function quickAddGroupPayload(
  defaultStateId: string | undefined,
  groupBy: string | null | undefined,
  groupValue: string,
  subGroupBy: string | null | undefined,
  subGroupValue: string
) {
  if (!groupBy) return {};
  return {
    state_id: defaultStateId,
    ...groupValuePayload(groupBy, groupValue),
    ...groupValuePayload(subGroupBy, subGroupValue),
  };
}

function groupValuePayload(key: string | null | undefined, value: string): Record<string, string | string[]> {
  if (!key || key === "created_by") return {};
  switch (key) {
    case "state":
      return { state_id: value };
    case "cycle":
      return { cycle_id: value };
    case "module":
      return { module_ids: [value] };
    case "labels":
      if (value !== "None") return { label_ids: [value] };
      break;
    case "assignees":
      if (value !== "None") return { assignee_ids: [value] };
      break;
  }
  return { [key]: value };
}
