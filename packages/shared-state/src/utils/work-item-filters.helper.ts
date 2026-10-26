/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  TBuildFilterExpressionParams,
  TFilterConditionForBuild,
  TFilterValue,
  TWorkItemFilterExpression,
  TWorkItemFilterProperty,
} from "@plane/types";
import { addSidebarFilterCondition } from "../store/rich-filters/filter-helpers";
import { readFilterExpression, writeFilterExpression } from "../store/rich-filters/conversion";
import { workItemFiltersAdapter } from "../store/work-item-filters/adapter";
import { buildTempFilterExpressionFromConditions } from "./rich-filter.helper";

export type TWorkItemFilterCondition = TFilterConditionForBuild<TWorkItemFilterProperty, TFilterValue>;

export function updateWorkItemFilterFromSidebar(
  source: TWorkItemFilterExpression | undefined,
  condition: TWorkItemFilterCondition
) {
  const current = readFilterExpression(workItemFiltersAdapter, source);
  if (current.error) return { ok: false as const, error: current.error };
  const expression = addSidebarFilterCondition(current.expression, condition);
  return writeFilterExpression(workItemFiltersAdapter, expression, current.committedConditionIds, current.error);
}

export const buildWorkItemFilterExpressionFromConditions = (
  params: Omit<
    TBuildFilterExpressionParams<TWorkItemFilterProperty, TFilterValue, TWorkItemFilterExpression>,
    "adapter"
  >
): TWorkItemFilterExpression | undefined => {
  const expression = buildTempFilterExpressionFromConditions({
    ...params,
    adapter: workItemFiltersAdapter,
  });
  if (!expression) console.error("Failed to build work item filter expression from conditions");
  return expression;
};
