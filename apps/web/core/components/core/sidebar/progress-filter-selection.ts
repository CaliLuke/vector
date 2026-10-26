/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TWorkItemFilterExpression } from "@plane/types";
import { conjunctiveFilterCondition, readFilterExpression, workItemFiltersAdapter } from "@plane/shared-state";
import type { TSelectedFilterProgressStats } from "./progress-stats/shared";

export function readProgressFilterSelection(
  richFilters: TWorkItemFilterExpression | undefined
): TSelectedFilterProgressStats {
  const { expression } = readFilterExpression(workItemFiltersAdapter, richFilters);
  return {
    assignees: conjunctiveFilterCondition(expression, "assignee_id", "in"),
    labels: conjunctiveFilterCondition(expression, "label_id", "in"),
    stateGroups: conjunctiveFilterCondition(expression, "state_group", "in"),
  };
}
