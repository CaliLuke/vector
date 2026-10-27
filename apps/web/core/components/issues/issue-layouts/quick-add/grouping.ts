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
